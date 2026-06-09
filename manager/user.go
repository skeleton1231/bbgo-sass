package main

import (
	"encoding/json"
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
}

type UserMode struct {
	UserID string
	Mode   string
}

type DefaultsProvider interface {
	GetDefaults(strategyID string) map[string]any
}

// --- YAML config types (shared with instance_store.go) ---

type databaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type syncUserDataStreamConfig struct {
	Trades       bool `yaml:"trades"`
	FilledOrders bool `yaml:"filledOrders"`
}

type syncConfig struct {
	UserDataStream *syncUserDataStreamConfig `yaml:"userDataStream"`
}

type bbgoConfig struct {
	Database                *databaseConfig           `yaml:"database,omitempty"`
	Sessions                map[string]sessionConfig  `yaml:"sessions,omitempty"`
	Exchange                map[string]exchangeConfig `yaml:"exchange"`
	Environment             *environmentConfig        `yaml:"environment,omitempty"`
	Sync                    *syncConfig               `yaml:"sync,omitempty"`
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

func buildBacktestYAML(strategy string, rawConfig json.RawMessage, startTime, endTime, overrideExchange, overrideSymbol string, defaults DefaultsProvider) ([]byte, error) {
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

	btCfg := struct {
		Exchange map[string]struct {
			Symbol string `yaml:"symbol"`
		} `yaml:"exchange"`
		Sessions map[string]struct {
			Exchange     string `yaml:"exchange"`
			EnvVarPrefix string `yaml:"envVarPrefix"`
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
			Exchange     string `yaml:"exchange"`
			EnvVarPrefix string `yaml:"envVarPrefix"`
		}{
			exchange: {Exchange: exchange, EnvVarPrefix: prefix},
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
