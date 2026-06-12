package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Tests in this file guard the leverage propagation chain:
//
//   Frontend FuturesConfig.leverage
//     → api.go CreateStrategy injects config.leverage
//       → buildInstanceYAML emits session.symbolLeverage + strategy params leverage
//
// A regression in any link silently makes the strategy run at the wrong leverage,
// causing incorrect position sizing and liquidation math.

func TestLeverageChain_FuturesConfigPropagatesToYAML(t *testing.T) {
	rawConfig, _ := json.Marshal(map[string]any{
		"interval": "1h", "quantity": 0.001, "symbol": "BTCUSDT",
	})
	inst := &StrategyInstance{
		UserID:        testUUID,
		Mode:          ModePaper,
		Strategy:      "pivotshort",
		Exchange:      "binance",
		Symbol:        "BTCUSDT",
		Config:        rawConfig,
		FuturesConfig: &FuturesConfig{Leverage: 10, MarginType: "cross"},
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "symbolLeverage:") {
		t.Errorf("expected symbolLeverage in YAML for futures strategy\n%s", s)
	}
	if !strings.Contains(s, "BTCUSDT: 10") {
		t.Errorf("expected symbolLeverage BTCUSDT: 10 in YAML\n%s", s)
	}
	if !strings.Contains(s, "leverage: 10") {
		t.Errorf("expected strategy params leverage: 10 in YAML\n%s", s)
	}
}

func TestLeverageChain_FuturesConfigOverridesConfigLeverage(t *testing.T) {
	rawConfig, _ := json.Marshal(map[string]any{
		"interval": "1h", "quantity": 0.001, "symbol": "BTCUSDT", "leverage": 3,
	})
	inst := &StrategyInstance{
		UserID:        testUUID,
		Mode:          ModePaper,
		Strategy:      "pivotshort",
		Exchange:      "binance",
		Symbol:        "BTCUSDT",
		Config:        rawConfig,
		FuturesConfig: &FuturesConfig{Leverage: 7},
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "BTCUSDT: 7") {
		t.Errorf("FuturesConfig.Leverage=7 should override config leverage=3\n%s", s)
	}
}

func TestLeverageChain_ConfigLeverageFallbackWhenNoFuturesConfig(t *testing.T) {
	rawConfig, _ := json.Marshal(map[string]any{
		"interval": "1h", "quantity": 0.001, "symbol": "BTCUSDT", "leverage": 5,
	})
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModePaper,
		Strategy: "pivotshort",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   rawConfig,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "BTCUSDT: 5") {
		t.Errorf("config leverage=5 should be used when FuturesConfig is nil\n%s", s)
	}
}

func TestLeverageChain_IsolatedMarginSetsSessionFlags(t *testing.T) {
	rawConfig, _ := json.Marshal(map[string]any{
		"interval": "1h", "quantity": 0.001, "symbol": "BTCUSDT",
	})
	inst := &StrategyInstance{
		UserID:        testUUID,
		Mode:          ModePaper,
		Strategy:      "pivotshort",
		Exchange:      "binance",
		Symbol:        "BTCUSDT",
		Config:        rawConfig,
		FuturesConfig: &FuturesConfig{Leverage: 5, MarginType: "isolated"},
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "isolatedFutures: true") {
		t.Errorf("marginType=isolated should set isolatedFutures: true\n%s", s)
	}
	if !strings.Contains(s, "isolatedFuturesSymbol: BTCUSDT") {
		t.Errorf("marginType=isolated should set isolatedFuturesSymbol\n%s", s)
	}
}

func TestLeverageChain_CreateStrategyInjectsConfigLeverage(t *testing.T) {
	var raw map[string]any
	originalConfig, _ := json.Marshal(map[string]any{
		"interval": "1h", "quantity": 0.001, "symbol": "BTCUSDT",
	})
	if err := json.Unmarshal(originalConfig, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	fc := &FuturesConfig{Leverage: 15, MarginType: "cross"}
	if fc.Leverage > 0 {
		raw["leverage"] = fc.Leverage
		if fc.MarginType != "" {
			raw["marginType"] = fc.MarginType
		}
	}
	injected, _ := json.Marshal(raw)

	inst := &StrategyInstance{
		UserID:        testUUID,
		Mode:          ModePaper,
		Strategy:      "pivotshort",
		Exchange:      "binance",
		Symbol:        "BTCUSDT",
		Config:        injected,
		FuturesConfig: fc,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "leverage: 15") {
		t.Errorf("injected config leverage=15 should appear in strategy params\n%s", s)
	}
	if !strings.Contains(s, "BTCUSDT: 15") {
		t.Errorf("injected leverage=15 should propagate to symbolLeverage\n%s", s)
	}
}

func TestLeverageChain_CrossExchangeFuturesConfigAppliesToSession(t *testing.T) {
	rawConfig, _ := json.Marshal(map[string]any{
		"symbol": "BTCUSDT", "spread": 0.001,
	})
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "xmaker",
		Symbol:   "BTCUSDT",
		Config:   rawConfig,
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance", Futures: true},
			{Name: "hedge", Exchange: "bybit", Futures: true},
		},
		CrossExchange: true,
		FuturesConfig: &FuturesConfig{Leverage: 8, MarginType: "cross"},
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "BTCUSDT: 8") {
		t.Errorf("cross-exchange FuturesConfig.Leverage=8 should set symbolLeverage\n%s", s)
	}
	if !strings.Contains(s, "leverage: 8") {
		t.Errorf("cross-exchange FuturesConfig should mirror leverage into params\n%s", s)
	}
}
