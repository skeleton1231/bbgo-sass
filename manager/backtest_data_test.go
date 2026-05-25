package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasDataForRange_AlwaysReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil)
	api := &API{container: cm}

	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("should always return false to force sync")
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
