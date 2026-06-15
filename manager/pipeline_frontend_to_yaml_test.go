package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Tests in this file guard the frontend → manager → YAML config pipeline.
//
// The frontend sends NESTED config (via nestConfig + ensureTypes in strategies.ts).
// The manager receives it, deepMerges registry defaults, and generates YAML for
// the bbgo container. A regression in any transformation silently makes the
// strategy run with wrong parameters.
//
// These tests simulate the exact JSON payload the frontend sends and verify
// the resulting YAML contains correct values at every nesting level.

// simulateFrontendConfig mimics what CreateStrategyDialog.handleSubmit produces:
// flat form values → ensureTypes (coerce strings to numbers/bools) → nestConfig
// (flat dotted keys → nested objects). The result is what POST /strategies sends.
func simulateFrontendConfig(flat map[string]any) json.RawMessage {
	nested := make(map[string]any)
	for k, v := range flat {
		if v == nil || v == "" {
			continue
		}
		keys := strings.Split(k, ".")
		current := nested
		for i := 0; i < len(keys)-1; i++ {
			if _, ok := current[keys[i]]; !ok {
				current[keys[i]] = make(map[string]any)
			}
			current = current[keys[i]].(map[string]any)
		}
		current[keys[len(keys)-1]] = v
	}
	b, _ := json.Marshal(nested)
	return b
}

// ============================================================
// Nested Config Deep-Merge Tests
// ============================================================

func TestPipeline_Bollmaker_NestedConfigDeepMergesWithRegistry(t *testing.T) {
	frontendConfig := simulateFrontendConfig(map[string]any{
		"interval":                  "5m",
		"bidQuantity":               0.01,
		"askQuantity":               0.01,
		"defaultBollinger.interval": "5m",
		"defaultBollinger.window":   30,
		"neutralBollinger.interval": "5m",
		"neutralBollinger.window":   30,
	})

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "bollmaker",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   frontendConfig,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "window", "30")
	assertYAMLContains(t, s, "bandWidth", "3")
	assertYAMLContains(t, s, "bidQuantity", "0.01")
}

func TestPipeline_Bollmaker_UserNestedValueOverridesRegistry(t *testing.T) {
	frontendConfig := simulateFrontendConfig(map[string]any{
		"interval":                   "1h",
		"bidQuantity":                0.001,
		"askQuantity":                0.001,
		"defaultBollinger.bandWidth": 5.0,
	})

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "bollmaker",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   frontendConfig,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "bandWidth", "5")
	if strings.Contains(s, "bandWidth: 3") {
		t.Errorf("registry bandWidth=3 should be overridden by user bandWidth=5\n%s", s)
	}
}

func TestPipeline_Pivotshort_NestedExitsArrayFromRegistry(t *testing.T) {
	frontendConfig := simulateFrontendConfig(map[string]any{
		"interval":          "1h",
		"breakLow.interval": "1h",
		"breakLow.window":   7,
	})

	inst := &StrategyInstance{
		UserID:        testUUID,
		Mode:          ModeLive,
		Strategy:      "pivotshort",
		Exchange:      "binance",
		Symbol:        "BTCUSDT",
		Config:        frontendConfig,
		FuturesConfig: &FuturesConfig{Leverage: 5, MarginType: "cross"},
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "exits")
	assertYAMLContains(t, s, "roiStopLoss")
	assertYAMLContains(t, s, "percentage")
}

// ============================================================
// Empty / Missing Config Edge Cases
// ============================================================

func TestPipeline_EmptyConfigGetsAllRegistryDefaults(t *testing.T) {
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "grid",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   json.RawMessage(`{}`),
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "gridNumber", "10")
	assertYAMLContains(t, s, "upperPrice", "70000")
	assertYAMLContains(t, s, "lowerPrice", "50000")
}

func TestPipeline_NullConfigHandledGracefully(t *testing.T) {
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "grid",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   json.RawMessage(`null`),
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "gridNumber", "10")
}

func TestPipeline_UnknownStrategyNoRegistryDefaults(t *testing.T) {
	frontendConfig := simulateFrontendConfig(map[string]any{
		"interval": "5m",
		"quantity": 0.01,
	})

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "custom-unknown-strategy",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   frontendConfig,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "custom-unknown-strategy:")
	assertYAMLContains(t, s, "interval", "5m")
	assertYAMLContains(t, s, "quantity", "0.01")
}

// ============================================================
// Symbol Extraction Pipeline
// ============================================================

func TestPipeline_SymbolFlowsToSessionAndExchange(t *testing.T) {
	frontendConfig := simulateFrontendConfig(map[string]any{
		"symbol":   "ETHUSDT",
		"interval": "1h",
	})

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "emacross",
		Exchange: "binance",
		Symbol:   "ETHUSDT",
		Config:   frontendConfig,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "ETHUSDT")
}

func TestPipeline_SymbolFallbackToBTCUSDT(t *testing.T) {
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "emacross",
		Exchange: "binance",
		Symbol:   "",
		Config:   json.RawMessage(`{"interval":"1h"}`),
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "BTCUSDT")
}

// ============================================================
// Paper Mode Pipeline
// ============================================================

func TestPipeline_PaperModeSetsPaperBalancesAndPublicOnly(t *testing.T) {
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModePaper,
		Strategy: "grid",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   json.RawMessage(`{"gridNumber":5}`),
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "paperBalances")
	assertYAMLContains(t, s, "publicOnly")
	assertYAMLContains(t, s, "PAPER_TRADE")
}

func TestPipeline_LiveModeWithCredsNotPublicOnly(t *testing.T) {
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "grid",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   json.RawMessage(`{"gridNumber":5}`),
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if strings.Contains(s, "publicOnly: true") {
		t.Errorf("live mode with credentials should not set publicOnly: true\n%s", s)
	}
}

// ============================================================
// Futures Config Propagation
// ============================================================

func TestPipeline_FuturesStrategySetsSessionFutures(t *testing.T) {
	inst := &StrategyInstance{
		UserID:        testUUID,
		Mode:          ModeLive,
		Strategy:      "pivotshort",
		Exchange:      "binance",
		Symbol:        "BTCUSDT",
		Config:        json.RawMessage(`{"interval":"1h"}`),
		FuturesConfig: &FuturesConfig{Leverage: 10, MarginType: "cross"},
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "futures: true")
	assertYAMLContains(t, s, "symbolLeverage:")
	assertYAMLContains(t, s, "BTCUSDT: 10")
}

func TestPipeline_FuturesStrategyEnablesPositionSync(t *testing.T) {
	inst := &StrategyInstance{
		UserID:        testUUID,
		Mode:          ModeLive,
		Strategy:      "pivotshort",
		Exchange:      "binance",
		Symbol:        "BTCUSDT",
		Config:        json.RawMessage(`{"interval":"1h"}`),
		FuturesConfig: &FuturesConfig{Leverage: 5},
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	assertYAMLContains(t, s, "futuresPosition: true")
	assertYAMLContains(t, s, "futuresPositionSyncInterval: 30s")
}

func TestPipeline_SpotStrategyNoFuturesPositionSync(t *testing.T) {
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "grid",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   json.RawMessage(`{"gridNumber":5}`),
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if strings.Contains(s, "futuresPosition: true") {
		t.Errorf("spot strategy should not enable futuresPosition sync\n%s", s)
	}
}
