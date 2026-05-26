package main

import (
	"testing"
)

// Known bbgo registered strategy IDs (from bbgo/pkg/strategy/*/strategy.go RegisterStrategy calls).
var knownBBGoStrategies = map[string]bool{
	"grid": true, "grid2": true, "bollgrid": true, "bollmaker": true,
	"linregmaker": true, "rsmaker": true, "fixedmaker": true, "fmaker": true,
	"scmaker": true, "supertrend": true, "emacross": true, "trendtrader": true,
	"pivotshort": true, "swing": true, "dca": true, "dca2": true, "dca3": true,
	"autobuy": true, "flashcrash": true, "wall": true, "sentinel": true,
	"random": true, "rebalance": true, "xhedgegrid": true, "audacitymaker": true,
	"liquiditymaker": true, "atrpin": true, "drift": true, "elliottwave": true,
	"factorzoo": true, "ewo_dgtrd": true, "harmonic": true, "irr": true,
	"schedule": true, "xvs": true, "techsignal": true, "autoborrow": true,
	"convert": true, "deposit2transfer": true, "support": true,
	"xmaker": true, "xbalance": true, "xalign": true, "xpremium": true,
	"xfixedmaker": true, "xnav": true, "xgap": true, "xdepthmaker": true,
	"xfunding": true, "xfundingv2": true,
}

// Frontend strategy IDs from web/src/lib/bbgo/strategies.ts.
var frontendStrategyIDs = []string{
	"grid", "grid2", "bollgrid", "bollmaker", "linregmaker", "rsmaker",
	"fixedmaker", "fmaker", "scmaker", "supertrend", "emacross", "trendtrader",
	"pivotshort", "swing", "dca", "dca2", "dca3", "autobuy", "flashcrash",
	"wall", "sentinel", "sentinel_anomaly", "random", "rebalance",
	"rebalance_portfolio", "xhedgegrid", "audacitymaker", "liquiditymaker",
	"atrpin", "drift", "elliottwave", "factorzoo", "ewo_dgtrd", "harmonic",
	"irr", "schedule", "xvs", "techsignal", "autoborrow", "convert",
	"deposit2transfer", "autobuy_scheduled", "support",
	"xmaker", "xbalance", "xalign", "xpremium", "xfixedmaker", "xnav",
	"xgap", "xdepthmaker", "xfunding", "xfundingv2",
}

func TestNormalizeStrategyConfig_ResolvesAllFrontendIDs(t *testing.T) {
	for _, id := range frontendStrategyIDs {
		resolved, _ := normalizeStrategyConfig(id, map[string]interface{}{})
		if !knownBBGoStrategies[resolved] {
			t.Errorf("frontend strategy %q resolves to %q which is NOT a known bbgo strategy ID", id, resolved)
		}
	}
}

func TestNormalizeStrategyConfig_AliasDirection(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sentinel_anomaly", "sentinel"},
		{"autobuy_scheduled", "autobuy"},
		{"rebalance_portfolio", "rebalance"},
		{"ewoDgtrd", "ewo_dgtrd"},
		{"grid", "grid"},
		{"sentinel", "sentinel"},
	}
	for _, tt := range tests {
		resolved, _ := normalizeStrategyConfig(tt.input, map[string]interface{}{})
		if resolved != tt.expected {
			t.Errorf("normalizeStrategyConfig(%q) = %q, want %q", tt.input, resolved, tt.expected)
		}
	}
}

func TestLegacyFieldAliases(t *testing.T) {
	params := map[string]interface{}{"interval": "1h"}
	_, result := normalizeStrategyConfig("dca", params)
	if _, has := result["investmentInterval"]; !has {
		t.Error("dca 'interval' should be renamed to 'investmentInterval'")
	}
	if _, has := result["interval"]; has {
		t.Error("dca old 'interval' key should be removed")
	}
}
