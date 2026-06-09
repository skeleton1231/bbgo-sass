package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossLayerLiveOnlyAlignment verifies the backend liveOnly strategies
// covers all strategy IDs that the frontend marks as liveOnly.
func TestCrossLayerLiveOnlyAlignment(t *testing.T) {
	for id, meta := range StrategyRegistry {
		if meta.LiveOnly {
			assert.True(t, testRegistry.IsLiveOnly(id), "StrategyRegistry liveOnly strategy %s should be liveOnly in testRegistry", id)
		}
	}

	for id := range testRegistry.liveOnly {
		meta, exists := StrategyRegistry[id]
		if exists {
			assert.True(t, meta.LiveOnly, "testRegistry liveOnly strategy %s should be liveOnly in StrategyRegistry", id)
		}
	}

	assert.False(t, testRegistry.IsLiveOnly("sentinel_anomaly"))
}

// TestCrossLayerExchangePrefixes verifies all 8 exchanges have env var prefixes.
func TestCrossLayerExchangePrefixes(t *testing.T) {
	require.Len(t, exchangePrefixes, 8)
	assert.Equal(t, "BINANCE", exchangePrefixes["binance"])
	assert.Equal(t, "OKEX", exchangePrefixes["okex"])
	assert.Equal(t, "KUCOIN", exchangePrefixes["kucoin"])
	assert.Equal(t, "BYBIT", exchangePrefixes["bybit"])
	assert.Equal(t, "BITGET", exchangePrefixes["bitget"])
	assert.Equal(t, "MAX", exchangePrefixes["max"])
	assert.Equal(t, "COINBASE", exchangePrefixes["coinbase"])
	assert.Equal(t, "BITFINEX", exchangePrefixes["bitfinex"])
	assert.Equal(t, "EXCHANGE", exchangeEnvPrefix("unknown_exchange"))
}

func storeCred(t *testing.T, creds *CredentialStore, enc *Encryptor, userID, exchange, key, secret, pass string) {
	t.Helper()
	keyEnc, err := enc.Encrypt(key)
	require.NoError(t, err)
	secEnc, err := enc.Encrypt(secret)
	require.NoError(t, err)
	var passEnc string
	if pass != "" {
		passEnc, err = enc.Encrypt(pass)
		require.NoError(t, err)
	}
	require.NoError(t, creds.Upsert(ExchangeCredential{
		UserID:              userID,
		Exchange:            exchange,
		APIKeyEncrypted:     keyEnc,
		APISecretEncrypted:  secEnc,
		PassphraseEncrypted: passEnc,
	}))
}

// TestCrossLayerDockerEnvArgsLive verifies live mode has credentials, no PAPER_TRADE.
func TestCrossLayerDockerEnvArgsLive(t *testing.T) {
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(t.TempDir(), enc)
	storeCred(t, creds, enc, testUUID, "binance", "mykey", "mysecret", "mypass")

	cm := &ContainerManager{
		cfg:   &Config{DataVolume: "bbgo-data"},
		creds: creds,
	}

	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		InstanceID: "grid2-BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	s := strings.Join(args, " ")

	assert.NotContains(t, s, "PAPER_TRADE")
	assert.Contains(t, s, "BINANCE_API_KEY=mykey")
	assert.Contains(t, s, "BINANCE_API_SECRET=mysecret")
	assert.Contains(t, s, "BINANCE_API_PASSPHRASE=mypass")
	assert.Contains(t, s, "DB_DRIVER=supabase")
}

// TestCrossLayerDockerEnvArgsPaper verifies paper mode has PAPER_TRADE=1 and NO credentials.
func TestCrossLayerDockerEnvArgsPaper(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{DataVolume: "bbgo-data"},
	}

	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModePaper, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		InstanceID: "grid2-BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	s := strings.Join(args, " ")

	assert.Contains(t, s, "PAPER_TRADE=1")
	assert.NotContains(t, s, "BINANCE_API_KEY")
	assert.NotContains(t, s, "BINANCE_TESTNET=1")
}

// TestCrossLayerDockerEnvArgsCrossExchange verifies multi-exchange credential injection.
func TestCrossLayerDockerEnvArgsCrossExchange(t *testing.T) {
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(t.TempDir(), enc)
	storeCred(t, creds, enc, testUUID, "binance", "bkey", "bsecret", "")
	storeCred(t, creds, enc, testUUID, "bybit", "ykey", "ysecret", "")

	cm := &ContainerManager{
		cfg:   &Config{DataVolume: "bbgo-data"},
		creds: creds,
	}

	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "xmaker",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Exchange: "binance", Name: "maker"},
			{Exchange: "bybit", Name: "taker"},
		},
		InstanceID: "xmaker-BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	s := strings.Join(args, " ")

	assert.Contains(t, s, "BINANCE_API_KEY=bkey")
	assert.Contains(t, s, "BYBIT_API_KEY=ykey")
	assert.NotContains(t, s, "PAPER_TRADE")
}

// TestCrossLayerMarketDataEnv verifies MARKET_DATA_SERVICE_URL injection.
func TestCrossLayerMarketDataEnv(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{
		DataVolume:     "bbgo-data",
		MarketDataAddr: "market-data:9090",
	}}

	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		InstanceID: "grid2-BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	assert.Contains(t, strings.Join(args, " "), "MARKET_DATA_SERVICE_URL=market-data:9090")
}

// TestCrossLayerBuildInstanceYAMLLive verifies YAML generation for a live instance.
func TestCrossLayerBuildInstanceYAMLLive(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModeLive, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config:     json.RawMessage(`{"symbol":"BTCUSDT","quantity":0.001,"gridNumber":10}`),
		InstanceID: "grid2-BTCUSDT-10",
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return true }, nil)
	require.NoError(t, err)

	yaml := string(yamlBytes)
	assert.Contains(t, yaml, "binance:")
	assert.Contains(t, yaml, "grid2:")
}

// TestCrossLayerBuildInstanceYAMLPaper verifies YAML generation for a paper instance.
func TestCrossLayerBuildInstanceYAMLPaper(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModePaper, Strategy: "grid2",
		Exchange: "binance", Symbol: "ETHUSDT",
		Config:     json.RawMessage(`{"symbol":"ETHUSDT","quantity":0.01,"gridNumber":5}`),
		InstanceID: "grid2-ETHUSDT-5",
	}
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return false }, nil)
	require.NoError(t, err)

	yaml := string(yamlBytes)
	assert.Contains(t, yaml, "binance:")
	assert.Contains(t, yaml, "grid2:")
}
