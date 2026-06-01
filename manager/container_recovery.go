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
// from the current strategies. Compares parsed YAML structures to avoid false
// negatives from map key ordering differences.
func (cm *ContainerManager) ConfigMatches(userID, mode string) bool {
	dataDir := filepath.Join(cm.cfg.DataDir, userID)
	if mode == ModePaper {
		dataDir = filepath.Join(cm.cfg.DataDir, userID+"-paper")
	}
	yamlPath := filepath.Join(dataDir, "bbgo.yaml")
	existing, err := os.ReadFile(yamlPath)
	if err != nil {
		return false
	}
	strategies, _ := parseStrategiesFromYAML(existing)
	expected, err := buildUserYAML(userID, mode, strategies, func(exchange string) bool {
		_, err := cm.creds.GetByMode(userID, exchange, mode == ModePaper)
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
func (cm *ContainerManager) tryRecoverViaDockerStart(userID, mode string) bool {
	if !cm.TryStart(userID, mode) {
		return false
	}

	var running bool
	for i := 0; i < 5; i++ {
		running, _ = cm.CheckRunning(userID, mode)
		if running {
			break
		}
		time.Sleep(time.Second)
	}
	if !running {
		return false
	}

	if cm.ConfigMatches(userID, mode) {
		log.Printf("container %s recovered via docker start", cm.containerName(userID, mode))
		return true
	}

	log.Printf("container %s config stale, recreating", cm.containerName(userID, mode))
	return false
}

// CheckAndRecover checks all running containers in parallel and restarts
// any that have died. Uses a goroutine pool (max 5) for parallel docker inspect.
func (cm *ContainerManager) CheckAndRecover(users []UserMode) []HealthCheckResult {
	results := make([]HealthCheckResult, len(users))
	var mu sync.Mutex

	for i, um := range users {
		idx, um := i, um
		if err := cm.pool.Submit(func() {
			running, _ := cm.CheckRunning(um.UserID, um.Mode)
			if running {
				mu.Lock()
				results[idx] = HealthCheckResult{UserID: um.UserID, Mode: um.Mode, Alive: true}
				mu.Unlock()
				return
			}
			log.Printf("health check: container %s died, attempting recovery", cm.containerName(um.UserID, um.Mode))

			if cm.tryRecoverViaDockerStart(um.UserID, um.Mode) {
				mu.Lock()
				results[idx] = HealthCheckResult{UserID: um.UserID, Mode: um.Mode, Alive: true, Restarted: true}
				mu.Unlock()
				return
			}

			log.Printf("recreating container %s", cm.containerName(um.UserID, um.Mode))
			if err := cm.CreateAndStart(um.UserID, um.Mode); err != nil {
				mu.Lock()
				results[idx] = HealthCheckResult{UserID: um.UserID, Mode: um.Mode, Alive: false, Error: err.Error()}
				mu.Unlock()
				return
			}
			mu.Lock()
			results[idx] = HealthCheckResult{UserID: um.UserID, Mode: um.Mode, Alive: true, Restarted: true}
			mu.Unlock()
		}); err != nil {
			results[idx] = HealthCheckResult{UserID: um.UserID, Mode: um.Mode, Alive: false, Error: err.Error()}
		}
	}
	cm.pool.Wait()
	return results
}

func (cm *ContainerManager) RecoverUsers(users []UserMode) []RecoveryResult {
	results := make([]RecoveryResult, len(users))
	var mu sync.Mutex

	for i, um := range users {
		idx, um := i, um
		if err := cm.pool.Submit(func() {
			name := cm.containerName(um.UserID, um.Mode)
			out, _ := cm.docker("inspect", "-f", "{{.State.Running}}", name)
			if out == "true" {
				log.Printf("recovered container %s (running)", name)
				mu.Lock()
				results[idx] = RecoveryResult{UserID: um.UserID, Mode: um.Mode, Status: StatusRunning}
				mu.Unlock()
				return
			}

			log.Printf("recovering container %s for user %s", name, um.UserID)

			if cm.tryRecoverViaDockerStart(um.UserID, um.Mode) {
				mu.Lock()
				results[idx] = RecoveryResult{UserID: um.UserID, Mode: um.Mode, Status: StatusRunning}
				mu.Unlock()
				return
			}

			log.Printf("recreating container %s", name)
			if err := cm.CreateAndStart(um.UserID, um.Mode); err != nil {
				log.Printf("recover user %s failed: %v", um.UserID, err)
				mu.Lock()
				results[idx] = RecoveryResult{UserID: um.UserID, Mode: um.Mode, Status: StatusError}
				mu.Unlock()
				return
			}
			mu.Lock()
			results[idx] = RecoveryResult{UserID: um.UserID, Mode: um.Mode, Status: StatusRunning}
			mu.Unlock()
		}); err != nil {
			results[idx] = RecoveryResult{UserID: um.UserID, Mode: um.Mode, Status: StatusError}
		}
	}
	cm.pool.Wait()
	return results
}

// CleanupStopped removes stopped bbgo containers that aren't tracked.
func (cm *ContainerManager) CleanupStopped(tracked []UserMode) int {
	trackedNames := make(map[string]bool, len(tracked))
	for _, um := range tracked {
		trackedNames[cm.containerName(um.UserID, um.Mode)] = true
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
		if trackedNames[name] {
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
