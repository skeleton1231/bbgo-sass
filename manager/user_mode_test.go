package main

import (
	"strings"
	"testing"
)

func TestBuildInstanceYAML_PaperMode_SetsEnvironment(t *testing.T) {
	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModePaper,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		InstanceID: "grid2-BTCUSDT",
	}
	yamlBytes, err := buildInstanceYAML(inst, func(exchange string) bool { return false }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "PAPER_TRADE:") {
		t.Error("expected PAPER_TRADE in YAML for paper mode")
	}
	if !strings.Contains(yaml, `"1"`) {
		t.Error("expected PAPER_TRADE value '1'")
	}
}

func TestBuildInstanceYAML_LiveMode_NoPaperTrade(t *testing.T) {
	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		InstanceID: "grid2-BTCUSDT",
	}
	yamlBytes, err := buildInstanceYAML(inst, func(exchange string) bool { return true }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should NOT appear in live mode YAML")
	}
}

func TestBuildInstanceYAML_CrossExchangePaperMode(t *testing.T) {
	inst := &StrategyInstance{
		UserID:        "test-user",
		Mode:          ModePaper,
		Strategy:      "xmaker",
		CrossExchange: true,
		Symbol:        "BTCUSDT",
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
			{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
		},
		Config:     rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		InstanceID: "xmaker-BTCUSDT",
	}
	yamlBytes, err := buildInstanceYAML(inst, func(exchange string) bool { return false }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "PAPER_TRADE:") {
		t.Error("expected PAPER_TRADE for cross-exchange paper mode")
	}
	if !strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("expected crossExchangeStrategies section")
	}
}

func TestBuildInstanceYAML_CrossExchangeLiveMode(t *testing.T) {
	inst := &StrategyInstance{
		UserID:        "test-user",
		Mode:          ModeLive,
		Strategy:      "xmaker",
		CrossExchange: true,
		Symbol:        "BTCUSDT",
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
			{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
		},
		Config:     rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		InstanceID: "xmaker-BTCUSDT",
	}
	yamlBytes, err := buildInstanceYAML(inst, func(exchange string) bool { return true }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should NOT appear for live cross-exchange mode")
	}
}
