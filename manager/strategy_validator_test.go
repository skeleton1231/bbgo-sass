package main

import (
	"encoding/json"
	"testing"
)

// ============================================================
// Grid Strategy Validation
// ============================================================

func TestValidate_Grid2_MissingQuantity(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000}`)
	warnings := ValidateStrategyConfig("grid2", config)
	if len(warnings) == 0 {
		t.Fatal("expected warnings for grid2 without quantity/investment")
	}
	found := false
	for _, w := range warnings {
		if w.ID == "missing_quantity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_quantity warning, got: %v", warnings)
	}
}

func TestValidate_Grid2_ValidWithQuantity(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`)
	warnings := ValidateStrategyConfig("grid2", config)
	assertNoWarning(t, warnings, "missing_quantity")
}

func TestValidate_Grid2_ValidWithQuoteInvestment(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quoteInvestment":1000}`)
	warnings := ValidateStrategyConfig("grid2", config)
	assertNoWarning(t, warnings, "missing_quantity")
}

func TestValidate_Grid_InvalidPriceRange(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":50000,"lowerPrice":70000,"quantity":0.001}`)
	warnings := ValidateStrategyConfig("grid", config)
	found := false
	for _, w := range warnings {
		if w.ID == "invalid_price_range" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected invalid_price_range, got: %v", warnings)
	}
}

func TestValidate_Grid_TooFewGrids(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":1,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`)
	warnings := ValidateStrategyConfig("grid", config)
	found := false
	for _, w := range warnings {
		if w.ID == "grid_too_few" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected grid_too_few, got: %v", warnings)
	}
}

func TestValidate_Grid_MissingProfitSpread(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`)
	warnings := ValidateStrategyConfig("grid", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_profit_spread" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_profit_spread for grid, got: %v", warnings)
	}
}

func TestValidate_Bollgrid_MissingGridPips(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1h","quantity":0.001,"profitSpread":50}`)
	warnings := ValidateStrategyConfig("bollgrid", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_grid_pips" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_grid_pips, got: %v", warnings)
	}
}

// ============================================================
// Maker Strategy Validation
// ============================================================

func TestValidate_Fixedmaker_MissingHalfSpread(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1m","quantity":0.001}`)
	warnings := ValidateStrategyConfig("fixedmaker", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_spread" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_spread, got: %v", warnings)
	}
}

func TestValidate_Fixedmaker_Valid(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1m","quantity":0.001,"halfSpread":0.0005}`)
	warnings := ValidateStrategyConfig("fixedmaker", config)
	assertNoWarning(t, warnings, "missing_spread")
	assertNoWarning(t, warnings, "missing_quantity")
}

func TestValidate_Bollmaker_MissingBidAskQuantity(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"5m"}`)
	warnings := ValidateStrategyConfig("bollmaker", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_quantity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_quantity for bollmaker without bid/ask qty, got: %v", warnings)
	}
}

func TestValidate_Bollmaker_ValidWithBidAsk(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"5m","bidQuantity":0.01,"askQuantity":0.01}`)
	warnings := ValidateStrategyConfig("bollmaker", config)
	assertNoWarning(t, warnings, "missing_quantity")
}

func TestValidate_Fmaker_MissingSpread(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"5m","quantity":0.01}`)
	warnings := ValidateStrategyConfig("fmaker", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_spread" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_spread, got: %v", warnings)
	}
}

func TestValidate_Scmaker_MissingQuantity(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"5m","window":20,"k":2.0}`)
	warnings := ValidateStrategyConfig("scmaker", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_quantity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_quantity for scmaker, got: %v", warnings)
	}
}

// ============================================================
// Trend Strategy Validation
// ============================================================

func TestValidate_Supertrend_MissingQuantity(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1h"}`)
	warnings := ValidateStrategyConfig("supertrend", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_quantity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_quantity, got: %v", warnings)
	}
}

func TestValidate_Supertrend_Valid(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1h","quantity":0.001}`)
	warnings := ValidateStrategyConfig("supertrend", config)
	assertNoWarning(t, warnings, "missing_quantity")
}

func TestValidate_Emacross_MissingWindows(t *testing.T) {
	config := rawJSON(`{"symbol":"ETHUSDT","interval":"4h"}`)
	warnings := ValidateStrategyConfig("emacross", config)
	foundFW := false
	foundSW := false
	for _, w := range warnings {
		if w.ID == "missing_fast_window" {
			foundFW = true
		}
		if w.ID == "missing_slow_window" {
			foundSW = true
		}
	}
	if !foundFW || !foundSW {
		t.Errorf("expected missing fast/slow window, got: %v", warnings)
	}
}

func TestValidate_Emacross_Valid(t *testing.T) {
	config := rawJSON(`{"symbol":"ETHUSDT","interval":"4h","fastWindow":7,"slowWindow":25,"quantity":0.01}`)
	warnings := ValidateStrategyConfig("emacross", config)
	assertNoWarning(t, warnings, "missing_fast_window")
	assertNoWarning(t, warnings, "missing_slow_window")
}

func TestValidate_Pivotshort_MissingInterval(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`)
	warnings := ValidateStrategyConfig("pivotshort", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_interval" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_interval, got: %v", warnings)
	}
}

func TestValidate_Swing_MissingMAWindow(t *testing.T) {
	config := rawJSON(`{"symbol":"ETHUSDT","interval":"4h","baseQuantity":0.0001}`)
	warnings := ValidateStrategyConfig("swing", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_ma_window" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_ma_window, got: %v", warnings)
	}
}

// ============================================================
// DCA / Schedule Strategy Validation
// ============================================================

func TestValidate_DCA_MissingBudget(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","investmentInterval":"1h"}`)
	warnings := ValidateStrategyConfig("dca", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_budget" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_budget, got: %v", warnings)
	}
}

func TestValidate_DCA_Valid(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","investmentInterval":"1h","budget":500}`)
	warnings := ValidateStrategyConfig("dca", config)
	assertNoWarning(t, warnings, "missing_budget")
}

func TestValidate_Schedule_MissingSide(t *testing.T) {
	config := rawJSON(`{"symbol":"ETHUSDT","interval":"1h","quantity":0.01}`)
	warnings := ValidateStrategyConfig("schedule", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_side" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_side, got: %v", warnings)
	}
}

// ============================================================
// Cross-Exchange Strategy Validation
// ============================================================

func TestValidate_Xmaker_MissingQuantity(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","spread":0.001}`)
	warnings := ValidateStrategyConfig("xmaker", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_quantity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_quantity for xmaker, got: %v", warnings)
	}
}

// ============================================================
// Futures Strategy Validation
// ============================================================

func TestValidate_Pivotshort_FuturesWithoutLeverage(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1h","quantity":0.001}`)
	warnings := ValidateStrategyConfig("pivotshort", config)
	found := false
	for _, w := range warnings {
		if w.ID == "futures_no_leverage" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected futures_no_leverage for pivotshort, got: %v", warnings)
	}
}

func TestValidate_Pivotshort_FuturesWithLeverage(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1h","quantity":0.001,"leverage":5}`)
	warnings := ValidateStrategyConfig("pivotshort", config)
	assertNoWarning(t, warnings, "futures_no_leverage")
}

// ============================================================
// Unknown / No-Validation Strategy
// ============================================================

func TestValidate_UnknownStrategy_NoWarnings(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT"}`)
	warnings := ValidateStrategyConfig("unknown_strategy", config)
	if len(warnings) != 0 {
		t.Errorf("unknown strategy should have no warnings, got: %v", warnings)
	}
}

func TestValidate_Techsignal_NoQuantityNeeded(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","interval":"1h"}`)
	warnings := ValidateStrategyConfig("techsignal", config)
	assertNoWarning(t, warnings, "missing_quantity")
}

func TestValidate_Random_MissingSchedule(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`)
	warnings := ValidateStrategyConfig("random", config)
	found := false
	for _, w := range warnings {
		if w.ID == "missing_schedule" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_schedule, got: %v", warnings)
	}
}

// ============================================================
// Defaults Merge Integration
// ============================================================

func TestValidate_Grid2_DefaultsMergeCoversQuantity(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000}`)
	merged := deepMerge(testRegistry.GetDefaults("grid2"), parseConfigMap(config))
	mergedJSON, _ := json.Marshal(merged)
	warnings := ValidateStrategyConfig("grid2", mergedJSON)
	assertNoWarning(t, warnings, "missing_quantity")
}

func TestValidate_Fixedmaker_DefaultsCoverHalfSpread(t *testing.T) {
	config := rawJSON(`{"symbol":"BTCUSDT"}`)
	merged := deepMerge(testRegistry.GetDefaults("fixedmaker"), parseConfigMap(config))
	mergedJSON, _ := json.Marshal(merged)
	warnings := ValidateStrategyConfig("fixedmaker", mergedJSON)
	assertNoWarning(t, warnings, "missing_spread")
	assertNoWarning(t, warnings, "missing_quantity")
}

// ============================================================
// Data-Driven Validation (from registry fields)
// ============================================================

func TestValidate_DataDriven_Grid2MissingQuantity(t *testing.T) {
	orig := globalFieldsForTest
	globalFieldsForTest = map[string][]FieldDef{
		"grid2": {
			{Key: "symbol", Type: "text", Required: true},
			{Key: "quantity", Type: "number", Required: true, Min: ptrFloat(0.00001)},
			{Key: "gridNumber", Type: "number", Required: true, Min: ptrFloat(2)},
			{Key: "upperPrice", Type: "number", Required: true},
			{Key: "lowerPrice", Type: "number", Required: true},
			{Key: "profitSpread", Type: "number", Required: false},
		},
	}
	defer func() { globalFieldsForTest = orig }()

	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000}`)
	warnings := ValidateStrategyConfig("grid2", config)

	found := false
	for _, w := range warnings {
		if w.ID == "missing_quantity" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_quantity from data-driven validation, got: %v", warnings)
	}
}

func TestValidate_DataDriven_Grid2Valid(t *testing.T) {
	orig := globalFieldsForTest
	globalFieldsForTest = map[string][]FieldDef{
		"grid2": {
			{Key: "symbol", Type: "text", Required: true},
			{Key: "quantity", Type: "number", Required: true, Min: ptrFloat(0.00001)},
			{Key: "gridNumber", Type: "number", Required: true, Min: ptrFloat(2)},
		},
	}
	defer func() { globalFieldsForTest = orig }()

	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`)
	warnings := ValidateStrategyConfig("grid2", config)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for valid grid2, got: %v", warnings)
	}
}

func TestValidate_DataDriven_InvalidPriceRange(t *testing.T) {
	orig := globalFieldsForTest
	globalFieldsForTest = map[string][]FieldDef{
		"grid2": {
			{Key: "quantity", Type: "number", Required: true},
		},
	}
	defer func() { globalFieldsForTest = orig }()

	config := rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":50000,"lowerPrice":70000,"quantity":0.001}`)
	warnings := ValidateStrategyConfig("grid2", config)

	found := false
	for _, w := range warnings {
		if w.ID == "invalid_price_range" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected invalid_price_range from cross-cutting check, got: %v", warnings)
	}
}

func TestValidate_DataDriven_TextFieldRequired(t *testing.T) {
	orig := globalFieldsForTest
	globalFieldsForTest = map[string][]FieldDef{
		"schedule": {
			{Key: "side", Type: "select", Required: true},
			{Key: "quantity", Type: "number", Required: true},
		},
	}
	defer func() { globalFieldsForTest = orig }()

	config := rawJSON(`{"symbol":"ETHUSDT","interval":"1h","quantity":0.01}`)
	warnings := ValidateStrategyConfig("schedule", config)

	found := false
	for _, w := range warnings {
		if w.ID == "missing_side" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_side from data-driven validation, got: %v", warnings)
	}
}

func ptrFloat(v float64) *float64 { return &v }

// ============================================================
// Helper
// ============================================================

func parseConfigMap(data []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func assertNoWarning(t *testing.T, warnings []StrategyWarning, id string) {
	t.Helper()
	for _, w := range warnings {
		if w.ID == id {
			t.Errorf("did not expect warning %q but got it: %s", id, w.Message)
		}
	}
}
