package main

import (
	"strings"
	"testing"
)

func TestParseHealthOutput_Running(t *testing.T) {
	h := parseHealthOutput("running|true|0|0")
	if h.Status != HealthStatusRunning {
		t.Errorf("status: got %q want %q", h.Status, HealthStatusRunning)
	}
	if !h.Running {
		t.Error("expected Running=true")
	}
	if h.RestartCount != 0 || h.ExitCode != 0 {
		t.Errorf("restart_count=%d exit_code=%d, want 0/0", h.RestartCount, h.ExitCode)
	}
}

func TestParseHealthOutput_CrashLoop(t *testing.T) {
	// Docker reports running=true in the brief window between crash cycles,
	// but the high RestartCount + non-zero ExitCode reveals the truth.
	h := parseHealthOutput("running|true|5|1")
	if h.Status != HealthStatusError {
		t.Errorf("status: got %q want %q (crashloop)", h.Status, HealthStatusError)
	}
	if h.RestartCount != 5 || h.ExitCode != 1 {
		t.Errorf("restart_count=%d exit_code=%d, want 5/1", h.RestartCount, h.ExitCode)
	}
	if !strings.Contains(h.Reason, "crashloop") {
		t.Errorf("reason %q should mention crashloop", h.Reason)
	}
}

func TestParseHealthOutput_Restarting(t *testing.T) {
	h := parseHealthOutput("restarting|false|3|1")
	if h.Status != HealthStatusError {
		t.Errorf("status: got %q want %q", h.Status, HealthStatusError)
	}
	if !h.Restarting {
		t.Error("expected Restarting=true")
	}
}

func TestParseHealthOutput_RunningZeroExitHighRestart_NotCrashLoop(t *testing.T) {
	// High restart count with exit code 0 — Docker may have been manually
	// restarted multiple times. Not a crashloop.
	h := parseHealthOutput("running|true|5|0")
	if h.Status != HealthStatusRunning {
		t.Errorf("status: got %q want %q (zero exit = not crashloop)", h.Status, HealthStatusRunning)
	}
}

func TestParseHealthOutput_LowRestartCount_NotCrashLoop(t *testing.T) {
	// Below threshold, even with non-zero exit, give the container a chance
	// (might be a transient init issue that resolved itself).
	h := parseHealthOutput("running|true|1|1")
	if h.Status != HealthStatusRunning {
		t.Errorf("status: got %q want %q", h.Status, HealthStatusRunning)
	}
}

func TestParseHealthOutput_Exited(t *testing.T) {
	h := parseHealthOutput("exited|false|0|0")
	if h.Status != HealthStatusStopped {
		t.Errorf("status: got %q want %q", h.Status, HealthStatusStopped)
	}
}

func TestParseHealthOutput_Created(t *testing.T) {
	h := parseHealthOutput("created|false|0|0")
	if h.Status != HealthStatusStarting {
		t.Errorf("status: got %q want %q", h.Status, HealthStatusStarting)
	}
}

func TestParseHealthOutput_Malformed(t *testing.T) {
	h := parseHealthOutput("garbage")
	if h.Status != HealthStatusStopped {
		t.Errorf("status: got %q want %q", h.Status, HealthStatusStopped)
	}
	if !strings.Contains(h.Reason, "unparseable") {
		t.Errorf("reason %q should mention unparseable", h.Reason)
	}
}

func TestCheckInstanceHealth_RoutesViaDocker(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		if args[0] == "inspect" {
			return "running|true|0|0", nil
		}
		return "", nil
	}
	h, err := cm.CheckInstanceHealth("user-1", ModeLive, "grid2-BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != HealthStatusRunning {
		t.Errorf("status: got %q want running", h.Status)
	}
}

func TestCheckInstanceHealth_InspectError(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "", errTestNotFound
	}
	h, err := cm.CheckInstanceHealth("user-1", ModeLive, "grid2-BTCUSDT")
	if err == nil {
		t.Error("expected error from docker inspect")
	}
	if h.Status != HealthStatusStopped {
		t.Errorf("status: got %q want stopped", h.Status)
	}
}

func TestCheckInstanceHealth_TestHook(t *testing.T) {
	cm := testContainerManager(t)
	cm.checkHealthFn = func(name string) (ContainerHealth, error) {
		return ContainerHealth{Status: HealthStatusError, RestartCount: 5, ExitCode: 1, Reason: "crashloop detected"}, nil
	}
	h, _ := cm.CheckInstanceHealth("user-1", ModeLive, "grid2-BTCUSDT")
	if h.Status != HealthStatusError {
		t.Errorf("status: got %q want error", h.Status)
	}
}

func TestExtractFatalLog_PicksFatalLine(t *testing.T) {
	logs := `time="2026-06-17T14:40:09Z" level=info msg="querying market info from binance..." session=binance
time="2026-06-17T14:40:10Z" level=info msg="attaching strategy *grid2.Strategy on binance..."
time="2026-06-17T14:40:10Z" level=fatal msg="cannot execute command" error="found invalid strategy config: spread is too small, please try to reduce your gridNum or increase the price range (upperPrice and lowerPrice): profitSpread 0.001000 0% is too small for lower price, less than the grid fee rate: 0.1507%"`
	got := extractFatalLog(logs)
	if !strings.Contains(got, "level=fatal") {
		t.Errorf("expected level=fatal line, got %q", got)
	}
	if !strings.Contains(got, "spread is too small") {
		t.Errorf("expected error detail in extracted line, got %q", got)
	}
}

func TestExtractFatalLog_FallsBackToLastError(t *testing.T) {
	// No logrus level markers — e.g. panic stack to stdout. Should fall back
	// to the last non-empty line.
	logs := "panic: runtime error: index out of range [1] with length 1\n\ngoroutine 1 [running]:\nmain.main()\n\t/main.go:42"
	got := extractFatalLog(logs)
	if !strings.Contains(got, "main.go:42") {
		t.Errorf("expected last non-empty line, got %q", got)
	}
}

func TestExtractFatalLog_Empty(t *testing.T) {
	if extractFatalLog("") != "" {
		t.Error("expected empty result for empty input")
	}
}

func TestTruncateError_Long(t *testing.T) {
	long := strings.Repeat("a", 2000)
	got := truncateError(long)
	if len(got) > 1100 {
		t.Errorf("truncate produced %d chars, expected <= ~1100", len(got))
	}
	if !strings.Contains(got, "[truncated]") {
		t.Error("expected [truncated] suffix")
	}
}

func TestTruncateError_Short(t *testing.T) {
	short := "short error"
	if got := truncateError(short); got != short {
		t.Errorf("got %q want %q", got, short)
	}
}

func TestCaptureContainerError_UsesDockerLogs(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		if args[0] == "logs" {
			return `time="2026-06-17T14:40:10Z" level=fatal msg="cannot execute command" error="spread is too small"`, nil
		}
		return "", nil
	}
	got := cm.CaptureContainerError("user-1", ModeLive, "grid2-BTCUSDT")
	if !strings.Contains(got, "spread is too small") {
		t.Errorf("expected captured error to contain spread message, got %q", got)
	}
}

func TestIsCrashLooping_True(t *testing.T) {
	cm := testContainerManager(t)
	cm.checkHealthFn = func(name string) (ContainerHealth, error) {
		return ContainerHealth{Status: HealthStatusError}, nil
	}
	inst := &StrategyInstance{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"}
	if !cm.isCrashLooping(inst) {
		t.Error("expected isCrashLooping=true")
	}
}

func TestIsCrashLooping_FalseWhenRunning(t *testing.T) {
	cm := testContainerManager(t)
	cm.checkHealthFn = func(name string) (ContainerHealth, error) {
		return ContainerHealth{Status: HealthStatusRunning}, nil
	}
	inst := &StrategyInstance{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"}
	if cm.isCrashLooping(inst) {
		t.Error("expected isCrashLooping=false for running container")
	}
}

func TestIsCrashLooping_FalseOnInspectError(t *testing.T) {
	cm := testContainerManager(t)
	cm.checkHealthFn = func(name string) (ContainerHealth, error) {
		return ContainerHealth{}, errTestNotFound
	}
	inst := &StrategyInstance{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"}
	if cm.isCrashLooping(inst) {
		t.Error("expected isCrashLooping=false when inspect fails (don't mistake missing for crashloop)")
	}
}
