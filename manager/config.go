package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port          int
	DataDir       string
	DataVolume    string
	SupabaseURL   string
	SupabaseKey   string
	SupabaseDBURL string
	EncryptionKey string
	DockerNetwork string
	BBGOImage     string
	BBGOPort      int
	BBGOGRPCPort    int
	ManagerToken    string
	MarketDataAddr      string
	BacktestSymbols     []string
	BacktestExchanges   []string
	BacktestStartTime   string
	BacktestEndTime     string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:          getEnvInt("MANAGER_PORT", 8090),
		DataDir:       getEnv("DATA_DIR", "./data"),
		DataVolume:    getEnv("DATA_VOLUME", "bbgo-data"),
		SupabaseURL:   getEnv("SUPABASE_URL", ""),
		SupabaseKey:   getEnv("SUPABASE_SERVICE_KEY", ""),
		SupabaseDBURL: getEnv("SUPABASE_DB_URL", ""),
		EncryptionKey: getEnv("ENCRYPTION_KEY", ""),
		DockerNetwork: getEnv("DOCKER_NETWORK", "bbgo-net"),
		BBGOImage:     getEnv("BBGO_IMAGE", "bbgo-base:latest"),
		BBGOPort:      getEnvInt("BBGO_PORT", 8080),
		BBGOGRPCPort:  getEnvInt("BBGO_GRPC_PORT", 9090),
		ManagerToken:   getEnv("MANAGER_TOKEN", ""),
		MarketDataAddr:    getEnv("MARKETDATA_ADDR", "bbgo-marketdata:9090"),
		BacktestSymbols:   getEnvSlice("BACKTEST_SYMBOLS", []string{"BTCUSDT", "ETHUSDT"}),
		BacktestExchanges: getEnvSlice("BACKTEST_EXCHANGES", []string{"binance"}),
		BacktestStartTime: getEnv("BACKTEST_START_TIME", "2023-12-01"),
		BacktestEndTime:   getEnv("BACKTEST_END_TIME", "2025-12-31"),
	}

	if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_SERVICE_KEY are required")
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
