package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MarketSub struct {
	Exchange string
	Session  string // session name for routing to spot or futures (e.g., "binance_futures")
	Channel  string
	Symbol   string
	Interval string
	Depth    string
}

type ContainerResources struct {
	Memory     string // --memory, e.g. "256m"
	MemorySwap string // --memory-swap, e.g. "512m"
	CPUs       string // --cpus, e.g. "0.5"
	PidsLimit  int    // --pids-limit
	LogMaxSize string // --log-opt max-size
	LogMaxFile int    // --log-opt max-file
}

type Config struct {
	Port                      int
	DataDir                   string
	DataVolume                string
	SupabaseURL               string
	SupabaseKey               string
	SupabaseDBURL             string
	EncryptionKey             string
	DockerNetwork             string
	BBGOImage                 string
	BBGOPort                  int
	BBGOGRPCPort              int
	BBGOUID                   int
	BBGOGID                   int
	ManagerToken              string
	WSAllowedOrigins          []string
	MarketDataAddr            string
	MarketDataRESTAddr        string
	MarketDataTestnetAddr     string
	MarketDataTestnetRESTAddr string
	MarketSubscriptions       []MarketSub
	BacktestSymbols           []string
	BacktestExchanges         []string
	BacktestStartTime         string
	BacktestEndTime           string
	BacktestSharedDir         string
	InstanceResources         ContainerResources
	BacktestResources         ContainerResources
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:                      getEnvInt("MANAGER_PORT", 8090),
		DataDir:                   getEnv("DATA_DIR", "./data"),
		DataVolume:                getEnv("DATA_VOLUME", "bbgo-data"),
		SupabaseURL:               getEnv("SUPABASE_URL", ""),
		SupabaseKey:               getEnv("SUPABASE_SERVICE_KEY", ""),
		SupabaseDBURL:             getEnv("SUPABASE_DB_URL", ""),
		EncryptionKey:             getEnv("ENCRYPTION_KEY", ""),
		DockerNetwork:             getEnv("DOCKER_NETWORK", "bbgo-net"),
		BBGOImage:                 getEnv("BBGO_IMAGE", "bbgo-base:latest"),
		BBGOPort:                  getEnvInt("BBGO_PORT", 8080),
		BBGOGRPCPort:              getEnvInt("BBGO_GRPC_PORT", 9090),
		BBGOUID:                   getEnvInt("BBGO_UID", 10001),
		BBGOGID:                   getEnvInt("BBGO_GID", 10001),
		ManagerToken:              getEnv("MANAGER_TOKEN", ""),
		WSAllowedOrigins:          getEnvSlice("WS_ALLOWED_ORIGINS", nil),
		MarketDataAddr:            getEnv("MARKETDATA_ADDR", "bbgo-marketdata:9090"),
		MarketDataRESTAddr:        getEnv("MARKETDATA_REST_ADDR", "bbgo-marketdata:8080"),
		MarketDataTestnetAddr:     getEnv("MARKETDATA_TESTNET_ADDR", ""),
		MarketDataTestnetRESTAddr: getEnv("MARKETDATA_TESTNET_REST_ADDR", ""),
		MarketSubscriptions:       parseMarketSubs(getEnvSlice("MARKET_SUBSCRIPTIONS", nil)),
		BacktestSymbols:           getEnvSlice("BACKTEST_SYMBOLS", []string{"BTCUSDT", "ETHUSDT"}),
		BacktestExchanges:         getEnvSlice("BACKTEST_EXCHANGES", []string{"binance"}),
		BacktestStartTime:         getEnv("BACKTEST_START_TIME", "2023-12-01"),
		BacktestEndTime:           getEnv("BACKTEST_END_TIME", "2025-12-31"),
		BacktestSharedDir:         getEnv("BACKTEST_SHARED_DIR", ""),
		InstanceResources: ContainerResources{
			Memory:     getEnv("CONTAINER_MEMORY", "256m"),
			MemorySwap: getEnv("CONTAINER_MEMORY_SWAP", "512m"),
			CPUs:       getEnv("CONTAINER_CPUS", "0.25"),
			PidsLimit:  getEnvInt("CONTAINER_PIDS_LIMIT", 64),
			LogMaxSize: getEnv("CONTAINER_LOG_MAX_SIZE", "10m"),
			LogMaxFile: getEnvInt("CONTAINER_LOG_MAX_FILE", 3),
		},
		BacktestResources: ContainerResources{
			Memory:     getEnv("BACKTEST_MEMORY", "256m"),
			MemorySwap: getEnv("BACKTEST_MEMORY_SWAP", "512m"),
			CPUs:       getEnv("BACKTEST_CPUS", "0.5"),
			PidsLimit:  getEnvInt("BACKTEST_PIDS_LIMIT", 64),
			LogMaxSize: getEnv("BACKTEST_LOG_MAX_SIZE", "10m"),
			LogMaxFile: getEnvInt("BACKTEST_LOG_MAX_FILE", 3),
		},
	}

	if cfg.SupabaseDBURL == "" && (cfg.SupabaseURL == "" || cfg.SupabaseKey == "") {
		return nil, fmt.Errorf("SUPABASE_DB_URL (or SUPABASE_URL + SUPABASE_SERVICE_KEY) is required")
	}
	if cfg.ManagerToken == "" {
		return nil, fmt.Errorf("MANAGER_TOKEN is required (shared secret for API authentication)")
	}
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required (base64-encoded 32-byte key for credential encryption)")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseMarketSubs(entries []string) []MarketSub {
	if len(entries) == 0 {
		return []MarketSub{
			{Exchange: "binance", Channel: "trade", Symbol: "BTCUSDT"},
			{Exchange: "binance", Channel: "kline", Symbol: "BTCUSDT", Interval: "1m"},
			{Exchange: "binance", Channel: "book", Symbol: "BTCUSDT", Depth: "5"},
		}
	}
	var subs []MarketSub
	for _, e := range entries {
		parts := strings.Split(e, ":")
		if len(parts) < 3 {
			continue
		}
		sub := MarketSub{Exchange: parts[0], Symbol: parts[1], Channel: parts[2]}
		if len(parts) > 3 {
			if sub.Channel == "kline" {
				sub.Interval = parts[3]
			} else {
				sub.Depth = parts[3]
			}
		}
		if len(parts) > 4 {
			sub.Session = parts[4]
		}
		subs = append(subs, sub)
	}
	return subs
}
