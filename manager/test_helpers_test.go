package main

import (
	"os"
	"path/filepath"
	"testing"
)

const testEncryptionKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// newTestStore creates a StrategyStore backed by a temp directory.
func newTestStore(t *testing.T) (*StrategyStore, string) {
	t.Helper()
	dir := t.TempDir()
	return NewStrategyStore(dir), dir
}

// writeTestStrategies creates a bbgo.yaml for the given user/mode with the specified strategies.
func writeTestStrategies(t *testing.T, store *StrategyStore, userID, mode string, strategies []StrategyEntry) {
	t.Helper()
	dir := store.hostDir(userID, mode)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	yaml, err := buildUserYAML(userID, mode, strategies, func(string) bool { return false })
	if err != nil {
		t.Fatalf("build yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bbgo.yaml"), yaml, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}

// containerRunningFor returns a containerRunning hook that reports the given user/mode pairs as running.
func containerRunningFor(running map[string]bool) func(string, string) bool {
	return func(userID, mode string) bool {
		return running[userID+":"+mode]
	}
}

func rawJSON(s string) []byte {
	return []byte(s)
}
