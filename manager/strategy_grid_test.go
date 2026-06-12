package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ============================================================
// Grid Strategy Business Tests
// Real quant trading scenarios: BTC range, ETH moderate, altcoin volatile
// ============================================================

func TestStrategy_Grid_ClassicBTCRange(t *testing.T) {
	config := map[string]any{
		"gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000,
		"quantity": 0.001, "profitSpread": 50, "side": "both",
	}
	yaml := mustBuildInstanceYAML(t, "grid", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "gridNumber", "10")
	assertYAMLContains(t, s, "upperPrice", "70000")
	assertYAMLContains(t, s, "lowerPrice", "50000")
	assertYAMLContains(t, s, "quantity", "0.001")
	assertYAMLContains(t, s, "profitSpread", "50")
	assertYAMLContains(t, s, "side", "both")
	assertYAMLContains(t, s, "grid:")
}

func TestStrategy_Grid_OneSidedBuyOnly(t *testing.T) {
	config := map[string]any{
		"gridNumber": 5, "upperPrice": 65000, "lowerPrice": 55000,
		"quantity": 0.01, "profitSpread": 100, "side": "buy",
	}
	yaml := mustBuildInstanceYAML(t, "grid", "BTCUSDT", "binance", "paper", config)
	s := string(yaml)

	assertYAMLContains(t, s, "side", "buy")
	if !strings.Contains(s, "PAPER_TRADE") {
		t.Error("paper mode should have PAPER_TRADE in environment")
	}
}

func TestStrategy_Grid2_AdvancedCompound(t *testing.T) {
	config := map[string]any{
		"gridNumber": 15, "upperPrice": 75000, "lowerPrice": 45000,
		"profitSpread": 0, "quoteInvestment": 5000,
		"compound": true, "earnBase": true,
	}
	yaml := mustBuildInstanceYAML(t, "grid2", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "compound", "true")
	assertYAMLContains(t, s, "earnBase", "true")
	assertYAMLContains(t, s, "quoteInvestment", "5000")
}

func TestStrategy_Grid2_WithRiskManagement(t *testing.T) {
	config := map[string]any{
		"gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000,
		"quoteInvestment": 1000,
		"triggerPrice": 60000, "stopLossPrice": 48000, "takeProfitPrice": 75000,
	}
	yaml := mustBuildInstanceYAML(t, "grid2", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "stopLossPrice")
	assertYAMLContains(t, s, "takeProfitPrice")
	assertYAMLContains(t, s, "triggerPrice")
}

func TestStrategy_Grid2_DefaultsInjected(t *testing.T) {
	config := map[string]any{
		"gridNumber": 10, "upperPrice": 65000, "lowerPrice": 55000,
	}
	yaml := mustBuildInstanceYAMLWithDefaults(t, "grid2", "BTCUSDT", "binance", "paper", config)
	s := string(yaml)
	assertYAMLContains(t, s, "quoteInvestment")
}

func TestStrategy_Bollgrid_DynamicBoundaries(t *testing.T) {
	config := map[string]any{
		"interval": "4h", "gridNumber": 8, "gridPips": 50,
		"quantity": 0.01, "profitSpread": 100,
	}
	yaml := mustBuildInstanceYAML(t, "bollgrid", "ETHUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "interval", "4h")
	assertYAMLContains(t, s, "gridNumber", "8")
	assertYAMLContains(t, s, "bollgrid:")
}

func TestStrategy_Xhedgegrid_HedgeMode(t *testing.T) {
	config := map[string]any{
		"gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000,
		"quoteInvestment": 2000,
	}
	yaml := mustBuildInstanceYAML(t, "xhedgegrid", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "xhedgegrid:")
	assertYAMLContains(t, s, "quoteInvestment", "2000")
}

// ============================================================
// Grid Backtest Scenarios
// ============================================================

func TestBacktest_Grid_FullParams(t *testing.T) {
	config := `{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001,"profitSpread":50,"side":"both"}`
	yaml, err := buildBacktestYAML("grid", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "gridNumber", "10")
	assertYAMLContains(t, s, "side", "both")
}

func TestBacktest_Grid2_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT","gridNumber":10}`
	yaml, err := buildBacktestYAML("grid2", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "quoteInvestment", "1000")
}

// ============================================================
// API: Grid strategy creation flow
// ============================================================

func TestAPICreate_Grid2_PaperSuccess(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "BTC Grid", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"gridNumber": 10, "upperPrice": 65000, "lowerPrice": 55000},
	})
	if w.Code != 201 {
		t.Errorf("grid2 paper creation should succeed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPICreate_Grid2_DuplicateSameSymbolSameConfig_409(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	body := map[string]any{
		"strategy": "grid2", "name": "BTC Grid", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"gridNumber": 10, "upperPrice": 65000, "lowerPrice": 55000},
	}
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", body)
	if w.Code != 201 {
		t.Fatalf("first create should succeed, got %d", w.Code)
	}

	w2 := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", body)
	if w2.Code != 409 {
		t.Errorf("duplicate should return 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

// ============================================================
// Helpers (shared across all strategy_*_test.go files)
// ============================================================

func mustBuildInstanceYAML(t *testing.T, strategy, symbol, exchange, mode string, config map[string]any) []byte {
	t.Helper()
	rawConfig, _ := json.Marshal(config)
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     mode,
		Strategy: strategy,
		Exchange: exchange,
		Symbol:   symbol,
		Config:   rawConfig,
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatalf("buildInstanceYAML(%s): %v", strategy, err)
	}
	return yaml
}

func mustBuildInstanceYAMLWithDefaults(t *testing.T, strategy, symbol, exchange, mode string, config map[string]any) []byte {
	t.Helper()
	rawConfig, _ := json.Marshal(config)
	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     mode,
		Strategy: strategy,
		Exchange: exchange,
		Symbol:   symbol,
		Config:   rawConfig,
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatalf("buildInstanceYAML(%s): %v", strategy, err)
	}
	return yaml
}

func assertYAMLContains(t *testing.T, yaml, field string, value ...string) {
	t.Helper()
	if !strings.Contains(yaml, field) {
		t.Errorf("YAML should contain field %q, got:\n%s", field, yaml)
	}
	if len(value) > 0 && !strings.Contains(yaml, value[0]) {
		t.Errorf("YAML should contain %s=%s, got:\n%s", field, value[0], yaml)
	}
}
