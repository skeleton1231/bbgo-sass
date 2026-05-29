package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusError    = "error"
	StatusStarting = "starting"
)

const (
	ModeLive  = "live"
	ModePaper = "paper"
)

// legacyStrategyAliases maps frontend strategy IDs to the correct bbgo registered IDs.
var legacyStrategyAliases = map[string]string{
	"ewoDgtrd":            "ewo_dgtrd",
	"sentinel_anomaly":    "sentinel",
	"autobuy_scheduled":   "autobuy",
	"rebalance_portfolio": "rebalance",
}

// liveOnlyStrategies are strategies that only work in live mode.
var liveOnlyStrategies = map[string]bool{
	"bollmaker":        true,
	"linregmaker":      true,
	"rsmaker":          true,
	"scmaker":          true,
	"supertrend":       true,
	"dca2":             true,
	"dca3":             true,
	"wall":             true,
	"sentinel":         true,
	"audacitymaker":    true,
	"liquiditymaker":   true,
	"drift":            true,
	"elliottwave":      true,
	"factorzoo":        true,
	"xvs":              true,
	"autoborrow":       true,
	"convert":          true,
	"deposit2transfer": true,
	"autobuy":          true,
	"rebalance":        true,
	"support":          true,
	"xpremium":         true,
	"xnav":             true,
}

// legacyFieldAliases maps strategy IDs to old→new field renames.
var legacyFieldAliases = map[string]map[string]string{
	"dca":         {"interval": "investmentInterval"},
	"fixedmaker":  {"spread": "halfSpread", "minProfitSpread": ""},
	"xfixedmaker": {"spread": "halfSpread"},
	"wall":        {"spread": "layerSpread"},
	"autobuy":     {"interval": "schedule", "buyQuantity": "quantity"},
	"rebalance":   {"interval": "schedule"},
	"drift":       {"drawGraph": "generateGraph"},
}

// normalizeStrategyConfig fixes old strategy IDs and field names from legacy DB records.
func normalizeStrategyConfig(strategy string, params map[string]interface{}) (string, map[string]interface{}) {
	if alias, ok := legacyStrategyAliases[strategy]; ok {
		strategy = alias
	}
	if fields, ok := legacyFieldAliases[strategy]; ok {
		for oldKey, newKey := range fields {
			if v, exists := params[oldKey]; exists {
				if _, hasNew := params[newKey]; !hasNew {
					params[newKey] = v
				}
				delete(params, oldKey)
			}
		}
	}
	return strategy, params
}

// userContainerKey returns the composite key for the user container map.
func userContainerKey(userID, mode string) string {
	return userID + ":" + mode
}

type SessionRoleConfig struct {
	Name         string `json:"name"`
	Exchange     string `json:"exchange"`
	EnvVarPrefix string `json:"envVarPrefix"`
	Futures      bool   `json:"futures"`
}

type StrategyEntry struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Exchange      string              `json:"exchange"`
	Strategy      string              `json:"strategy"`
	Config        json.RawMessage     `json:"config"`
	Mode          string              `json:"mode"`
	CrossExchange bool                `json:"crossExchange"`
	Sessions      []SessionRoleConfig `json:"sessions,omitempty"`
}

type UserContainer struct {
	UserID     string          `json:"user_id"`
	Mode       string          `json:"mode"`
	Status     string          `json:"status"`
	Strategies []StrategyEntry `json:"strategies"`
}

type UserContainerManager struct {
	mu    sync.RWMutex
	users map[string]*UserContainer // key: "{userID}:{mode}"
}

func NewUserContainerManager() *UserContainerManager {
	return &UserContainerManager{users: make(map[string]*UserContainer)}
}

func (m *UserContainerManager) Get(userID, mode string) (*UserContainer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	uc, ok := m.users[userContainerKey(userID, mode)]
	if !ok {
		return nil, false
	}
	return cloneUserContainer(uc), true
}

// GetByUser returns all containers for a user (live and/or paper).
func (m *UserContainerManager) GetByUser(userID string) []*UserContainer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*UserContainer
	for _, uc := range m.users {
		if uc.UserID == userID {
			result = append(result, cloneUserContainer(uc))
		}
	}
	return result
}

// FindStrategy searches all containers for a user to find one containing the given strategyID.
// Returns the container mode and whether it was found.
func (m *UserContainerManager) FindStrategy(userID, strategyID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, uc := range m.users {
		if uc.UserID != userID {
			continue
		}
		for _, s := range uc.Strategies {
			if s.ID == strategyID {
				return uc.Mode, true
			}
		}
	}
	return "", false
}

func (m *UserContainerManager) UpdateStatus(userID, mode, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if uc, ok := m.users[userContainerKey(userID, mode)]; ok {
		uc.Status = status
	}
}

func (m *UserContainerManager) CompareAndSetStatus(userID, mode, oldStatus, newStatus string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if uc, ok := m.users[userContainerKey(userID, mode)]; ok && uc.Status == oldStatus {
		uc.Status = newStatus
		return true
	}
	return false
}

func (m *UserContainerManager) AddStrategy(userID, mode string, entry StrategyEntry) (*UserContainer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := userContainerKey(userID, mode)
	uc, ok := m.users[key]
	created := !ok
	if !ok {
		uc = &UserContainer{
			UserID:     userID,
			Mode:       mode,
			Status:     StatusStopped,
			Strategies: []StrategyEntry{},
		}
		m.users[key] = uc
	}
	uc.Strategies = append(uc.Strategies, entry)
	return cloneUserContainer(uc), created
}

// RemoveStrategy removes a strategy from whichever container holds it.
// Returns (found, modeOfContainer).
func (m *UserContainerManager) RemoveStrategy(userID, strategyID string) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, uc := range m.users {
		if uc.UserID != userID {
			continue
		}
		for i, s := range uc.Strategies {
			if s.ID == strategyID {
				uc.Strategies = append(uc.Strategies[:i], uc.Strategies[i+1:]...)
				if len(uc.Strategies) == 0 {
					delete(m.users, key)
				}
				return true, uc.Mode
			}
		}
	}
	return false, ""
}

func (m *UserContainerManager) ListUsers() []*UserContainer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*UserContainer, 0, len(m.users))
	for _, uc := range m.users {
		list = append(list, cloneUserContainer(uc))
	}
	return list
}

func (m *UserContainerManager) Restore(users []*UserContainer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = make(map[string]*UserContainer)
	for _, uc := range users {
		if uc.Mode == "" {
			uc.Mode = ModeLive
		}
		m.users[userContainerKey(uc.UserID, uc.Mode)] = uc
	}
}

type databaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type bbgoConfig struct {
	Database                *databaseConfig           `yaml:"database,omitempty"`
	Sessions                map[string]sessionConfig  `yaml:"sessions,omitempty"`
	Exchange                map[string]exchangeConfig `yaml:"exchange"`
	Environment             *environmentConfig        `yaml:"environment,omitempty"`
	ExchangeStrategies      []map[string]interface{}  `yaml:"exchangeStrategies,omitempty"`
	CrossExchangeStrategies []map[string]interface{}  `yaml:"crossExchangeStrategies,omitempty"`
}

type sessionConfig struct {
	Exchange     string `yaml:"exchange"`
	EnvVarPrefix string `yaml:"envVarPrefix"`
	Futures      bool   `yaml:"futures,omitempty"`
	PublicOnly   bool   `yaml:"publicOnly,omitempty"`
}

type exchangeConfig struct {
	Symbol string `yaml:"symbol"`
}

type environmentConfig struct {
	PaperTrade                 string `yaml:"PAPER_TRADE,omitempty"`
	DisableStartupBalanceQuery bool   `yaml:"disablestartupbalancequery"`
}

// buildUserYAML generates the bbgo YAML config for a container. The mode is determined
// by uc.Mode — paper containers get PAPER_TRADE=1, live containers do not.
func buildUserYAML(uc *UserContainer, hasCredentials func(exchange string) bool) ([]byte, error) {
	exchanges := map[string]exchangeConfig{}
	sessions := map[string]sessionConfig{}
	var exchangeStrategies []map[string]interface{}
	var crossStrategies []map[string]interface{}

	for _, s := range uc.Strategies {
		var params map[string]interface{}
		if err := json.Unmarshal(s.Config, &params); err != nil {
			var rawStr string
			if err2 := json.Unmarshal(s.Config, &rawStr); err2 == nil {
				strategyID := s.Strategy
				if alias, ok := legacyStrategyAliases[strategyID]; ok {
					strategyID = alias
				}
				entry := map[string]interface{}{
					"on":       s.Exchange,
					strategyID: rawStr,
				}
				exchangeStrategies = append(exchangeStrategies, entry)
			}
			continue
		}

		s.Strategy, params = normalizeStrategyConfig(s.Strategy, params)

		if s.CrossExchange {
			csEntry := buildCrossExchangeStrategy(s, params, sessions, exchanges, hasCredentials, uc.Mode)
			crossStrategies = append(crossStrategies, csEntry)
			continue
		}

		symbol := "BTCUSDT"
		if v, ok := params["symbol"].(string); ok && v != "" {
			symbol = v
		}
		if _, exists := exchanges[s.Exchange]; !exists {
			exchanges[s.Exchange] = exchangeConfig{Symbol: symbol}
			prefix := exchangeEnvPrefix(s.Exchange)
			sessions[s.Exchange] = sessionConfig{
				Exchange:     s.Exchange,
				EnvVarPrefix: prefix,
				PublicOnly:   !hasCredentials(s.Exchange),
			}
		}

		entry := map[string]interface{}{
			"on":       s.Exchange,
			s.Strategy: params,
		}
		exchangeStrategies = append(exchangeStrategies, entry)
	}

	dataDir := uc.UserID
	if uc.Mode == ModePaper {
		dataDir = uc.UserID + "-paper"
	}
	cfg := bbgoConfig{
		Database: &databaseConfig{
			Driver: "sqlite3",
			DSN:    fmt.Sprintf("file:/data/%s/bbgo.db?cache=shared&_journal_mode=WAL", dataDir),
		},
		Sessions:                sessions,
		Exchange:                exchanges,
		ExchangeStrategies:      exchangeStrategies,
		CrossExchangeStrategies: crossStrategies,
	}
	cfg.Environment = &environmentConfig{DisableStartupBalanceQuery: true}
	if uc.Mode == ModePaper {
		cfg.Environment.PaperTrade = "1"
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal bbgo config for user %s: %w", uc.UserID, err)
	}
	return out, nil
}

func buildCrossExchangeStrategy(s StrategyEntry, params map[string]interface{}, sessions map[string]sessionConfig, exchanges map[string]exchangeConfig, hasCredentials func(string) bool, mode string) map[string]interface{} {
	for _, sr := range s.Sessions {
		prefix := sr.EnvVarPrefix
		if prefix == "" {
			prefix = exchangeEnvPrefix(sr.Exchange)
		}
		if _, exists := sessions[sr.Name]; !exists {
			sessions[sr.Name] = sessionConfig{
				Exchange:     sr.Exchange,
				EnvVarPrefix: prefix,
				Futures:      sr.Futures,
				PublicOnly:   !hasCredentials(sr.Exchange),
			}
		}
		symbol := "BTCUSDT"
		if v, ok := params["symbol"].(string); ok && v != "" {
			symbol = v
		}
		if _, exists := exchanges[sr.Exchange]; !exists {
			exchanges[sr.Exchange] = exchangeConfig{Symbol: symbol}
		}
	}

	return map[string]interface{}{
		s.Strategy: params,
	}
}

func buildBacktestYAML(strategy string, rawConfig json.RawMessage, startTime, endTime, overrideExchange, overrideSymbol string) ([]byte, error) {
	var allParams map[string]interface{}
	if err := json.Unmarshal(rawConfig, &allParams); err != nil {
		return nil, err
	}

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
		ExchangeStrategies []map[string]interface{} `yaml:"exchangeStrategies"`
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
		ExchangeStrategies: []map[string]interface{}{
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
	return map[string]string{quote: "10000"}
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

func cloneUserContainer(uc *UserContainer) *UserContainer {
	cp := *uc
	if len(uc.Strategies) > 0 {
		cp.Strategies = make([]StrategyEntry, len(uc.Strategies))
		copy(cp.Strategies, uc.Strategies)
	}
	return &cp
}
