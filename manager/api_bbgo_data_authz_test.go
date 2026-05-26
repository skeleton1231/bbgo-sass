package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestBBGoDataEndpoints_UserIDMismatch_Rejected validates that all bbgo data
// endpoints (ping, sessions, trades, etc.) reject requests where the URL path
// user ID doesn't match the X-User-Id header.
func TestBBGoDataEndpoints_UserIDMismatch_Rejected(t *testing.T) {
	victimID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	attackerID := "11111111-2222-3333-4444-555555555555"

	users := NewUserContainerManager()
	users.users[victimID] = &UserContainer{
		UserID:     victimID,
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"trades": []interface{}{},
		})
	}))
	defer bbgoSrv.Close()
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", attackerID)
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	endpoints := []struct {
		name   string
		path   string
		method string
	}{
		{"ping", "/api/users/" + victimID + "/bbgo/ping", "GET"},
		{"sessions", "/api/users/" + victimID + "/bbgo/sessions", "GET"},
		{"trades", "/api/users/" + victimID + "/bbgo/trades", "GET"},
		{"closed_orders", "/api/users/" + victimID + "/bbgo/orders/closed", "GET"},
		{"assets", "/api/users/" + victimID + "/bbgo/assets", "GET"},
		{"strategies", "/api/users/" + victimID + "/bbgo/strategies", "GET"},
		{"trading_volume", "/api/users/" + victimID + "/bbgo/trading-volume", "GET"},
		{"session_detail", "/api/users/" + victimID + "/bbgo/session/binance", "GET"},
		{"session_trades", "/api/users/" + victimID + "/bbgo/session/binance/trades", "GET"},
		{"session_orders", "/api/users/" + victimID + "/bbgo/session/binance/open-orders", "GET"},
		{"session_account", "/api/users/" + victimID + "/bbgo/session/binance/account", "GET"},
		{"session_balances", "/api/users/" + victimID + "/bbgo/session/binance/balances", "GET"},
		{"session_symbols", "/api/users/" + victimID + "/bbgo/session/binance/symbols", "GET"},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("expected 403 for user ID mismatch on %s, got %d: %s", ep.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestBBGoDataEndpoints_MatchingUserID_Accepted verifies that bbgo data
// endpoints succeed when X-User-Id matches the URL path user ID.
func TestBBGoDataEndpoints_MatchingUserID_Accepted(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	users := NewUserContainerManager()
	users.users[userID] = &UserContainer{
		UserID:     userID,
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "ok"})
	}))
	defer bbgoSrv.Close()
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", userID)
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for matching user ID, got %d: %s", w.Code, w.Body.String())
	}
}

// TestPnLEndpoint_UserIDMismatch_Rejected validates the PnL endpoint also
// checks user ID authorization (it uses userFromURL, not resolveUserID).
func TestPnLEndpoint_UserIDMismatch_Rejected(t *testing.T) {
	victimID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	attackerID := "11111111-2222-3333-4444-555555555555"

	users := NewUserContainerManager()
	users.users[victimID] = &UserContainer{
		UserID:     victimID,
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"trades": []interface{}{},
		})
	}))
	defer bbgoSrv.Close()
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", attackerID)
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+victimID+"/bbgo/pnl", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for PnL user ID mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCredentialCreate_TriggersContainerRestart verifies that adding
// credentials to a running container triggers a restart.
func TestCredentialCreate_TriggersContainerRestart(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live"}},
	}

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)

	var restartCalled bool
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)
	api.containerStart = func(_ *UserContainer) error {
		restartCalled = true
		return nil
	}
	api.containerRunning = func(_ string) bool { return true }

	r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	body := map[string]interface{}{
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

	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for credential-triggered container restart")
		case <-ticker.C:
			if restartCalled {
				return
			}
		}
	}
}

// TestCredentialDelete_TriggersContainerRestart verifies that removing
// credentials from a running container triggers a restart.
func TestCredentialDelete_TriggersContainerRestart(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live"}},
	}

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)

	insertTestCredential(t, creds, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "binance", "k", "s")
	stored, _ := creds.List("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	credID := stored[0].ID

	var restartCalled bool
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)
	api.containerStart = func(_ *UserContainer) error {
		restartCalled = true
		return nil
	}
	api.containerRunning = func(_ string) bool { return true }

	r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	req := httptest.NewRequest("DELETE", "/api/credentials/"+credID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for credential-delete container restart")
		case <-ticker.C:
			if restartCalled {
				return
			}
		}
	}
}
