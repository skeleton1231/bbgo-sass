package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// --- hubForMode unit tests ---

func TestHubForMode_LiveMode(t *testing.T) {
	liveHub := &MarketDataHub{}
	api := &API{hub: liveHub, testnetHub: nil}

	got := api.hubForMode(ModeLive)
	if got != liveHub {
		t.Error("expected live hub for live mode")
	}
}

func TestHubForMode_EmptyMode(t *testing.T) {
	liveHub := &MarketDataHub{}
	api := &API{hub: liveHub, testnetHub: nil}

	got := api.hubForMode("")
	if got != liveHub {
		t.Error("expected live hub for empty mode")
	}
}

func TestHubForMode_PaperMode_AlwaysUsesMainnet(t *testing.T) {
	liveHub := &MarketDataHub{}
	testnetHub := &MarketDataHub{}
	api := &API{hub: liveHub, testnetHub: testnetHub}

	got := api.hubForMode(ModePaper)
	if got != liveHub {
		t.Error("paper mode should always use mainnet hub")
	}
}

func TestHubForMode_PaperMode_NoTestnetHub(t *testing.T) {
	liveHub := &MarketDataHub{}
	api := &API{hub: liveHub, testnetHub: nil}

	got := api.hubForMode(ModePaper)
	if got != liveHub {
		t.Error("expected live hub fallback for paper mode when testnetHub is nil")
	}
}

func TestHubForMode_BothHubs_Nil(t *testing.T) {
	api := &API{hub: nil, testnetHub: nil}

	got := api.hubForMode(ModePaper)
	if got != nil {
		t.Error("expected nil when both hubs are nil")
	}
}

// --- Container env args: market data routing ---

func TestEnvArgs_PaperMode_MainnetMarketDataAddr(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{
		ManagerToken:          "test-token",
		DataDir:               dir,
		DataVolume:            "bbgo-data",
		DockerNetwork:         "bbgo-net",
		BBGOImage:             "bbgo-base:latest",
		BBGOPort:              8080,
		MarketDataAddr:        "bbgo-marketdata:9090",
		MarketDataTestnetAddr: "bbgo-marketdata-testnet:9090",
	}
	cm := NewContainerManager(cfg, creds, nil)

	args := cm.envArgs("test-user", ModePaper, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	findEnv := func(val string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == val {
				return true
			}
		}
		return false
	}
	if !findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata:9090") {
		t.Errorf("paper container should use mainnet marketdata addr, got %v", args)
	}
	if findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata-testnet:9090") {
		t.Error("paper container should NOT use testnet marketdata addr")
	}
}

func TestEnvArgs_LiveMode_LiveMarketDataAddr(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{
		ManagerToken:          "test-token",
		DataDir:               dir,
		DataVolume:            "bbgo-data",
		DockerNetwork:         "bbgo-net",
		BBGOImage:             "bbgo-base:latest",
		BBGOPort:              8080,
		MarketDataAddr:        "bbgo-marketdata:9090",
		MarketDataTestnetAddr: "bbgo-marketdata-testnet:9090",
	}
	cm := NewContainerManager(cfg, creds, nil)

	args := cm.envArgs("test-user", ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	findEnv := func(val string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == val {
				return true
			}
		}
		return false
	}
	if !findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata:9090") {
		t.Errorf("live container should use live marketdata addr, got %v", args)
	}
	if findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata-testnet:9090") {
		t.Error("live container should NOT use testnet marketdata addr")
	}
}

func TestEnvArgs_PaperMode_NoTestnetAddr_FallsBackToLive(t *testing.T) {
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
		MarketDataAddr: "bbgo-marketdata:9090",
	}
	cm := NewContainerManager(cfg, creds, nil)

	args := cm.envArgs("test-user", ModePaper, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	findEnv := func(val string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == val {
				return true
			}
		}
		return false
	}
	if !findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata:9090") {
		t.Errorf("paper container should fall back to live marketdata addr when testnet addr is not configured, got %v", args)
	}
}

// --- MarketData API mode routing tests ---

func TestMarketTicker_PaperMode_NoTestnetHub_FallsBackToLive(t *testing.T) {
	_, router := setupMarketDataAPIWithTestnet(nil, nil)

	req := httptest.NewRequest("GET", "/api/markets/binance/ticker?symbol=BTCUSDT&mode=paper", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when both hubs nil with paper mode, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_PaperMode_AlwaysUsesLiveHub(t *testing.T) {
	// Klines are public data — paper mode must use live hub (testnet has limited history).
	testnetHub := &MarketDataHub{}
	_, router := setupMarketDataAPIWithTestnet(nil, testnetHub)

	req := httptest.NewRequest("GET", "/api/markets/binance/klines?symbol=BTCUSDT&mode=paper", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for paper klines when live hub nil (ignores testnet hub), got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketTicker_LiveMode_IgnoresTestnetHub(t *testing.T) {
	testnetHub := &MarketDataHub{}
	_, router := setupMarketDataAPIWithTestnet(nil, testnetHub)

	req := httptest.NewRequest("GET", "/api/markets/binance/ticker?symbol=BTCUSDT&mode=live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for live mode when live hub is nil, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_LiveMode_IgnoresTestnetHub(t *testing.T) {
	testnetHub := &MarketDataHub{}
	_, router := setupMarketDataAPIWithTestnet(nil, testnetHub)

	req := httptest.NewRequest("GET", "/api/markets/binance/klines?symbol=BTCUSDT&mode=live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for live mode when live hub is nil, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketTicker_NoModeParam_DefaultsToLive(t *testing.T) {
	_, router := setupMarketDataAPIWithTestnet(nil, nil)

	req := httptest.NewRequest("GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when no mode param and hub nil, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketSymbols_PaperMode_UsesMainnetRESTAddr(t *testing.T) {
	cfg := &Config{
		SupabaseURL:               "http://localhost:1",
		SupabaseKey:               "test",
		ManagerToken:              "test-token",
		MarketDataAddr:            "bbgo-marketdata:9090",
		MarketDataRESTAddr:        "bbgo-marketdata:8080",
		MarketDataTestnetAddr:     "bbgo-marketdata-testnet:9090",
		MarketDataTestnetRESTAddr: "bbgo-marketdata-testnet:8080",
	}
	store := NewStrategyStore("", nil)
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

	var capturedBaseURL string
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		capturedBaseURL = baseURL
		return &BBGoClient{baseURL: baseURL}
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/markets/binance/symbols?mode=paper", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedBaseURL != "http://bbgo-marketdata:8080" {
		t.Errorf("paper mode should use mainnet REST addr, got %s", capturedBaseURL)
	}
}

func TestMarketSymbols_LiveMode_UsesLiveRESTAddr(t *testing.T) {
	cfg := &Config{
		SupabaseURL:               "http://localhost:1",
		SupabaseKey:               "test",
		ManagerToken:              "test-token",
		MarketDataAddr:            "bbgo-marketdata:9090",
		MarketDataRESTAddr:        "bbgo-marketdata:8080",
		MarketDataTestnetAddr:     "bbgo-marketdata-testnet:9090",
		MarketDataTestnetRESTAddr: "bbgo-marketdata-testnet:8080",
	}
	store := NewStrategyStore("", nil)
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

	var capturedBaseURL string
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		capturedBaseURL = baseURL
		return &BBGoClient{baseURL: baseURL}
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/markets/binance/symbols?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedBaseURL != "http://bbgo-marketdata:8080" {
		t.Errorf("live mode should use live REST addr, got %s", capturedBaseURL)
	}
}

func TestMarketSymbols_NoMode_DefaultsToLiveRESTAddr(t *testing.T) {
	cfg := &Config{
		SupabaseURL:               "http://localhost:1",
		SupabaseKey:               "test",
		ManagerToken:              "test-token",
		MarketDataAddr:            "bbgo-marketdata:9090",
		MarketDataRESTAddr:        "bbgo-marketdata:8080",
		MarketDataTestnetAddr:     "bbgo-marketdata-testnet:9090",
		MarketDataTestnetRESTAddr: "bbgo-marketdata-testnet:8080",
	}
	store := NewStrategyStore("", nil)
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

	var capturedBaseURL string
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		capturedBaseURL = baseURL
		return &BBGoClient{baseURL: baseURL}
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/markets/binance/symbols", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedBaseURL != "http://bbgo-marketdata:8080" {
		t.Errorf("no mode should default to live REST addr, got %s", capturedBaseURL)
	}
}
