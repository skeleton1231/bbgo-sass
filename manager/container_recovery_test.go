package main

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/c9s/bbgo/saas/manager/pool"
)

var errTestNotFound = errors.New("container not found")

func TestCheckInstanceRunning_Running(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "true", nil
	}
	running, err := cm.CheckInstanceRunning("user-1", ModeLive, "grid2-BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Error("expected running=true")
	}
}

func TestCheckInstanceRunning_Exited(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "false", nil
	}
	running, err := cm.CheckInstanceRunning("user-1", ModeLive, "grid2-BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Error("expected running=false for stopped container")
	}
}

func TestCheckInstanceRunning_NotFound(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "", errTestNotFound
	}
	running, err := cm.CheckInstanceRunning("user-1", ModeLive, "grid2-BTCUSDT")
	if err == nil {
		t.Error("expected error for missing container")
	}
	if running {
		t.Error("expected running=false on error")
	}
}

func TestTryRecoverViaDockerStart_ExitedContainer(t *testing.T) {
	cm := testContainerManager(t)
	var calls []string
	started := false
	cm.dockerFn = func(args ...string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		if args[0] == "inspect" && len(args) >= 3 && args[2] == "{{.State.Status}}" {
			return "exited", nil
		}
		if args[0] == "inspect" && len(args) >= 3 && args[2] == "{{.State.Running}}" {
			if started {
				return "true", nil
			}
			return "false", nil
		}
		if args[0] == "start" {
			started = true
			return "", nil
		}
		return "true", nil
	}
	inst := &StrategyInstance{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"}
	if !cm.tryRecoverViaDockerStart(inst) {
		t.Fatal("tryRecoverViaDockerStart should succeed for exited container")
	}
	var didStart bool
	for _, c := range calls {
		if strings.HasPrefix(c, "start bbgo-user-1") {
			didStart = true
		}
	}
	if !didStart {
		t.Error("expected docker start to be called")
	}
}

func TestTryRecoverViaDockerStart_AlreadyRunning(t *testing.T) {
	cm := testContainerManager(t)
	var callCount int
	cm.dockerFn = func(args ...string) (string, error) {
		callCount++
		return "running", nil
	}
	inst := &StrategyInstance{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"}
	if !cm.tryRecoverViaDockerStart(inst) {
		t.Fatal("tryRecoverViaDockerStart should return true for running container")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (inspect only, no start)", callCount)
	}
}

func TestTryRecoverViaDockerStart_DockerStartFails(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		if args[0] == "inspect" {
			return "exited", nil
		}
		return "error: container not found", errTestNotFound
	}
	inst := &StrategyInstance{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"}
	if cm.tryRecoverViaDockerStart(inst) {
		t.Error("tryRecoverViaDockerStart should return false when docker start fails")
	}
}

func TestTryRecoverViaDockerStart_NoContainer(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "", errTestNotFound
	}
	inst := &StrategyInstance{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"}
	if cm.tryRecoverViaDockerStart(inst) {
		t.Error("tryRecoverViaDockerStart should return false when container doesn't exist")
	}
}

func TestCleanupStopped_RemovesExitedAndDead(t *testing.T) {
	cm := testContainerManager(t)
	var removed atomic.Int32
	cm.dockerFn = func(args ...string) (string, error) {
		cmd := strings.Join(args, " ")
		switch {
		case strings.Contains(cmd, "status=exited") && strings.Contains(cmd, "status=dead"):
			return "bbgo-user-sto-live-grid2-btcusdt\nbbgo-user-res-live-grid2-btcusdt\nbbgo-user-dea-live-grid2-btcusdt", nil
		case args[0] == "rm":
			removed.Add(int32(len(args) - 1))
			return "", nil
		}
		return "", nil
	}

	tracked := []StrategyInstance{
		{UserID: "user-restarting", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"},
	}
	cleaned := cm.CleanupStopped(tracked)

	if cleaned != 2 {
		t.Errorf("cleaned = %d, want 2", cleaned)
	}
	if removed.Load() != 2 {
		t.Errorf("removed = %d, want 2", removed.Load())
	}
}

func TestCleanupStopped_SkipsRunningContainers(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		cmd := strings.Join(args, " ")
		if strings.Contains(cmd, "status=exited") && strings.Contains(cmd, "status=dead") {
			return "bbgo-user-1-live-grid2-btcusdt", nil
		}
		if args[0] == "rm" {
			t.Error("should not remove container still tracked as running")
		}
		return "", nil
	}

	tracked := []StrategyInstance{
		{UserID: "user-1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"},
	}
	cleaned := cm.CleanupStopped(tracked)
	if cleaned != 0 {
		t.Errorf("cleaned = %d, want 0", cleaned)
	}
}

func TestCleanupStopped_EmptyDocker(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "", nil
	}
	cleaned := cm.CleanupStopped(nil)
	if cleaned != 0 {
		t.Errorf("cleaned = %d, want 0", cleaned)
	}
}

func TestCheckAndRecover_AllAlive(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "true", nil
	}

	instances := []StrategyInstance{
		{UserID: "u1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"},
		{UserID: "u2", Mode: ModeLive, InstanceID: "grid2-ETHUSDT"},
	}

	results := cm.CheckAndRecover(instances)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Alive {
			t.Errorf("expected %s/%s alive", r.UserID, r.InstanceID)
		}
		if r.Restarted {
			t.Errorf("expected %s/%s not restarted", r.UserID, r.InstanceID)
		}
	}
}

func TestCheckAndRecover_DeadThenRestarted(t *testing.T) {
	cm := testContainerManager(t)
	started := false
	cm.dockerFn = func(args ...string) (string, error) {
		cmd := strings.Join(args, " ")
		if strings.Contains(cmd, "State.Running") {
			if started {
				return "true", nil
			}
			return "false", nil
		}
		if args[0] == "inspect" && len(args) >= 3 && args[2] == "{{.State.Status}}" {
			return "exited", nil
		}
		if args[0] == "start" {
			started = true
			return "", nil
		}
		return "true", nil
	}

	instances := []StrategyInstance{
		{UserID: "u1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"},
	}
	results := cm.CheckAndRecover(instances)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Alive {
		t.Error("expected alive after restart")
	}
	if !results[0].Restarted {
		t.Error("expected restarted=true")
	}
}

// TestCheckAndRecover_CrashLoop_SkipsRecovery verifies that when the container
// is in a crashloop, CheckAndRecover does NOT call docker start or recreate —
// it just marks the result as failed so the user can fix the underlying
// config. This is the abstract fix for the phantom-active problem: any
// strategy that crashes inside the container (validation, panic, missing
// data, etc.) is detected via Docker's RestartCount/ExitCode rather than
// silently retried forever.
func TestCheckAndRecover_CrashLoop_SkipsRecovery(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	var calls []string
	cm.dockerFn = func(args ...string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		if args[0] == "inspect" && len(args) >= 3 {
			tpl := args[2]
			switch tpl {
			case "{{.State.Running}}":
				// Docker reports not-running here because we're between crash cycles
				// or docker is mid-restart.
				return "false", nil
			case "{{.State.Status}}":
				return "exited", nil
			default:
				// Composed-template inspect from CheckInstanceHealth returns crashloop signal.
				return "restarting|false|5|1", nil
			}
		}
		if args[0] == "logs" {
			return `time="2026-06-17T14:40:10Z" level=fatal msg="cannot execute command" error="spread is too small"`, nil
		}
		return "", nil
	}

	createTestInstance(t, store, "u1", "live", "grid2", "BTCUSDT", nil)
	instances := []StrategyInstance{
		{UserID: "u1", Mode: ModeLive, InstanceID: "grid2-BTCUSDT"},
	}
	results := cm.CheckAndRecover(instances)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Alive {
		t.Error("expected Alive=false for crashlooping container")
	}
	if results[0].Error == "" {
		t.Error("expected non-empty Error for crashlooping container")
	}
	for _, c := range calls {
		if strings.HasPrefix(c, "start ") || strings.HasPrefix(c, "run ") {
			t.Errorf("recovery must not start/run a crashlooping container, got call: %s", c)
		}
	}
}

func TestRecoverUsers_CrashLoop_MarksError(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	cm.dockerFn = func(args ...string) (string, error) {
		// args[0]=inspect args[1]=-f args[2]=<template> args[3]=<name>
		if args[0] == "inspect" && len(args) >= 3 {
			tpl := args[2]
			switch tpl {
			case "{{.State.Running}}":
				return "false", nil
			case "{{.State.Status}}":
				return "exited", nil
			default:
				// Composed template from CheckInstanceHealth.
				return "running|true|7|1", nil
			}
		}
		if args[0] == "logs" {
			return `time="2026-06-17T14:40:10Z" level=fatal msg="cannot execute command" error="bollmaker bandWidth must be > 0"`, nil
		}
		return "", nil
	}

	createTestInstance(t, store, "u1", "live", "bollmaker", "BTCUSDT", nil)
	users := []UserMode{{UserID: "u1", Mode: ModeLive}}
	results := cm.RecoverUsers(users)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusError {
		t.Errorf("expected status=%q for crashlooping container, got %q", StatusError, results[0].Status)
	}
}

func TestRecoverUsers_AllRunning(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	cm.dockerFn = func(args ...string) (string, error) {
		return "true", nil
	}

	createTestInstance(t, store, "u1", "live", "grid2", "BTCUSDT", nil)

	users := []UserMode{{UserID: "u1", Mode: ModeLive}}
	results := cm.RecoverUsers(users)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusRunning {
		t.Errorf("expected running, got %s", results[0].Status)
	}
}

func testContainerManager(t *testing.T) *ContainerManager {
	t.Helper()
	dir := t.TempDir()
	cfg := &Config{
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
		BBGOGRPCPort:  9090,
	}
	p := pool.New(5)
	t.Cleanup(func() { p.Release() })
	return NewContainerManager(cfg, nil, p, nil)
}
