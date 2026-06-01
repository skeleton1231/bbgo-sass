package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// paperExchange is the only exchange supported in paper mode.
const paperExchange = "binance"

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
func normalizeStrategyConfig(strategy string, params map[string]any) (string, map[string]any) {
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
	Name          string              `json:"name"`
	Exchange      string              `json:"exchange"`
	Strategy      string              `json:"strategy"`
	Config        json.RawMessage     `json:"config"`
	Mode          string              `json:"mode"`
	CrossExchange bool                `json:"crossExchange"`
	Sessions      []SessionRoleConfig `json:"sessions,omitempty"`
}

// UserMode identifies a user+mode pair for container tracking.
type UserMode struct {
	UserID string
	Mode   string
}

// StrategyStore manages bbgo.yaml files on disk as the source of truth
// for user strategy configurations. No in-memory state, no database sync.
type StrategyStore struct {
	dataDir string
}

func NewStrategyStore(dataDir string) *StrategyStore {
	return &StrategyStore{dataDir: dataDir}
}

// yamlPath returns the path to bbgo.yaml for a given user/mode.
func (s *StrategyStore) yamlPath(userID, mode string) string {
	dir := filepath.Join(s.dataDir, userID)
	if mode == ModePaper {
		dir += "-paper"
	}
	return filepath.Join(dir, "bbgo.yaml")
}

// hostDir returns the host directory for user/mode config files.
func (s *StrategyStore) hostDir(userID, mode string) string {
	dir := filepath.Join(s.dataDir, userID)
	if mode == ModePaper {
		dir += "-paper"
	}
	return dir
}

// ReadYAML reads the raw bbgo.yaml content for a user/mode.
func (s *StrategyStore) ReadYAML(userID, mode string) ([]byte, error) {
	return os.ReadFile(s.yamlPath(userID, mode))
}

// YAMLExists returns true if a bbgo.yaml exists for the user/mode.
func (s *StrategyStore) YAMLExists(userID, mode string) bool {
	_, err := os.Stat(s.yamlPath(userID, mode))
	return err == nil
}

// ListStrategies parses bbgo.yaml and returns the configured strategy entries.
func (s *StrategyStore) ListStrategies(userID, mode string) ([]StrategyEntry, error) {
	data, err := s.ReadYAML(userID, mode)
	if err != nil {
		return nil, err
	}
	return parseStrategiesFromYAML(data)
}

// parseStrategiesFromYAML extracts StrategyEntry list from bbgo config YAML.
func parseStrategiesFromYAML(data []byte) ([]StrategyEntry, error) {
	var cfg bbgoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse bbgo.yaml: %w", err)
	}

	var entries []StrategyEntry

	for _, es := range cfg.ExchangeStrategies {
		entry, ok := parseExchangeStrategyEntry(es)
		if ok {
			entries = append(entries, entry)
		}
	}

	for _, cs := range cfg.CrossExchangeStrategies {
		entry, ok := parseCrossStrategyEntry(cs)
		if ok {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// parseExchangeStrategyEntry extracts a StrategyEntry from an exchangeStrategy YAML map.
// Format: {"on": "binance", "grid2": {symbol: "BTCUSDT", ...}}
func parseExchangeStrategyEntry(m map[string]any) (StrategyEntry, bool) {
	exchange, _ := m["on"].(string)
	if exchange == "" {
		return StrategyEntry{}, false
	}

	for key, val := range m {
		if key == "on" {
			continue
		}
		config, err := json.Marshal(val)
		if err != nil {
			continue
		}
		return StrategyEntry{
			Exchange: exchange,
			Strategy: key,
			Config:   config,
		}, true
	}
	return StrategyEntry{}, false
}

// parseCrossStrategyEntry extracts a StrategyEntry from a crossExchangeStrategy YAML map.
// Format: {"xmaker": {symbol: "BTCUSDT", ...}}
func parseCrossStrategyEntry(m map[string]any) (StrategyEntry, bool) {
	for key, val := range m {
		config, err := json.Marshal(val)
		if err != nil {
			continue
		}
		return StrategyEntry{
			Strategy:      key,
			Config:        config,
			CrossExchange: true,
		}, true
	}
	return StrategyEntry{}, false
}

// AddStrategy adds a new strategy, regenerates bbgo.yaml, and writes to disk.
func (s *StrategyStore) AddStrategy(userID, mode string, entry StrategyEntry, hasCredentials func(exchange string) bool) error {
	var existing []StrategyEntry
	if data, err := s.ReadYAML(userID, mode); err == nil {
		existing, _ = parseStrategiesFromYAML(data)
	}
	existing = append(existing, entry)

	return s.writeYAML(userID, mode, existing, hasCredentials)
}

// RemoveStrategy removes a strategy by type+symbol match, regenerates bbgo.yaml.
// Returns (found bool, err error).
func (s *StrategyStore) RemoveStrategy(userID, mode, strategy, symbol string) (bool, error) {
	entries, err := s.ListStrategies(userID, mode)
	if err != nil {
		return false, err
	}

	filtered := make([]StrategyEntry, 0, len(entries))
	found := false
	for _, e := range entries {
		eSymbol := extractSymbolFromConfig(e.Config)
		if e.Strategy == strategy && eSymbol == symbol && !found {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return false, nil
	}

	if len(filtered) == 0 {
		os.Remove(s.yamlPath(userID, mode))
		return true, nil
	}

	hasCred := func(string) bool { return false }
	return true, s.writeYAML(userID, mode, filtered, hasCred)
}

// writeYAML regenerates bbgo.yaml from the given strategy entries.
func (s *StrategyStore) writeYAML(userID, mode string, entries []StrategyEntry, hasCredentials func(exchange string) bool) error {
	dir := s.hostDir(userID, mode)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	yamlContent, err := buildUserYAML(userID, mode, entries, hasCredentials)
	if err != nil {
		return err
	}

	return os.WriteFile(s.yamlPath(userID, mode), yamlContent, 0o644)
}

// extractSymbolFromConfig returns the symbol from a strategy config JSON.
func extractSymbolFromConfig(config json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return ""
	}
	s, _ := m["symbol"].(string)
	return s
}

// ScanUsers scans DATA_DIR for bbgo.yaml files and returns discovered user/mode pairs.
func (s *StrategyStore) ScanUsers() []UserMode {
	var result []UserMode
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		yamlPath := filepath.Join(s.dataDir, name, "bbgo.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			userID := name
			result = append(result, UserMode{UserID: userID, Mode: ModeLive})
		}
		if strings.HasSuffix(name, "-paper") {
			yamlPath := filepath.Join(s.dataDir, name, "bbgo.yaml")
			if _, err := os.Stat(yamlPath); err == nil {
				userID := strings.TrimSuffix(name, "-paper")
				result = append(result, UserMode{UserID: userID, Mode: ModePaper})
			}
		}
	}
	return result
}

// --- YAML config types ---

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
	ExchangeStrategies      []map[string]any  `yaml:"exchangeStrategies,omitempty"`
	CrossExchangeStrategies []map[string]any  `yaml:"crossExchangeStrategies,omitempty"`
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

// buildUserYAML generates the bbgo YAML config from strategy entries.
func buildUserYAML(userID, mode string, strategies []StrategyEntry, hasCredentials func(exchange string) bool) ([]byte, error) {
	exchanges := map[string]exchangeConfig{}
	sessions := map[string]sessionConfig{}
	var exchangeStrategies []map[string]any
	var crossStrategies []map[string]any

	for _, s := range strategies {
		var params map[string]any
		if len(s.Config) == 0 {
			params = map[string]any{}
		} else if err := json.Unmarshal(s.Config, &params); err != nil {
			var rawStr string
			if err2 := json.Unmarshal(s.Config, &rawStr); err2 == nil {
				strategyID := s.Strategy
				if alias, ok := legacyStrategyAliases[strategyID]; ok {
					strategyID = alias
				}
				entry := map[string]any{
					"on":       s.Exchange,
					strategyID: rawStr,
				}
				exchangeStrategies = append(exchangeStrategies, entry)
			}
			continue
		}

		s.Strategy, params = normalizeStrategyConfig(s.Strategy, params)

		if s.CrossExchange {
			csEntry := buildCrossExchangeStrategy(s, params, sessions, exchanges, hasCredentials, mode)
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

		entry := map[string]any{
			"on":       s.Exchange,
			s.Strategy: params,
		}
		exchangeStrategies = append(exchangeStrategies, entry)
	}

	dataDir := userID
	if mode == ModePaper {
		dataDir = userID + "-paper"
	}
	cfg := bbgoConfig{
		Database: &databaseConfig{
			Driver: "sqlite3",
			DSN:    fmt.Sprintf("file:/data/%s/bbgo.db?cache=shared&_journal_mode=WAL", dataDir),
		},
		Sessions: sessions,
		Exchange: exchanges,
		Sync: &syncConfig{
			UserDataStream: &syncUserDataStreamConfig{
				Trades:       true,
				FilledOrders: true,
			},
		},
		ExchangeStrategies:      exchangeStrategies,
		CrossExchangeStrategies: crossStrategies,
	}
	cfg.Environment = &environmentConfig{}
	if mode == ModePaper {
		cfg.Environment.PaperTrade = "1"
	} else {
		cfg.Environment.DisableStartupBalanceQuery = true
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal bbgo config for user %s: %w", userID, err)
	}
	return out, nil
}

func buildCrossExchangeStrategy(s StrategyEntry, params map[string]any, sessions map[string]sessionConfig, exchanges map[string]exchangeConfig, hasCredentials func(string) bool, mode string) map[string]any {
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

	return map[string]any{
		s.Strategy: params,
	}
}

func buildBacktestYAML(strategy string, rawConfig json.RawMessage, startTime, endTime, overrideExchange, overrideSymbol string) ([]byte, error) {
	var allParams map[string]any
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
