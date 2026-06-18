package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusError    = "error"
	StatusStarting = "starting"
)

var defaultPaperBalances = map[string]float64{
	"USDT": 10000,
	"BTC":  0.5,
}

const (
	ModeLive  = "live"
	ModePaper = "paper"
)

const paperExchange = "binance"

var legacyStrategyAliases = map[string]string{
	"ewoDgtrd":            "ewo_dgtrd",
	"sentinel_anomaly":    "sentinel",
	"autobuy_scheduled":   "autobuy",
	"rebalance_portfolio": "rebalance",
}

var legacyFieldAliases = map[string]map[string]string{
	"dca":         {"interval": "investmentInterval"},
	"fixedmaker":  {"spread": "halfSpread", "minProfitSpread": ""},
	"xfixedmaker": {"spread": "halfSpread"},
	"wall":        {"spread": "layerSpread"},
	"autobuy":     {"interval": "schedule", "buyQuantity": "quantity"},
	"rebalance":   {"interval": "schedule"},
	"drift":       {"drawGraph": "generateGraph"},
}

func deepMerge(base, overlay map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		if baseMap, ok := base[k].(map[string]any); ok {
			if overlayMap, ok := v.(map[string]any); ok {
				result[k] = deepMerge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}

func normalizeStrategyConfig(strategy string, params map[string]any) (string, map[string]any) {
	if alias, ok := legacyStrategyAliases[strategy]; ok {
		strategy = alias
	}
	if fields, ok := legacyFieldAliases[strategy]; ok {
		for oldKey, newKey := range fields {
			if v, exists := params[oldKey]; exists {
				if newKey != "" {
					if _, hasNew := params[newKey]; !hasNew {
						params[newKey] = v
					}
				}
				delete(params, oldKey)
			}
		}
	}
	return strategy, params
}

type FuturesConfig struct {
	Leverage   int    `json:"leverage,omitempty"`
	MarginType string `json:"marginType,omitempty"`
}

// RiskConfig carries universal risk parameters that bbgo's
// UniversalRiskController (auto-bound by GeneralOrderExecutor.Bind when
// BBGO_UNIVERSAL_RISK_* env vars are present) enforces on any strategy.
// All fields use value-zero = unset semantics so they merge cleanly with
// PATCH semantics (zero is also how bbgo treats "disabled").
type RiskConfig struct {
	StopLossPrice      float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice    float64 `json:"takeProfitPrice,omitempty"`
	RoiStopLoss        float64 `json:"roiStopLoss,omitempty"`
	RoiTakeProfit      float64 `json:"roiTakeProfit,omitempty"`
	TrailingActivation float64 `json:"trailingActivation,omitempty"`
	TrailingCallback   float64 `json:"trailingCallback,omitempty"`
	MaxPositionQty     float64 `json:"maxPositionQty,omitempty"`
}

// HasAny reports whether the config carries at least one positive threshold.
// Zero-valued fields are treated as "unset" by bbgo and skipped at emission.
func (rc *RiskConfig) HasAny() bool {
	if rc == nil {
		return false
	}
	return rc.StopLossPrice > 0 ||
		rc.TakeProfitPrice > 0 ||
		rc.RoiStopLoss > 0 ||
		rc.RoiTakeProfit > 0 ||
		rc.TrailingActivation > 0 ||
		rc.TrailingCallback > 0 ||
		rc.MaxPositionQty > 0
}

// EnvArgs returns the `-e KEY=VAL` Docker args for the BBGO_UNIVERSAL_RISK_*
// env vars. Returns nil when the config is empty/unset so the controller
// stays disabled.
func (rc *RiskConfig) EnvArgs() []string {
	if !rc.HasAny() {
		return nil
	}
	var args []string
	add := func(name string, v float64) {
		if v > 0 {
			args = append(args, "-e", name+"="+strconv.FormatFloat(v, 'f', -1, 64))
		}
	}
	add("BBGO_UNIVERSAL_RISK_STOP_LOSS_PRICE", rc.StopLossPrice)
	add("BBGO_UNIVERSAL_RISK_TAKE_PROFIT_PRICE", rc.TakeProfitPrice)
	add("BBGO_UNIVERSAL_RISK_ROI_STOP_LOSS", rc.RoiStopLoss)
	add("BBGO_UNIVERSAL_RISK_ROI_TAKE_PROFIT", rc.RoiTakeProfit)
	add("BBGO_UNIVERSAL_RISK_TRAILING_ACTIVATION", rc.TrailingActivation)
	add("BBGO_UNIVERSAL_RISK_TRAILING_CALLBACK", rc.TrailingCallback)
	add("BBGO_UNIVERSAL_RISK_MAX_POSITION_QTY", rc.MaxPositionQty)
	return args
}

// Validate returns an error if any risk field has an invalid (non-positive) value.
// Empty / zero-valued fields are valid (treated as "unset").
func (rc *RiskConfig) Validate() error {
	if rc == nil {
		return nil
	}
	check := func(name string, v float64) error {
		if v < 0 {
			return fmt.Errorf("%s must be >= 0", name)
		}
		return nil
	}
	for _, c := range []struct {
		name string
		v    float64
	}{
		{"stopLossPrice", rc.StopLossPrice},
		{"takeProfitPrice", rc.TakeProfitPrice},
		{"roiStopLoss", rc.RoiStopLoss},
		{"roiTakeProfit", rc.RoiTakeProfit},
		{"trailingActivation", rc.TrailingActivation},
		{"trailingCallback", rc.TrailingCallback},
		{"maxPositionQty", rc.MaxPositionQty},
	} {
		if err := check(c.name, c.v); err != nil {
			return err
		}
	}
	return nil
}

type SessionRoleConfig struct {
	Name         string `json:"name"`
	Exchange     string `json:"exchange"`
	EnvVarPrefix string `json:"envVarPrefix"`
	Futures      bool   `json:"futures"`
}

type StrategyEntry struct {
	Name          string              `json:"name"`
	Exchange      string              `json:"exchange"`
	Strategy      string              `json:"strategy"`
	Config        json.RawMessage     `json:"config"`
	Mode          string              `json:"mode"`
	CrossExchange bool                `json:"crossExchange"`
	Sessions      []SessionRoleConfig `json:"sessions,omitempty"`
	FuturesConfig *FuturesConfig      `json:"futuresConfig,omitempty"`
	RiskConfig    *RiskConfig         `json:"riskConfig,omitempty"`
}

type UserMode struct {
	UserID string
	Mode   string
}

type DefaultsProvider interface {
	GetDefaults(strategyID string) map[string]any
	RequiresFutures(strategyID string) bool
}

// --- YAML config types (shared with instance_store.go) ---

type databaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type syncUserDataStreamConfig struct {
	Trades                      bool   `yaml:"trades"`
	FilledOrders                bool   `yaml:"filledOrders"`
	FuturesPosition             bool   `yaml:"futuresPosition,omitempty"`
	FuturesPositionSyncInterval string `yaml:"futuresPositionSyncInterval,omitempty"`
}

type syncConfig struct {
	UserDataStream *syncUserDataStreamConfig `yaml:"userDataStream"`
}

type persistenceConfig struct {
	Json *jsonPersistenceConfig `yaml:"json,omitempty"`
}

type jsonPersistenceConfig struct {
	Directory string `yaml:"directory"`
}

type bbgoConfig struct {
	InstanceID              string                    `yaml:"instanceId,omitempty"`
	Database                *databaseConfig           `yaml:"database,omitempty"`
	Sessions                map[string]sessionConfig  `yaml:"sessions,omitempty"`
	Exchange                map[string]exchangeConfig `yaml:"exchange"`
	Environment             *environmentConfig        `yaml:"environment,omitempty"`
	Sync                    *syncConfig               `yaml:"sync,omitempty"`
	Persistence             *persistenceConfig        `yaml:"persistence,omitempty"`
	ExchangeStrategies      []map[string]any          `yaml:"exchangeStrategies,omitempty"`
	CrossExchangeStrategies []map[string]any          `yaml:"crossExchangeStrategies,omitempty"`
}

type sessionConfig struct {
	Exchange              string             `yaml:"exchange"`
	EnvVarPrefix          string             `yaml:"envVarPrefix"`
	Futures               bool               `yaml:"futures,omitempty"`
	IsolatedFutures       bool               `yaml:"isolatedFutures,omitempty"`
	IsolatedFuturesSymbol string             `yaml:"isolatedFuturesSymbol,omitempty"`
	SymbolLeverage        map[string]int     `yaml:"symbolLeverage,omitempty"`
	PublicOnly            bool               `yaml:"publicOnly,omitempty"`
	PaperBalances         map[string]float64 `yaml:"paperBalances,omitempty"`
}

type exchangeConfig struct {
	Symbol string `yaml:"symbol"`
}

type environmentConfig struct {
	PaperTrade                 string `yaml:"PAPER_TRADE,omitempty"`
	DisableStartupBalanceQuery bool   `yaml:"disablestartupbalancequery,omitempty"`
}

func buildBacktestYAML(strategy string, rawConfig json.RawMessage, startTime, endTime, overrideExchange, overrideSymbol string, defaults DefaultsProvider, futuresConfig *FuturesConfig) ([]byte, error) {
	var allParams map[string]any
	if len(rawConfig) == 0 || string(rawConfig) == "null" {
		allParams = map[string]any{}
	} else if err := json.Unmarshal(rawConfig, &allParams); err != nil {
		return nil, err
	}

	if defaults != nil {
		if tmpl := defaults.GetDefaults(strategy); tmpl != nil {
			allParams = deepMerge(tmpl, allParams)
		}
	}

	// FuturesConfig is the UI source of truth — mirror it into params before any
	// downstream logic (validator-equivalent, session YAML) runs. This matches
	// the live-instance path in api.go CreateStrategy.
	if futuresConfig != nil {
		if futuresConfig.Leverage > 0 {
			allParams["leverage"] = futuresConfig.Leverage
		}
		if futuresConfig.MarginType != "" {
			allParams["marginType"] = futuresConfig.MarginType
		}
	}

	strategy, allParams = normalizeStrategyConfig(strategy, allParams)

	exchange := overrideExchange
	if exchange == "" {
		if v, ok := allParams["exchange"].(string); ok && v != "" {
			exchange = v
		}
	}
	if exchange == "" {
		exchange = "binance"
	}

	if startTime == "" {
		startTime = "2024-01-01"
	}
	if endTime == "" {
		endTime = "2024-06-01"
	}

	prefix := exchangeEnvPrefix(exchange)
	delete(allParams, "exchange")

	symbol := overrideSymbol
	if symbol == "" {
		if v, ok := allParams["symbol"].(string); ok && v != "" {
			symbol = v
		}
	}
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	allParams["symbol"] = symbol

	var sessionFutures bool
	var sessionSymbolLeverage map[string]int
	var sessionIsolatedFutures bool
	var sessionIsolatedFuturesSymbol string

	if defaults != nil && defaults.RequiresFutures(strategy) {
		sessionFutures = true
		// Priority: FuturesConfig > params["leverage"] (already includes registry default) > 0
		// No silent 20x default — if user didn't pick a leverage and registry has none,
		// session runs without symbolLeverage (exchange/paper uses its own default).
		leverage := 0
		if futuresConfig != nil && futuresConfig.Leverage > 0 {
			leverage = futuresConfig.Leverage
		}
		if leverage == 0 {
			if lv := toFloat(allParams["leverage"]); lv >= 1 {
				leverage = int(lv)
			}
		}
		if leverage > 0 {
			sessionSymbolLeverage = map[string]int{symbol: leverage}
		}

		marginType, _ := allParams["marginType"].(string)
		if futuresConfig != nil && futuresConfig.MarginType != "" {
			marginType = futuresConfig.MarginType
		}
		if marginType == "isolated" {
			sessionIsolatedFutures = true
			sessionIsolatedFuturesSymbol = symbol
		}
	}

	btCfg := struct {
		Exchange map[string]struct {
			Symbol string `yaml:"symbol"`
		} `yaml:"exchange"`
		Sessions map[string]struct {
			Exchange              string         `yaml:"exchange"`
			EnvVarPrefix          string         `yaml:"envVarPrefix"`
			Futures               bool           `yaml:"futures,omitempty"`
			SymbolLeverage        map[string]int `yaml:"symbolLeverage,omitempty"`
			IsolatedFutures       bool           `yaml:"isolatedFutures,omitempty"`
			IsolatedFuturesSymbol string         `yaml:"isolatedFuturesSymbol,omitempty"`
		} `yaml:"sessions"`
		ExchangeStrategies []map[string]any `yaml:"exchangeStrategies"`
		Backtest           struct {
			Sessions  []string `yaml:"sessions"`
			Symbols   []string `yaml:"symbols"`
			StartTime string   `yaml:"startTime"`
			EndTime   string   `yaml:"endTime"`
			Accounts  map[string]struct {
				Balances map[string]string `yaml:"balances"`
			} `yaml:"accounts"`
		} `yaml:"backtest"`
	}{
		Exchange: map[string]struct {
			Symbol string `yaml:"symbol"`
		}{
			exchange: {Symbol: symbol},
		},
		Sessions: map[string]struct {
			Exchange              string         `yaml:"exchange"`
			EnvVarPrefix          string         `yaml:"envVarPrefix"`
			Futures               bool           `yaml:"futures,omitempty"`
			SymbolLeverage        map[string]int `yaml:"symbolLeverage,omitempty"`
			IsolatedFutures       bool           `yaml:"isolatedFutures,omitempty"`
			IsolatedFuturesSymbol string         `yaml:"isolatedFuturesSymbol,omitempty"`
		}{
			exchange: {
				Exchange:              exchange,
				EnvVarPrefix:          prefix,
				Futures:               sessionFutures,
				SymbolLeverage:        sessionSymbolLeverage,
				IsolatedFutures:       sessionIsolatedFutures,
				IsolatedFuturesSymbol: sessionIsolatedFuturesSymbol,
			},
		},
		ExchangeStrategies: []map[string]any{
			{
				"on":     exchange,
				strategy: allParams,
			},
		},
		Backtest: struct {
			Sessions  []string `yaml:"sessions"`
			Symbols   []string `yaml:"symbols"`
			StartTime string   `yaml:"startTime"`
			EndTime   string   `yaml:"endTime"`
			Accounts  map[string]struct {
				Balances map[string]string `yaml:"balances"`
			} `yaml:"accounts"`
		}{
			Sessions:  []string{exchange},
			Symbols:   []string{symbol},
			StartTime: startTime,
			EndTime:   endTime,
			Accounts: map[string]struct {
				Balances map[string]string `yaml:"balances"`
			}{
				exchange: {Balances: backtestBalances(symbol)},
			},
		},
	}

	return yaml.Marshal(btCfg)
}

var commonQuoteCurrencies = []string{"USDT", "BUSD", "USDC", "TUSD", "FDUSD", "BTC", "ETH", "BNB"}

func extractQuoteCurrency(symbol string) string {
	for _, q := range commonQuoteCurrencies {
		if strings.HasSuffix(symbol, q) {
			return q
		}
	}
	return "USDT"
}

func backtestBalances(symbol string) map[string]string {
	quote := extractQuoteCurrency(symbol)
	base := strings.TrimSuffix(symbol, quote)
	return map[string]string{quote: "10000", base: "10"}
}

func filterTradingPairs(symbols []string) []string {
	filtered := make([]string, 0, len(symbols))
	for _, s := range symbols {
		if isValidTradingPair(s) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func isValidTradingPair(symbol string) bool {
	for _, q := range commonQuoteCurrencies {
		if strings.HasSuffix(symbol, q) && len(symbol) > len(q) {
			return true
		}
	}
	return false
}
