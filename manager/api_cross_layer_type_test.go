package main

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestCrossExchangeYAML_XMaker verifies the full YAML generation for an
// xmaker cross-exchange strategy with maker+hedge sessions.
func TestCrossExchangeYAML_XMaker(t *testing.T) {
	strategies := []StrategyEntry{
		{
			Strategy:      "xmaker",
			Mode:          "live",
			CrossExchange: true,
			Config: rawJSON(`{
				"symbol": "BTCUSDT",
				"spread": 0.001,
				"quantity": 0.01,
				"updateInterval": "1m",
				"hedgeInterval": "5m"
			}`),
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
				{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
			},
		},
	}

	yamlBytes, err := buildUserYAML("test-user", ModeLive, strategies, func(ex string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)

	if !strings.Contains(yamlStr, "maker:") {
		t.Error("expected maker session in YAML")
	}
	if !strings.Contains(yamlStr, "hedge:") {
		t.Error("expected hedge session in YAML")
	}
	if !strings.Contains(yamlStr, "exchange: binance") {
		t.Error("expected binance exchange for maker session")
	}
	if !strings.Contains(yamlStr, "exchange: bybit") {
		t.Error("expected bybit exchange for hedge session")
	}
	if !strings.Contains(yamlStr, "futures: true") {
		t.Error("expected futures:true for hedge session")
	}
	if !strings.Contains(yamlStr, "crossExchangeStrategies:") {
		t.Error("expected crossExchangeStrategies key")
	}
	if strings.Contains(yamlStr, "exchangeStrategies:") {
		t.Error("should NOT have exchangeStrategies for cross-exchange-only config")
	}
	if strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should not be set for live cross-exchange strategy")
	}
	if !strings.Contains(yamlStr, "xmaker:") {
		t.Error("expected xmaker strategy in YAML")
	}
	if !strings.Contains(yamlStr, "BTCUSDT") {
		t.Error("expected BTCUSDT symbol in YAML")
	}
}

// TestCrossExchangeYAML_PaperMode verifies paper mode for cross-exchange.
func TestCrossExchangeYAML_PaperMode(t *testing.T) {
	strategies := []StrategyEntry{
		{
			Strategy:      "xmaker",
			Mode:          "paper",
			CrossExchange: true,
			Config:        rawJSON(`{"symbol": "ETHUSDT", "quantity": 0.1}`),
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
				{Name: "hedge", Exchange: "okex", EnvVarPrefix: "OKEX", Futures: true},
			},
		},
	}

	yamlBytes, err := buildUserYAML("test-user", ModePaper, strategies, func(ex string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)

	if !strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Error("expected PAPER_TRADE for paper mode cross-exchange strategy")
	}
	if !strings.Contains(yamlStr, "ETHUSDT") {
		t.Error("expected ETHUSDT symbol")
	}
}

// TestEnvArgs_MultiExchangeCredentials verifies that docker env args include
// credentials for all exchanges used by a cross-exchange strategy.
func TestEnvArgs_MultiExchangeCredentials(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "test-user", "binance", "binance-key", "binance-secret")
	insertTestCredential(t, creds, "test-user", "bybit", "bybit-key", "bybit-secret")

	cfg := &Config{DataVolume: "vol", DockerNetwork: "net", BBGOImage: "img"}
	cm := NewContainerManager(cfg, creds, nil)

	strategies := []StrategyEntry{
		{
			Strategy:      "xmaker",
			Mode:          "live",
			CrossExchange: true,
			Config:        rawJSON(`{"symbol":"BTCUSDT","quantity":0.01}`),
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance"},
				{Name: "hedge", Exchange: "bybit", Futures: true},
			},
		},
	}

	args := cm.envArgs("test-user", ModeLive, strategies)
	cmdStr := strings.Join(args, " ")

	if !strings.Contains(cmdStr, "BINANCE_API_KEY=binance-key") {
		t.Error("expected BINANCE_API_KEY injection")
	}
	if !strings.Contains(cmdStr, "BINANCE_API_SECRET=binance-secret") {
		t.Error("expected BINANCE_API_SECRET injection")
	}
	if !strings.Contains(cmdStr, "BYBIT_API_KEY=bybit-key") {
		t.Error("expected BYBIT_API_KEY injection")
	}
	if !strings.Contains(cmdStr, "BYBIT_API_SECRET=bybit-secret") {
		t.Error("expected BYBIT_API_SECRET injection")
	}
	if strings.Contains(cmdStr, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should not be set for live cross-exchange")
	}
}

// TestEnvArgs_CredentialDeduplication verifies that if multiple strategies
// use the same exchange, credentials are only injected once.
func TestEnvArgs_CredentialDeduplication(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "test-user", "binance", "key", "secret")

	cfg := &Config{DataVolume: "vol", DockerNetwork: "net", BBGOImage: "img"}
	cm := NewContainerManager(cfg, creds, nil)

	strategies := []StrategyEntry{
		{Exchange: "binance", Strategy: "grid", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		{Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"ETHUSDT"}`)},
	}

	args := cm.envArgs("test-user", ModeLive, strategies)
	count := strings.Count(strings.Join(args, " "), "BINANCE_API_KEY=key")
	if count != 1 {
		t.Errorf("expected BINANCE_API_KEY injected exactly once, got %d", count)
	}
}

// TestYAMLGeneration_VariousStrategies verifies YAML generation for
// multiple strategy types to ensure config field mapping works.
func TestYAMLGeneration_VariousStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		config   string
		mode     string
		want     []string
		dontWant []string
	}{
		{
			name:     "grid paper mode",
			strategy: "grid",
			config:   `{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`,
			mode:     "paper",
			want:     []string{"grid:", "BTCUSDT", "gridNumber:", "PAPER_TRADE"},
		},
		{
			name:     "supertrend live mode",
			strategy: "supertrend",
			config:   `{"symbol":"ETHUSDT","interval":"4h","quantity":0.01,"supertrendMultiplier":3}`,
			mode:     "live",
			want:     []string{"supertrend:", "ETHUSDT", "supertrendMultiplier:"},
			dontWant: []string{"PAPER_TRADE"},
		},
		{
			name:     "dca paper mode",
			strategy: "dca",
			config:   `{"symbol":"BTCUSDT","investmentInterval":"1h","budget":100}`,
			mode:     "paper",
			want:     []string{"dca:", "BTCUSDT", "investmentInterval:", "PAPER_TRADE"},
		},
		{
			name:     "bollgrid paper mode",
			strategy: "bollgrid",
			config:   `{"symbol":"BTCUSDT","interval":"1h","gridNumber":8,"gridPips":50,"quantity":0.001}`,
			mode:     "paper",
			want:     []string{"bollgrid:", "BTCUSDT", "gridPips:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerMode := ModeLive
			if tt.mode == "paper" {
				containerMode = ModePaper
			}
			strategies := []StrategyEntry{
				{
					Strategy: tt.strategy,
					Mode:     tt.mode,
					Config:   rawJSON(tt.config),
					Exchange: "binance",
				},
			}

			yamlBytes, err := buildUserYAML("test-user", containerMode, strategies, func(ex string) bool { return tt.mode == "live" })
			if err != nil {
				t.Fatal(err)
			}
			yamlStr := string(yamlBytes)

			for _, w := range tt.want {
				if !strings.Contains(yamlStr, w) {
					t.Errorf("expected %q in YAML\n%s", w, yamlStr)
				}
			}
			for _, dw := range tt.dontWant {
				if strings.Contains(yamlStr, dw) {
					t.Errorf("did NOT expect %q in YAML\n%s", dw, yamlStr)
				}
			}
		})
	}
}

// TestLiveOnlyStrategies_FrontendBackendAlignment verifies that every strategy
// marked liveOnly in the frontend schema is also liveOnly in the backend.
func TestLiveOnlyStrategies_FrontendBackendAlignment(t *testing.T) {
	frontendLiveOnly := map[string]bool{
		"autoborrow": true, "convert": true, "deposit2transfer": true,
		"sentinel": true,
	}

	for alias, canonical := range legacyStrategyAliases {
		if frontendLiveOnly[canonical] {
			frontendLiveOnly[alias] = true
		}
	}

	for strat := range liveOnlyStrategies {
		if !frontendLiveOnly[strat] {
			t.Errorf("backend liveOnly strategy %q not in frontend liveOnly set", strat)
		}
	}

	for strat := range frontendLiveOnly {
		normalized := strat
		if alias, ok := legacyStrategyAliases[strat]; ok {
			normalized = alias
		}
		if !liveOnlyStrategies[normalized] {
			t.Errorf("frontend liveOnly strategy %q (normalized: %q) not in backend liveOnly set", strat, normalized)
		}
	}
}

// TestExchangePrefixes_FrontendBackendAlignment verifies the env var prefix
// mapping is identical between frontend and backend.
func TestExchangePrefixes_FrontendBackendAlignment(t *testing.T) {
	frontendPrefixes := map[string]string{
		"binance":  "BINANCE",
		"okex":     "OKEX",
		"bybit":    "BYBIT",
		"bitget":   "BITGET",
		"kucoin":   "KUCOIN",
		"max":      "MAX",
		"coinbase": "COINBASE",
		"bitfinex": "BITFINEX",
	}

	for exchange, expectedPrefix := range frontendPrefixes {
		got := exchangeEnvPrefix(exchange)
		if got != expectedPrefix {
			t.Errorf("exchange %q: frontend=%q backend=%q", exchange, expectedPrefix, got)
		}
	}
}

// TestYAMLStructure_ParsesCorrectly verifies generated YAML is valid bbgo config.
func TestYAMLStructure_ParsesCorrectly(t *testing.T) {
	strategies := []StrategyEntry{
		{
			Exchange: "binance",
			Strategy: "grid2",
			Mode:     "paper",
			Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`),
		},
	}

	yamlBytes, err := buildUserYAML("test-user", ModePaper, strategies, func(ex string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	var cfg bbgoConfig
	if err := yaml.Unmarshal(yamlBytes, &cfg); err != nil {
		t.Fatalf("generated YAML is invalid: %v\n%s", err, string(yamlBytes))
	}

	if len(cfg.Sessions) == 0 {
		t.Error("expected at least one session")
	}
	if cfg.Sessions["binance"].Exchange != "binance" {
		t.Error("expected binance session")
	}
	if cfg.Sessions["binance"].PublicOnly != true {
		t.Error("expected PublicOnly=true when no credentials")
	}
	if cfg.Environment == nil || cfg.Environment.PaperTrade != "1" {
		t.Error("expected PAPER_TRADE=1 in environment")
	}
	if len(cfg.ExchangeStrategies) != 1 {
		t.Fatalf("expected 1 exchange strategy, got %d", len(cfg.ExchangeStrategies))
	}
	es := cfg.ExchangeStrategies[0]
	if es["on"] != "binance" {
		t.Error("expected strategy on binance session")
	}
	if _, ok := es["grid2"]; !ok {
		t.Error("expected grid2 key in strategy config")
	}
}
