package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestBuildSyncConfig(t *testing.T) {
	tests := []struct {
		name       string
		exchange   string
		symbol     string
		startTime  string
		endTime    string
		wantInYAML []string
	}{
		{
			name:      "binance_btcusdt",
			exchange:  "binance",
			symbol:    "BTCUSDT",
			startTime: "2024-01-01",
			endTime:   "2024-06-01",
			wantInYAML: []string{"binance:", "BTCUSDT", "2024-01-01", "2024-06-01"},
		},
		{
			name:      "bybit_ethusdt",
			exchange:  "bybit",
			symbol:    "ETHUSDT",
			startTime: "2024-03-01",
			endTime:   "2024-03-31",
			wantInYAML: []string{"bybit:", "ETHUSDT", "2024-03-01", "2024-03-31"},
		},
		{
			name:      "kucoin_solusdt",
			exchange:  "kucoin",
			symbol:    "SOLUSDT",
			startTime: "2023-01-01",
			endTime:   "2024-01-01",
			wantInYAML: []string{"kucoin:", "SOLUSDT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, err := buildSyncConfig(tt.exchange, tt.symbol, tt.startTime, tt.endTime)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s := string(yaml)
			for _, want := range tt.wantInYAML {
				if !strings.Contains(s, want) {
					t.Errorf("expected YAML to contain %q\n--- YAML ---\n%s", want, s)
				}
			}
		})
	}
}
