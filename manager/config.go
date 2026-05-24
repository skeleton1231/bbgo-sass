package main

import (
	"fmt"
	"os"
	"strconv"
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
	MarketDataAddr  string
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
		MarketDataAddr: getEnv("MARKETDATA_ADDR", "bbgo-marketdata:9090"),
	}

	if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_SERVICE_KEY are required")
	}
	if cfg.ManagerToken == "" {
		return nil, fmt.Errorf("MANAGER_TOKEN is required (shared secret for API authentication)")
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
