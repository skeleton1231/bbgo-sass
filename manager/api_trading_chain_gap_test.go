package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// --- DeleteStrategy chain tests ---

// TestDeleteLastStrat_StopsContainer verifies the full chain:
// removing the last strategy stops the container and sets status=stopped.
func TestDeleteLastStrat_StopsContainer(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.users.users[userID+":"+ModeLive].Status = StatusRunning
	api.users.users[userID+":"+ModeLive].Strategies = []StrategyEntry{
		{ID: "strat-1", Exchange: "binance", Strategy: "grid", Mode: "paper"},
	}
	api.containerRunning = func(string, _ string) bool { return true }

	stopCalled := false
	api.containerStop = func(userID string, _ string) {
		stopCalled = true
	}
	api.containerStart = func(uc *UserContainer) error { return nil }

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID+"/strategies/strat-1", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !stopCalled {
		t.Error("expected container stop when last strategy deleted")
	}
	uc, _ := api.users.Get(userID, ModeLive)
	if uc != nil {
		t.Errorf("expected container deleted after last strategy removed, got status %s", uc.Status)
	}
}

// TestDeleteStrategy_RemainingStrategies_RestartsRunningContainer verifies:
// deleting a strategy from a running container triggers async restart.
func TestDeleteStrategy_RemainingStrategies_RestartsRunningContainer(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.users.users[userID+":"+ModeLive].Status = StatusRunning
	api.users.users[userID+":"+ModeLive].Strategies = []StrategyEntry{
		{ID: "strat-1", Exchange: "binance", Strategy: "grid", Mode: "paper"},
		{ID: "strat-2", Exchange: "binance", Strategy: "grid2", Mode: "paper"},
	}
	api.containerRunning = func(string, _ string) bool { return true }

	startCalled := false
	api.containerStart = func(uc *UserContainer) error {
		startCalled = true
		return nil
	}

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID+"/strategies/strat-1", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	uc, _ := api.users.Get(userID, ModeLive)
	if len(uc.Strategies) != 1 || uc.Strategies[0].ID != "strat-2" {
		t.Errorf("expected 1 remaining strategy (strat-2), got %v", uc.Strategies)
	}

	time.Sleep(100 * time.Millisecond)
	if !startCalled {
		t.Error("expected container restart after strategy deletion on running container")
	}
}

// TestDeleteStrategy_StoppedContainer_NoRestart verifies:
// deleting a strategy when container is stopped does NOT trigger restart.
func TestDeleteStrategy_StoppedContainer_NoRestart(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.users.users[userID+":"+ModeLive].Status = StatusStopped
	api.users.users[userID+":"+ModeLive].Strategies = []StrategyEntry{
		{ID: "strat-1", Exchange: "binance", Strategy: "grid", Mode: "paper"},
		{ID: "strat-2", Exchange: "binance", Strategy: "grid2", Mode: "paper"},
	}

	startCalled := false
	api.containerStart = func(uc *UserContainer) error {
		startCalled = true
		return nil
	}

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID+"/strategies/strat-1", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	if startCalled {
		t.Error("expected NO container restart when container is stopped")
	}
}

// TestDeleteMissingStrat_NotFound verifies 404 for missing strategy.
func TestDeleteMissingStrat_NotFound(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.users.users[userID+":"+ModeLive].Strategies = []StrategyEntry{
		{ID: "strat-1", Exchange: "binance", Strategy: "grid", Mode: "paper"},
	}

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID+"/strategies/nonexistent", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- BBGo data endpoint when container stopped ---

// TestBBGoDataEndpoint_ContainerStopped_Returns503 verifies that
// bbgo data endpoints return 503 when the container is not running.
func TestBBGoDataEndpoint_ContainerStopped_Returns503(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.users.users[userID+":"+ModeLive].Status = StatusStopped

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID+"/bbgo/ping", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when container stopped, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Proxy path construction ---

// TestProxyToBot_PathStripping verifies the proxy correctly strips
// the /api/bbgo/{userID} prefix and re-adds /api.
func TestProxyToBot_PathStripping(t *testing.T) {
	proxyCalls := []string{}
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyCalls = append(proxyCalls, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer bbgoSrv.Close()

	cm := &ContainerManager{cfg: &Config{}}
	proxy := NewBotProxy(cm)
	proxy.resolveAddr = func(string, _ string) string { return bbgoSrv.URL }

	for _, tc := range []struct {
		input    string
		expected string
	}{
		{"/api/bbgo/user-1/sessions", "/api/sessions"},
		{"/api/bbgo/user-1/session/main/trades", "/api/session/main/trades"},
		{"/api/bbgo/user-1/ping", "/api/ping"},
	} {
		proxyCalls = nil
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, tc.input, nil)
		proxy.ProxyToBot(w, r, "user-1", ModeLive)

		if len(proxyCalls) == 0 || proxyCalls[0] != tc.expected {
			t.Errorf("proxy(%s): expected path %q, got %v", tc.input, tc.expected, proxyCalls)
		}
		if w.Code != http.StatusOK {
			t.Errorf("proxy(%s): expected 200, got %d", tc.input, w.Code)
		}
	}
}

// TestProxyToBot_RemovesSensitiveHeaders verifies sensitive headers are removed
// before forwarding to the bbgo container.
func TestProxyToBot_RemovesSensitiveHeaders(t *testing.T) {
	receivedHeaders := http.Header{}
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer bbgoSrv.Close()

	cm := &ContainerManager{cfg: &Config{}}
	proxy := NewBotProxy(cm)
	proxy.resolveAddr = func(string, _ string) string { return bbgoSrv.URL }

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/bbgo/user-1/sessions", nil)
	r.Header.Set("X-Manager-Token", "secret-token")
	r.Header.Set("X-User-Id", "user-123")
	r.Header.Set("Accept", "application/json")
	proxy.ProxyToBot(w, r, "user-1", ModeLive)

	if receivedHeaders.Get("X-Manager-Token") != "" {
		t.Error("X-Manager-Token should be stripped before forwarding to bbgo container")
	}
	if receivedHeaders.Get("X-User-Id") != "" {
		t.Error("X-User-Id should be stripped before forwarding to bbgo container")
	}
	if receivedHeaders.Get("Accept") != "application/json" {
		t.Error("Accept header should be preserved")
	}
}

// --- Full trading chain: paper mode end-to-end ---

// TestTradingChain_PaperMode_FullFlow tests the complete paper trading chain:
// create strategy (paper) → start container → verify YAML has PAPER_TRADE.
func TestTradingChain_PaperMode_FullFlow(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)

	users := NewUserContainerManager()
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	tnKey, _ := enc.Encrypt("tn-key")
	tnSec, _ := enc.Encrypt("tn-secret")
	creds.Upsert(ExchangeCredential{
		ID: "tn1", UserID: userID, Exchange: "binance",
		APIKeyEncrypted: tnKey, APISecretEncrypted: tnSec, IsTestnet: true,
	})

	cfg := &Config{
		DataDir:       tmpDir,
		ManagerToken:  "test-token",
		SupabaseURL:   "http://localhost:1",
		SupabaseKey:   "test",
		BBGOPort:      8080,
		BBGOGRPCPort:  9090,
		BBGOImage:     "bbgo-base:latest",
		DockerNetwork: "test-net",
		DataVolume:    "test-data",
	}
	cm := &ContainerManager{cfg: cfg, creds: creds}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, NewBacktestJobStore(tmpDir))

	var capturedArgs []string
	api.containerStart = func(uc *UserContainer) error {
		capturedArgs = cm.envArgs(uc)
		yamlContent, _ := buildUserYAML(uc, func(exchange string) bool {
			if cm.creds == nil {
				return false
			}
			_, _, _, err := cm.creds.GetDecrypted(uc.UserID, exchange)
			return err == nil
		})
		if yamlContent != nil {
			os.MkdirAll(tmpDir+"/"+userID, 0o755)
			os.WriteFile(tmpDir+"/"+userID+"/bbgo.yaml", yamlContent, 0o644)
		}
		return nil
	}
	api.containerRunning = func(string, _ string) bool { return false }
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`ok`)) }))
		t.Cleanup(srv.Close)
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}

	r := testRouter(api)

	// Step 1: Create paper strategy
	body := `{"name":"Paper Grid","exchange":"binance","strategy":"grid2","config":{"symbol":"BTCUSDT"},"mode":"paper"}`
	req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create strategy: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: Start user container
	req2 := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/start?mode=paper", nil)
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusAccepted {
		t.Fatalf("start user: expected 202, got %d: %s", w2.Code, w2.Body.String())
	}

	time.Sleep(200 * time.Millisecond)

	// Step 3: Verify YAML written to disk has PAPER_TRADE
	yamlBytes, err := os.ReadFile(tmpDir + "/" + userID + "/bbgo.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(yamlBytes), "PAPER_TRADE") {
		t.Errorf("expected PAPER_TRADE in YAML, got:\n%s", string(yamlBytes))
	}

	// Step 4: Verify env args have PAPER_TRADE=1
	hasPaper := false
	for _, a := range capturedArgs {
		if a == "PAPER_TRADE=1" {
			hasPaper = true
		}
	}
	if !hasPaper {
		t.Errorf("expected PAPER_TRADE=1 in env args, got %v", capturedArgs)
	}

	// Step 5: Verify testnet API keys are injected for paper mode
	hasApiKey := false
	for _, a := range capturedArgs {
		if strings.Contains(a, "BINANCE_API_KEY=tn-key") {
			hasApiKey = true
		}
	}
	if !hasApiKey {
		t.Errorf("expected testnet API key injection for paper mode, got %v", capturedArgs)
	}
}

// --- Full trading chain: live mode end-to-end ---

// TestTradingChain_LiveMode_FullFlow tests the complete live trading chain:
// create credentials → create strategy (live) → start → verify API keys injected.
func TestTradingChain_LiveMode_FullFlow(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)

	users := NewUserContainerManager()
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	cfg := &Config{
		DataDir:        tmpDir,
		ManagerToken:   "test-token",
		SupabaseURL:    "http://localhost:1",
		SupabaseKey:    "test",
		BBGOPort:       8080,
		BBGOGRPCPort:   9090,
		BBGOImage:      "bbgo-base:latest",
		DockerNetwork:  "test-net",
		DataVolume:     "test-data",
		MarketDataAddr: "marketdata:9090",
	}
	cm := &ContainerManager{cfg: cfg, creds: creds}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, NewBacktestJobStore(tmpDir))

	var capturedArgs []string
	api.containerStart = func(uc *UserContainer) error {
		capturedArgs = cm.envArgs(uc)
		yamlContent, _ := buildUserYAML(uc, func(exchange string) bool {
			if cm.creds == nil {
				return false
			}
			_, _, _, err := cm.creds.GetDecrypted(uc.UserID, exchange)
			return err == nil
		})
		if yamlContent != nil {
			os.MkdirAll(tmpDir+"/"+userID, 0o755)
			os.WriteFile(tmpDir+"/"+userID+"/bbgo.yaml", yamlContent, 0o644)
		}
		return nil
	}
	api.containerRunning = func(string, _ string) bool { return false }
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`ok`)) }))
		t.Cleanup(srv.Close)
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}

	r := testRouter(api)

	// Step 1: Create credentials
	credBody := `{"exchange":"binance","api_key":"mykey123","api_secret":"mysecret456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(credBody))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create credential: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: Create live strategy
	stratBody := `{"name":"Live Grid","exchange":"binance","strategy":"grid2","config":{"symbol":"BTCUSDT"},"mode":"live"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", strings.NewReader(stratBody))
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("create live strategy: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	// Step 3: Start container
	req3 := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/start?mode=live", nil)
	req3.Header.Set("X-User-Id", userID)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusAccepted {
		t.Fatalf("start: expected 202, got %d: %s", w3.Code, w3.Body.String())
	}

	time.Sleep(200 * time.Millisecond)

	// Step 4: Verify YAML has NO PAPER_TRADE
	yamlBytes, err := os.ReadFile(tmpDir + "/" + userID + "/bbgo.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(yamlBytes), "PAPER_TRADE") {
		t.Errorf("live mode should NOT have PAPER_TRADE in YAML, got:\n%s", string(yamlBytes))
	}

	// Step 5: Verify env args have API keys and no PAPER_TRADE
	hasAPIKey := false
	hasAPISecret := false
	for _, a := range capturedArgs {
		if a == "PAPER_TRADE=1" {
			t.Error("live mode should not set PAPER_TRADE=1")
		}
		if a == "BINANCE_API_KEY=mykey123" {
			hasAPIKey = true
		}
		if a == "BINANCE_API_SECRET=mysecret456" {
			hasAPISecret = true
		}
	}
	if !hasAPIKey {
		t.Error("expected BINANCE_API_KEY in env args for live mode")
	}
	if !hasAPISecret {
		t.Error("expected BINANCE_API_SECRET in env args for live mode")
	}

	// Step 6: Verify YAML session is NOT PublicOnly
	if strings.Contains(string(yamlBytes), "publicOnly") {
		t.Errorf("live mode with credentials should NOT have publicOnly in YAML, got:\n%s", string(yamlBytes))
	}
}

// --- Credential endpoint guard when encryptor is nil ---

func TestCreateCredential_NilEncryptor_Returns503(t *testing.T) {
	users := NewUserContainerManager()
	cfg := &Config{ManagerToken: "test-token", SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	proxy := NewBotProxy(cm)
	// No encryptor, no creds
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(string, _ string) bool { return false }

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(`{"exchange":"binance","api_key":"k","api_secret":"s"}`))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when encryptor is nil, got %d: %s", w.Code, w.Body.String())
	}
}
