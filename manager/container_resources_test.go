package main

import (
	"strings"
	"testing"
)

func TestInstanceResourceArgs_WithLimits(t *testing.T) {
	cfg := &Config{
		InstanceResources: ContainerResources{
			Memory: "256m", MemorySwap: "512m", CPUs: "0.5",
			PidsLimit: 128, LogMaxSize: "10m", LogMaxFile: 3,
		},
	}
	cm := NewContainerManager(cfg, nil, nil, nil)

	args := cm.instanceResourceArgs()
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--memory 256m") {
		t.Errorf("expected --memory 256m in %q", joined)
	}
	if !strings.Contains(joined, "--memory-swap 512m") {
		t.Errorf("expected --memory-swap 512m in %q", joined)
	}
	if !strings.Contains(joined, "--cpus 0.5") {
		t.Errorf("expected --cpus 0.5 in %q", joined)
	}
	if !strings.Contains(joined, "--pids-limit 128") {
		t.Errorf("expected --pids-limit 128 in %q", joined)
	}
	if !strings.Contains(joined, "max-size=10m") {
		t.Errorf("expected max-size=10m in %q", joined)
	}
	if !strings.Contains(joined, "max-file=3") {
		t.Errorf("expected max-file=3 in %q", joined)
	}
}

func TestInstanceResourceArgs_EmptyFields(t *testing.T) {
	cfg := &Config{
		InstanceResources: ContainerResources{
			Memory: "256m",
		},
	}
	cm := NewContainerManager(cfg, nil, nil, nil)

	args := cm.instanceResourceArgs()
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--memory 256m") {
		t.Errorf("expected --memory 256m in %q", joined)
	}
	if strings.Contains(joined, "--memory-swap") {
		t.Errorf("should not include --memory-swap when empty, got %q", joined)
	}
	if strings.Contains(joined, "--cpus") {
		t.Errorf("should not include --cpus when empty, got %q", joined)
	}
	if strings.Contains(joined, "--pids-limit") {
		t.Errorf("should not include --pids-limit when 0, got %q", joined)
	}
	if strings.Contains(joined, "max-file") {
		t.Errorf("should not include max-file when 0, got %q", joined)
	}
}

func TestCreateAndStartInstance_IncludeResourceLimits(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
		BBGOGRPCPort:  9090,
		InstanceResources: ContainerResources{
			Memory: "256m", MemorySwap: "512m", CPUs: "0.5",
			PidsLimit: 128, LogMaxSize: "10m", LogMaxFile: 3,
		},
	}
	store := NewInstanceStore(dir, nil)
	cm := NewContainerManager(cfg, nil, nil, store)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	inst := &StrategyInstance{
		UserID: "test-user", Mode: ModeLive, Strategy: "grid",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config: rawJSON(`{"symbol":"BTCUSDT"}`), InstanceID: "grid-BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return false })

	if err := cm.CreateAndStartInstance(inst); err != nil {
		t.Fatalf("CreateAndStartInstance: %v", err)
	}

	cmdStr := strings.Join(capturedArgs, " ")

	if !strings.Contains(cmdStr, "--memory 256m") {
		t.Error("expected --memory 256m in docker run command")
	}
	if !strings.Contains(cmdStr, "--cpus 0.5") {
		t.Error("expected --cpus 0.5 in docker run command")
	}
	if !strings.Contains(cmdStr, "--pids-limit 128") {
		t.Error("expected --pids-limit 128 in docker run command")
	}
	if !strings.Contains(cmdStr, "max-size=10m") {
		t.Error("expected log max-size=10m in docker run command")
	}
}

func TestCreateAndStartInstance_NoResourceLimits_WhenEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
		BBGOGRPCPort:  9090,
	}
	store := NewInstanceStore(dir, nil)
	cm := NewContainerManager(cfg, nil, nil, store)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	inst := &StrategyInstance{
		UserID: "test-user", Mode: ModeLive, Strategy: "grid",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config: rawJSON(`{"symbol":"BTCUSDT"}`), InstanceID: "grid-BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return false })

	if err := cm.CreateAndStartInstance(inst); err != nil {
		t.Fatalf("CreateAndStartInstance: %v", err)
	}

	cmdStr := strings.Join(capturedArgs, " ")

	if strings.Contains(cmdStr, "--memory") {
		t.Error("should not include --memory when not configured")
	}
	if strings.Contains(cmdStr, "--cpus") {
		t.Error("should not include --cpus when not configured")
	}
	if strings.Contains(cmdStr, "--pids-limit") {
		t.Error("should not include --pids-limit when not configured")
	}
}
