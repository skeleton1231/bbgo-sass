package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// testRouterWithUser is like testRouter but also sets X-User-Id header.
func testRouterWithUser(api *API, userID string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			if userID != "" {
				r.Header.Set("X-User-Id", userID)
			}
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)
	return r
}

func TestResolveUserID_Mismatch_Rejected(t *testing.T) {
	store := NewStrategyStore("")
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }

	r := testRouterWithUser(api, "11111111-2222-3333-4444-555555555555")
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user ID mismatch, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "mismatch") {
		t.Errorf("expected mismatch error, got: %s", w.Body.String())
	}
}

func TestResolveUserID_MatchingHeader_Accepted(t *testing.T) {
	store := NewStrategyStore("")
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }

	r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for matching user ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResolveUserID_NoHeader_UsesURL(t *testing.T) {
	store := NewStrategyStore("")
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when no header (uses URL), got %d: %s", w.Code, w.Body.String())
	}
}

func TestResolveUserID_InvalidUUID_Rejected(t *testing.T) {
	store := NewStrategyStore("")
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResolveUserID_CreateStrategy_MismatchBlocked(t *testing.T) {
	store, dir := newTestStore(t)
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)

	r := testRouterWithUser(api, "11111111-2222-3333-4444-555555555555")

	body := map[string]interface{}{
		"name":     "Evil Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
		"mode":     "paper",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for strategy creation with mismatched user, got %d: %s", w.Code, w.Body.String())
	}

	strategies, _ := store.ListStrategies("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive)
	if len(strategies) != 0 {
		t.Error("strategy should NOT have been created for mismatched user")
	}
}

func TestCredentialEndpoint_UsesHeaderUserID(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)

	store := NewStrategyStore(dir)
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, nil)

	r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	body := map[string]interface{}{
		"exchange":   "binance",
		"api_key":    "my-key",
		"api_secret": "my-secret",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/credentials", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	stored, err := creds.List("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("list credentials: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(stored))
	}
	if stored[0].Exchange != "binance" {
		t.Errorf("expected binance exchange, got %s", stored[0].Exchange)
	}
}

func TestCredentialEndpoint_MissingUserID_Rejected(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)

	store := NewStrategyStore(dir)
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, enc, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)

	body := map[string]interface{}{
		"exchange":   "binance",
		"api_key":    "my-key",
		"api_secret": "my-secret",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/credentials", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing user ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAndStart_WritesYAMLToDisk(t *testing.T) {
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
		MarketDataAddr: "bbgo-marketdata:9090",
	}
	_ = NewContainerManager(cfg, creds, nil)

	strategies := []StrategyEntry{
		{
			Name:     "Paper Grid",
			Exchange: "binance",
			Strategy: "grid2",
			Mode:     "paper",
			Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`),
		},
	}

	yaml, err := buildUserYAML("test-user", ModePaper, strategies, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("build yaml: %v", err)
	}

	hostDir := filepath.Join(dir, "test-user")
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	yamlPath := filepath.Join(hostDir, "bbgo.yaml")
	if err := os.WriteFile(yamlPath, yaml, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, "PAPER_TRADE:") {
		t.Error("YAML missing PAPER_TRADE for paper mode")
	}
	if !strings.Contains(s, "grid2:") {
		t.Error("YAML missing grid2 strategy")
	}
	if !strings.Contains(s, "BTCUSDT") {
		t.Error("YAML missing symbol BTCUSDT")
	}
	if !strings.Contains(s, "publicOnly: true") {
		t.Error("YAML missing publicOnly (no credentials)")
	}
	if !strings.Contains(s, "binance:") {
		t.Error("YAML missing binance session")
	}
}

func TestDockerArgs_LiveMode_EnvAssembly(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "test-user", "binance", "live-key", "live-secret")

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
	cm := NewContainerManager(cfg, creds, nil)

	liveStrategies := []StrategyEntry{
		{
			Name:     "Live Grid",
			Exchange: "binance",
			Strategy: "grid2",
			Mode:     "live",
			Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`),
		},
	}

	args := cm.envArgs("test-user", ModeLive, liveStrategies)

	findEnv := func(key string) bool {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-e" && args[i+1] == key {
				return true
			}
		}
		return false
	}

	if findEnv("PAPER_TRADE=1") {
		t.Error("live mode should NOT have PAPER_TRADE=1")
	}
	if !findEnv("BINANCE_API_KEY=live-key") {
		t.Error("expected BINANCE_API_KEY for live mode")
	}
	if !findEnv("BINANCE_API_SECRET=live-secret") {
		t.Error("expected BINANCE_API_SECRET for live mode")
	}
	if !findEnv("DB_DRIVER=supabase") {
		t.Error("expected DB_DRIVER")
	}
	if !findEnv("MARKET_DATA_SERVICE_URL=bbgo-marketdata:9090") {
		t.Error("expected MARKET_DATA_SERVICE_URL")
	}
}
