package main

import (
	"strings"
	"testing"
)

func TestBuildUserYAML_PaperMode_SetsEnvironment(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModePaper,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy: "grid2",
				Exchange: "binance",
				Mode:     "paper",
				Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return false })
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

func TestBuildUserYAML_LiveMode_NoPaperTrade(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy: "grid2",
				Exchange: "binance",
				Mode:     "live",
				Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return true })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should NOT appear in live mode YAML")
	}
}

func TestBuildUserYAML_PaperContainer_MultipleStrategies(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModePaper,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy: "grid2",
				Exchange: "binance",
				Mode:     "paper",
				Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
			{
				Strategy: "dca",
				Exchange: "binance",
				Mode:     "paper",
				Config:   rawJSON(`{"symbol":"ETHUSDT"}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "PAPER_TRADE:") {
		t.Error("expected PAPER_TRADE for paper container with multiple strategies")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy")
	}
	if !strings.Contains(yaml, "dca:") {
		t.Error("expected dca strategy")
	}
}

func TestBuildUserYAML_CrossExchangePaperMode(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModePaper,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "paper",
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
				},
				Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return false })
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

func TestBuildUserYAML_CrossExchangeLiveMode(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "live",
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
				},
				Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return true })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should NOT appear for live cross-exchange mode")
	}
}

func TestBuildUserYAML_MultipleStrategies_AllLive(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy: "grid2",
				Exchange: "binance",
				Mode:     "live",
				Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
			{
				Strategy: "dca",
				Exchange: "binance",
				Mode:     "live",
				Config:   rawJSON(`{"symbol":"ETHUSDT"}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return true })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should NOT appear when all strategies are live")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy")
	}
	if !strings.Contains(yaml, "dca:") {
		t.Error("expected dca strategy")
	}
}
