package main

import (
	"log"
	"strings"
	"sync"
	"time"
)

type HealthCheckResult struct {
	UserID     string
	Mode       string
	InstanceID string
	Alive      bool
	Restarted  bool
	Error      string
}

type RecoveryResult struct {
	UserID string
	Mode   string
	Status string
}

// CheckAndRecover checks all running instance containers in parallel and restarts
// any that have died. Uses a goroutine pool (max 5) for parallel docker inspect.
//
// Crashloop backoff: if CheckInstanceHealth detects the container is in a
// visible crashloop (restarting state, or restart_count >= threshold with
// non-zero exit), we do NOT try to recover it. Recreating a container whose
// bbgo process crashes on startup would just produce another crashloop and
// burn CPU on a docker-level infinite loop. The captured error is already in
// strategy_instances.last_error via startInstanceContainer or the recovery
// capture below; user must fix the underlying config and re-start.
func (cm *ContainerManager) CheckAndRecover(instances []StrategyInstance) []HealthCheckResult {
	results := make([]HealthCheckResult, len(instances))
	var mu sync.Mutex

	for i, inst := range instances {
		idx, inst := i, inst
		if err := cm.pool.Submit(func() {
			name := cm.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
			running, _ := cm.CheckInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID)
			if running && !cm.isCrashLooping(&inst) {
				mu.Lock()
				results[idx] = HealthCheckResult{
					UserID: inst.UserID, Mode: inst.Mode, InstanceID: inst.InstanceID, Alive: true,
				}
				mu.Unlock()
				return
			}

			if running {
				log.Printf("health check: instance container %s is crashlooping (running between cycles), capturing error", name)
				cm.captureErrorIfNeeded(&inst)
				mu.Lock()
				results[idx] = HealthCheckResult{
					UserID: inst.UserID, Mode: inst.Mode, InstanceID: inst.InstanceID,
					Alive: false, Error: "crashloop detected — recovery skipped",
				}
				mu.Unlock()
				return
			}

			log.Printf("health check: instance container %s died, attempting recovery", name)

			if cm.isCrashLooping(&inst) {
				log.Printf("health check: instance container %s is crashlooping, skipping recovery (user must fix config and restart)", name)
				cm.captureErrorIfNeeded(&inst)
				mu.Lock()
				results[idx] = HealthCheckResult{
					UserID: inst.UserID, Mode: inst.Mode, InstanceID: inst.InstanceID,
					Alive: false, Error: "crashloop detected — recovery skipped",
				}
				mu.Unlock()
				return
			}

			if cm.tryRecoverViaDockerStart(&inst) {
				mu.Lock()
				results[idx] = HealthCheckResult{
					UserID: inst.UserID, Mode: inst.Mode, InstanceID: inst.InstanceID,
					Alive: true, Restarted: true,
				}
				mu.Unlock()
				return
			}

			log.Printf("recreating instance container %s", name)
			if err := cm.CreateAndStartInstance(&inst); err != nil {
				mu.Lock()
				results[idx] = HealthCheckResult{
					UserID: inst.UserID, Mode: inst.Mode, InstanceID: inst.InstanceID,
					Alive: false, Error: err.Error(),
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			results[idx] = HealthCheckResult{
				UserID: inst.UserID, Mode: inst.Mode, InstanceID: inst.InstanceID,
				Alive: true, Restarted: true,
			}
			mu.Unlock()
		}); err != nil {
			results[idx] = HealthCheckResult{
				UserID: inst.UserID, Mode: inst.Mode, InstanceID: inst.InstanceID,
				Alive: false, Error: err.Error(),
			}
		}
	}
	cm.pool.Wait()
	return results
}

// isCrashLooping returns true when the container shows visible crashloop
// signals — either Docker reports status=restarting, or the container has
// accumulated >= crashLoopRestartThreshold restarts with a non-zero exit
// code. We treat high restart count + non-zero exit as crashloop even when
// Docker momentarily reports running=true, because the next crash is
// imminent.
func (cm *ContainerManager) isCrashLooping(inst *StrategyInstance) bool {
	health, err := cm.CheckInstanceHealth(inst.UserID, inst.Mode, inst.InstanceID)
	if err != nil {
		return false
	}
	return health.Status == HealthStatusError
}

// captureErrorIfNeeded grabs the container's last log lines and stores them
// on the strategy_instances row. Called from recovery when a crashloop is
// detected so the user can see *why* — without this, the row's last_error
// would stay empty if the initial startInstanceContainer captured the error
// before the crashloop fully developed.
func (cm *ContainerManager) captureErrorIfNeeded(inst *StrategyInstance) {
	if cm.store == nil {
		return
	}
	captured := cm.CaptureContainerError(inst.UserID, inst.Mode, inst.InstanceID)
	if captured == "" {
		return
	}
	cm.store.MarkInstanceError(inst.UserID, inst.Mode, inst.InstanceID, captured)
}

// RecoverUsers discovers all instances for tracked users and recovers their containers.
func (cm *ContainerManager) RecoverUsers(users []UserMode) []RecoveryResult {
	var allInstances []StrategyInstance
	for _, um := range users {
		instances, err := cm.store.ListInstances(um.UserID, um.Mode)
		if err != nil {
			continue
		}
		allInstances = append(allInstances, instances...)
	}

	results := make([]RecoveryResult, len(allInstances))
	var mu sync.Mutex

	for i, inst := range allInstances {
		idx, inst := i, inst
		if err := cm.pool.Submit(func() {
			name := cm.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
			running, _ := cm.CheckInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID)
			if running && !cm.isCrashLooping(&inst) {
				log.Printf("recovered instance container %s (running)", name)
				mu.Lock()
				results[idx] = RecoveryResult{UserID: inst.UserID, Mode: inst.Mode, Status: StatusRunning}
				mu.Unlock()
				return
			}

			if running {
				log.Printf("recover: instance container %s is crashlooping (running between cycles), capturing error", name)
				cm.captureErrorIfNeeded(&inst)
				mu.Lock()
				results[idx] = RecoveryResult{UserID: inst.UserID, Mode: inst.Mode, Status: StatusError}
				mu.Unlock()
				return
			}

			log.Printf("recovering instance container %s", name)

			if cm.isCrashLooping(&inst) {
				log.Printf("recover: instance container %s is crashlooping, skipping (user must fix config)", name)
				cm.captureErrorIfNeeded(&inst)
				mu.Lock()
				results[idx] = RecoveryResult{UserID: inst.UserID, Mode: inst.Mode, Status: StatusError}
				mu.Unlock()
				return
			}

			if cm.tryRecoverViaDockerStart(&inst) {
				mu.Lock()
				results[idx] = RecoveryResult{UserID: inst.UserID, Mode: inst.Mode, Status: StatusRunning}
				mu.Unlock()
				return
			}

			if err := cm.CreateAndStartInstance(&inst); err != nil {
				log.Printf("recover instance %s failed: %v", name, err)
				mu.Lock()
				results[idx] = RecoveryResult{UserID: inst.UserID, Mode: inst.Mode, Status: StatusError}
				mu.Unlock()
				return
			}
			mu.Lock()
			results[idx] = RecoveryResult{UserID: inst.UserID, Mode: inst.Mode, Status: StatusRunning}
			mu.Unlock()
		}); err != nil {
			results[idx] = RecoveryResult{UserID: inst.UserID, Mode: inst.Mode, Status: StatusError}
		}
	}
	cm.pool.Wait()
	return results
}

func (cm *ContainerManager) tryRecoverViaDockerStart(inst *StrategyInstance) bool {
	name := cm.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
	out, err := cm.docker("inspect", "-f", "{{.State.Status}}", name)
	if err != nil {
		return false
	}
	status := strings.TrimSpace(out)
	if status == "running" {
		return true
	}
	if status != "exited" && status != "created" {
		return false
	}

	if _, err := cm.docker("start", name); err != nil {
		log.Printf("docker start %s failed (was %s): %v", name, status, err)
		return false
	}
	log.Printf("instance container %s restarted via docker start (was %s)", name, status)

	var running bool
	for i := 0; i < 5; i++ {
		running, _ = cm.CheckInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID)
		if running {
			break
		}
		time.Sleep(time.Second)
	}
	return running
}

// CleanupStopped removes stopped bbgo containers that aren't tracked.
func (cm *ContainerManager) CleanupStopped(trackedInstances []StrategyInstance) int {
	trackedNames := make(map[string]bool, len(trackedInstances))
	for _, inst := range trackedInstances {
		trackedNames[cm.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)] = true
	}

	var names []string
	out, err := cm.docker("ps", "-a",
		"--filter", "name="+containerPrefix,
		"--filter", "status=exited",
		"--filter", "status=dead",
		"--format", "{{.Names}}",
	)
	if err != nil {
		log.Printf("cleanup stopped: %v", err)
	} else {
		for _, n := range strings.Split(out, "\n") {
			if n = strings.TrimSpace(n); n != "" {
				names = append(names, n)
			}
		}
	}

	var toRemove []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || !strings.HasPrefix(name, containerPrefix) {
			continue
		}
		if trackedNames[name] {
			continue
		}
		toRemove = append(toRemove, name)
	}
	if len(toRemove) == 0 {
		return 0
	}
	rmArgs := append([]string{"rm"}, toRemove...)
	if _, err := cm.docker(rmArgs...); err != nil {
		log.Printf("cleanup stopped: batch rm failed (%v), falling back to per-container", err)
		cleaned := 0
		for _, name := range toRemove {
			if _, err := cm.docker("rm", name); err != nil {
				log.Printf("cleanup stopped: failed to remove %s: %v", name, err)
			} else {
				cleaned++
			}
		}
		return cleaned
	}
	for _, name := range toRemove {
		log.Printf("cleanup stopped: removed container %s", name)
	}
	return len(toRemove)
}
