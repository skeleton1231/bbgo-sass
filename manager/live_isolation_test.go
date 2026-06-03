package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- Live container env isolation tests ---

// TestLiveContainer_NoTestnetEnv verifies live containers do NOT receive
// TESTNET or PAPER_TRADE environment variables.
func TestLiveContainer_NoTestnetEnv(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	liveKey, _ := enc.Encrypt("live-key")
	liveSec, _ := enc.Encrypt("live-secret")
	creds.Upsert(ExchangeCredential{
		UserID: userID, Exchange: "binance",
		APIKeyEncrypted: liveKey, APISecretEncrypted: liveSec, IsTestnet: false, IsVerified: true,
	})

	cfg := &Config{
		DataDir: tmpDir, ManagerToken: "test-token",
		SupabaseURL: "http://localhost:1", SupabaseKey: "test",
		BBGOPort: 8080, BBGOGRPCPort: 9090,
		BBGOImage: "bbgo-base:latest", DockerNetwork: "test-net", DataVolume: "test-data",
	}
	cm := &ContainerManager{cfg: cfg, creds: creds}
	proxy := NewBotProxy(cm)
	store := NewStrategyStore(tmpDir)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, NewBacktestJobStore(tmpDir), nil)
	api.verifyCredFn = func(_, _, _, _ string, _ bool) VerifyResult { return VerifyResult{Verified: true} }

	var capturedArgs []string
	api.containerStart = func(startUserID, startMode string) error {
		strategies, _ := api.strategies.ListStrategies(startUserID, startMode)
		capturedArgs = cm.envArgs(startUserID, startMode, strategies)
		return nil
	}
	api.containerRunning = func(string, string) bool { return false }
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`ok`)) }))
		t.Cleanup(srv.Close)
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}

	r := testRouter(api)

	body := `{"name":"Live Grid","exchange":"binance","strategy":"grid2","config":{"symbol":"BTCUSDT"},"mode":"live"}`
	req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create strategy: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/start?mode=live", nil)
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusAccepted {
		t.Fatalf("start: expected 202, got %d: %s", w2.Code, w2.Body.String())
	}

	time.Sleep(200 * time.Millisecond)

	for _, a := range capturedArgs {
		if a == "PAPER_TRADE=1" {
			t.Error("live container should NOT have PAPER_TRADE=1")
		}
		if a == "BINANCE_TESTNET=1" {
			t.Error("live container should NOT have BINANCE_TESTNET=1")
		}
	}

	hasLiveKey := false
	for _, a := range capturedArgs {
		if a == "BINANCE_API_KEY=live-key" {
			hasLiveKey = true
		}
	}
	if !hasLiveKey {
		t.Errorf("expected live API key in env args, got %v", capturedArgs)
	}
}

// TestLiveContainer_YAML_DisableStartupBalanceQuery verifies live mode YAML
// has disablestartupbalancequery set and no PAPER_TRADE.
func TestLiveContainer_YAML_DisableStartupBalanceQuery(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	liveKey, _ := enc.Encrypt("live-key")
	liveSec, _ := enc.Encrypt("live-secret")
	creds.Upsert(ExchangeCredential{
		UserID: userID, Exchange: "binance",
		APIKeyEncrypted: liveKey, APISecretEncrypted: liveSec, IsTestnet: false, IsVerified: true,
	})

	strategies := []StrategyEntry{}

	yamlContent, err := buildUserYAML(userID, ModeLive, strategies, func(exchange string) bool {
		if creds == nil {
			return false
		}
		_, _, _, e := creds.GetDecryptedByMode(userID, exchange, false)
		return e == nil
	})
	if err != nil {
		t.Fatal(err)
	}

	yamlStr := string(yamlContent)
	if strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Errorf("live YAML should NOT contain PAPER_TRADE, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "disablestartupbalancequery") {
		t.Errorf("live YAML should contain disablestartupbalancequery, got:\n%s", yamlStr)
	}
	if strings.Contains(yamlStr, "publicOnly") {
		t.Errorf("live YAML with credentials should NOT have publicOnly, got:\n%s", yamlStr)
	}
}

// TestDualContainer_CredentialIsolation verifies live and paper containers
// each receive their own mode-specific credentials.
func TestDualContainer_CredentialIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	liveKey, _ := enc.Encrypt("live-key")
	liveSec, _ := enc.Encrypt("live-secret")
	creds.Upsert(ExchangeCredential{
		UserID: userID, Exchange: "binance",
		APIKeyEncrypted: liveKey, APISecretEncrypted: liveSec, IsTestnet: false, IsVerified: true,
	})

	tnKey, _ := enc.Encrypt("tn-key")
	tnSec, _ := enc.Encrypt("tn-secret")
	creds.Upsert(ExchangeCredential{
		UserID: userID, Exchange: "binance",
		APIKeyEncrypted: tnKey, APISecretEncrypted: tnSec, IsTestnet: true, IsVerified: true,
	})

	cfg := &Config{
		DataDir: tmpDir, ManagerToken: "test-token",
		SupabaseURL: "http://localhost:1", SupabaseKey: "test",
		BBGOPort: 8080, BBGOGRPCPort: 9090,
		BBGOImage: "bbgo-base:latest", DockerNetwork: "test-net", DataVolume: "test-data",
	}
	cm := &ContainerManager{cfg: cfg, creds: creds}

	strategies := []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		}

	liveArgs := cm.envArgs(userID, ModeLive, strategies)
	paperArgs := cm.envArgs(userID, ModePaper, strategies)

	assertEnv(t, liveArgs, "BINANCE_API_KEY=live-key", "live container should have live API key")
	assertNoEnv(t, liveArgs, "BINANCE_TESTNET=1", "live container should NOT have TESTNET")
	assertNoEnv(t, liveArgs, "PAPER_TRADE=1", "live container should NOT have PAPER_TRADE")

	assertEnv(t, paperArgs, "BINANCE_API_KEY=tn-key", "paper container should have testnet API key")
	assertEnv(t, paperArgs, "BINANCE_TESTNET=1", "paper container should have TESTNET")
	assertEnv(t, paperArgs, "PAPER_TRADE=1", "paper container should have PAPER_TRADE")
}

// TestCreateCredential_RestartIsolation verifies that creating a live credential
// only restarts the live container, not the paper container.
func TestCreateCredential_RestartIsolation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	origURLs := exchangeBaseURLs
	exchangeBaseURLs = map[string]struct{ live, testnet string }{
		"binance": {live: srv.URL, testnet: srv.URL},
	}
	defer func() { exchangeBaseURLs = origURLs }()

	api, cleanup := setupCredsAPI(t)
	api.verifyCredFn = nil
	defer cleanup()

	userID := credsUID

	// Write strategies for both modes so both containers are considered "running"
	writeTestStrategies(t, api.strategies, userID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})
	writeTestStrategies(t, api.strategies, userID, ModePaper, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})
	api.containerRunning = containerRunningFor(map[string]bool{
		userID + ":" + ModeLive:  true,
		userID + ":" + ModePaper: true,
	})

	var liveStarts, paperStarts int
	api.containerStart = func(startUserID, startMode string) error {
		if startMode == ModeLive {
			liveStarts++
		} else if startMode == ModePaper {
			paperStarts++
		}
		return nil
	}

	// Create LIVE credential → should only restart live container
	body := `{"exchange":"binance","api_key":"live-key","api_secret":"live-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	testRouter(api).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	time.Sleep(100 * time.Millisecond)

	if liveStarts != 1 {
		t.Errorf("expected 1 live restart after live cred, got %d", liveStarts)
	}
	if paperStarts != 0 {
		t.Errorf("expected 0 paper restarts after live cred, got %d", paperStarts)
	}

	// Create TESTNET credential → should only restart paper container
	paperStarts, liveStarts = 0, 0
	tnBody := `{"exchange":"binance","api_key":"tn-key","api_secret":"tn-secret","is_testnet":true}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(tnBody))
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	testRouter(api).ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for testnet cred, got %d: %s", w2.Code, w2.Body.String())
	}
	time.Sleep(100 * time.Millisecond)

	if paperStarts != 1 {
		t.Errorf("expected 1 paper restart after testnet cred, got %d", paperStarts)
	}
	if liveStarts != 0 {
		t.Errorf("expected 0 live restarts after testnet cred, got %d", liveStarts)
	}
}

func assertEnv(t *testing.T, args []string, expected, msg string) {
	t.Helper()
	for _, a := range args {
		if a == expected {
			return
		}
	}
	t.Errorf("%s: not found in %v", msg, args)
}

func assertNoEnv(t *testing.T, args []string, forbidden, msg string) {
	t.Helper()
	for _, a := range args {
		if a == forbidden {
			t.Errorf("%s: found forbidden %s", msg, forbidden)
			return
		}
	}
}
