package main

import (
	"strings"
	"testing"
)

func TestResourceArgs_LiveMode(t *testing.T) {
	cfg := &Config{
		LiveResources: ContainerResources{
			Memory: "256m", MemorySwap: "512m", CPUs: "0.5",
			PidsLimit: 128, LogMaxSize: "10m", LogMaxFile: 3,
		},
	}
	cm := NewContainerManager(cfg, nil, nil)

	args := cm.resourceArgs(ModeLive)
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

func TestResourceArgs_PaperMode(t *testing.T) {
	cfg := &Config{
		PaperResources: ContainerResources{
			Memory: "128m", MemorySwap: "256m", CPUs: "0.25",
			PidsLimit: 64, LogMaxSize: "10m", LogMaxFile: 3,
		},
	}
	cm := NewContainerManager(cfg, nil, nil)

	args := cm.resourceArgs(ModePaper)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--memory 128m") {
		t.Errorf("expected --memory 128m in %q", joined)
	}
	if !strings.Contains(joined, "--cpus 0.25") {
		t.Errorf("expected --cpus 0.25 in %q", joined)
	}
	if !strings.Contains(joined, "--pids-limit 64") {
		t.Errorf("expected --pids-limit 64 in %q", joined)
	}
}

func TestResourceArgs_EmptyFields(t *testing.T) {
	cfg := &Config{
		LiveResources: ContainerResources{
			Memory: "256m",
		},
	}
	cm := NewContainerManager(cfg, nil, nil)

	args := cm.resourceArgs(ModeLive)
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

func TestCreateAndStart_IncludeResourceLimits_Live(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
		BBGOGRPCPort:  9090,
		LiveResources: ContainerResources{
			Memory: "256m", MemorySwap: "512m", CPUs: "0.5",
			PidsLimit: 128, LogMaxSize: "10m", LogMaxFile: 3,
		},
	}
	cm := NewContainerManager(cfg, nil, nil)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid", Mode: "live",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	if err := cm.CreateAndStart(uc); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	cmdStr := strings.Join(capturedArgs, " ")

	if !strings.Contains(cmdStr, "--memory 256m") {
		t.Error("expected --memory 256m in live docker run command")
	}
	if !strings.Contains(cmdStr, "--cpus 0.5") {
		t.Error("expected --cpus 0.5 in live docker run command")
	}
	if !strings.Contains(cmdStr, "--pids-limit 128") {
		t.Error("expected --pids-limit 128 in live docker run command")
	}
	if !strings.Contains(cmdStr, "max-size=10m") {
		t.Error("expected log max-size=10m in live docker run command")
	}
}

func TestCreateAndStart_IncludeResourceLimits_Paper(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManagerToken:   "tok",
		DataDir:        dir,
		DataVolume:     "bbgo-data",
		DockerNetwork:  "bbgo-net",
		BBGOImage:      "bbgo-base:latest",
		BBGOPort:       8080,
		BBGOGRPCPort:   9090,
		PaperResources: ContainerResources{
			Memory: "128m", CPUs: "0.25", PidsLimit: 64,
		},
	}
	cm := NewContainerManager(cfg, nil, nil)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	uc := &UserContainer{
		Mode:   ModePaper,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid", Mode: "paper",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	if err := cm.CreateAndStart(uc); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	cmdStr := strings.Join(capturedArgs, " ")

	if !strings.Contains(cmdStr, "--memory 128m") {
		t.Error("expected --memory 128m in paper docker run command")
	}
	if !strings.Contains(cmdStr, "--cpus 0.25") {
		t.Error("expected --cpus 0.25 in paper docker run command")
	}
	if !strings.Contains(cmdStr, "--pids-limit 64") {
		t.Error("expected --pids-limit 64 in paper docker run command")
	}
	if strings.Contains(cmdStr, "--memory 256m") {
		t.Error("live memory limit should not appear in paper container")
	}
}

func TestCreateAndStart_NoResourceLimits_WhenEmpty(t *testing.T) {
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
	cm := NewContainerManager(cfg, nil, nil)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid", Mode: "live",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	if err := cm.CreateAndStart(uc); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
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
