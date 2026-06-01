package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- MarketSymbols endpoint ---

func TestAPI_MarketSymbols_ProxiesToMarketDataREST(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	symbolsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"symbols": []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "INVALIDPAIR", "123456"},
		})
	}))
	defer symbolsSrv.Close()

	api.cfg.MarketDataRESTAddr = strings.TrimPrefix(symbolsSrv.URL, "http://")
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(baseURL)
	}

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodGet, "/api/markets/binance/symbols", nil)
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	symbols, ok := resp["symbols"].([]interface{})
	if !ok || len(symbols) != 3 {
		t.Fatalf("expected 3 filtered symbols, got %d: %v", len(symbols), resp["symbols"])
	}
	for _, s := range symbols {
		str, ok := s.(string)
		if !ok {
			t.Fatalf("symbol is not a string: %v", s)
		}
		if str == "INVALIDPAIR" || str == "123456" {
			t.Errorf("invalid symbol %q should have been filtered out", str)
		}
	}
}

func TestAPI_MarketSymbols_BackendDown(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	api.cfg.MarketDataRESTAddr = "localhost:1"
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(baseURL)
	}

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodGet, "/api/markets/binance/symbols", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

// --- Container restart on strategy create while starting ---

func TestAPI_CreateStrategy_StartingContainer_NoExtraStart(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	tnKey, _ := api.encryptor.Encrypt("tn-key")
	tnSec, _ := api.encryptor.Encrypt("tn-secret")
	api.creds.Upsert(ExchangeCredential{
		UserID: userID, Exchange: "binance",
		APIKeyEncrypted: tnKey, APISecretEncrypted: tnSec, IsTestnet: true,
	})

	writeTestStrategies(t, api.strategies, userID, ModePaper, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	// Simulate "starting" state via the starting sync.Map
	api.starting.Store(userID+":"+ModePaper, true)

	startCount := 0
	api.containerStart = func(userID, mode string) error {
		startCount++
		return nil
	}

	r := testRouter(api)
	stratBody := `{"name":"Test Grid","exchange":"binance","strategy":"grid2","config":{},"mode":"paper"}`
	req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", strings.NewReader(stratBody))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	if startCount != 0 {
		t.Errorf("expected no container start for starting container, got %d starts", startCount)
	}
}

// --- Mode inheritance: strategy without mode inherits existing mode ---

func TestAPI_CreateStrategy_ModeInheritsFromExisting(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	tnKey, _ := api.encryptor.Encrypt("tn-key")
	tnSec, _ := api.encryptor.Encrypt("tn-secret")
	api.creds.Upsert(ExchangeCredential{
		UserID: userID, Exchange: "binance",
		APIKeyEncrypted: tnKey, APISecretEncrypted: tnSec, IsTestnet: true,
	})

	r := testRouter(api)
	stratBody := `{"name":"No Mode","exchange":"binance","strategy":"grid","config":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", strings.NewReader(stratBody))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for no-mode strategy with existing paper, got %d: %s", w.Code, w.Body.String())
	}
}

// --- envArgs: paper strategy sets PAPER_TRADE=1 ---

func TestEnvArgs_PaperStrategy_SetsPaperTrade(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	cm := &ContainerManager{cfg: &Config{DataDir: tmpDir, BBGOPort: 8080, BBGOGRPCPort: 9090}, creds: creds}

	strategies := []StrategyEntry{
		{Exchange: "binance", Strategy: "grid", Mode: "paper"},
	}
	args := cm.envArgs("test-user", ModePaper, strategies)

	hasPaper := false
	for i, a := range args {
		if a == "PAPER_TRADE=1" {
			hasPaper = true
			break
		}
		if i > 0 && args[i-1] == "-e" && a == "PAPER_TRADE=1" {
			hasPaper = true
			break
		}
	}
	if !hasPaper {
		t.Error("expected PAPER_TRADE=1 in env args for paper strategy")
	}
}

func TestEnvArgs_LiveStrategy_NoPaperTrade(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	keyEnc, _ := enc.Encrypt("key")
	secretEnc, _ := enc.Encrypt("secret")
	creds.Upsert(ExchangeCredential{
		UserID:             "test-user",
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
	})
	cm := &ContainerManager{cfg: &Config{DataDir: tmpDir, BBGOPort: 8080, BBGOGRPCPort: 9090, MarketDataAddr: "marketdata:9090"}, creds: creds}

	strategies := []StrategyEntry{
		{Exchange: "binance", Strategy: "grid", Mode: "live"},
	}
	args := cm.envArgs("test-user", ModeLive, strategies)

	for _, a := range args {
		if a == "PAPER_TRADE=1" {
			t.Error("PAPER_TRADE=1 should not be set for live strategy")
		}
	}

	hasMarketData := false
	hasDBDriver := false
	for _, a := range args {
		if strings.HasPrefix(a, "MARKET_DATA_SERVICE_URL=") {
			hasMarketData = true
		}
		if a == "DB_DRIVER=supabase" {
			hasDBDriver = true
		}
	}
	if !hasMarketData {
		t.Error("expected MARKET_DATA_SERVICE_URL in env args")
	}
	if !hasDBDriver {
		t.Error("expected DB_DRIVER=supabase in env args")
	}
}

// --- Credential delete triggers container restart for running user ---

func TestAPI_DeleteCredential_RunningContainer_SetsStarting(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	writeTestStrategies(t, api.strategies, userID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})
	api.containerRunning = func(string, _ string) bool { return true }

	var restartCalled bool
	api.containerStart = func(userID, mode string) error {
		restartCalled = true
		return nil
	}

	credBody := `{"exchange":"binance","api_key":"testkey","api_secret":"testsecret"}`
	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(credBody))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create credential: expected 201, got %d", w.Code)
	}

	var credResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&credResp)
	credID, _ := credResp["id"].(string)

	req2 := httptest.NewRequest(http.MethodDelete, "/api/credentials/"+credID, nil)
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("delete credential: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	time.Sleep(200 * time.Millisecond)
	if !restartCalled {
		t.Error("expected container restart after credential delete on running container")
	}
}
