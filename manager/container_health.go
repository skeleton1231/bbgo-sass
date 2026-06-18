package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Container crashloop detection threshold. With --restart=unless-stopped, a
// bbgo process that exits non-zero (strategy validation error, panic, missing
// market data, bad credentials — any internal failure) is restarted by Docker
// immediately. After this many restarts within the docker daemon's restart
// window we classify the container as crashlooping rather than briefly running
// between crash cycles.
const crashLoopRestartThreshold = 3

// heartbeatStaleThreshold bounds how old the bbgo-written heartbeat file may
// be before we treat a "running" container as silently wedged. bbgo refreshes
// every 60s; 5 minutes absorbs three missed ticks plus docker's restart lag.
const heartbeatStaleThreshold = 5 * time.Minute

// ContainerHealth is the manager-side view of an instance container's
// liveness. Docker's `.State.Running` flag is necessary but not sufficient —
// `--restart=unless-stopped` makes Docker report `running=true` during the
// brief window between crash cycles, so a container whose bbgo process exits
// non-zero on startup can look alive from outside. This struct captures the
// richer signal needed to distinguish "actually serving traffic" from
// "phantom-active".
type ContainerHealth struct {
	Status       string // running | starting | error | stopped
	Running      bool   // raw docker .State.Running
	Restarting   bool   // raw docker .State.Status == "restarting"
	RestartCount int    // docker .RestartCount
	ExitCode     int    // docker .State.ExitCode (last exit)
	Reason       string // human-readable explanation
}

// HealthStatus values. StatusError covers both crashloop and unreachable-
// after-start-window — callers can read Health.Reason to distinguish.
const (
	HealthStatusRunning  = "running"
	HealthStatusStarting = "starting"
	HealthStatusError    = "error"
	HealthStatusStopped  = "stopped"
)

// CheckInstanceHealth inspects the container's docker state and classifies it.
// The classification is conservative: only return running when the container
// is genuinely running AND not in a visible crashloop AND its heartbeat file
// (if present) is fresh. Caller decides how to weigh the initial-start grace
// window.
//
// Single `docker inspect` call with a composed template — avoids the N+1 of
// running four separate inspect calls per container.
func (cm *ContainerManager) CheckInstanceHealth(userID, mode, instanceID string) (ContainerHealth, error) {
	name := cm.InstanceContainerName(userID, mode, instanceID)
	h, err := cm.checkHealthByName(name)
	if err != nil {
		return h, err
	}
	// Heartbeat check only applies when docker says the container is healthy
	// — a crashed or stopped container has bigger problems than a stale file.
	if h.Status != HealthStatusRunning {
		return h, nil
	}
	if hb := cm.checkHeartbeat(userID, mode, instanceID); hb != "" {
		h.Status = HealthStatusError
		h.Reason = hb
	}
	return h, nil
}

// checkHeartbeat returns a non-empty reason string when the heartbeat file
// indicates the bbgo process inside the container is silently wedged.
// Returns "" when fresh, missing, or unreadable (we don't fail health on
// file-access errors — those are manager-side issues, not container issues,
// and an older bbgo image that doesn't write heartbeats shouldn't be marked
// unhealthy just because the file is absent).
func (cm *ContainerManager) checkHeartbeat(userID, mode, instanceID string) string {
	if cm.store == nil {
		return ""
	}
	path := filepath.Join(cm.store.InstanceDir(userID, mode, instanceID), "heartbeat")
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	age := time.Since(info.ModTime())
	if age <= heartbeatStaleThreshold {
		return ""
	}
	return fmt.Sprintf("heartbeat stale (last beat %s ago)", age.Truncate(time.Second))
}

// checkHealthByName is the testable core — accepts a container name directly
// so tests can drive it without constructing a StrategyInstance.
func (cm *ContainerManager) checkHealthByName(name string) (ContainerHealth, error) {
	if cm.checkHealthFn != nil {
		return cm.checkHealthFn(name)
	}
	// One docker inspect, four values via a composed Go-template. Order matters —
	// the parser below reads them positionally.
	tpl := "{{.State.Status}}|{{.State.Running}}|{{.RestartCount}}|{{.State.ExitCode}}"
	out, err := cm.docker("inspect", "-f", tpl, name)
	if err != nil {
		return ContainerHealth{Status: HealthStatusStopped, Reason: fmt.Sprintf("docker inspect: %v", err)}, err
	}
	return parseHealthOutput(strings.TrimSpace(out)), nil
}

func parseHealthOutput(out string) ContainerHealth {
	parts := strings.SplitN(out, "|", 4)
	if len(parts) != 4 {
		return ContainerHealth{Status: HealthStatusStopped, Reason: "unparseable inspect output: " + out}
	}
	statusStr := strings.TrimSpace(parts[0])
	runningStr := strings.TrimSpace(parts[1])
	restartCount, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	exitCode, _ := strconv.Atoi(strings.TrimSpace(parts[3]))

	running := runningStr == "true"
	restarting := statusStr == "restarting"

	h := ContainerHealth{
		Running:      running,
		Restarting:   restarting,
		RestartCount: restartCount,
		ExitCode:     exitCode,
	}

	switch {
	case statusStr == "restarting":
		h.Status = HealthStatusError
		h.Reason = fmt.Sprintf("container is restarting (restart_count=%d, last_exit=%d)", restartCount, exitCode)
	case running && restartCount >= crashLoopRestartThreshold && exitCode != 0:
		// Docker reports running=true during the brief window between crash
		// cycles. Multiple restarts + non-zero exit = bbgo is failing every time
		// it starts, regardless of the momentary running=true.
		h.Status = HealthStatusError
		h.Reason = fmt.Sprintf("crashloop detected (restart_count=%d, last_exit=%d)", restartCount, exitCode)
	case running:
		h.Status = HealthStatusRunning
	case statusStr == "created":
		h.Status = HealthStatusStarting
		h.Reason = "container created but not started"
	case statusStr == "exited" || statusStr == "dead":
		h.Status = HealthStatusStopped
		h.Reason = fmt.Sprintf("container %s (exit_code=%d)", statusStr, exitCode)
	default:
		h.Status = HealthStatusStopped
		h.Reason = "container state: " + statusStr
	}
	return h
}

// CaptureContainerError grabs the last N lines of the container's logs and
// extracts the lines that look like the actual failure message. Used to
// surface *why* a container is crashlooping (e.g. "spread is too small" for
// grid2, or any level=fatal from any other strategy) to the user via the
// `last_error` column on strategy_instances.
//
// Strategy-agnostic by design: we look for logrus level=fatal / level=error
// markers and the bbgo "cannot execute command" logrus-fatal wrapper, not for
// strategy-specific keywords. Whatever crashed the strategy gets captured.
func (cm *ContainerManager) CaptureContainerError(userID, mode, instanceID string) string {
	name := cm.InstanceContainerName(userID, mode, instanceID)
	return cm.captureErrorByName(name)
}

func (cm *ContainerManager) captureErrorByName(name string) string {
	if cm.captureErrorFn != nil {
		return cm.captureErrorFn(name)
	}
	out, err := cm.docker("logs", "--tail", "200", name)
	if err != nil {
		return truncateError(err.Error())
	}
	return extractFatalLog(out)
}

// extractFatalLog pulls the last `level=fatal` / `level=error` line plus a
// short tail from raw docker log output. Returns the trimmed, single-line
// message suitable for storage in strategy_instances.last_error.
func extractFatalLog(logs string) string {
	if logs == "" {
		return ""
	}
	lines := strings.Split(logs, "\n")
	var lastFatal string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// logrus format: `level=fatal msg="..."` or `level=error msg="..."`.
		// bbgo's startup failures always go through logrus.Fatalf which
		// produces level=fatal.
		if strings.Contains(line, "level=fatal") || strings.Contains(line, "level=error") {
			lastFatal = line
		}
	}
	if lastFatal != "" {
		return truncateError(lastFatal)
	}
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return truncateError(line)
		}
	}
	return ""
}

// truncateError caps stored error messages so a runaway stack trace doesn't
// bloat the strategy_instances row. 1 KiB is enough for the logrus line plus
// the immediately preceding context line.
func truncateError(s string) string {
	const maxLen = 1024
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
