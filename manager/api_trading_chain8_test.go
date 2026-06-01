package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupCredsAPI creates an API with credential store and encryption ready.
func setupCredsAPI(t *testing.T) (*API, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(tmpDir, enc)
	store, _ := newTestStore(t)

	cfg := &Config{ManagerToken: "test-token", DataDir: tmpDir}
	cm := &ContainerManager{cfg: cfg}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, NewBacktestJobStore(tmpDir))
	api.containerRunning = func(string, _ string) bool { return false }
	api.containerStart = func(userID, mode string) error { return nil }
	api.verifyCredFn = func(_, _, _, _ string, _ bool) VerifyResult { return VerifyResult{Verified: true} }
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`ok`))
		}))
		t.Cleanup(srv.Close)
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}
	return api, func() { api.Close() }
}

const credsUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeee000070"

// --- Credential CRUD through API ---

func TestAPI_CreateCredential_Success(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	body := `{"exchange":"binance","api_key":"mykey","api_secret":"mysecret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create credential: status = %d, want 201: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["exchange"] != "binance" {
		t.Errorf("expected exchange binance, got %v", resp["exchange"])
	}
	if resp["id"] == "" {
		t.Error("expected non-empty id")
	}
}

func TestAPI_CreateCredential_MissingFields(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	body := `{"exchange":"binance","api_key":"mykey"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing secret: status = %d, want 400", w.Code)
	}
}

func TestAPI_CreateCredential_UnsupportedExchange(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	body := `{"exchange":"unknown","api_key":"k","api_secret":"s"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unsupported exchange: status = %d, want 400", w.Code)
	}
}

func TestAPI_ListCredentials_AfterCreate(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	body := `{"exchange":"binance","api_key":"k","api_secret":"s"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/credentials", nil)
	req2.Header.Set("X-User-Id", credsUID)
	w2 := httptest.NewRecorder()
	api.ListCredentials(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list: status = %d, want 200", w2.Code)
	}
	var list []map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(list))
	}
	if list[0]["exchange"] != "binance" {
		t.Errorf("expected binance, got %v", list[0]["exchange"])
	}
}

func TestAPI_DeleteCredential_Success(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	body := `{"exchange":"binance","api_key":"k","api_secret":"s"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	credID := created["id"].(string)

	req2 := chiReq("DELETE", "/api/credentials/"+credID, "", "id", credID)
	req2.Header.Set("X-User-Id", credsUID)
	w2 := httptest.NewRecorder()
	api.DeleteCredential(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("delete: status = %d, want 200: %s", w2.Code, w2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/credentials", nil)
	req3.Header.Set("X-User-Id", credsUID)
	w3 := httptest.NewRecorder()
	api.ListCredentials(w3, req3)
	var list []map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&list)
	if len(list) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(list))
	}
}

func TestAPI_DeleteCredential_NotFound(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	req := chiReq("DELETE", "/api/credentials/nonexistent", "", "id", "nonexistent")
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.DeleteCredential(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("not found: status = %d, want 404", w.Code)
	}
}

// --- Credential with passphrase (OKEx) ---

func TestAPI_CreateCredential_WithPassphrase(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	body := `{"exchange":"okex","api_key":"k","api_secret":"s","passphrase":"p"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("okex credential: status = %d, want 201: %s", w.Code, w.Body.String())
	}
}

// --- Credential triggers restart when container running ---

func TestAPI_CreateCredential_TriggersRestart(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	// Write strategies and mark as running
	writeTestStrategies(t, api.strategies, credsUID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})
	api.containerRunning = func(string, _ string) bool { return true }

	var restartCalled bool
	api.containerStart = func(userID, mode string) error {
		restartCalled = true
		return nil
	}

	body := `{"exchange":"binance","api_key":"k","api_secret":"s"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	if !restartCalled {
		t.Error("expected container restart after credential create while running")
	}
}

// --- PnL container fallback when no syncer ---

func TestAPI_PnL_ContainerFallback_Stopped(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	req := chiReq("GET", "/api/users/"+credsUID+"/bbgo/pnl", "", "userID", credsUID)
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.BBGoPnL(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("stopped container: status = %d, want 503", w.Code)
	}
}

func TestAPI_PnL_ContainerFallback_Running(t *testing.T) {
	tmpDir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trades" {
			w.Write([]byte(`{"trades":[{"symbol":"BTCUSDT","side":"BUY","price":"50000","quantity":"1","fee":"10","traded_at":"2024-01-01"},{"symbol":"BTCUSDT","side":"SELL","price":"55000","quantity":"1","fee":"10","traded_at":"2024-01-02"}]}`))
			return
		}
		w.Write([]byte(`{"message":"ok"}`))
	}))
	defer bbgoSrv.Close()

	store, _ := newTestStore(t)
	writeTestStrategies(t, store, credsUID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	cm := &ContainerManager{cfg: &Config{DataDir: tmpDir, BBGOPort: 8080}}
	creds := NewCredentialStore(tmpDir, enc)
	proxy := NewBotProxy(cm)

	api := NewAPI(&Config{DataDir: tmpDir, ManagerToken: "t"}, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, NewBacktestJobStore(tmpDir))
	api.containerRunning = func(string, _ string) bool { return true }
	api.containerStart = func(userID, mode string) error { return nil }
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
	}
	defer api.Close()

	req := chiReq("GET", "/api/users/"+credsUID+"/bbgo/pnl", "", "userID", credsUID)
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.BBGoPnL(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("pnl running: status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var report PnLReport
	json.NewDecoder(w.Body).Decode(&report)
	if report.TotalRealizedPnL != 5000 {
		t.Errorf("expected realized PnL 5000, got %f", report.TotalRealizedPnL)
	}
}

// --- WebSocket ticket auth ---

func TestAPI_WSTicket_IssuesValidTicket(t *testing.T) {
	api, cleanup := setupCredsAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/ws/ticket", nil)
	req.Header.Set("X-User-Id", credsUID)
	w := httptest.NewRecorder()
	api.IssueWSTicket(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ticket: status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	ticket, ok := resp["ticket"].(string)
	if !ok || ticket == "" {
		t.Fatal("expected non-empty ticket")
	}
}
