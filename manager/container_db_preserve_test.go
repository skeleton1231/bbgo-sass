package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAndStartInstance_PreservesDB(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)

	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT"}`),
		InstanceID: "grid2-BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return false })

	instanceDir := store.InstanceDir("test-user", ModeLive, "grid2-BTCUSDT")
	dbPath := filepath.Join(instanceDir, "bbgo.db")
	os.MkdirAll(instanceDir, 0o755)
	os.WriteFile(dbPath, []byte("existing-db-state"), 0o644)

	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
	}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.dockerFn = func(args ...string) (string, error) {
		return "container-id", nil
	}

	if err := cm.CreateAndStartInstance(inst); err != nil {
		t.Fatalf("CreateAndStartInstance: %v", err)
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("bbgo.db should be preserved: %v", err)
	}
	if string(data) != "existing-db-state" {
		t.Errorf("DB content corrupted: got %q", string(data))
	}
}

func TestCreateAndStartInstance_SecondRestart_KeepsDB(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)

	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT"}`),
		InstanceID: "grid2-BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return false })

	instanceDir := store.InstanceDir("test-user", ModeLive, "grid2-BTCUSDT")
	dbPath := filepath.Join(instanceDir, "bbgo.db")
	os.MkdirAll(instanceDir, 0o755)
	os.WriteFile(dbPath, []byte("first-state"), 0o644)

	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
	}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.dockerFn = func(args ...string) (string, error) {
		return "container-id", nil
	}

	cm.CreateAndStartInstance(inst)
	os.WriteFile(dbPath, []byte("second-state"), 0o644)

	cm.CreateAndStartInstance(inst)

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("bbgo.db should survive second restart: %v", err)
	}
	if string(data) != "second-state" {
		t.Errorf("DB should keep latest state, got %q", string(data))
	}
}

func TestCreateAndStartInstance_NoDB_NoBackup(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)

	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT"}`),
		InstanceID: "grid2-BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return false })

	instanceDir := store.InstanceDir("test-user", ModeLive, "grid2-BTCUSDT")
	os.MkdirAll(instanceDir, 0o755)

	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
	}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.dockerFn = func(args ...string) (string, error) {
		return "container-id", nil
	}

	if err := cm.CreateAndStartInstance(inst); err != nil {
		t.Fatalf("CreateAndStartInstance: %v", err)
	}

	entries, _ := os.ReadDir(instanceDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bbgo.db.backup.") {
			t.Errorf("no backup should be created when no DB exists, found %s", e.Name())
		}
	}
}
