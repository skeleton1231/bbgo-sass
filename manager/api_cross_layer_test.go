package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossLayerLiveOnlyAlignment verifies the backend liveOnlyStrategies map
// covers all strategy IDs that the frontend marks as liveOnly.
func TestCrossLayerLiveOnlyAlignment(t *testing.T) {
	require.Len(t, liveOnlyStrategies, 23, "liveOnlyStrategies count changed — update frontend STRATEGY_SCHEMAS too")

	// Core maker strategies
	assert.True(t, liveOnlyStrategies["bollmaker"])
	assert.True(t, liveOnlyStrategies["linregmaker"])
	assert.True(t, liveOnlyStrategies["rsmaker"])
	assert.True(t, liveOnlyStrategies["scmaker"])
	assert.True(t, liveOnlyStrategies["audacitymaker"])
	assert.True(t, liveOnlyStrategies["liquiditymaker"])

	// Trend strategies
	assert.True(t, liveOnlyStrategies["supertrend"])
	assert.True(t, liveOnlyStrategies["drift"])
	assert.True(t, liveOnlyStrategies["elliottwave"])
	assert.True(t, liveOnlyStrategies["factorzoo"])

	// DCA strategies
	assert.True(t, liveOnlyStrategies["dca2"])
	assert.True(t, liveOnlyStrategies["dca3"])
	assert.True(t, liveOnlyStrategies["autobuy"])

	// Other strategies
	assert.True(t, liveOnlyStrategies["wall"])
	assert.True(t, liveOnlyStrategies["sentinel"])
	assert.True(t, liveOnlyStrategies["rebalance"])
	assert.True(t, liveOnlyStrategies["support"])

	// Volatility
	assert.True(t, liveOnlyStrategies["xvs"])

	// Utility strategies
	assert.True(t, liveOnlyStrategies["autoborrow"])
	assert.True(t, liveOnlyStrategies["convert"])
	assert.True(t, liveOnlyStrategies["deposit2transfer"])

	// Cross-exchange liveOnly
	assert.True(t, liveOnlyStrategies["xpremium"])
	assert.True(t, liveOnlyStrategies["xnav"])

	// Legacy aliases should NOT be in liveOnlyStrategies — they get normalized
	assert.False(t, liveOnlyStrategies["sentinel_anomaly"])
	assert.False(t, liveOnlyStrategies["autobuy_scheduled"])
	assert.False(t, liveOnlyStrategies["rebalance_portfolio"])

	// But they should exist in legacyStrategyAliases and their targets must be liveOnly
	require.Contains(t, legacyStrategyAliases, "sentinel_anomaly")
	require.Contains(t, legacyStrategyAliases, "autobuy_scheduled")
	require.Contains(t, legacyStrategyAliases, "rebalance_portfolio")
	assert.True(t, liveOnlyStrategies[legacyStrategyAliases["sentinel_anomaly"]])
	assert.True(t, liveOnlyStrategies[legacyStrategyAliases["autobuy_scheduled"]])
	assert.True(t, liveOnlyStrategies[legacyStrategyAliases["rebalance_portfolio"]])
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
		ID:                  "cred-" + exchange,
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

	args := cm.envArgs(&UserContainer{
		UserID: testUUID,
		Strategies: []StrategyEntry{
			{ID: "grid", Exchange: "binance", Mode: "live"},
		},
	})
	s := strings.Join(args, " ")

	assert.NotContains(t, s, "PAPER_TRADE")
	assert.Contains(t, s, "BINANCE_API_KEY=mykey")
	assert.Contains(t, s, "BINANCE_API_SECRET=mysecret")
	assert.Contains(t, s, "BINANCE_PASSPHRASE=mypass")
	assert.Contains(t, s, "DB_DRIVER=sqlite3")
}

// TestCrossLayerDockerEnvArgsPaper verifies paper mode has PAPER_TRADE=1 + credentials.
func TestCrossLayerDockerEnvArgsPaper(t *testing.T) {
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(t.TempDir(), enc)
	storeCred(t, creds, enc, testUUID, "binance", "mykey", "mysecret", "")

	cm := &ContainerManager{
		cfg:   &Config{DataVolume: "bbgo-data"},
		creds: creds,
	}

	args := cm.envArgs(&UserContainer{
		UserID: testUUID,
		Strategies: []StrategyEntry{
			{ID: "grid", Exchange: "binance", Mode: "paper"},
		},
	})
	s := strings.Join(args, " ")

	assert.Contains(t, s, "PAPER_TRADE=1")
	assert.Contains(t, s, "BINANCE_API_KEY=mykey")
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

	args := cm.envArgs(&UserContainer{
		UserID: testUUID,
		Strategies: []StrategyEntry{
			{
				ID: "xmaker", CrossExchange: true, Mode: "live",
				Sessions: []SessionRoleConfig{
					{Exchange: "binance", Name: "maker"},
					{Exchange: "bybit", Name: "taker"},
				},
			},
		},
	})
	s := strings.Join(args, " ")

	assert.Contains(t, s, "BINANCE_API_KEY=bkey")
	assert.Contains(t, s, "BYBIT_API_KEY=ykey")
	assert.NotContains(t, s, "PAPER_TRADE")
}

// TestCrossLayerPaperModeIsGlobal verifies PAPER_TRADE appears exactly once.
func TestCrossLayerPaperModeIsGlobal(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{DataVolume: "bbgo-data"}}

	args := cm.envArgs(&UserContainer{
		UserID: testUUID,
		Strategies: []StrategyEntry{
			{ID: "grid", Exchange: "binance", Mode: "paper"},
			{ID: "twap", Exchange: "okex", Mode: "paper"},
		},
	})
	count := strings.Count(strings.Join(args, " "), "PAPER_TRADE=1")
	assert.Equal(t, 1, count)
}

// TestCrossLayerMarketDataEnv verifies MARKET_DATA_SERVICE_URL injection.
func TestCrossLayerMarketDataEnv(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{
		DataVolume:     "bbgo-data",
		MarketDataAddr: "market-data:9090",
	}}

	args := cm.envArgs(&UserContainer{
		UserID: testUUID,
		Strategies: []StrategyEntry{
			{ID: "grid", Exchange: "binance", Mode: "live"},
		},
	})
	assert.Contains(t, strings.Join(args, " "), "MARKET_DATA_SERVICE_URL=market-data:9090")
}

// TestCrossLayerBuildYAMLLive verifies YAML generation for a live strategy.
func TestCrossLayerBuildYAMLLive(t *testing.T) {
	yamlBytes, err := buildUserYAML(&UserContainer{
		UserID: testUUID,
		Strategies: []StrategyEntry{
			{
				Strategy: "grid", Exchange: "binance", Mode: "live",
				Config: json.RawMessage(`{"symbol":"BTCUSDT","quantity":0.001,"gridNumber":10}`),
			},
		},
	}, func(string) bool { return true })
	require.NoError(t, err)

	yaml := string(yamlBytes)
	assert.Contains(t, yaml, "binance:")
	assert.Contains(t, yaml, "grid:")
}

// TestCrossLayerBuildYAMLPaper verifies YAML generation for a paper strategy.
func TestCrossLayerBuildYAMLPaper(t *testing.T) {
	yamlBytes, err := buildUserYAML(&UserContainer{
		UserID: testUUID,
		Strategies: []StrategyEntry{
			{
				Strategy: "grid", Exchange: "binance", Mode: "paper",
				Config: json.RawMessage(`{"symbol":"ETHUSDT","quantity":0.01,"gridNumber":5}`),
			},
		},
	}, func(string) bool { return false })
	require.NoError(t, err)

	yaml := string(yamlBytes)
	assert.Contains(t, yaml, "binance:")
	assert.Contains(t, yaml, "grid:")
}
