package main

import (
	"encoding/json"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusError    = "error"
	StatusStarting = "starting"
)

// legacyStrategyAliases maps old frontend IDs to the correct bbgo strategy IDs.
var legacyStrategyAliases = map[string]string{
	"ewoDgtrd":        "ewo_dgtrd",
	"sentinel":        "sentinel_anomaly",
	"autobuy":         "autobuy_scheduled",
	"rebalance":       "rebalance_portfolio",
}

// legacyFieldAliases maps strategy IDs to old→new field renames.
var legacyFieldAliases = map[string]map[string]string{
	"dca": {"interval": "investmentInterval"},
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
	Status     string          `json:"status"`
	Strategies []StrategyEntry `json:"strategies"`
}

type UserContainerManager struct {
	mu    sync.RWMutex
	users map[string]*UserContainer
}

func NewUserContainerManager() *UserContainerManager {
	return &UserContainerManager{users: make(map[string]*UserContainer)}
}

func (m *UserContainerManager) Get(userID string) (*UserContainer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	uc, ok := m.users[userID]
	if !ok {
		return nil, false
	}
	return cloneUserContainer(uc), true
}

func (m *UserContainerManager) UpdateStatus(userID, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if uc, ok := m.users[userID]; ok {
		uc.Status = status
	}
}

func (m *UserContainerManager) AddStrategy(userID string, entry StrategyEntry) (*UserContainer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uc, ok := m.users[userID]
	created := !ok
	if !ok {
		uc = &UserContainer{
			UserID:     userID,
			Status:     StatusStopped,
			Strategies: []StrategyEntry{},
		}
		m.users[userID] = uc
	}
	uc.Strategies = append(uc.Strategies, entry)
	return cloneUserContainer(uc), created
}

func (m *UserContainerManager) RemoveStrategy(userID, strategyID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	uc, ok := m.users[userID]
	if !ok {
		return false
	}
	for i, s := range uc.Strategies {
		if s.ID == strategyID {
			uc.Strategies = append(uc.Strategies[:i], uc.Strategies[i+1:]...)
			return true
		}
	}
	return false
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
	for _, uc := range users {
		m.users[uc.UserID] = uc
	}
}

type bbgoConfig struct {
	Sessions               map[string]sessionConfig    `yaml:"sessions,omitempty"`
	Exchange               map[string]exchangeConfig   `yaml:"exchange"`
	Environment            *environmentConfig          `yaml:"environment,omitempty"`
	ExchangeStrategies     []map[string]interface{}    `yaml:"exchangeStrategies,omitempty"`
	CrossExchangeStrategies []map[string]interface{}   `yaml:"crossExchangeStrategies,omitempty"`
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
	PaperTrade                string `yaml:"PAPER_TRADE,omitempty"`
	DisableStartupBalanceQuery bool   `yaml:"disablestartupbalancequery"`
}

func buildUserYAML(uc *UserContainer, hasCredentials func(exchange string) bool) string {
	exchanges := map[string]exchangeConfig{}
	sessions := map[string]sessionConfig{}
	var exchangeStrategies []map[string]interface{}
	var crossStrategies []map[string]interface{}
	hasPaper := false

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
					"on":         s.Exchange,
					strategyID:   rawStr,
				}
				exchangeStrategies = append(exchangeStrategies, entry)
			}
			continue
		}

		s.Strategy, params = normalizeStrategyConfig(s.Strategy, params)

		if s.CrossExchange {
			csEntry := buildCrossExchangeStrategy(s, params, sessions, exchanges, hasCredentials)
			crossStrategies = append(crossStrategies, csEntry)
			if s.Mode == "paper" {
				hasPaper = true
			}
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
				"on":         s.Exchange,
				s.Strategy:   params,
			}
		exchangeStrategies = append(exchangeStrategies, entry)

		if s.Mode == "paper" {
			hasPaper = true
		}
	}

	cfg := bbgoConfig{
		Sessions:                sessions,
		Exchange:                exchanges,
		ExchangeStrategies:      exchangeStrategies,
		CrossExchangeStrategies: crossStrategies,
	}
	cfg.Environment = &environmentConfig{DisableStartupBalanceQuery: true}
	if hasPaper {
		cfg.Environment.PaperTrade = "1"
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return ""
	}
	return string(out)
}

func buildCrossExchangeStrategy(s StrategyEntry, params map[string]interface{}, sessions map[string]sessionConfig, exchanges map[string]exchangeConfig, hasCredentials func(string) bool) map[string]interface{} {
	for _, sr := range s.Sessions {
		if _, exists := sessions[sr.Name]; !exists {
			sessions[sr.Name] = sessionConfig{
				Exchange:     sr.Exchange,
				EnvVarPrefix: sr.EnvVarPrefix,
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

func buildBacktestYAML(strategy string, rawConfig json.RawMessage, startTime, endTime string) ([]byte, error) {
	var allParams map[string]interface{}
	if err := json.Unmarshal(rawConfig, &allParams); err != nil {
		return nil, err
	}

	exchange := "binance"
	if v, ok := allParams["exchange"].(string); ok && v != "" {
		exchange = v
	}

	if startTime == "" {
		startTime = "2024-01-01"
	}
	if endTime == "" {
		endTime = "2024-06-01"
	}

	prefix := exchangeEnvPrefix(exchange)
	delete(allParams, "exchange")

	strategy, allParams = normalizeStrategyConfig(strategy, allParams)

	symbol := "BTCUSDT"
	if v, ok := allParams["symbol"].(string); ok && v != "" {
		symbol = v
	}

	// Ensure interval is set for strategies that require kline subscriptions
	if _, ok := allParams["interval"]; !ok {
		allParams["interval"] = "1h"
	}

	btCfg := struct {
		Exchange map[string]struct {
			Symbol string `yaml:"symbol"`
		} `yaml:"exchange"`
		Sessions map[string]struct {
			Exchange     string `yaml:"exchange"`
			EnvVarPrefix string `yaml:"envVarPrefix"`
		} `yaml:"sessions"`
		ExchangeStrategies []map[string]interface{} `yaml:"exchangeStrategies"`
		Backtest struct {
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
				"on":       exchange,
				strategy:   allParams,
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
				exchange: {Balances: map[string]string{"USDT": "10000", "BTC": "0.1"}},
			},
		},
	}

	return yaml.Marshal(btCfg)
}

func cloneUserContainer(uc *UserContainer) *UserContainer {
	cp := *uc
	if len(uc.Strategies) > 0 {
		cp.Strategies = make([]StrategyEntry, len(uc.Strategies))
		copy(cp.Strategies, uc.Strategies)
	}
	return &cp
}
