package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestCreateStrategy_StoppedContainer_NoAutoStart verifies that creating a
// strategy for a stopped container stores it but does NOT trigger a start.
func TestCreateStrategy_StoppedContainer_NoAutoStart(t *testing.T) {
	store, _ := newTestStore(t)
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

	var startCalled bool
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerStart = func(userID, mode string) error {
		startCalled = true
		return nil
	}
	api.containerRunning = func(_, _ string) bool { return false }

	r := testRouter(api)

	body := map[string]any{
		"name":     "Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]any{"symbol": "BTCUSDT"},
		"mode":     "live",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	strats, _ := store.ListStrategies(userID, ModeLive)
	if len(strats) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(strats))
	}
	if startCalled {
		t.Error("creating strategy on stopped container should NOT trigger auto-start")
	}
}

// TestCreateStrategy_ErrorContainer_NoAutoStart verifies that creating a
// strategy when container is in error state does NOT trigger a start.
func TestCreateStrategy_ErrorContainer_NoAutoStart(t *testing.T) {
	store, _ := newTestStore(t)
	_ = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

	var startCalled bool
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerStart = func(userID, mode string) error {
		startCalled = true
		return nil
	}
	api.containerRunning = func(_, _ string) bool { return false }

	r := testRouter(api)

	body := map[string]any{
		"name":     "Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]any{"symbol": "BTCUSDT"},
		"mode":     "live",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if startCalled {
		t.Error("creating strategy on error container should NOT trigger auto-start")
	}
}

// TestCreateStrategy_RunningContainer_TriggersRestart verifies that creating
// a strategy while the container is running triggers an async restart.
func TestCreateStrategy_RunningContainer_TriggersRestart(t *testing.T) {
	store, _ := newTestStore(t)
	_ = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()

	var startCalled bool
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerStart = func(userID, mode string) error {
		startCalled = true
		return nil
	}
	api.containerRunning = func(_, _ string) bool { return true }
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := testRouter(api)

	body := map[string]any{
		"name":     "Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]any{"symbol": "BTCUSDT"},
		"mode":     "live",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for strategy-triggered container restart")
		case <-ticker.C:
			if startCalled {
				return
			}
		}
	}
}

// TestContainerLogs_UserIDMismatch_Rejected verifies logs endpoint checks auth.
func TestContainerLogs_UserIDMismatch_Rejected(t *testing.T) {
	victimID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	attackerID := "11111111-2222-3333-4444-555555555555"

	store, _ := newTestStore(t)
	writeTestStrategies(t, store, victimID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", attackerID)
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+victimID+"/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for logs user ID mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCredentialCreate_StoppedContainer_NoRestart verifies that adding
// credentials when the container is stopped does NOT trigger a restart.
func TestCredentialCreate_StoppedContainer_NoRestart(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	store, _ := newTestStore(t)
	_ = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)

	var restartCalled bool
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, nil, nil)
	api.containerStart = func(userID, mode string) error {
		restartCalled = true
		return nil
	}
	api.containerRunning = func(_, _ string) bool { return false }

	r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	body := map[string]any{
		"exchange":   "binance",
		"api_key":    "new-key",
		"api_secret": "new-secret",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/credentials", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	time.Sleep(200 * time.Millisecond)
	if restartCalled {
		t.Error("creating credentials for stopped container should NOT trigger restart")
	}
}

// TestProxyToBot_StripsAuthHeaders verifies the proxy removes sensitive
// headers before forwarding to the bbgo container.
func TestProxyToBot_StripsAuthHeaders(t *testing.T) {
	var receivedHeaders http.Header
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer bbgoSrv.Close()

	store, _ := newTestStore(t)
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	writeTestStrategies(t, store, userID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	proxy.resolveAddr = func(_, _ string) string { return bbgoSrv.URL }
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "secret-manager-token")
			r.Header.Set("X-User-Id", userID)
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/bbgo/"+userID+"/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if receivedHeaders.Get("X-Manager-Token") != "" {
		t.Error("proxy should strip X-Manager-Token before forwarding to bbgo container")
	}
	if receivedHeaders.Get("X-User-Id") != "" {
		t.Error("proxy should strip X-User-Id before forwarding to bbgo container")
	}
}

// TestBuildUserYAML_AllExchanges verifies all supported exchanges produce
// correct env var prefixes in the generated YAML.
func TestBuildUserYAML_AllExchanges(t *testing.T) {
	exchanges := []struct {
		name   string
		prefix string
	}{
		{"binance", "BINANCE"},
		{"okex", "OKEX"},
		{"bybit", "BYBIT"},
		{"bitget", "BITGET"},
		{"kucoin", "KUCOIN"},
	}

	for _, ex := range exchanges {
		t.Run(ex.name, func(t *testing.T) {
			strategies := []StrategyEntry{
				{
					Strategy: "grid2",
					Exchange: ex.name,
					Mode:     "paper",
					Config:   rawJSON(`{"symbol":"BTCUSDT"}`),
				},
			}
			yaml, err := buildUserYAML("test-user", ModeLive, strategies, func(_ string) bool { return false })
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s := string(yaml)
			if !strings.Contains(s, ex.name+":") {
				t.Errorf("expected session for %s", ex.name)
			}
			if !strings.Contains(s, "envVarPrefix: "+ex.prefix) {
				t.Errorf("expected envVarPrefix %s, got:\n%s", ex.prefix, s)
			}
		})
	}
}

// TestEnvArgs_AllExchanges verifies credential injection uses correct
// env var prefix for each supported exchange.
func TestEnvArgs_AllExchanges(t *testing.T) {
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
		BBGOGRPCPort:   9090,
		MarketDataAddr: "",
	}
	cm := NewContainerManager(cfg, creds, nil)

	exchanges := []struct {
		name   string
		prefix string
	}{
		{"binance", "BINANCE"},
		{"okex", "OKEX"},
		{"bybit", "BYBIT"},
		{"bitget", "BITGET"},
		{"kucoin", "KUCOIN"},
	}

	for _, ex := range exchanges {
		t.Run(ex.name, func(t *testing.T) {
			insertTestCredential(t, creds, "test-user", ex.name, "key-"+ex.name, "secret-"+ex.name)

			strategies := []StrategyEntry{
				{
					Strategy: "grid2",
					Exchange: ex.name,
					Mode:     "live",
					Config:   rawJSON(`{"symbol":"BTCUSDT"}`),
				},
			}

			args := cm.envArgs("test-user", ModeLive, strategies)
			findEnv := func(key string) bool {
				for i := 0; i < len(args)-1; i++ {
					if args[i] == "-e" && args[i+1] == key {
						return true
					}
				}
				return false
			}
			if !findEnv(ex.prefix + "_API_KEY=key-" + ex.name) {
				t.Errorf("expected %s_API_KEY for %s in env args", ex.prefix, ex.name)
			}
			if !findEnv(ex.prefix + "_API_SECRET=secret-" + ex.name) {
				t.Errorf("expected %s_API_SECRET for %s in env args", ex.prefix, ex.name)
			}
		})
	}
}
