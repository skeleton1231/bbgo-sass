package main

import (
	"strings"
	"testing"
)

func TestBuildUserYAML_SingleExchange(t *testing.T) {
	yamlBytes, err := buildUserYAML("test-user", ModeLive, []StrategyEntry{
		{
			Strategy: "grid2",
			Exchange: "binance",
			Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		},
	}, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)
	if !strings.Contains(yaml, "exchange:") {
		t.Error("expected exchange section")
	}
	if !strings.Contains(yaml, "binance") {
		t.Error("expected binance in YAML")
	}
	if !strings.Contains(yaml, "exchangeStrategies:") {
		t.Error("expected exchangeStrategies section")
	}
	if !strings.Contains(yaml, `"on": binance`) {
		t.Error("expected '\"on\": binance' session binding")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy config")
	}
	if strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("should not have crossExchangeStrategies for single exchange")
	}
}

func TestBuildUserYAML_CrossExchange(t *testing.T) {
	yamlBytes, err := buildUserYAML("test-user", ModeLive, []StrategyEntry{
		{
			Strategy:      "xmaker",
			Exchange:      "",
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
				{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
			},
			Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.001,"spread":0.001}`),
		},
	}, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("expected crossExchangeStrategies section")
	}
	if !strings.Contains(yaml, "xmaker:") {
		t.Error("expected xmaker strategy in YAML")
	}
	if !strings.Contains(yaml, "binance") {
		t.Error("expected binance session in YAML")
	}
	if !strings.Contains(yaml, "bybit") {
		t.Error("expected bybit session in YAML")
	}
	if !strings.Contains(yaml, "futures: true") {
		t.Error("expected futures: true for hedge session")
	}
	if strings.Contains(yaml, "exchangeStrategies:") {
		t.Error("should not have exchangeStrategies section")
	}
}

func TestBuildUserYAML_Mixed(t *testing.T) {
	yamlBytes, err := buildUserYAML("test-user", ModeLive, []StrategyEntry{
		{
			Strategy: "grid2",
			Exchange: "binance",
			Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		},
		{
			Strategy:      "xfunding",
			Exchange:      "",
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "spot", Exchange: "binance", EnvVarPrefix: "BINANCE"},
				{Name: "futures", Exchange: "okex", EnvVarPrefix: "OKEX", Futures: true},
			},
			Config: rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		},
	}, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "exchangeStrategies:") {
		t.Error("expected exchangeStrategies section for grid2")
	}
	if !strings.Contains(yaml, `"on": binance`) {
		t.Error("expected '\"on\": binance' session binding for grid2")
	}
	if !strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("expected crossExchangeStrategies section for xfunding")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy")
	}
	if !strings.Contains(yaml, "xfunding:") {
		t.Error("expected xfunding strategy")
	}
}

func TestBuildBacktestYAML(t *testing.T) {
	yaml, err := buildBacktestYAML("grid2", rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`), "2024-01-01", "2024-06-01", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yaml)
	if !strings.Contains(s, "sessions:") {
		t.Error("expected sessions section in backtest YAML")
	}
	if !strings.Contains(s, "exchangeStrategies:") {
		t.Error("expected exchangeStrategies in backtest YAML")
	}
	if !strings.Contains(s, "grid2:") {
		t.Error("expected grid2 strategy in backtest YAML")
	}
	if !strings.Contains(s, "BTCUSDT") {
		t.Error("expected symbol BTCUSDT in strategy params")
	}
	if !strings.Contains(s, "backtest:") {
		t.Error("expected backtest section")
	}
	if !strings.Contains(s, "accounts:") {
		t.Error("expected accounts in backtest section")
	}
	if !strings.Contains(s, "2024-01-01") {
		t.Error("expected start time in backtest YAML")
	}
}

func TestBuildBacktestYAML_Table(t *testing.T) {
	tests := []struct {
		name       string
		strategy   string
		config     string
		startTime  string
		endTime    string
		exchange   string
		symbol     string
		wantErr    bool
		wantInYAML []string
	}{
		{
			name:       "grid2_with_all_fields",
			strategy:   "grid2",
			config:     `{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":50000,"lowerPrice":40000}`,
			startTime:  "2024-01-01",
			endTime:    "2024-06-01",
			wantInYAML: []string{"grid2:", "gridNumber:", "upperPrice:", "lowerPrice:", "BTCUSDT", "2024-01-01", "2024-06-01"},
		},
		{
			name:       "empty_config_uses_defaults",
			strategy:   "grid2",
			config:     `{}`,
			startTime:  "",
			endTime:    "",
			wantInYAML: []string{"grid2:", "BTCUSDT", "2024-01-01", "binance:"},
		},
		{
			name:       "symbol_from_config_when_override_empty",
			strategy:   "dca",
			config:     `{"symbol":"ETHUSDT","investmentInterval":"1h"}`,
			symbol:     "",
			wantInYAML: []string{"dca:", "ETHUSDT", "investmentInterval:"},
		},
		{
			name:       "override_symbol_takes_priority",
			strategy:   "grid2",
			config:     `{"symbol":"BTCUSDT"}`,
			symbol:     "SOLUSDT",
			wantInYAML: []string{"grid2:", "SOLUSDT"},
		},
		{
			name:       "override_exchange_takes_priority",
			strategy:   "grid2",
			config:     `{"exchange":"kucoin","symbol":"BTCUSDT"}`,
			exchange:   "bybit",
			wantInYAML: []string{"bybit:", "BYBIT"},
		},
		{
			name:       "exchange_from_config_when_override_empty",
			strategy:   "grid2",
			config:     `{"exchange":"okex","symbol":"BTCUSDT"}`,
			exchange:   "",
			wantInYAML: []string{"okex:", "OKEX"},
		},
		{
			name:       "supertrend_strategy",
			strategy:   "supertrend",
			config:     `{"symbol":"BTCUSDT","interval":"15m"}`,
			wantInYAML: []string{"supertrend:", "BTCUSDT", "15m"},
		},
		{
			name:       "bollmaker_strategy",
			strategy:   "bollmaker",
			config:     `{"symbol":"ETHUSDT"}`,
			wantInYAML: []string{"bollmaker:", "ETHUSDT"},
		},
		{
			name:     "invalid_json_config",
			strategy: "grid2",
			config:   `{invalid`,
			wantErr:  true,
		},
		{
			name:       "interval_not_injected_when_missing",
			strategy:   "grid2",
			config:     `{"symbol":"BTCUSDT"}`,
			wantInYAML: []string{"grid2:"},
		},
		{
			name:       "interval_preserved_when_set",
			strategy:   "grid2",
			config:     `{"symbol":"BTCUSDT","interval":"5m"}`,
			wantInYAML: []string{"interval: 5m"},
		},
		{
			name:       "unknown_exchange_gets_default_prefix",
			strategy:   "grid2",
			config:     `{"symbol":"BTCUSDT"}`,
			exchange:   "unknown_exchange",
			wantInYAML: []string{"unknown_exchange:", "EXCHANGE"},
		},
		{
			name:       "kucoin_exchange",
			strategy:   "grid2",
			config:     `{"symbol":"BTCUSDT"}`,
			exchange:   "kucoin",
			wantInYAML: []string{"kucoin:", "KUCOIN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, err := buildBacktestYAML(tt.strategy, rawJSON(tt.config), tt.startTime, tt.endTime, tt.exchange, tt.symbol)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s := string(yaml)
			for _, want := range tt.wantInYAML {
				if !strings.Contains(s, want) {
					t.Errorf("expected YAML to contain %q\n--- YAML ---\n%s", want, s)
				}
			}
		})
	}
}

// TestBuildBacktestYAML_AllExercises validates that every supported exchange × strategy
// combination produces valid YAML with correct sections.
func TestBuildBacktestYAML_AllExercises(t *testing.T) {
	allExchanges := []string{"binance", "okex", "kucoin", "bybit", "bitget", "max", "coinbase", "bitfinex"}
	allPrefixes := map[string]string{
		"binance": "BINANCE", "okex": "OKEX", "kucoin": "KUCOIN", "bybit": "BYBIT",
		"bitget": "BITGET", "max": "MAX", "coinbase": "COINBASE", "bitfinex": "BITFINEX",
	}

	// Single-exchange strategies with representative configs.
	// Each has: strategy ID, default config params for backtest.
	strategies := []struct {
		id     string
		config string
	}{
		{"grid", `{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`},
		{"grid2", `{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`},
		{"bollgrid", `{"symbol":"BTCUSDT","interval":"1h","gridNumber":10,"gridPips":50,"quantity":0.001}`},
		{"fixedmaker", `{"symbol":"BTCUSDT","quantity":0.001,"spread":0.001,"minProfitSpread":0.001}`},
		{"fmaker", `{"symbol":"BTCUSDT","quantity":0.001,"spread":0.001,"minProfitSpread":0.001}`},
		{"emacross", `{"symbol":"BTCUSDT","interval":"1h","fastLength":7,"slowLength":25}`},
		{"trendtrader", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"pivotshort", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"swing", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"dca", `{"symbol":"BTCUSDT"}`},
		{"autobuy", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"flashcrash", `{"symbol":"BTCUSDT"}`},
		{"sentinel", `{"symbol":"BTCUSDT"}`},
		{"random", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"rebalance", `{"symbol":"BTCUSDT"}`},
		{"xhedgegrid", `{"symbol":"BTCUSDT","gridNumber":10}`},
		{"atrpin", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"ewo_dgtrd", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"harmonic", `{"symbol":"BTCUSDT","interval":"1h"}`},
		{"irr", `{"symbol":"BTCUSDT"}`},
		{"schedule", `{"symbol":"BTCUSDT"}`},
		{"techsignal", `{"symbol":"BTCUSDT","interval":"1h"}`},
	}

	for _, ex := range allExchanges {
		for _, st := range strategies {
			name := st.id + "/" + ex
			t.Run(name, func(t *testing.T) {
				yaml, err := buildBacktestYAML(st.id, []byte(st.config), "2024-01-01", "2024-03-01", ex, "BTCUSDT")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				s := string(yaml)

				// Must contain the exchange section
				if !strings.Contains(s, ex+":") {
					t.Errorf("missing exchange %q section", ex)
				}
				// Must contain the correct env var prefix
				if !strings.Contains(s, allPrefixes[ex]) {
					t.Errorf("missing env prefix %q", allPrefixes[ex])
				}
				// Must contain the strategy key
				if !strings.Contains(s, st.id+":") {
					t.Errorf("missing strategy %q section", st.id)
				}
				// Must contain symbol
				if !strings.Contains(s, "BTCUSDT") {
					t.Error("missing symbol BTCUSDT")
				}
				// Must contain backtest section
				if !strings.Contains(s, "backtest:") {
					t.Error("missing backtest section")
				}
				// Must contain date range
				if !strings.Contains(s, "2024-01-01") || !strings.Contains(s, "2024-03-01") {
					t.Error("missing date range")
				}
				// Must contain accounts with default balances
				if !strings.Contains(s, "accounts:") {
					t.Error("missing accounts section")
				}
			})
		}
	}
}

func TestIsValidTradingPair(t *testing.T) {
	tests := []struct {
		symbol string
		want   bool
	}{
		{"BTCUSDT", true},
		{"ETHBTC", true},
		{"BNBBUSD", true},
		{"SOLUSDC", true},
		{"XRPFDUSD", true},
		{"BTCETH", true},
		{"DOGEUSDT", true},
		{"BTGETH", true},
		{"FORUSDT", true},
		{"USDT", false},
		{"BTC", false},
		{"", false},
		{"123ABC", false},
		{"BINANCE-PERP", false},
		{"BTCTWD", false},
	}

	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			if got := isValidTradingPair(tt.symbol); got != tt.want {
				t.Errorf("isValidTradingPair(%q) = %v, want %v", tt.symbol, got, tt.want)
			}
		})
	}
}

func TestFilterTradingPairs(t *testing.T) {
	input := []string{"BTCUSDT", "BTGETH", "FORUSDT", "123ABC", "ETHBTC", "USDT", "DOGEUSDT", "BINANCE-PERP", "SOLUSDC"}
	filtered := filterTradingPairs(input)

	want := []string{"BTCUSDT", "BTGETH", "FORUSDT", "ETHBTC", "DOGEUSDT", "SOLUSDC"}
	if len(filtered) != len(want) {
		t.Fatalf("expected %d symbols, got %d: %v", len(want), len(filtered), filtered)
	}
	for i, s := range want {
		if filtered[i] != s {
			t.Errorf("filtered[%d] = %q, want %q", i, filtered[i], s)
		}
	}
}

// TestBuildBacktestYAML_AllExchangesValidConfig verifies SyncBacktest config for all exchanges.
func TestBuildSyncConfig_AllExchanges(t *testing.T) {
	allExchanges := []string{"binance", "okex", "kucoin", "bybit", "bitget", "max", "coinbase", "bitfinex"}

	for _, ex := range allExchanges {
		t.Run(ex, func(t *testing.T) {
			yaml, err := buildSyncConfig(ex, "BTCUSDT", "2024-01-01", "2024-06-01")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s := string(yaml)
			if !strings.Contains(s, ex+":") {
				t.Errorf("missing exchange %q in sync config", ex)
			}
			if !strings.Contains(s, "BTCUSDT") {
				t.Error("missing symbol in sync config")
			}
			if !strings.Contains(s, "2024-01-01") {
				t.Error("missing start time in sync config")
			}
			if !strings.Contains(s, "2024-06-01") {
				t.Error("missing end time in sync config")
			}
		})
	}
}
