package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// ============================================================
// Trend Following Strategy Business Tests
// ============================================================

func TestStrategy_Supertrend_BTCBreakout(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "quantity": 0.001,
		"supertrendMultiplier": 3.0, "takeProfitAtrMultiplier": 2.0,
		"stopByReversedSupertrend": true,
	}
	yaml := mustBuildInstanceYAML(t, "supertrend", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "supertrend:")
	assertYAMLContains(t, s, "supertrendMultiplier", "3")
}

func TestStrategy_Emacross_GoldenCross(t *testing.T) {
	config := map[string]any{
		"interval": "4h", "fastWindow": 7, "slowWindow": 25, "quantity": 0.01,
	}
	yaml := mustBuildInstanceYAML(t, "emacross", "ETHUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "emacross:")
	assertYAMLContains(t, s, "fastWindow", "7")
	assertYAMLContains(t, s, "slowWindow", "25")
}

func TestStrategy_Trendtrader_LineBreakout(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "quantity": 0.01, "leverage": 3,
	}
	yaml := mustBuildInstanceYAML(t, "trendtrader", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "trendtrader:")
	assertYAMLContains(t, s, "leverage", "3")
}

func TestStrategy_Atrpin_PinBarDetection(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "quantity": 0.001, "multiplier": 2.0,
	}
	yaml := mustBuildInstanceYAML(t, "atrpin", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "atrpin:")
	assertYAMLContains(t, s, "multiplier", "2")
}

func TestStrategy_Drift_MAPrediction(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "window": 20, "quantity": 0.001,
		"stoploss": 0.02, "useStopLoss": true, "useAtr": true,
	}
	yaml := mustBuildInstanceYAML(t, "drift", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "drift:")
	assertYAMLContains(t, s, "stoploss", "0.02")
}

func TestStrategy_Elliottwave_Oscillator(t *testing.T) {
	config := map[string]any{
		"interval": "4h", "stoploss": 0.03, "useHeikinAshi": true,
	}
	yaml := mustBuildInstanceYAML(t, "elliottwave", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "elliottwave:")
	assertYAMLContains(t, s, "useHeikinAshi", "true")
}

func TestStrategy_Factorzoo_MultiFactor(t *testing.T) {
	config := map[string]any{"interval": "1h", "window": 30}
	yaml := mustBuildInstanceYAML(t, "factorzoo", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "factorzoo:")
}

// ============================================================
// Mean Reversion Strategy Business Tests
// ============================================================

func TestStrategy_Pivotshort_ShortBreakdown(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "quantity": 0.001, "leverage": 2,
	}
	yaml := mustBuildInstanceYAML(t, "pivotshort", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "pivotshort:")
	assertYAMLContains(t, s, "leverage", "2")
}

func TestStrategy_Swing_MACrossover(t *testing.T) {
	config := map[string]any{
		"interval": "4h", "baseQuantity": 0.0001,
		"movingAverageType": "EMA", "movingAverageWindow": 50,
	}
	yaml := mustBuildInstanceYAML(t, "swing", "ETHUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "swing:")
	assertYAMLContains(t, s, "movingAverageType", "EMA")
}

func TestStrategy_EwoDgtrd_Divergence(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "sigWin": 5, "stoploss": 0.02,
		"cciStochFilterHigh": 200, "cciStochFilterLow": -200,
	}
	yaml := mustBuildInstanceYAML(t, "ewo_dgtrd", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "ewo_dgtrd:")
	assertYAMLContains(t, s, "sigWin", "5")
}

func TestStrategy_Harmonic_PatternDetection(t *testing.T) {
	config := map[string]any{
		"interval": "4h", "window": 20, "quantity": 0.001,
	}
	yaml := mustBuildInstanceYAML(t, "harmonic", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "harmonic:")
}

func TestStrategy_Irr_NegativeReturn(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "window": 20, "quantity": 0.001,
	}
	yaml := mustBuildInstanceYAML(t, "irr", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "irr:")
}

// ============================================================
// DCA Strategy Business Tests
// ============================================================

func TestStrategy_DCA_DailyAccumulation(t *testing.T) {
	config := map[string]any{
		"investmentInterval": "1h", "budget": 500, "budgetPeriod": "day",
	}
	yaml := mustBuildInstanceYAML(t, "dca", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "dca:")
	assertYAMLContains(t, s, "budget", "500")
}

func TestStrategy_DCA_DefaultsInjected(t *testing.T) {
	config := map[string]any{"symbol": "BTCUSDT"}
	yaml := mustBuildInstanceYAMLWithDefaults(t, "dca", "BTCUSDT", "binance", "paper", config)
	s := string(yaml)
	assertYAMLContains(t, s, "budget")
}

func TestStrategy_DCA2_AdvancedWithTakeProfit(t *testing.T) {
	config := map[string]any{
		"quoteInvestment": 100, "maxOrderCount": 10,
		"priceDeviation": 0.05, "takeProfitRatio": 0.1,
	}
	yaml := mustBuildInstanceYAML(t, "dca2", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "dca2:")
	assertYAMLContains(t, s, "takeProfitRatio", "0.1")
}

func TestStrategy_Autobuy_Scheduled(t *testing.T) {
	config := map[string]any{
		"schedule": "0 8 * * *", "quantity": 0.01,
	}
	yaml := mustBuildInstanceYAML(t, "autobuy", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "autobuy:")
	assertYAMLContains(t, s, "schedule")
}

func TestStrategy_Schedule_WeeklyBuy(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "side": "buy", "quantity": 0.01,
		"useLimitOrder": true,
	}
	yaml := mustBuildInstanceYAML(t, "schedule", "ETHUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "schedule:")
	assertYAMLContains(t, s, "side", "buy")
}

// ============================================================
// Volatility Strategy Business Tests
// ============================================================

func TestStrategy_Flashcrash_BuyTheDip(t *testing.T) {
	config := map[string]any{
		"interval": "1h", "gridNumber": 10, "percentage": 0.05,
		"baseQuantity": 0.001,
	}
	yaml := mustBuildInstanceYAML(t, "flashcrash", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "flashcrash:")
	assertYAMLContains(t, s, "percentage", "0.05")
}

func TestStrategy_Xvs_VolumeSurge(t *testing.T) {
	config := map[string]any{
		"quantity": 0.01, "maxExposure": 0.1,
		"volumeInterval": "5m", "volumeThreshold": 3.0, "stoploss": 0.02,
	}
	yaml := mustBuildInstanceYAML(t, "xvs", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "xvs:")
	assertYAMLContains(t, s, "volumeThreshold", "3")
}

// ============================================================
// Other Strategy Business Tests
// ============================================================

func TestStrategy_Wall_SellWall(t *testing.T) {
	config := map[string]any{
		"side": "sell", "interval": "1h", "quantity": 1.0,
		"numLayers": 5, "layerSpread": 0.001,
	}
	yaml := mustBuildInstanceYAML(t, "wall", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "wall:")
	assertYAMLContains(t, s, "numLayers", "5")
}

func TestStrategy_Sentinel_Monitor(t *testing.T) {
	config := map[string]any{
		"interval": "5m", "threshold": 0.02, "window": 100,
		"numSamples": 10, "proportion": 0.8,
	}
	yaml := mustBuildInstanceYAML(t, "sentinel", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "sentinel:")
	assertYAMLContains(t, s, "threshold", "0.02")
}

func TestStrategy_Random_TestMode(t *testing.T) {
	config := map[string]any{
		"schedule": "*/30 * * * *", "dryRun": true, "quantity": 0.001,
	}
	yaml := mustBuildInstanceYAML(t, "random", "BTCUSDT", "binance", "paper", config)
	s := string(yaml)

	assertYAMLContains(t, s, "random:")
	assertYAMLContains(t, s, "dryRun", "true")
}

func TestStrategy_Rebalance_Portfolio(t *testing.T) {
	config := map[string]any{
		"schedule": "0 0 * * *", "quoteCurrency": "USDT",
		"threshold": 0.05, "dryRun": false,
	}
	yaml := mustBuildInstanceYAML(t, "rebalance", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "rebalance:")
	assertYAMLContains(t, s, "threshold", "0.05")
}

// ============================================================
// Indicator & Utility Strategy Business Tests
// ============================================================

func TestStrategy_Techsignal_SupportDetection(t *testing.T) {
	config := map[string]any{"interval": "1h"}
	yaml := mustBuildInstanceYAML(t, "techsignal", "BTCUSDT", "binance", "live", config)
	s := string(yaml)
	assertYAMLContains(t, s, "techsignal:")
}

func TestStrategy_Autoborrow_MarginManagement(t *testing.T) {
	config := map[string]any{
		"interval": "5m", "minMarginLevel": 1.5, "maxMarginLevel": 3.0,
	}
	yaml := mustBuildInstanceYAML(t, "autoborrow", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "autoborrow:")
	assertYAMLContains(t, s, "minMarginLevel", "1.5")
}

func TestStrategy_Convert_DustConversion(t *testing.T) {
	config := map[string]any{"from": "BNB", "to": "USDT"}
	yaml := mustBuildInstanceYAML(t, "convert", "BNBUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "convert:")
	assertYAMLContains(t, s, "from", "BNB")
}

func TestStrategy_Deposit2transfer_AutoTransfer(t *testing.T) {
	config := map[string]any{
		"assets": "BTC,ETH", "interval": "1m", "ignoreDust": true,
	}
	yaml := mustBuildInstanceYAML(t, "deposit2transfer", "BTCUSDT", "binance", "live", config)
	s := string(yaml)

	assertYAMLContains(t, s, "deposit2transfer:")
}

func TestStrategy_Support_Monitor(t *testing.T) {
	config := map[string]any{"interval": "1h", "quantity": 0.001}
	yaml := mustBuildInstanceYAML(t, "support", "BTCUSDT", "binance", "live", config)
	s := string(yaml)
	assertYAMLContains(t, s, "support:")
}

// ============================================================
// LiveOnly enforcement tests
// ============================================================

func TestAPICreate_Supertrend_PaperBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "supertrend", "name": "ST Bot", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"quantity": 0.001},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("supertrend in paper mode should be blocked (liveOnly), got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPICreate_Drift_PaperBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "drift", "name": "Drift Bot", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"window": 20},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("drift in paper mode should be blocked (liveOnly), got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPICreate_DCA2_PaperBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "dca2", "name": "DCA2", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"quoteInvestment": 100, "maxOrderCount": 10},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("dca2 in paper mode should be blocked (liveOnly), got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================
// Backtest defaults verification
// ============================================================

func TestBacktest_Emacross_FullParams(t *testing.T) {
	config := `{"symbol":"ETHUSDT","interval":"4h","fastWindow":7,"slowWindow":25,"quantity":0.01}`
	yaml, err := buildBacktestYAML("emacross", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "emacross:")
	assertYAMLContains(t, s, "fastWindow", "7")
}

func TestBacktest_Pivotshort_ExitsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("pivotshort", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if !strings.Contains(s, "percentage: -0.05") {
		t.Errorf("pivotshort should have roiStopLoss injected, got:\n%s", s)
	}
}

func TestBacktest_Swing_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("swing", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "movingAverageType", "SMA")
	assertYAMLContains(t, s, "baseQuantity", "0.0001")
}

func TestBacktest_DCA_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("dca", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "budget")
}

func TestBacktest_Techsignal_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("techsignal", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "interval", "1h")
}

func TestBacktest_EwoDgtrd_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("ewo_dgtrd", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "sigWin", "5")
}

func TestBacktest_Supertrend_DefaultsInjected(t *testing.T) {
	config := `{"symbol":"BTCUSDT","factor":3}`
	yaml, err := buildBacktestYAML("supertrend", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "interval", "1h")
}
