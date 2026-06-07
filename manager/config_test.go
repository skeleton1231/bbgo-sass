package main

import (
	"os"
	"testing"
)

// configEnvKeys are all environment variables read by LoadConfig.
var configEnvKeys = []string{
	"SUPABASE_URL", "SUPABASE_SERVICE_KEY",
	"MANAGER_TOKEN", "ENCRYPTION_KEY", "MANAGER_PORT",
	"DATA_DIR", "DATA_VOLUME", "DOCKER_NETWORK",
	"BBGO_IMAGE", "BBGO_PORT", "BBGO_GRPC_PORT",
	"MARKETDATA_ADDR", "MARKETDATA_REST_ADDR",
	"WS_ALLOWED_ORIGINS", "MARKET_SUBSCRIPTIONS",
	"BACKTEST_SYMBOLS", "BACKTEST_EXCHANGES",
	"BACKTEST_START_TIME", "BACKTEST_END_TIME", "BACKTEST_SHARED_DIR",
	"CONTAINER_MEMORY", "CONTAINER_MEMORY_SWAP", "CONTAINER_CPUS",
	"CONTAINER_PIDS_LIMIT", "CONTAINER_LOG_MAX_SIZE", "CONTAINER_LOG_MAX_FILE",
}

// setConfigEnv clears only config env vars then sets the given ones.
func setConfigEnv(t *testing.T, kv ...string) {
	t.Helper()
	for _, k := range configEnvKeys {
		os.Unsetenv(k)
	}
	for i := 0; i < len(kv); i += 2 {
		os.Setenv(kv[i], kv[i+1])
	}
}

func TestLoadConfig_RequiredVars(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://example.supabase.co",
		"SUPABASE_SERVICE_KEY", "test-key",
		"MANAGER_TOKEN", "my-token",
		"ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef",
	)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8090 {
		t.Errorf("expected default port 8090, got %d", cfg.Port)
	}
	if cfg.SupabaseURL != "https://example.supabase.co" {
		t.Errorf("expected supabase URL, got %s", cfg.SupabaseURL)
	}
	if cfg.ManagerToken != "my-token" {
		t.Errorf("expected manager token, got %s", cfg.ManagerToken)
	}
	if cfg.BBGOImage != "bbgo-base:latest" {
		t.Errorf("expected default bbgo image, got %s", cfg.BBGOImage)
	}
	if cfg.DockerNetwork != "bbgo-net" {
		t.Errorf("expected default docker network, got %s", cfg.DockerNetwork)
	}
}

func TestLoadConfig_CustomPort(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://x.supabase.co",
		"SUPABASE_SERVICE_KEY", "k",
		"MANAGER_TOKEN", "t",
		"ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef",
		"MANAGER_PORT", "9090",
	)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	setConfigEnv(t)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing required vars")
	}
}

func TestLoadConfig_MissingSupabaseKey(t *testing.T) {
	setConfigEnv(t, "SUPABASE_URL", "https://x.supabase.co")
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing SUPABASE_SERVICE_KEY")
	}
}

func TestLoadConfig_MissingManagerToken(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://x.supabase.co",
		"SUPABASE_SERVICE_KEY", "k",
	)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing MANAGER_TOKEN")
	}
}

func TestLoadConfig_MissingEncryptionKey(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://x.supabase.co",
		"SUPABASE_SERVICE_KEY", "k",
		"MANAGER_TOKEN", "t",
	)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing ENCRYPTION_KEY")
	}
}

func TestLoadConfig_MarketSubscriptions(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://x.supabase.co",
		"SUPABASE_SERVICE_KEY", "k",
		"MANAGER_TOKEN", "t",
		"ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef",
		"MARKET_SUBSCRIPTIONS", "binance:ETHUSDT:trade,binance:ETHUSDT:kline:5m",
	)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MarketSubscriptions) < 2 {
		t.Fatalf("expected 2 market subs, got %d", len(cfg.MarketSubscriptions))
	}
	if cfg.MarketSubscriptions[0].Symbol != "ETHUSDT" {
		t.Errorf("expected first sub symbol ETHUSDT, got %s", cfg.MarketSubscriptions[0].Symbol)
	}
	if cfg.MarketSubscriptions[0].Channel != "trade" {
		t.Errorf("expected first sub channel trade, got %s", cfg.MarketSubscriptions[0].Channel)
	}
	if cfg.MarketSubscriptions[1].Interval != "5m" {
		t.Errorf("expected second sub interval 5m, got %s", cfg.MarketSubscriptions[1].Interval)
	}
}

func TestLoadConfig_BacktestDefaults(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://x.supabase.co",
		"SUPABASE_SERVICE_KEY", "k",
		"MANAGER_TOKEN", "t",
		"ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef",
	)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.BacktestSymbols) != 2 || cfg.BacktestSymbols[0] != "BTCUSDT" {
		t.Errorf("expected default backtest symbols, got %v", cfg.BacktestSymbols)
	}
	if len(cfg.BacktestExchanges) != 1 || cfg.BacktestExchanges[0] != "binance" {
		t.Errorf("expected default backtest exchanges, got %v", cfg.BacktestExchanges)
	}
}

func TestSafeShortID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "short"},
		{"abcdefghijklmnop", "abcdefgh"},
		{"1234567890", "12345678"},
		{"", ""},
	}
	for _, tt := range tests {
		got := safeShortID(tt.input)
		if got != tt.want {
			t.Errorf("safeShortID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLoadConfig_ResourceDefaults(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://x.supabase.co",
		"SUPABASE_SERVICE_KEY", "k",
		"MANAGER_TOKEN", "t",
		"ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef",
	)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.InstanceResources.Memory != "256m" {
		t.Errorf("memory: got %q, want 256m", cfg.InstanceResources.Memory)
	}
	if cfg.InstanceResources.MemorySwap != "512m" {
		t.Errorf("swap: got %q, want 512m", cfg.InstanceResources.MemorySwap)
	}
	if cfg.InstanceResources.CPUs != "0.25" {
		t.Errorf("cpus: got %q, want 0.25", cfg.InstanceResources.CPUs)
	}
	if cfg.InstanceResources.PidsLimit != 64 {
		t.Errorf("pids: got %d, want 64", cfg.InstanceResources.PidsLimit)
	}
	if cfg.InstanceResources.LogMaxSize != "10m" {
		t.Errorf("log size: got %q, want 10m", cfg.InstanceResources.LogMaxSize)
	}
	if cfg.InstanceResources.LogMaxFile != 3 {
		t.Errorf("log files: got %d, want 3", cfg.InstanceResources.LogMaxFile)
	}
}

func TestLoadConfig_ResourceOverrides(t *testing.T) {
	setConfigEnv(t,
		"SUPABASE_URL", "https://x.supabase.co",
		"SUPABASE_SERVICE_KEY", "k",
		"MANAGER_TOKEN", "t",
		"ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef",
		"CONTAINER_MEMORY", "512m",
		"CONTAINER_CPUS", "1.0",
		"CONTAINER_PIDS_LIMIT", "256",
	)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.InstanceResources.Memory != "512m" {
		t.Errorf("memory: got %q, want 512m", cfg.InstanceResources.Memory)
	}
	if cfg.InstanceResources.CPUs != "1.0" {
		t.Errorf("cpus: got %q, want 1.0", cfg.InstanceResources.CPUs)
	}
	if cfg.InstanceResources.PidsLimit != 256 {
		t.Errorf("pids: got %d, want 256", cfg.InstanceResources.PidsLimit)
	}
}
