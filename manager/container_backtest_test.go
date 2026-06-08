package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunBacktest_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, nil)

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
	cm := NewContainerManager(cfg, nil, nil, nil)

	jobID := "bt-test123"
	backtestDir := filepath.Join(dir, "backtest", "user-1", jobID)
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

func TestCleanupBacktest_NonExistentDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, nil)

	cm.CleanupBacktest("user-1", "bt-nonexistent")
}

func TestBacktestReportDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, nil)

	got := cm.BacktestReportDir("user-1", "bt-abc")
	want := filepath.Join(dir, "backtest", "user-1", "bt-abc")
	if got != want {
		t.Errorf("BacktestReportDir = %q, want %q", got, want)
	}
}

func TestReadBacktestReport(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, nil)

	jobID := "bt-report1"
	reportDir := filepath.Join(dir, "backtest", "user-1", jobID)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}

	summary := `{"totalTrades": 10, "profit": 500}`
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}
	equity := "time\tequity\n2024-01-01\t10000\n2024-06-01\t10500"
	if err := os.WriteFile(filepath.Join(reportDir, "equity_curve.tsv"), []byte(equity), 0o644); err != nil {
		t.Fatal(err)
	}

	report, equityCurve, err := cm.ReadBacktestReport("user-1", jobID)
	if err != nil {
		t.Fatalf("ReadBacktestReport: %v", err)
	}
	if string(report) != summary {
		t.Errorf("report = %q, want %q", string(report), summary)
	}
	if string(equityCurve) != equity {
		t.Errorf("equityCurve = %q, want %q", string(equityCurve), equity)
	}
}

func TestReadBacktestReport_MissingSummary(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, nil)

	_, _, err := cm.ReadBacktestReport("user-1", "bt-missing")
	if err == nil {
		t.Error("expected error for missing summary.json")
	}
}
