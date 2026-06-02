package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBacktestDefaults_InjectedWhenMissing(t *testing.T) {
	tests := []struct {
		strategy   string
		config     string
		wantField  string
		wantValue  string
	}{
		{"emacross", `{"symbol":"BTCUSDT","quantity":0.1}`, "interval", "1h"},
		{"supertrend", `{"symbol":"ETHUSDT","factor":3}`, "interval", "1h"},
		{"bollgrid", `{"symbol":"BTCUSDT","gridNumber":8}`, "interval", "1h"},
		{"trendtrader", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
		{"bollmaker", `{"symbol":"ETHUSDT"}`, "interval", "1h"},
		{"pivotshort", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
		{"swing", `{"symbol":"ETHUSDT"}`, "interval", "1h"},
		{"flashcrash", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
	}

	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			yaml, err := buildBacktestYAML(tt.strategy, json.RawMessage(tt.config), "2024-01-01", "2024-06-01", "binance", "")
			if err != nil {
				t.Fatal(err)
			}
			s := string(yaml)
			if !strContains(s, tt.wantField) {
				t.Errorf("backtest YAML for %s should contain %s, got:\n%s", tt.strategy, tt.wantField, s)
			}
			if !strContains(s, tt.wantValue) {
				t.Errorf("backtest YAML for %s should contain value %s, got:\n%s", tt.strategy, tt.wantValue, s)
			}
		})
	}
}

func TestBacktestDefaults_NotOverriddenWhenPresent(t *testing.T) {
	config := `{"symbol":"BTCUSDT","interval":"4h","quantity":0.1}`
	yaml, err := buildBacktestYAML("emacross", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "")
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if !strContains(s, "4h") {
		t.Errorf("user-provided interval=4h should be preserved, got:\n%s", s)
	}
}

func TestBacktestDefaults_BollgridProfitSpread(t *testing.T) {
	config := `{"symbol":"BTCUSDT","gridNumber":8,"quantity":0.001}`
	yaml, err := buildBacktestYAML("bollgrid", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "")
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if !strContains(s, "profitSpread") {
		t.Errorf("bollgrid should have profitSpread injected, got:\n%s", s)
	}
	if !strContains(s, "gridPips") {
		t.Errorf("bollgrid should have gridPips injected, got:\n%s", s)
	}
}

func strContains(s, sub string) bool {
	return strings.Contains(s, sub)
}
