package main

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

var errTestNotFound = errors.New("container not found")

func TestContainerStatus_Running(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		if args[0] == "inspect" {
			return "running", nil
		}
		return "", nil
	}
	running, status, err := cm.ContainerStatus("user-1", ModeLive)
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Error("expected running=true")
	}
	if status != "running" {
		t.Errorf("status = %q, want %q", status, "running")
	}
}

func TestContainerStatus_Exited(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "exited", nil
	}
	running, status, err := cm.ContainerStatus("user-1", ModeLive)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Error("expected running=false for exited container")
	}
	if status != "exited" {
		t.Errorf("status = %q, want %q", status, "exited")
	}
}

func TestContainerStatus_NotFound(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "", errTestNotFound
	}
	_, _, err := cm.ContainerStatus("user-1", ModeLive)
	if err == nil {
		t.Error("expected error for missing container")
	}
}

func TestTryStart_ExitedContainer(t *testing.T) {
	cm := testContainerManager(t)
	var calls []string
	cm.dockerFn = func(args ...string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		if args[0] == "inspect" {
			return "exited", nil
		}
		return "", nil
	}
	if !cm.TryStart("user-1", ModeLive) {
		t.Fatal("TryStart should succeed for exited container")
	}
	var started bool
	for _, c := range calls {
		if strings.HasPrefix(c, "start bbgo-user-1") {
			started = true
		}
	}
	if !started {
		t.Error("expected docker start to be called")
	}
}

func TestTryStart_AlreadyRunning(t *testing.T) {
	cm := testContainerManager(t)
	var callCount int
	cm.dockerFn = func(args ...string) (string, error) {
		callCount++
		return "running", nil
	}
	if !cm.TryStart("user-1", ModeLive) {
		t.Fatal("TryStart should return true for running container")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (inspect only, no start)", callCount)
	}
}

func TestTryStart_DockerStartFails(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		if args[0] == "inspect" {
			return "exited", nil
		}
		return "error: container not found", errTestNotFound
	}
	if cm.TryStart("user-1", ModeLive) {
		t.Error("TryStart should return false when docker start fails")
	}
}

func TestTryStart_NoContainer(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "", errTestNotFound
	}
	if cm.TryStart("user-1", ModeLive) {
		t.Error("TryStart should return false when container doesn't exist")
	}
}

func TestCleanupStopped_RemovesExitedAndDead(t *testing.T) {
	cm := testContainerManager(t)
	var removed atomic.Int32
	cm.dockerFn = func(args ...string) (string, error) {
		cmd := strings.Join(args, " ")
		switch {
		case strings.Contains(cmd, "status=exited"):
			return "bbgo-user-stopped\nbbgo-user-restarting", nil
		case strings.Contains(cmd, "status=dead"):
			return "bbgo-user-dead", nil
		case args[0] == "rm":
			removed.Add(1)
			return "", nil
		}
		return "", nil
	}

	// user-restarting is tracked, so bbgo-user-restarting is skipped
	users := []UserMode{
		{UserID: "user-restarting", Mode: ModeLive},
	}
	cleaned := cm.CleanupStopped(users)

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
		if strings.Contains(cmd, "status=exited") {
			return "bbgo-user-1", nil
		}
		if strings.Contains(cmd, "status=dead") {
			return "", nil
		}
		if args[0] == "rm" {
			t.Error("should not remove container still tracked as running")
		}
		return "", nil
	}

	// user-1 is tracked, so bbgo-user-1 should be skipped
	users := []UserMode{
		{UserID: "user-1", Mode: ModeLive},
	}
	cleaned := cm.CleanupStopped(users)
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
	return NewContainerManager(cfg, nil, nil)
}
