package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_PaperTradingChain validates the complete paper trading chain:
// strategy(paper) → buildUserYAML → YAML has PAPER_TRADE → envArgs has PAPER_TRADE=1 → no credential injection
func TestE2E_PaperTradingChain(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{
		ManagerToken:   "test-token",
		DataDir:        dir,
		DataVolume:     "bbgo-data",
		DockerNetwork:  "bbgo-net",
		BBGOImage:      "bbgo-base:latest",
		BBGOPort:       8080,
		BBGOGRPCPort:   9090,
		MarketDataAddr: "bbgo-marketdata:9090",
	}
	cm := NewContainerManager(cfg, creds, nil)

	strategies := []StrategyEntry{
		{
			Name:     "Paper Grid",
			Exchange: "binance",
			Strategy: "grid2",
			Mode:     "paper",
			Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`),
		},
	}

	// Step 1: Build YAML — must have PAPER_TRADE and publicOnly
	yaml, err := buildUserYAML("test-user", ModePaper, strategies, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("build yaml: %v", err)
	}
	yamlStr := string(yaml)
	if !strings.Contains(yamlStr, "PAPER_TRADE:") {
		t.Error("step 1: YAML missing PAPER_TRADE")
	}
	if !strings.Contains(yamlStr, "publicOnly: true") {
		t.Error("step 1: YAML missing publicOnly: true (no credentials)")
	}
	if !strings.Contains(yamlStr, "grid2:") {
		t.Error("step 1: YAML missing grid2 strategy")
	}

	// Step 2: Write YAML to disk
	userDir := filepath.Join(dir, "test-user")
	os.MkdirAll(userDir, 0o755)
	yamlPath := filepath.Join(userDir, "bbgo.yaml")
	if err := os.WriteFile(yamlPath, yaml, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	data, _ := os.ReadFile(yamlPath)
	if string(data) != yamlStr {
		t.Error("step 2: YAML on disk differs from generated")
	}

	// Step 3: envArgs — must have PAPER_TRADE=1, must NOT have API keys
	args := cm.envArgs("test-user", ModePaper, strategies)
	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if !findEnv("PAPER_TRADE=1") {
		t.Error("step 3: envArgs missing PAPER_TRADE=1")
	}
	if findEnv("BINANCE_API_KEY") {
		t.Error("step 3: envArgs should NOT have BINANCE_API_KEY for paper without credentials")
	}

	// Step 4: Verify DB and market data env vars present
	if !findEnv("DB_DRIVER=sqlite3") {
		t.Error("step 4: envArgs missing DB_DRIVER=sqlite3 for paper")
	}
	if findEnv("SUPABASE_URL") {
		t.Error("step 4: paper mode must NOT inject SUPABASE_URL")
	}
	if !findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata:9090") {
		t.Error("step 4: envArgs missing MARKET_DATA_SERVICE_URL")
	}
}

// TestE2E_LiveTradingChain validates the complete live trading chain:
// strategy(live) + credentials → buildUserYAML → no PAPER_TRADE → envArgs injects API keys
func TestE2E_LiveTradingChain(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{
		ManagerToken:   "test-token",
		DataDir:        dir,
		DataVolume:     "bbgo-data",
		DockerNetwork:  "bbgo-net",
		BBGOImage:      "bbgo-base:latest",
		BBGOPort:       8080,
		BBGOGRPCPort:   9090,
		MarketDataAddr: "bbgo-marketdata:9090",
	}
	cm := NewContainerManager(cfg, creds, nil)

	insertTestCredential(t, creds, "test-user", "binance", "live-key-123", "live-secret-456")

	strategies := []StrategyEntry{
		{
			Name:     "Live Grid",
			Exchange: "binance",
			Strategy: "grid2",
			Mode:     "live",
			Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`),
		},
	}

	// Step 1: Build YAML — no PAPER_TRADE, no publicOnly
	yaml, err := buildUserYAML("test-user", ModeLive, strategies, func(exchange string) bool {
		_, _, _, err := creds.GetDecrypted("test-user", exchange)
		return err == nil
	})
	if err != nil {
		t.Fatalf("build yaml: %v", err)
	}
	yamlStr := string(yaml)
	if strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Error("step 1: YAML should NOT have PAPER_TRADE for live mode")
	}
	if strings.Contains(yamlStr, "publicOnly: true") {
		t.Error("step 1: YAML should NOT have publicOnly when credentials exist")
	}

	// Step 2: envArgs — API keys injected, no PAPER_TRADE
	args := cm.envArgs("test-user", ModeLive, strategies)
	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if findEnv("PAPER_TRADE=1") {
		t.Error("step 2: envArgs should NOT have PAPER_TRADE=1 for live mode")
	}
	if !findEnv("BINANCE_API_KEY=live-key-123") {
		t.Error("step 2: envArgs missing BINANCE_API_KEY")
	}
	if !findEnv("BINANCE_API_SECRET=live-secret-456") {
		t.Error("step 2: envArgs missing BINANCE_API_SECRET")
	}
}

// TestE2E_CrossExchangeChain validates cross-exchange strategy YAML and env injection.
func TestE2E_CrossExchangeChain(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{
		ManagerToken:   "test-token",
		DataDir:        dir,
		DataVolume:     "bbgo-data",
		DockerNetwork:  "bbgo-net",
		BBGOImage:      "bbgo-base:latest",
		BBGOPort:       8080,
		BBGOGRPCPort:   9090,
		MarketDataAddr: "",
	}
	cm := NewContainerManager(cfg, creds, nil)

	insertTestCredential(t, creds, "test-user", "binance", "bin-key", "bin-secret")
	insertTestCredential(t, creds, "test-user", "bybit", "byb-key", "byb-secret")

	strategies := []StrategyEntry{
		{
			Name:          "XMaker",
			Strategy:      "xmaker",
			Mode:          "live",
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
				{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
			},
			Config: rawJSON(`{"symbol":"BTCUSDT","spread":0.001}`),
		},
	}

	// Step 1: YAML has crossExchangeStrategies, both exchanges, futures flag
	yaml, err := buildUserYAML("test-user", ModeLive, strategies, func(exchange string) bool {
		_, _, _, err := creds.GetDecrypted("test-user", exchange)
		return err == nil
	})
	if err != nil {
		t.Fatalf("build yaml: %v", err)
	}
	yamlStr := string(yaml)
	if !strings.Contains(yamlStr, "crossExchangeStrategies:") {
		t.Error("step 1: missing crossExchangeStrategies section")
	}
	if !strings.Contains(yamlStr, "xmaker:") {
		t.Error("step 1: missing xmaker strategy")
	}
	if !strings.Contains(yamlStr, "futures: true") {
		t.Error("step 1: missing futures: true for bybit session")
	}

	// Step 2: envArgs injects both exchange credentials
	args := cm.envArgs("test-user", ModeLive, strategies)
	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if !findEnv("BINANCE_API_KEY=bin-key") {
		t.Error("step 2: missing BINANCE_API_KEY")
	}
	if !findEnv("BYBIT_API_KEY=byb-key") {
		t.Error("step 2: missing BYBIT_API_KEY")
	}
}
