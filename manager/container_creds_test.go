package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupContainerManager(t *testing.T) (*ContainerManager, *CredentialStore) {
	t.Helper()
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
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
	cm := NewContainerManager(cfg, creds, nil, nil)
	return cm, creds
}

func testInst(userID, mode, strategy, exchange, symbol string) *StrategyInstance {
	return &StrategyInstance{
		UserID:     userID,
		Mode:       mode,
		Strategy:   strategy,
		Exchange:   exchange,
		Symbol:     symbol,
		InstanceID: strategy + "-" + symbol,
	}
}

func insertTestCredential(t *testing.T, cs *CredentialStore, userID, exchange, apiKey, apiSecret string) {
	t.Helper()
	enc := cs.crypto
	keyEnc, err := enc.Encrypt(apiKey)
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	secretEnc, err := enc.Encrypt(apiSecret)
	if err != nil {
		t.Fatalf("encrypt api secret: %v", err)
	}
	cred := ExchangeCredential{
		ID:                 generateID("cred"),
		UserID:             userID,
		Exchange:           exchange,
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
	}
	if err := cs.Upsert(cred); err != nil {
		t.Fatalf("upsert credential: %v", err)
	}
}

func TestEnvArgs_PaperMode_SetsEnv(t *testing.T) {
	cm, _ := setupContainerManager(t)
	args := cm.instanceEnvArgs(testInst("test-user", ModePaper, "grid2", "binance", "BTCUSDT"))

	found := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" && args[i+1] == "PAPER_TRADE=1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected PAPER_TRADE=1 env var for paper mode")
	}
}

func TestEnvArgs_LiveMode_NoPaperTradeEnv(t *testing.T) {
	cm, _ := setupContainerManager(t)
	args := cm.instanceEnvArgs(testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT"))

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" && args[i+1] == "PAPER_TRADE=1" {
			t.Error("PAPER_TRADE=1 should NOT be set for live mode")
		}
	}
}

func TestEnvArgs_InjectsCredentials(t *testing.T) {
	cm, creds := setupContainerManager(t)
	insertTestCredential(t, creds, "test-user", "binance", "my-api-key", "my-api-secret")

	args := cm.instanceEnvArgs(testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT"))

	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if !findEnv("BINANCE_API_KEY=my-api-key") {
		t.Error("expected BINANCE_API_KEY env var")
	}
	if !findEnv("BINANCE_API_SECRET=my-api-secret") {
		t.Error("expected BINANCE_API_SECRET env var")
	}
}

func TestEnvArgs_NoCredentials_NoInjection(t *testing.T) {
	cm, _ := setupContainerManager(t)
	args := cm.instanceEnvArgs(testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT"))

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" {
			val := args[i+1]
			if len(val) > 10 && val[:10] == "BINANCE_AP" {
				t.Errorf("should not inject credentials when none stored, got: %s", val)
			}
		}
	}
}

func TestEnvArgs_PaperMode_NoCredentialsInjected(t *testing.T) {
	cm, creds := setupContainerManager(t)
	insertTestCredential(t, creds, "test-user", "binance", "live-key", "live-secret")

	args := cm.instanceEnvArgs(testInst("test-user", ModePaper, "grid2", "binance", "BTCUSDT"))

	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if findEnv("BINANCE_API_KEY=live-key") {
		t.Error("paper mode should NOT inject any credentials")
	}
	if findEnv("BINANCE_TESTNET=1") {
		t.Error("paper mode should NOT set BINANCE_TESTNET=1")
	}
}

func TestEnvArgs_PaperMode_CrossExchange_NoCredentialsInjected(t *testing.T) {
	cm, creds := setupContainerManager(t)
	insertTestCredential(t, creds, "test-user", "binance", "binance-key", "binance-secret")
	insertTestCredential(t, creds, "test-user", "bybit", "bybit-key", "bybit-secret")

	inst := &StrategyInstance{
		UserID:        "test-user",
		Mode:          ModePaper,
		Strategy:      "xmaker",
		Symbol:        "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
			{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
		},
		InstanceID: "xmaker-BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)

	joined := strings.Join(args, " ")
	if strings.Contains(joined, "BINANCE_API_KEY") {
		t.Error("paper mode should NOT inject any credentials")
	}
	if strings.Contains(joined, "BYBIT_API_KEY") {
		t.Error("paper mode should NOT inject any credentials")
	}
}

func TestEnvArgs_CrossExchange_InjectsAllSessionExchanges(t *testing.T) {
	cm, creds := setupContainerManager(t)
	insertTestCredential(t, creds, "test-user", "binance", "binance-key", "binance-secret")
	insertTestCredential(t, creds, "test-user", "bybit", "bybit-key", "bybit-secret")

	inst := &StrategyInstance{
		UserID:        "test-user",
		Mode:          ModeLive,
		Strategy:      "xmaker",
		Symbol:        "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
			{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
		},
		InstanceID: "xmaker-BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)

	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if !findEnv("BINANCE_API_KEY=binance-key") {
		t.Error("expected BINANCE_API_KEY for xmaker maker session")
	}
	if !findEnv("BYBIT_API_KEY=bybit-key") {
		t.Error("expected BYBIT_API_KEY for xmaker hedge session")
	}
}

func TestEnvArgs_Passphrase_Injected(t *testing.T) {
	cm, creds := setupContainerManager(t)

	enc := creds.crypto
	keyEnc, _ := enc.Encrypt("key")
	secretEnc, _ := enc.Encrypt("secret")
	passEnc, _ := enc.Encrypt("mypass")
	creds.Upsert(ExchangeCredential{
		UserID: "test-user", Exchange: "okex",
		APIKeyEncrypted: keyEnc, APISecretEncrypted: secretEnc, PassphraseEncrypted: passEnc,
	})

	args := cm.instanceEnvArgs(testInst("test-user", ModeLive, "grid2", "okex", "BTCUSDT"))

	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if !findEnv("OKEX_API_KEY=key") {
		t.Error("expected OKEX_API_KEY")
	}
	if !findEnv("OKEX_API_SECRET=secret") {
		t.Error("expected OKEX_API_SECRET")
	}
	if !findEnv("OKEX_API_PASSPHRASE=mypass") {
		t.Error("expected OKEX_API_PASSPHRASE")
	}
}

func TestEnvArgs_MarketDataServiceURL(t *testing.T) {
	cm, _ := setupContainerManager(t)
	args := cm.instanceEnvArgs(testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT"))

	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}
	if !findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata:9090") {
		t.Error("expected MARKET_DATA_SERVICE_URL env var when MarketDataAddr is configured")
	}
}

func TestEnvArgs_NoMarketDataAddr(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{
		ManagerToken:   "test-token",
		DataDir:        dir,
		DataVolume:     "bbgo-data",
		DockerNetwork:  "bbgo-net",
		BBGOImage:      "bbgo-base:latest",
		BBGOPort:       8080,
		MarketDataAddr: "",
	}
	cm := NewContainerManager(cfg, creds, nil, nil)
	args := cm.instanceEnvArgs(testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT"))

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" && strings.HasPrefix(args[i+1], "MARKET_DATA_SERVICE_URL=") {
			t.Error("MARKET_DATA_SERVICE_URL should NOT be set when MarketDataAddr is empty")
		}
	}
}

func TestBuildInstanceYAML_PublicOnly_NoCredentials(t *testing.T) {
	inst := testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT")
	inst.Config = rawJSON(`{"symbol":"BTCUSDT"}`)
	yaml, err := buildInstanceYAML(inst, func(exchange string) bool { return false }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yaml)
	if !strings.Contains(s, "publicOnly: true") {
		t.Error("expected publicOnly: true when no credentials")
	}
}

func TestBuildInstanceYAML_PublicOnlyFalse_WithCredentials(t *testing.T) {
	inst := testInst("test-user", ModeLive, "grid2", "binance", "BTCUSDT")
	inst.Config = rawJSON(`{"symbol":"BTCUSDT"}`)
	yaml, err := buildInstanceYAML(inst, func(exchange string) bool { return true }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yaml)
	if strings.Contains(s, "publicOnly: true") {
		t.Error("should NOT have publicOnly: true when credentials exist")
	}
}

func TestBuildInstanceYAML_PaperMode_AlwaysPublicOnly(t *testing.T) {
	inst := testInst("test-user", ModePaper, "grid2", "okex", "BTCUSDT")
	inst.Config = rawJSON(`{"symbol":"BTCUSDT"}`)
	yaml, err := buildInstanceYAML(inst, func(exchange string) bool { return false }, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yaml)
	if !strings.Contains(s, "publicOnly: true") {
		t.Error("paper mode should set publicOnly: true")
	}
}

func TestWriteYAMLToDisk(t *testing.T) {
	store, dir := newTestStore(t)
	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModePaper,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`),
		InstanceID: "grid2-BTCUSDT",
	}
	if err := store.CreateInstance(inst, func(string) bool { return false }); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	yamlPath := filepath.Join(dir, "test-user", "paper", "grid2-btcusdt", "bbgo.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "grid2:") {
		t.Error("expected grid2 strategy in written YAML")
	}
	if !strings.Contains(s, "BTCUSDT") {
		t.Error("expected BTCUSDT symbol in written YAML")
	}
	if !strings.Contains(s, "PAPER_TRADE:") {
		t.Error("expected PAPER_TRADE in written YAML for paper mode")
	}
}
