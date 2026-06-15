package main

import (
	"strings"
	"testing"
)

func TestRiskConfig_HasAny(t *testing.T) {
	cases := []struct {
		name string
		rc   *RiskConfig
		want bool
	}{
		{"nil", nil, false},
		{"empty", &RiskConfig{}, false},
		{"stopLoss only", &RiskConfig{StopLossPrice: 19000}, true},
		{"takeProfit only", &RiskConfig{TakeProfitPrice: 22000}, true},
		{"roi only", &RiskConfig{RoiStopLoss: 0.05}, true},
		{"trailing only", &RiskConfig{TrailingCallback: 0.02}, true},
		{"maxQty only", &RiskConfig{MaxPositionQty: 5}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.rc.HasAny(); got != c.want {
				t.Fatalf("HasAny=%v, want %v", got, c.want)
			}
		})
	}
}

func TestRiskConfig_EnvArgs(t *testing.T) {
	rc := &RiskConfig{
		StopLossPrice:      19000,
		TakeProfitPrice:    22000,
		RoiStopLoss:        0.05,
		RoiTakeProfit:      0.10,
		TrailingActivation: 0.03,
		TrailingCallback:   0.02,
		MaxPositionQty:     5,
	}
	args := rc.EnvArgs()
	want := []string{
		"BBGO_UNIVERSAL_RISK_STOP_LOSS_PRICE=19000",
		"BBGO_UNIVERSAL_RISK_TAKE_PROFIT_PRICE=22000",
		"BBGO_UNIVERSAL_RISK_ROI_STOP_LOSS=0.05",
		"BBGO_UNIVERSAL_RISK_ROI_TAKE_PROFIT=0.1",
		"BBGO_UNIVERSAL_RISK_TRAILING_ACTIVATION=0.03",
		"BBGO_UNIVERSAL_RISK_TRAILING_CALLBACK=0.02",
		"BBGO_UNIVERSAL_RISK_MAX_POSITION_QTY=5",
	}
	for _, w := range want {
		found := false
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing env entry %q in %v", w, args)
		}
	}
}

func TestRiskConfig_EnvArgs_NilOrEmpty(t *testing.T) {
	cases := []struct {
		name string
		rc   *RiskConfig
	}{
		{"nil", nil},
		{"empty", &RiskConfig{}},
		{"all-zero", &RiskConfig{StopLossPrice: 0, TakeProfitPrice: 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if args := c.rc.EnvArgs(); args != nil {
				t.Errorf("expected nil args, got %v", args)
			}
		})
	}
}

func TestRiskConfig_Validate(t *testing.T) {
	if err := (&RiskConfig{StopLossPrice: -1}).Validate(); err == nil {
		t.Error("expected error for negative stopLossPrice")
	}
	if err := (&RiskConfig{RoiStopLoss: 0.05}).Validate(); err != nil {
		t.Errorf("expected no error for positive value: %v", err)
	}
	if err := (&RiskConfig{}).Validate(); err != nil {
		t.Errorf("expected no error for empty config: %v", err)
	}
	if err := (&RiskConfig{MaxPositionQty: -5}).Validate(); err == nil {
		t.Error("expected error for negative maxPositionQty")
	}
}

func TestMergeRiskConfig_PreservesUnsetFields(t *testing.T) {
	base := &RiskConfig{
		StopLossPrice:   19000,
		TakeProfitPrice: 22000,
	}
	patch := &RiskConfig{TakeProfitPrice: 25000}
	merged := mergeRiskConfig(base, patch)
	if merged.StopLossPrice != 19000 {
		t.Errorf("SL preserved: got %v, want 19000", merged.StopLossPrice)
	}
	if merged.TakeProfitPrice != 25000 {
		t.Errorf("TP overwritten: got %v, want 25000", merged.TakeProfitPrice)
	}
}

func TestMergeRiskConfig_AddsNewField(t *testing.T) {
	base := &RiskConfig{StopLossPrice: 19000}
	patch := &RiskConfig{MaxPositionQty: 5}
	merged := mergeRiskConfig(base, patch)
	if merged.StopLossPrice != 19000 {
		t.Errorf("SL should be preserved: got %v", merged.StopLossPrice)
	}
	if merged.MaxPositionQty != 5 {
		t.Errorf("maxQty should be added: got %v", merged.MaxPositionQty)
	}
}

func TestMergeRiskConfig_AllZeroReturnsNil(t *testing.T) {
	base := &RiskConfig{}
	patch := &RiskConfig{}
	merged := mergeRiskConfig(base, patch)
	if merged != nil {
		t.Errorf("expected nil for all-zero merge, got %+v", merged)
	}
}

func TestMergeRiskConfig_NilPatchReturnsBase(t *testing.T) {
	base := &RiskConfig{StopLossPrice: 19000}
	merged := mergeRiskConfig(base, nil)
	if merged != base {
		t.Errorf("expected base preserved when patch is nil")
	}
}

func TestInstanceEnvArgs_InjectsRiskConfig(t *testing.T) {
	cm, _ := setupContainerManager(t)
	inst := testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT")
	inst.RiskConfig = &RiskConfig{
		StopLossPrice:   19000,
		TakeProfitPrice: 22000,
	}
	args := cm.instanceEnvArgs(inst)

	want := []string{
		"BBGO_UNIVERSAL_RISK_STOP_LOSS_PRICE=19000",
		"BBGO_UNIVERSAL_RISK_TAKE_PROFIT_PRICE=22000",
	}
	for _, w := range want {
		found := false
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing env entry %q in args: %v", w, args)
		}
	}
}

func TestInstanceEnvArgs_NoRiskConfig_SkipsEnvVars(t *testing.T) {
	cm, _ := setupContainerManager(t)
	inst := testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT")
	args := cm.instanceEnvArgs(inst)

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" && strings.HasPrefix(args[i+1], "BBGO_UNIVERSAL_RISK_") {
			t.Errorf("expected no BBGO_UNIVERSAL_RISK_* env vars without RiskConfig, got %s", args[i+1])
		}
	}
}
