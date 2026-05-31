package main

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// HealthCheckResult reports the outcome of checking a single container.
type HealthCheckResult struct {
	UserID    string
	Mode      string
	Alive     bool
	Restarted bool
	Error     string
}

type RecoveryResult struct {
	UserID string
	Mode   string
	Status string
}

// ContainerStatus returns whether a container is running, its status string,
// and any error. Used by TryStart to decide whether docker start is viable.
func (cm *ContainerManager) ContainerStatus(userID, mode string) (bool, string, error) {
	name := cm.containerName(userID, mode)
	out, err := cm.docker("inspect", "-f", "{{.State.Status}}", name)
	if err != nil {
		return false, "", err
	}
	status := strings.TrimSpace(out)
	return status == "running", status, nil
}

// TryStart attempts to start an existing stopped container via "docker start".
// Returns true if the container was already running or successfully started.
func (cm *ContainerManager) TryStart(userID, mode string) bool {
	name := cm.containerName(userID, mode)
	_, status, err := cm.ContainerStatus(userID, mode)
	if err != nil {
		return false
	}
	if status == "running" {
		return true
	}
	if status == "exited" || status == "created" {
		if _, err := cm.docker("start", name); err != nil {
			log.Printf("docker start %s failed (was %s): %v", name, status, err)
			return false
		}
		log.Printf("container %s restarted via docker start (was %s)", name, status)
		return true
	}
	return false
}

// ConfigMatches checks if the on-disk bbgo.yaml matches what would be generated
// from the current UserContainer strategies. Compares parsed YAML structures to
// avoid false negatives from map key ordering differences.
func (cm *ContainerManager) ConfigMatches(uc *UserContainer) bool {
	dataDir := filepath.Join(cm.cfg.DataDir, uc.UserID)
	if uc.Mode == ModePaper {
		dataDir = filepath.Join(cm.cfg.DataDir, uc.UserID+"-paper")
	}
	yamlPath := filepath.Join(dataDir, "bbgo.yaml")
	existing, err := os.ReadFile(yamlPath)
	if err != nil {
		return false
	}
	expected, err := buildUserYAML(uc, func(exchange string) bool {
		_, err := cm.creds.GetByMode(uc.UserID, exchange, uc.Mode == ModePaper)
		return err == nil
	})
	if err != nil {
		return false
	}

	var existingMap, expectedMap map[string]any
	if err := yaml.Unmarshal(existing, &existingMap); err != nil {
		return false
	}
	if err := yaml.Unmarshal(expected, &expectedMap); err != nil {
		return false
	}
	return reflect.DeepEqual(existingMap, expectedMap)
}

// tryRecoverViaDockerStart attempts lightweight recovery via "docker start".
// Returns true if the container is up with matching config (caller can stop here).
// Returns false if recovery failed or config is stale (caller should recreate).
func (cm *ContainerManager) tryRecoverViaDockerStart(uc *UserContainer) bool {
	if !cm.TryStart(uc.UserID, uc.Mode) {
		return false
	}

	var running bool
	for i := 0; i < 5; i++ {
		running, _ = cm.CheckRunning(uc.UserID, uc.Mode)
		if running {
			break
		}
		time.Sleep(time.Second)
	}
	if !running {
		return false
	}

	if cm.ConfigMatches(uc) {
		log.Printf("container %s recovered via docker start", cm.containerName(uc.UserID, uc.Mode))
		return true
	}

	log.Printf("container %s config stale, recreating", cm.containerName(uc.UserID, uc.Mode))
	return false
}

// CheckAndRecover checks all running containers in parallel and restarts
// any that have died. Uses a goroutine pool (max 5) for parallel docker inspect.
// Prefers docker start over full recreation to preserve container state.
func (cm *ContainerManager) CheckAndRecover(users []*UserContainer) []HealthCheckResult {
	results := make([]HealthCheckResult, len(users))
	var mu sync.Mutex

	for i, uc := range users {
		if uc.Status != StatusRunning {
			results[i] = HealthCheckResult{UserID: uc.UserID, Mode: uc.Mode, Alive: false}
			continue
		}
		idx, uc := i, uc
		if err := cm.pool.Submit(func() {
			running, _ := cm.CheckRunning(uc.UserID, uc.Mode)
			if running {
				mu.Lock()
				results[idx] = HealthCheckResult{UserID: uc.UserID, Mode: uc.Mode, Alive: true}
				mu.Unlock()
				return
			}
			log.Printf("health check: container %s died, attempting recovery", cm.containerName(uc.UserID, uc.Mode))

			if cm.tryRecoverViaDockerStart(uc) {
				mu.Lock()
				results[idx] = HealthCheckResult{UserID: uc.UserID, Mode: uc.Mode, Alive: true, Restarted: true}
				mu.Unlock()
				return
			}

			log.Printf("recreating container %s", cm.containerName(uc.UserID, uc.Mode))
			if err := cm.CreateAndStart(uc); err != nil {
				mu.Lock()
				results[idx] = HealthCheckResult{UserID: uc.UserID, Mode: uc.Mode, Alive: false, Error: err.Error()}
				mu.Unlock()
				return
			}
			mu.Lock()
			results[idx] = HealthCheckResult{UserID: uc.UserID, Mode: uc.Mode, Alive: true, Restarted: true}
			mu.Unlock()
		}); err != nil {
			results[idx] = HealthCheckResult{UserID: uc.UserID, Mode: uc.Mode, Alive: false, Error: err.Error()}
		}
	}
	cm.pool.Wait()
	return results
}

func (cm *ContainerManager) RecoverUsers(users []*UserContainer) []RecoveryResult {
	results := make([]RecoveryResult, len(users))
	var mu sync.Mutex

	for i, uc := range users {
		idx, uc := i, uc
		if err := cm.pool.Submit(func() {
			name := cm.containerName(uc.UserID, uc.Mode)
			out, _ := cm.docker("inspect", "-f", "{{.State.Running}}", name)
			if out == "true" {
				log.Printf("recovered container %s (running)", name)
				mu.Lock()
				results[idx] = RecoveryResult{UserID: uc.UserID, Mode: uc.Mode, Status: StatusRunning}
				mu.Unlock()
				return
			}
			if uc.Status == StatusRunning {
				log.Printf("recovering container %s for user %s", name, uc.UserID)

				if cm.tryRecoverViaDockerStart(uc) {
					mu.Lock()
					results[idx] = RecoveryResult{UserID: uc.UserID, Mode: uc.Mode, Status: StatusRunning}
					mu.Unlock()
					return
				}

				log.Printf("recreating container %s", name)
				if err := cm.CreateAndStart(uc); err != nil {
					log.Printf("recover user %s failed: %v", uc.UserID, err)
					mu.Lock()
					results[idx] = RecoveryResult{UserID: uc.UserID, Mode: uc.Mode, Status: StatusError}
					mu.Unlock()
					return
				}
				mu.Lock()
				results[idx] = RecoveryResult{UserID: uc.UserID, Mode: uc.Mode, Status: StatusRunning}
				mu.Unlock()
				return
			}
			mu.Lock()
			results[idx] = RecoveryResult{UserID: uc.UserID, Mode: uc.Mode, Status: uc.Status}
			mu.Unlock()
		}); err != nil {
			results[idx] = RecoveryResult{UserID: uc.UserID, Mode: uc.Mode, Status: StatusError}
		}
	}
	cm.pool.Wait()
	return results
}

// CleanupStopped removes stopped bbgo containers that are no longer tracked
// as running or starting. This frees Docker resources (disk, network entries).
// Must only be called after CheckAndRecover completes to avoid racing with recovery.
func (cm *ContainerManager) CleanupStopped(allUsers []*UserContainer) int {
	tracked := make(map[string]bool, len(allUsers))
	for _, uc := range allUsers {
		if uc.Status == StatusRunning || uc.Status == StatusStarting {
			tracked[cm.containerName(uc.UserID, uc.Mode)] = true
		}
	}

	var names []string
	for _, status := range []string{"exited", "dead"} {
		out, err := cm.docker("ps", "-a", "--filter", "name="+containerPrefix, "--filter", "status="+status, "--format", "{{.Names}}")
		if err != nil {
			log.Printf("cleanup stopped (%s): %v", status, err)
			continue
		}
		for _, n := range strings.Split(out, "\n") {
			if n = strings.TrimSpace(n); n != "" {
				names = append(names, n)
			}
		}
	}

	var cleaned int
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || !strings.HasPrefix(name, containerPrefix) {
			continue
		}
		if tracked[name] {
			continue
		}
		if _, err := cm.docker("rm", name); err != nil {
			log.Printf("cleanup stopped: failed to remove %s: %v", name, err)
		} else {
			log.Printf("cleanup stopped: removed container %s", name)
			cleaned++
		}
	}
	return cleaned
}
