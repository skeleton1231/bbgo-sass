package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAndStart_PreservesDB(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "test-user")
	os.MkdirAll(userDir, 0o755)

	dbPath := filepath.Join(userDir, "bbgo.db")
	os.WriteFile(dbPath, []byte("existing-db-state"), 0o644)

	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
	}
	cm := NewContainerManager(cfg, nil, nil)
	cm.dockerFn = func(args ...string) (string, error) {
		return "container-id", nil
	}

	uc := &UserContainer{
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "live",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	if err := cm.CreateAndStart(uc); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("bbgo.db should be preserved: %v", err)
	}
	if string(data) != "existing-db-state" {
		t.Errorf("DB content corrupted: got %q", string(data))
	}

	entries, _ := os.ReadDir(userDir)
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bbgo.db.backup.") {
			found = true
			backupData, _ := os.ReadFile(filepath.Join(userDir, e.Name()))
			if string(backupData) != "existing-db-state" {
				t.Errorf("backup content mismatch: got %q", string(backupData))
			}
			break
		}
	}
	if !found {
		t.Error("expected safety backup to exist")
	}
}

func TestCreateAndStart_SecondRestart_KeepsDB(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "test-user")
	os.MkdirAll(userDir, 0o755)

	dbPath := filepath.Join(userDir, "bbgo.db")
	os.WriteFile(dbPath, []byte("first-state"), 0o644)

	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
	}
	cm := NewContainerManager(cfg, nil, nil)
	cm.dockerFn = func(args ...string) (string, error) {
		return "container-id", nil
	}

	uc := &UserContainer{
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "live",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}

	cm.CreateAndStart(uc)
	os.WriteFile(dbPath, []byte("second-state"), 0o644)

	cm.CreateAndStart(uc)

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("bbgo.db should survive second restart: %v", err)
	}
	if string(data) != "second-state" {
		t.Errorf("DB should keep latest state, got %q", string(data))
	}
}

func TestCreateAndStart_NoDB_NoBackup(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "test-user")
	os.MkdirAll(userDir, 0o755)

	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
	}
	cm := NewContainerManager(cfg, nil, nil)
	cm.dockerFn = func(args ...string) (string, error) {
		return "container-id", nil
	}

	uc := &UserContainer{
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "paper",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	if err := cm.CreateAndStart(uc); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	entries, _ := os.ReadDir(userDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bbgo.db.backup.") {
			t.Errorf("no backup should be created when no DB exists, found %s", e.Name())
		}
	}
}
