package main

import (
	"strings"
	"testing"
)

func TestBuildUserYAML_SingleExchange(t *testing.T) {
	uc := &UserContainer{
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy: "grid2",
				Exchange: "binance",
				Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "exchange:") {
		t.Error("expected exchange section")
	}
	if !strings.Contains(yaml, "binance") {
		t.Error("expected binance in YAML")
	}
	if !strings.Contains(yaml, "exchangeStrategies:") {
		t.Error("expected exchangeStrategies section")
	}
	if !strings.Contains(yaml, `"on": binance`) {
		t.Error("expected '\"on\": binance' session binding")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy config")
	}
	if strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("should not have crossExchangeStrategies for single exchange")
	}
}

func TestBuildUserYAML_CrossExchange(t *testing.T) {
	uc := &UserContainer{
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				Exchange:      "",
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
				},
				Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.001,"spread":0.001}`),
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("expected crossExchangeStrategies section")
	}
	if !strings.Contains(yaml, "xmaker:") {
		t.Error("expected xmaker strategy in YAML")
	}
	if !strings.Contains(yaml, "binance") {
		t.Error("expected binance session in YAML")
	}
	if !strings.Contains(yaml, "bybit") {
		t.Error("expected bybit session in YAML")
	}
	if !strings.Contains(yaml, "futures: true") {
		t.Error("expected futures: true for hedge session")
	}
	if strings.Contains(yaml, "exchangeStrategies:") {
		t.Error("should not have exchangeStrategies section")
	}
}

func TestBuildUserYAML_Mixed(t *testing.T) {
	uc := &UserContainer{
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{
				Strategy: "grid2",
				Exchange: "binance",
				Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
			},
			{
				Strategy:      "xfunding",
				Exchange:      "",
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "spot", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "futures", Exchange: "okex", EnvVarPrefix: "OKEX", Futures: true},
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

	if !strings.Contains(yaml, "exchangeStrategies:") {
		t.Error("expected exchangeStrategies section for grid2")
	}
	if !strings.Contains(yaml, `"on": binance`) {
		t.Error("expected '\"on\": binance' session binding for grid2")
	}
	if !strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("expected crossExchangeStrategies section for xfunding")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy")
	}
	if !strings.Contains(yaml, "xfunding:") {
		t.Error("expected xfunding strategy")
	}
}

func TestBuildBacktestYAML(t *testing.T) {
	yaml, err := buildBacktestYAML("grid2", rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`), "2024-01-01", "2024-06-01", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yaml)
	if !strings.Contains(s, "sessions:") {
		t.Error("expected sessions section in backtest YAML")
	}
	if !strings.Contains(s, "exchangeStrategies:") {
		t.Error("expected exchangeStrategies in backtest YAML")
	}
	if !strings.Contains(s, "grid2:") {
		t.Error("expected grid2 strategy in backtest YAML")
	}
	if !strings.Contains(s, "BTCUSDT") {
		t.Error("expected symbol BTCUSDT in strategy params")
	}
	if !strings.Contains(s, "backtest:") {
		t.Error("expected backtest section")
	}
	if !strings.Contains(s, "accounts:") {
		t.Error("expected accounts in backtest section")
	}
	if !strings.Contains(s, "2024-01-01") {
		t.Error("expected start time in backtest YAML")
	}
}

func rawJSON(s string) []byte {
	return []byte(s)
}
