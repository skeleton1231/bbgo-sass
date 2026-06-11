package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// ============================================================
// Market Maker Strategy Business Tests
// ============================================================

func TestStrategy_Bollmaker_BTCModerateSpread(t *testing.T) {
	config := map[string]any{
		"interval": "5m", "bidQuantity": 0.01, "askQuantity": 0.01,
		"spread": 0.002, "minProfitSpread": 0.001,
		"maxExposurePosition": 0.1, "disableShort": false,
	}
	yaml := mustBuildInstanceYAML(t, "bollmaker", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "bollmaker:")
	assertYAMLContains(t, s, "bidQuantity", "0.01")
	assertYAMLContains(t, s, "spread", "0.002")
}

func TestStrategy_Bollmaker_DisableShort(t *testing.T) {
	config := map[string]any{
		"interval": "1m", "bidQuantity": 0.001, "askQuantity": 0.001,
		"disableShort": true, "tradeInBand": true, "shadowProtection": true,
	}
	yaml := mustBuildInstanceYAML(t, "bollmaker", "ETHUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "disableShort", "true")
	assertYAMLContains(t, s, "shadowProtection", "true")
}

// TestStrategy_Bollmaker_RegistryFillsNestedBandWidth is a regression test for the
// collapsed-bollinger-bands bug. The user's stored config supplied bollinger
// interval+window but omitted bandWidth; without registry deep-merge, bandWidth
// stayed 0 and BOLL bands collapsed to the SMA line.
//
// Verifies that buildInstanceYAML, when given a registry with the production
// bollmaker defaults, fills bandWidth into both defaultBollinger and
// neutralBollinger even when the user config omits it entirely.
func TestStrategy_Bollmaker_RegistryFillsNestedBandWidth(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "bidQuantity": 0.001, "askQuantity": 0.001,
		"defaultBollinger": map[string]any{"interval": "1h", "window": 20},
		"neutralBollinger": map[string]any{"interval": "1h", "window": 20},
	}
	rawConfig, _ := json.Marshal(config)
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "bollmaker",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   rawConfig,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "bandWidth:") {
		t.Errorf("YAML missing bandWidth — registry defaults did not deep-merge into nested bollinger config\n%s", s)
	}
	assertYAMLContains(t, s, "bandWidth", "3")
}

// TestStrategy_Bollmaker_RegistryBandWidthNotOverriddenByUser verifies the inverse:
// when the user explicitly sets a bandWidth, that value wins over the registry
// default. Guards against the deep-merge silently overwriting intentional user choices.
func TestStrategy_Bollmaker_RegistryBandWidthNotOverriddenByUser(t *testing.T) {
	config := map[string]any{
		"interval": "1h",
		"defaultBollinger": map[string]any{
			"interval": "1h", "window": 30, "bandWidth": 1.5,
		},
		"neutralBollinger": map[string]any{
			"interval": "1h", "window": 30, "bandWidth": 2.5,
		},
	}
	rawConfig, _ := json.Marshal(config)
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "bollmaker",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   rawConfig,
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "window: 30") {
		t.Errorf("user-supplied window=30 should survive deep-merge\n%s", s)
	}
	if !strings.Contains(s, "bandWidth: 1.5") {
		t.Errorf("user-supplied defaultBollinger.bandWidth=1.5 should survive deep-merge\n%s", s)
	}
}

func TestStrategy_Linregmaker_TrendAware(t *testing.T) {
	config := map[string]any{
		"interval": "5m", "bidQuantity": 0.005, "askQuantity": 0.005,
		"spread": 0.001, "maxExposurePosition": 0.05,
	}
	yaml := mustBuildInstanceYAML(t, "linregmaker", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "linregmaker:")
	assertYAMLContains(t, s, "maxExposurePosition", "0.05")
}

func TestStrategy_Rsmaker_RelativeStrength(t *testing.T) {
	config := map[string]any{
		"interval": "15m", "bidQuantity": 0.01, "askQuantity": 0.01,
		"spread": 0.003, "minProfitSpread": 0.001,
	}
	yaml := mustBuildInstanceYAML(t, "rsmaker", "ETHUSDT", "okex", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "rsmaker:")
	assertYAMLContains(t, s, "okex")
}

func TestStrategy_Fixedmaker_SimpleSpread(t *testing.T) {
	config := map[string]any{
		"interval": "1m", "quantity": 0.001, "halfSpread": 0.0005,
	}
	yaml := mustBuildInstanceYAML(t, "fixedmaker", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "fixedmaker:")
	assertYAMLContains(t, s, "halfSpread", "0.0005")
}

func TestStrategy_Fixedmaker_DefaultsInjected(t *testing.T) {
	config := map[string]any{"symbol": "BTCUSDT"}
	yaml := mustBuildInstanceYAMLWithDefaults(t, "fixedmaker", "BTCUSDT", "binance", "paper", config)
	s := string(yaml)
	assertYAMLContains(t, s, "halfSpread")
}

func TestStrategy_Fmaker_FlexibleMaker(t *testing.T) {
	config := map[string]any{
		"interval": "5m", "quantity": 0.01, "spread": 0.002,
	}
	yaml := mustBuildInstanceYAML(t, "fmaker", "BTCUSDT", "bybit", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "fmaker:")
	assertYAMLContains(t, s, "bybit")
}

func TestStrategy_Scmaker_BollingerSafety(t *testing.T) {
	config := map[string]any{
		"interval": "5m", "window": 20, "k": 2.0,
		"numOfLiquidityLayers": 3, "maxExposure": 0.1,
	}
	yaml := mustBuildInstanceYAML(t, "scmaker", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "scmaker:")
	assertYAMLContains(t, s, "numOfLiquidityLayers", "3")
}

func TestStrategy_Audacitymaker_OrderFlow(t *testing.T) {
	config := map[string]any{"interval": "1m", "window": 100}
	yaml := mustBuildInstanceYAML(t, "audacitymaker", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "audacitymaker:")
	assertYAMLContains(t, s, "window", "100")
}

func TestStrategy_Liquiditymaker_Layered(t *testing.T) {
	config := map[string]any{
		"numOfLiquidityLayers": 5, "spread": 0.001,
		"askLiquidityAmount": 0.01, "bidLiquidityAmount": 0.01,
		"liquidityPriceRange": 0.005, "maxPositionExposure": 0.1,
	}
	yaml := mustBuildInstanceYAML(t, "liquiditymaker", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "liquiditymaker:")
	assertYAMLContains(t, s, "numOfLiquidityLayers", "5")
}

// ============================================================
// LiveOnly enforcement for market makers
// ============================================================

func TestAPICreate_Bollmaker_PaperBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "bollmaker", "name": "Boll Maker", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"bidQuantity": 0.01, "askQuantity": 0.01},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("bollmaker in paper mode should be blocked (liveOnly), got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPICreate_Liquiditymaker_PaperBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "liquiditymaker", "name": "Liq Maker", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"numOfLiquidityLayers": 3},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("liquiditymaker in paper mode should be blocked (liveOnly), got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================
// Market Maker Backtest Scenarios
// ============================================================

func TestBacktest_Fixedmaker_NumbersNotQuoted(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("fixedmaker", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if strings.Contains(s, `halfSpread: "`) {
		t.Errorf("halfSpread should not be quoted:\n%s", s)
	}
}

func TestBacktest_Fmaker_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("fmaker", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "spread", "0.001")
}

func TestBacktest_Bollmaker_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"ETHUSDT","bidQuantity":0.01,"askQuantity":0.01}`
	yaml, err := buildBacktestYAML("bollmaker", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "interval", "1h")
}
