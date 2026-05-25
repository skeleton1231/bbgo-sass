package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHasDataForRange_WhenDBNotExist(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil)
	api := &API{container: cm}

	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("should return false when DB does not exist")
	}
}

func TestHasDataForRange_WhenDBTooSmall(t *testing.T) {
	dir := t.TempDir()
	sharedDir := dir + "/backtest-shared"
	os.MkdirAll(sharedDir, 0o755)
	os.WriteFile(sharedDir+"/backtest.db", make([]byte, 100), 0o644)

	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil)
	api := &API{container: cm}

	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("should return false when DB is too small")
	}
}

func TestHasDataForRange_WhenDBExistsButStale(t *testing.T) {
	dir := t.TempDir()
	sharedDir := dir + "/backtest-shared"
	os.MkdirAll(sharedDir, 0o755)
	os.WriteFile(sharedDir+"/backtest.db", make([]byte, 2<<20), 0o644)
	oldTime := time.Now().Add(-48 * time.Hour)
	os.Chtimes(sharedDir+"/backtest.db", oldTime, oldTime)

	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil)
	api := &API{container: cm}

	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("should return false when DB modification time is stale")
	}
}

func TestHasDataForRange_WhenDBRecentAndLarge(t *testing.T) {
	dir := t.TempDir()
	sharedDir := dir + "/backtest-shared"
	os.MkdirAll(sharedDir, 0o755)
	os.WriteFile(sharedDir+"/backtest.db", make([]byte, 2<<20), 0o644)

	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil)
	api := &API{container: cm}

	if !api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("should return true when DB is recent and large enough")
	}
}

func TestHasDataForRange_WithCustomSharedDir(t *testing.T) {
	dir := t.TempDir()
	sharedDir := filepath.Join(dir, "custom-shared")
	os.MkdirAll(sharedDir, 0o755)
	os.WriteFile(sharedDir+"/backtest.db", make([]byte, 2<<20), 0o644)

	cfg := &Config{DataDir: dir, BacktestSharedDir: sharedDir}
	cm := NewContainerManager(cfg, nil)
	api := &API{container: cm}

	if !api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("should return true with custom shared dir")
	}
}

func TestCleanupBackups(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		f, _ := os.CreateTemp(dir, "bbgo.db.backup.*")
		f.Close()
	}

	cleanupBackups(dir, "bbgo.db.backup", 2)

	entries, _ := os.ReadDir(dir)
	backupCount := 0
	for _, e := range entries {
		if matched, _ := filepath.Match("bbgo.db.backup.*", e.Name()); matched {
			backupCount++
		}
	}
	if backupCount != 2 {
		t.Errorf("expected 2 backups remaining, got %d", backupCount)
	}
}
