package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type tradingChainSetup struct {
	api       *API
	container *ContainerManager
	creds     *CredentialStore
	enc       *Encryptor
}

func newTradingChainSetup(t *testing.T) *tradingChainSetup {
	t.Helper()
	enc, err := NewEncryptor(testEncryptionKey)
	require.NoError(t, err)
	creds := NewCredentialStore(t.TempDir(), enc)
	store, _ := newTestStore(t)
	cm := &ContainerManager{cfg: &Config{
		DataVolume: "bbgo-data", BBGOPort: 8080, BBGOGRPCPort: 9090,
	}, creds: creds}
	proxy := NewBotProxy(cm)
	api := NewAPI(&Config{Port: 8090, ManagerToken: "test"}, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, nil)
	api.containerStart = func(userID, mode string) error { return nil }
	api.containerStop = func(string, _ string) {}
	api.containerRunning = func(string, _ string) bool { return false }
	return &tradingChainSetup{api, cm, creds, enc}
}

func (s *tradingChainSetup) storeCred(t *testing.T, exchange, key, secret string) {
	t.Helper()
	keyEnc, err := s.enc.Encrypt(key)
	require.NoError(t, err)
	secEnc, err := s.enc.Encrypt(secret)
	require.NoError(t, err)
	require.NoError(t, s.creds.Upsert(ExchangeCredential{
		UserID: testUUID, Exchange: exchange,
		APIKeyEncrypted: keyEnc, APISecretEncrypted: secEnc,
	}))
}
// Test: Live grid strategy produces correct YAML + env args for bbgo container
func TestTradingChain_LiveGrid_FullYAMLAndEnv(t *testing.T) {
	s := newTradingChainSetup(t)
	s.storeCred(t, "binance", "livekey", "livesecret")

	strategies := []StrategyEntry{{
		Strategy: "grid", Exchange: "binance", Mode: "live",
		Config: json.RawMessage(`{"symbol":"BTCUSDT","quantity":0.001,"gridNumber":10,"upperPrice":50000,"lowerPrice":40000}`),
	}}

	yamlBytes, err := buildUserYAML(testUUID, ModeLive, strategies, func(string) bool { return true })
	require.NoError(t, err)

	var cfg bbgoConfig
	require.NoError(t, yaml.Unmarshal(yamlBytes, &cfg))

	require.Contains(t, cfg.Sessions, "binance")
	assert.Equal(t, "BINANCE", cfg.Sessions["binance"].EnvVarPrefix)
	assert.False(t, cfg.Sessions["binance"].PublicOnly)
	assert.Equal(t, "BTCUSDT", cfg.Exchange["binance"].Symbol)
	require.Len(t, cfg.ExchangeStrategies, 1)
	assert.Equal(t, "binance", cfg.ExchangeStrategies[0]["on"])
	assert.Contains(t, cfg.ExchangeStrategies[0], "grid")
	assert.Empty(t, cfg.Environment.PaperTrade)

	args := s.container.envArgs(testUUID, ModeLive, strategies)
	argsStr := strings.Join(args, " ")
	assert.NotContains(t, argsStr, "PAPER_TRADE")
	assert.Contains(t, argsStr, "BINANCE_API_KEY=livekey")
	assert.Contains(t, argsStr, "BINANCE_API_SECRET=livesecret")
}

// Test: Paper mode sets PAPER_TRADE in both YAML and env vars
func TestTradingChain_PaperGrid_YAMLAndEnv(t *testing.T) {
	s := newTradingChainSetup(t)

	strategies := []StrategyEntry{{
		Strategy: "grid", Exchange: "binance", Mode: "paper",
		Config: json.RawMessage(`{"symbol":"ETHUSDT","quantity":0.01,"gridNumber":5}`),
	}}

	yamlBytes, err := buildUserYAML(testUUID, ModePaper, strategies, func(string) bool { return false })
	require.NoError(t, err)

	var cfg bbgoConfig
	require.NoError(t, yaml.Unmarshal(yamlBytes, &cfg))
	assert.True(t, cfg.Sessions["binance"].PublicOnly)
	assert.Equal(t, "1", cfg.Environment.PaperTrade)

	args := s.container.envArgs(testUUID, ModePaper, strategies)
	assert.Contains(t, strings.Join(args, " "), "PAPER_TRADE=1")
}

// Test: Cross-exchange strategy builds correct sessions and injects multi-exchange credentials
func TestTradingChain_CrossExchange_YAML(t *testing.T) {
	s := newTradingChainSetup(t)
	s.storeCred(t, "binance", "bkey", "bsec")
	s.storeCred(t, "bybit", "ykey", "ysec")

	strategies := []StrategyEntry{{
		Strategy: "xmaker", Mode: "live", CrossExchange: true,
		Config: json.RawMessage(`{"symbol":"BTCUSDT","quantity":0.001}`),
		Sessions: []SessionRoleConfig{
			{Name: "maker", Exchange: "binance"},
			{Name: "taker", Exchange: "bybit"},
		},
	}}

	yamlBytes, err := buildUserYAML(testUUID, ModeLive, strategies, func(string) bool { return true })
	require.NoError(t, err)

	var cfg bbgoConfig
	require.NoError(t, yaml.Unmarshal(yamlBytes, &cfg))

	require.Contains(t, cfg.Sessions, "maker")
	require.Contains(t, cfg.Sessions, "taker")
	assert.Equal(t, "binance", cfg.Sessions["maker"].Exchange)
	assert.Equal(t, "bybit", cfg.Sessions["taker"].Exchange)
	assert.Empty(t, cfg.ExchangeStrategies)
	require.Len(t, cfg.CrossExchangeStrategies, 1)
	assert.Contains(t, cfg.CrossExchangeStrategies[0], "xmaker")

	args := s.container.envArgs(testUUID, ModeLive, strategies)
	argsStr := strings.Join(args, " ")
	assert.Contains(t, argsStr, "BINANCE_API_KEY=bkey")
	assert.Contains(t, argsStr, "BYBIT_API_KEY=ykey")
}

// Test: All legacy aliases produce valid bbgo strategy IDs in YAML
func TestTradingChain_LegacyAliases(t *testing.T) {
	for _, tc := range []struct{ frontend, bbgoID string }{
		{"sentinel_anomaly", "sentinel"},
		{"autobuy_scheduled", "autobuy"},
		{"rebalance_portfolio", "rebalance"},
		{"ewoDgtrd", "ewo_dgtrd"},
	} {
		t.Run(tc.frontend, func(t *testing.T) {
			strategies := []StrategyEntry{{
				Strategy: tc.frontend, Exchange: "binance", Mode: "live",
				Config: json.RawMessage(`{"symbol":"BTCUSDT"}`),
			}}
			yamlBytes, err := buildUserYAML(testUUID, ModeLive, strategies, func(string) bool { return true })
			require.NoError(t, err)
			assert.Contains(t, string(yamlBytes), tc.bbgoID+":")
		})
	}
}

// Test: DCA field rename interval->investmentInterval
func TestTradingChain_DCA_FieldRename(t *testing.T) {
	strategies := []StrategyEntry{{
		Strategy: "dca", Exchange: "binance", Mode: "live",
		Config: json.RawMessage(`{"symbol":"BTCUSDT","interval":"1h","quantity":0.001}`),
	}}
	yamlBytes, err := buildUserYAML(testUUID, ModeLive, strategies, func(string) bool { return true })
	require.NoError(t, err)
	yamlStr := string(yamlBytes)
	assert.Contains(t, yamlStr, "investmentInterval:")
}

// Test: Multiple strategies on different exchanges produce correct multi-session YAML
func TestTradingChain_MultipleExchanges(t *testing.T) {
	strategies := []StrategyEntry{
		{Strategy: "grid", Exchange: "binance", Mode: "live",
			Config: json.RawMessage(`{"symbol":"BTCUSDT","quantity":0.001,"gridNumber":10}`)},
		{Strategy: "twap", Exchange: "okex", Mode: "live",
			Config: json.RawMessage(`{"symbol":"ETHUSDT","quantity":0.1}`)},
	}

	yamlBytes, err := buildUserYAML(testUUID, ModeLive, strategies, func(string) bool { return true })
	require.NoError(t, err)

	var cfg bbgoConfig
	require.NoError(t, yaml.Unmarshal(yamlBytes, &cfg))

	require.Contains(t, cfg.Sessions, "binance")
	require.Contains(t, cfg.Sessions, "okex")
	assert.Equal(t, "BTCUSDT", cfg.Exchange["binance"].Symbol)
	assert.Equal(t, "ETHUSDT", cfg.Exchange["okex"].Symbol)
	require.Len(t, cfg.ExchangeStrategies, 2)
	assert.Empty(t, cfg.Environment.PaperTrade)
}

// Test: Backtest YAML has correct structure
func TestTradingChain_BacktestYAML(t *testing.T) {
	yamlBytes, err := buildBacktestYAML("grid",
		json.RawMessage(`{"symbol":"ETHUSDT","quantity":0.01,"gridNumber":5}`),
		"2024-01-01", "2024-06-01", "binance", "")
	require.NoError(t, err)
	yamlStr := string(yamlBytes)
	assert.Contains(t, yamlStr, "backtest:")
	assert.Contains(t, yamlStr, "ETHUSDT")
	assert.Contains(t, yamlStr, "grid:")
}

// Test: Container DNS address format
func TestTradingChain_ContainerAddresses(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{BBGOPort: 8080, BBGOGRPCPort: 9090}}
	assert.Equal(t, "http://bbgo-user-123:8080", cm.APIURL("user-123", ModeLive))
	assert.Equal(t, "bbgo-user-456:9090", cm.ContainerGRPCAddr("user-456", ModeLive))
}
