package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// ============================================================
// Cross-Exchange Strategy Business Tests
// ============================================================

func TestStrategy_Xmaker_CrossExchangeHedge(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xmaker",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance", Futures: false},
			{Name: "hedge", Exchange: "binance", Futures: true},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","spread":0.001,"quantity":0.01}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)

	assertYAMLContains(t, s, "xmaker:")
	assertYAMLContains(t, s, "crossExchangeStrategies")
}

func TestStrategy_Xmaker_MultiExchangeSessions(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xmaker",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance", Futures: false},
			{Name: "hedge", Exchange: "okex", Futures: true},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.01}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)

	assertYAMLContains(t, s, "binance")
	assertYAMLContains(t, s, "okex")
}

func TestStrategy_Xbalance_BalanceRebalance(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xbalance",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "source", Exchange: "binance"},
			{Name: "target", Exchange: "okex"},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","interval":"1m"}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "xbalance:")
}

func TestStrategy_Xalign_SpotFuturesArbitrage(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xalign",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "spot", Exchange: "binance", Futures: false},
			{Name: "futures", Exchange: "binance", Futures: true},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","interval":"5m"}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "xalign:")
	assertYAMLContains(t, s, "futures: true")
}

func TestStrategy_Xpremium_PremiumTrading(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xpremium",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "spot", Exchange: "binance"},
			{Name: "futures", Exchange: "binance", Futures: true},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.01}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "xpremium:")
}

func TestStrategy_Xfixedmaker_CrossFixedSpread(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xfixedmaker",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance"},
			{Name: "hedge", Exchange: "binance", Futures: true},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","halfSpread":0.0005,"quantity":0.01}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "xfixedmaker:")
}

func TestStrategy_Xgap_InterExchangeGap(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xgap",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "sessionA", Exchange: "binance"},
			{Name: "sessionB", Exchange: "okex"},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.01}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "xgap:")
}

func TestStrategy_Xfunding_FundingRateArb(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xfunding",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "spot", Exchange: "binance"},
			{Name: "futures", Exchange: "binance", Futures: true},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.01}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "xfunding:")
}

func TestStrategy_Xdepthmaker_DepthMaker(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xdepthmaker",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance"},
			{Name: "hedge", Exchange: "binance", Futures: true},
		},
		Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.01,"spread":0.001}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "xdepthmaker:")
}

// ============================================================
// Cross-exchange API validation
// ============================================================

func TestAPICreate_Xmaker_CrossExchangeRequiresSessions(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "xmaker", "name": "XMaker", "exchange": "binance",
		"mode": "live", "symbol": "BTCUSDT",
		"crossExchange": true,
		"config": map[string]any{"quantity": 0.01},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("cross-exchange without sessions should fail, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPICreate_Xmaker_PaperNonBinanceBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "xmaker", "name": "XMaker", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"crossExchange": true,
		"sessions": []map[string]any{
			{"name": "maker", "exchange": "binance"},
			{"name": "hedge", "exchange": "okex"},
		},
		"config": map[string]any{"quantity": 0.01},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("paper mode cross-exchange with non-binance session should be blocked, got %d", w.Code)
	}
}

// ============================================================
// Cross-exchange liveOnly enforcement
// ============================================================

func TestAPICreate_Xpremium_PaperBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "xpremium", "name": "Premium", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"crossExchange": true,
		"sessions": []map[string]any{
			{"name": "spot", "exchange": "binance"},
			{"name": "futures", "exchange": "binance"},
		},
		"config": map[string]any{"quantity": 0.01},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("xpremium in paper mode should be blocked (liveOnly), got %d", w.Code)
	}
}

func TestAPICreate_Xnav_PaperBlocked(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "xnav", "name": "NAV", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"crossExchange": true,
		"sessions": []map[string]any{
			{"name": "spot", "exchange": "binance"},
			{"name": "futures", "exchange": "binance"},
		},
		"config": map[string]any{},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("xnav in paper mode should be blocked (liveOnly), got %d", w.Code)
	}
}

// ============================================================
// Futures Config Business Tests
// ============================================================

func TestStrategy_Pivotshort_FuturesWithLeverage(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "pivotshort",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config: rawJSON(`{"symbol":"BTCUSDT","interval":"1h","quantity":0.001}`),
		FuturesConfig: &FuturesConfig{Leverage: 5, MarginType: "isolated"},
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)

	assertYAMLContains(t, s, "futures: true")
	assertYAMLContains(t, s, "isolatedFutures: true")
	assertYAMLContains(t, s, "symbolLeverage")
}

func TestStrategy_Bollmaker_FuturesInjected(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "bollmaker",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config: rawJSON(`{"symbol":"BTCUSDT","bidQuantity":0.01,"askQuantity":0.01}`),
		FuturesConfig: &FuturesConfig{Leverage: 3},
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "futures: true")
}

func TestStrategy_Grid2_NoFuturesForNonFuturesStrategy(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config: rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`),
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, testRegistry)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if strings.Contains(s, "futures: true") {
		t.Error("grid2 should NOT have futures config")
	}
}

// ============================================================
// Legacy Alias Tests
// ============================================================

func TestStrategy_LegacyAlias_EwoDgtrd(t *testing.T) {
	config := map[string]any{"interval": "1h", "sigWin": 5}
	yaml := mustBuildInstanceYAML(t, "ewoDgtrd", "BTCUSDT", "binance", "live", config)
	s := string(yaml)
	assertYAMLContains(t, s, "ewo_dgtrd:")
}

func TestStrategy_LegacyAlias_AutobuyScheduled(t *testing.T) {
	config := map[string]any{"schedule": "0 8 * * *", "buyQuantity": 0.01}
	rawConfig, _ := json.Marshal(config)
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "autobuy_scheduled",
		Exchange: "binance", Symbol: "BTCUSDT", Config: rawConfig,
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "autobuy:")
}

func TestStrategy_LegacyFieldAlias_DCA_Interval(t *testing.T) {
	config := map[string]any{"interval": "1h", "budget": 500}
	rawConfig, _ := json.Marshal(config)
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "dca",
		Exchange: "binance", Symbol: "BTCUSDT", Config: rawConfig,
	}
	yaml, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	assertYAMLContains(t, s, "investmentInterval")
}

// ============================================================
// Instance ID Stability
// ============================================================

func TestInstanceID_SameConfigSameID(t *testing.T) {
	config := rawJSON(`{"gridNumber":10,"upperPrice":65000,"lowerPrice":55000}`)
	id1 := computeInstanceID("grid2", "BTCUSDT", config)
	id2 := computeInstanceID("grid2", "BTCUSDT", config)
	if id1 != id2 {
		t.Errorf("same config should produce same instance ID: %s != %s", id1, id2)
	}
}

func TestInstanceID_DifferentConfigDifferentID(t *testing.T) {
	id1 := computeInstanceID("grid2", "BTCUSDT", rawJSON(`{"gridNumber":10}`))
	id2 := computeInstanceID("grid2", "BTCUSDT", rawJSON(`{"gridNumber":20}`))
	if id1 == id2 {
		t.Error("different config should produce different instance IDs")
	}
}

func TestInstanceID_DifferentStrategyDifferentID(t *testing.T) {
	config := rawJSON(`{"gridNumber":10}`)
	id1 := computeInstanceID("grid2", "BTCUSDT", config)
	id2 := computeInstanceID("grid", "BTCUSDT", config)
	if id1 == id2 {
		t.Error("different strategies should produce different instance IDs")
	}
}

// ============================================================
// Deep Merge Tests
// ============================================================

func TestDeepMerge_BasePreservedWhenNoOverlay(t *testing.T) {
	base := map[string]any{"interval": "1h", "quantity": 0.001}
	overlay := map[string]any{"symbol": "BTCUSDT"}
	result := deepMerge(base, overlay)

	if result["interval"] != "1h" {
		t.Errorf("base interval should be preserved, got: %v", result["interval"])
	}
}

func TestDeepMerge_OverlayOverridesBase(t *testing.T) {
	base := map[string]any{"interval": "1h", "quantity": 0.001}
	overlay := map[string]any{"interval": "4h"}
	result := deepMerge(base, overlay)

	if result["interval"] != "4h" {
		t.Errorf("overlay should override base, got: %v", result["interval"])
	}
}

func TestDeepMerge_NestedMapMerge(t *testing.T) {
	base := map[string]any{
		"trendLine": map[string]any{"interval": "1h", "quantity": 0.001},
	}
	overlay := map[string]any{
		"trendLine": map[string]any{"quantity": 0.01},
	}
	result := deepMerge(base, overlay)
	tl := result["trendLine"].(map[string]any)

	if tl["interval"] != "1h" {
		t.Errorf("nested base should be preserved, got: %v", tl["interval"])
	}
	if tl["quantity"] != 0.01 {
		t.Errorf("nested overlay should override, got: %v", tl["quantity"])
	}
}

// ============================================================
// Exchange Prefix Tests
// ============================================================

func TestExchangeEnvPrefix_AllExchanges(t *testing.T) {
	tests := []struct {
		exchange string
		prefix   string
	}{
		{"binance", "BINANCE"}, {"okex", "OKEX"}, {"kucoin", "KUCOIN"},
		{"bybit", "BYBIT"}, {"bitget", "BITGET"}, {"max", "MAX"},
		{"coinbase", "COINBASE"}, {"bitfinex", "BITFINEX"},
		{"unknown_exchange", "EXCHANGE"},
	}
	for _, tt := range tests {
		t.Run(tt.exchange, func(t *testing.T) {
			got := exchangeEnvPrefix(tt.exchange)
			if got != tt.prefix {
				t.Errorf("exchangeEnvPrefix(%s) = %s, want %s", tt.exchange, got, tt.prefix)
			}
		})
	}
}

// ============================================================
// Paper Mode Specific Tests
// ============================================================

func TestPaperMode_PaperBalancesInjected(t *testing.T) {
	config := map[string]any{"gridNumber": 10}
	yaml := mustBuildInstanceYAML(t, "grid2", "BTCUSDT", "binance", "paper", config)
	s := string(yaml)
	assertYAMLContains(t, s, "paperBalances")
	assertYAMLContains(t, s, "USDT")
}

func TestPaperMode_PublicOnlySet(t *testing.T) {
	config := map[string]any{"interval": "1h"}
	yaml := mustBuildInstanceYAML(t, "emacross", "BTCUSDT", "binance", "paper", config)
	s := string(yaml)
	assertYAMLContains(t, s, "publicOnly", "true")
}

func TestLiveMode_PublicOnlyFalseWithCredentials(t *testing.T) {
	yaml, err := buildInstanceYAML(&StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "emacross",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config: rawJSON(`{"interval":"1h"}`),
	}, func(string) bool { return true }, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if strings.Contains(s, "publicOnly: true") {
		t.Error("live mode with credentials should NOT have publicOnly=true")
	}
}
