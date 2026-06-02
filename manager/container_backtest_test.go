package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunBacktest_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil)

	t.Run("slash in jobID", func(t *testing.T) {
		_, err := cm.RunBacktest("user-1", "evil/../../etc", []byte("yaml"))
		if err == nil {
			t.Error("expected error for jobID with slash")
		}
	})

	t.Run("dotdot in jobID", func(t *testing.T) {
		_, err := cm.RunBacktest("user-1", "..secret", []byte("yaml"))
		if err == nil {
			t.Error("expected error for jobID with ..")
		}
	})
}

func TestCleanupBacktest_RemovesDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil)
	cm.checkRunningFn = func(string, string) (bool, error) { return true, nil }

	jobID := "bt-test123"
	// backtestMode prefers paper, so hostDir appends "-paper"
	backtestDir := dir + "/user-1-paper/backtest/" + jobID
	if err := os.MkdirAll(backtestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dummyFile := filepath.Join(backtestDir, "bbgo.yaml")
	if err := os.WriteFile(dummyFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	cm.CleanupBacktest("user-1", jobID)

	if _, err := os.Stat(backtestDir); !os.IsNotExist(err) {
		t.Error("expected backtest directory to be removed")
	}
}

func TestCleanupBacktest_NilReceiver(t *testing.T) {
	var cm *ContainerManager
	cm.CleanupBacktest("user-1", "bt-123")
}

func TestCleanupBacktest_NoContainerRunning(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil)
	cm.checkRunningFn = func(string, string) (bool, error) { return false, nil }

	backtestDir := dir + "/user-1/backtest/bt-123"
	if err := os.MkdirAll(backtestDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cm.CleanupBacktest("user-1", "bt-123")

	// Directory should still exist since backtestMode returns error
	if _, err := os.Stat(backtestDir); os.IsNotExist(err) {
		t.Error("expected directory to remain when no container is running")
	}
}
