package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// staticDefaults provides the old backtestDefaults as a DefaultsProvider for tests.
// Once the registry migration is applied, tests should use StrategyDefaultsCache instead.
var staticDefaults = &staticDefaultsProvider{defaults: map[string]map[string]any{
	"grid":        {"gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000, "quantity": 0.001, "profitSpread": 50, "side": "both"},
	"grid2":       {"gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000, "quantity": 0.001, "profitSpread": 0, "quoteInvestment": 1000},
	"xhedgegrid":  {"gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000, "quantity": 0.001, "profitSpread": 0, "quoteInvestment": 1000},
	"bollgrid":    {"interval": "1h", "profitSpread": 50, "gridPips": 50, "quantity": 0.001},
	"emacross":    {"interval": "1h", "fastWindow": 7, "slowWindow": 25},
	"trendtrader": {"interval": "1h", "trendLine": map[string]any{"interval": "1h", "quantity": 0.001, "pivotRightWindow": 5}},
	"supertrend":  {"interval": "1h", "quantity": 0.001},
	"atrpin":      {"interval": "1h", "quantity": 0.001, "multiplier": 2.0},
	"pivotshort":  {"interval": "1h", "breakLow": map[string]any{"interval": "1h", "window": 7, "ratio": 0.01}, "exits": []map[string]any{{"roiStopLoss": map[string]any{"percentage": -0.05}}}},
	"swing":       {"interval": "1h", "movingAverageType": "SMA", "movingAverageWindow": 20, "movingAverageInterval": "1h", "baseQuantity": 0.0001},
	"ewo_dgtrd":   {"interval": "1h", "sigWin": 5, "stoploss": 0.02},
	"irr":         {"interval": "1h", "window": 20, "quantity": 0.001},
	"flashcrash":  {"interval": "1h", "baseQuantity": 0.001},
	"fixedmaker":  {"interval": "1m", "quantity": 0.001, "halfSpread": 0.001},
	"fmaker":      {"interval": "1h", "spread": 0.001, "quantity": 0.001},
	"bollmaker":   {"interval": "1h"},
	"harmonic":    {"interval": "1h", "window": 20, "quantity": 0.001},
	"dca":         {"investmentInterval": "1h", "budget": 500, "budgetPeriod": "day"},
	"schedule":    {"interval": "1h", "quantity": 0.001, "side": "buy"},
	"random":      {"schedule": "*/30 * * * *", "dryRun": true, "quantity": 0.001},
	"techsignal":  {"interval": "1h", "supportDetection": []map[string]any{{"interval": "1h", "movingAverageInterval": "1h", "movingAverageWindow": 20, "movingAverageType": "SMA"}}},
}}

type staticDefaultsProvider struct {
	defaults map[string]map[string]any
}

func (s *staticDefaultsProvider) GetDefaults(strategyID string) map[string]any {
	return s.defaults[strategyID]
}

func (s *staticDefaultsProvider) RequiresFutures(strategyID string) bool {
	return false
}

func TestBacktestDefaults_InjectedWhenMissing(t *testing.T) {
	tests := []struct {
		strategy  string
		config    string
		wantField string
		wantValue string
	}{
		{"emacross", `{"symbol":"BTCUSDT","quantity":0.1}`, "interval", "1h"},
		{"supertrend", `{"symbol":"ETHUSDT","factor":3}`, "interval", "1h"},
		{"bollgrid", `{"symbol":"BTCUSDT","gridNumber":8}`, "interval", "1h"},
		{"trendtrader", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
		{"bollmaker", `{"symbol":"ETHUSDT"}`, "interval", "1h"},
		{"pivotshort", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
		{"swing", `{"symbol":"ETHUSDT"}`, "interval", "1h"},
		{"flashcrash", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
		{"fixedmaker", `{"symbol":"BTCUSDT"}`, "halfSpread", "0.001"},
		{"fmaker", `{"symbol":"BTCUSDT"}`, "spread", "0.001"},
		{"swing", `{"symbol":"BTCUSDT"}`, "movingAverageType", "SMA"},
		{"ewo_dgtrd", `{"symbol":"BTCUSDT"}`, "sigWin", "5"},
		{"harmonic", `{"symbol":"BTCUSDT"}`, "window", "20"},
		{"irr", `{"symbol":"BTCUSDT"}`, "window", "20"},
		{"schedule", `{"symbol":"BTCUSDT"}`, "side", "buy"},
		{"random", `{"symbol":"BTCUSDT"}`, "schedule", "*/30 * * * *"},
		{"xhedgegrid", `{"symbol":"BTCUSDT"}`, "gridNumber", "10"},
		{"atrpin", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
		{"techsignal", `{"symbol":"BTCUSDT"}`, "interval", "1h"},
		{"grid", `{"symbol":"BTCUSDT"}`, "gridNumber", "10"},
		{"grid2", `{"symbol":"BTCUSDT"}`, "quoteInvestment", "1000"},
		{"dca", `{"symbol":"BTCUSDT"}`, "budget", "100"},
	}

	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			yaml, err := buildBacktestYAML(tt.strategy, json.RawMessage(tt.config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
			if err != nil {
				t.Fatal(err)
			}
			s := string(yaml)
			if !strContains(s, tt.wantField) {
				t.Errorf("backtest YAML for %s should contain %s, got:\n%s", tt.strategy, tt.wantField, s)
			}
			if !strContains(s, tt.wantValue) {
				t.Errorf("backtest YAML for %s should contain value %s, got:\n%s", tt.strategy, tt.wantValue, s)
			}
		})
	}
}

func TestBacktestDefaults_NotOverriddenWhenPresent(t *testing.T) {
	config := `{"symbol":"BTCUSDT","interval":"4h","quantity":0.1}`
	yaml, err := buildBacktestYAML("emacross", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if !strContains(s, "4h") {
		t.Errorf("user-provided interval=4h should be preserved, got:\n%s", s)
	}
}

func TestBacktestDefaults_BollgridProfitSpread(t *testing.T) {
	config := `{"symbol":"BTCUSDT","gridNumber":8,"quantity":0.001}`
	yaml, err := buildBacktestYAML("bollgrid", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if !strContains(s, "profitSpread") {
		t.Errorf("bollgrid should have profitSpread injected, got:\n%s", s)
	}
	if !strContains(s, "gridPips") {
		t.Errorf("bollgrid should have gridPips injected, got:\n%s", s)
	}
}

func strContains(s, sub string) bool {
	return strings.Contains(s, sub)
}

func TestBacktestDefaults_PivotshortExits(t *testing.T) {
	config := `{"symbol":"BTCUSDT"}`
	yaml, err := buildBacktestYAML("pivotshort", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if !strContains(s, "roistoploss:") && !strContains(s, "roiStopLoss:") {
		t.Errorf("pivotshort should have roiStopLoss exit injected, got:\n%s", s)
	}
	if !strContains(s, "percentage: -0.05") {
		t.Errorf("pivotshort roiStopLoss should have percentage -0.05, got:\n%s", s)
	}
}

func TestBacktestYAML_NumbersNotQuoted(t *testing.T) {
	tests := []struct {
		strategy string
		config   string
		field    string
	}{
		{"fixedmaker", `{"symbol":"BTCUSDT"}`, "quantity"},
		{"fixedmaker", `{"symbol":"BTCUSDT"}`, "halfSpread"},
		{"swing", `{"symbol":"BTCUSDT"}`, "baseQuantity"},
		{"ewo_dgtrd", `{"symbol":"BTCUSDT"}`, "stoploss"},
		{"grid", `{"symbol":"BTCUSDT"}`, "profitSpread"},
		{"grid2", `{"symbol":"BTCUSDT"}`, "quoteInvestment"},
	}
	for _, tt := range tests {
		t.Run(tt.strategy+"_"+tt.field, func(t *testing.T) {
			yaml, err := buildBacktestYAML(tt.strategy, json.RawMessage(tt.config), "2024-01-01", "2024-06-01", "binance", "", staticDefaults)
			if err != nil {
				t.Fatal(err)
			}
			s := string(yaml)
			quoted := tt.field + `: "`
			if strings.Contains(s, quoted) {
				t.Errorf("%s field %s should not be a quoted string in YAML:\n%s", tt.strategy, tt.field, s)
			}
		})
	}
}

type futuresDefaultsProvider struct {
	staticDefaultsProvider
	futuresStrategies map[string]bool
}

func (f *futuresDefaultsProvider) RequiresFutures(strategyID string) bool {
	return f.futuresStrategies[strategyID]
}

func TestBacktestYAML_FuturesConfig(t *testing.T) {
	provider := &futuresDefaultsProvider{
		futuresStrategies: map[string]bool{"pivotshort": true},
	}

	t.Run("includes futures and leverage when strategy requires it", func(t *testing.T) {
		config := `{"symbol":"BTCUSDT","leverage":10}`
		yaml, err := buildBacktestYAML("pivotshort", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", provider)
		if err != nil {
			t.Fatal(err)
		}
		s := string(yaml)
		if !strContains(s, "futures: true") {
			t.Errorf("futures strategy should have futures: true, got:\n%s", s)
		}
		if !strContains(s, "symbolLeverage:") {
			t.Errorf("futures strategy should have symbolLeverage, got:\n%s", s)
		}
		if !strContains(s, "BTCUSDT: 10") {
			t.Errorf("leverage should be 10 for BTCUSDT, got:\n%s", s)
		}
	})

	t.Run("no futures config when strategy does not require it", func(t *testing.T) {
		config := `{"symbol":"BTCUSDT","quantity":0.1}`
		yaml, err := buildBacktestYAML("emacross", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", provider)
		if err != nil {
			t.Fatal(err)
		}
		s := string(yaml)
		if strContains(s, "futures: true") {
			t.Errorf("non-futures strategy should not have futures: true, got:\n%s", s)
		}
	})

	t.Run("isolated margin from marginType", func(t *testing.T) {
		config := `{"symbol":"BTCUSDT","leverage":5,"marginType":"isolated"}`
		yaml, err := buildBacktestYAML("pivotshort", json.RawMessage(config), "2024-01-01", "2024-06-01", "binance", "", provider)
		if err != nil {
			t.Fatal(err)
		}
		s := string(yaml)
		if !strContains(s, "isolatedFutures: true") {
			t.Errorf("isolated margin should have isolatedFutures: true, got:\n%s", s)
		}
		if !strContains(s, "isolatedFuturesSymbol: BTCUSDT") {
			t.Errorf("isolated margin should have isolatedFuturesSymbol, got:\n%s", s)
		}
	})
}
