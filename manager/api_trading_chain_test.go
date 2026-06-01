package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPI_CreateStrategy_LiveRequiresCredentials(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)

	store, _ := newTestStore(t)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)

	body := map[string]any{
		"name":     "Live Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]any{"symbol": "BTCUSDT"},
		"mode":     "live",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for live mode without credentials, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "credentials") {
		t.Errorf("expected error message about credentials, got: %s", w.Body.String())
	}
}

func TestAPI_CreateStrategy_LiveWithCredentials_Accepted(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "binance", "key", "secret")

	store, _ := newTestStore(t)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)

	body := map[string]any{
		"name":     "Live Grid",
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
		t.Fatalf("expected 201 for live mode with credentials, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_LiveCrossExchange_MissingOneCredential(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "binance", "key", "secret")

	store, _ := newTestStore(t)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)

	body := map[string]any{
		"name":          "XMaker",
		"strategy":      "xmaker",
		"crossExchange": true,
		"config":        map[string]any{"symbol": "BTCUSDT"},
		"mode":          "live",
		"sessions": []map[string]any{
			{"name": "maker", "exchange": "binance", "envVarPrefix": "BINANCE"},
			{"name": "hedge", "exchange": "bybit", "envVarPrefix": "BYBIT"},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing bybit credentials, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "bybit") {
		t.Errorf("error should mention bybit, got: %s", w.Body.String())
	}
}

func TestAPI_DeleteLastStrategy_StopsContainer(t *testing.T) {
	var stopCalled bool
	bbgoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]any{
				"strategies": []map[string]any{
					{"strategyInstanceID": "s1", "strategy": "grid2", "symbol": "BTCUSDT", "on": []any{"binance"}},
				},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	api, _ := setupTestAPIWithMockCM(t, bbgoHandler, true)
	api.containerStop = func(userID string, _ string) {
		stopCalled = true
	}
	r := testRouter(api)

	req := httptest.NewRequest("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if !stopCalled {
		t.Error("deleting last strategy should stop the container")
	}

	strategies, _ := api.strategies.ListStrategies("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive)
	if len(strategies) != 0 {
		t.Errorf("expected no strategies after last strategy removed, got %d", len(strategies))
	}
}

func TestAPI_DeleteStrategy_RunningContainer_TriggersRestart(t *testing.T) {
	var restartCalled bool
	bbgoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]any{
				"strategies": []map[string]any{
					{"strategyInstanceID": "s1", "strategy": "grid2", "symbol": "BTCUSDT", "on": []any{"binance"}},
					{"strategyInstanceID": "s2", "strategy": "supertrend", "symbol": "ETHUSDT", "on": []any{"binance"}},
				},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	api, _ := setupTestAPIWithMockCM(t, bbgoHandler, true)

		// Write both strategies so deleting one leaves another
		writeTestStrategies(t, api.strategies, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
			{Exchange: "binance", Strategy: "supertrend", Config: rawJSON(`{"symbol":"ETHUSDT"}`)},
		})


	api.containerStart = func(userID, mode string) error {
		restartCalled = true
		return nil
	}

	r := testRouter(api)
	req := httptest.NewRequest("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/s1", nil)
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
			t.Fatal("timed out waiting for restart to be triggered")
		case <-ticker.C:
			if restartCalled {
				return
			}
		}
	}
}

func TestBuildUserYAML_MultiStrategy_SameExchange_DedupSession(t *testing.T) {
	strategies := []StrategyEntry{
		{
			Strategy: "grid2",
			Exchange: "binance",
			Mode:     "live",
			Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
		},
		{
			Strategy: "dca",
			Exchange: "binance",
			Mode:     "live",
			Config:   rawJSON(`{"symbol":"ETHUSDT"}`),
		},
	}
	yaml, err := buildUserYAML("test-user", ModeLive, strategies, func(exchange string) bool { return true })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yaml)

	if !strings.Contains(s, "grid2:") {
		t.Error("expected grid2 strategy")
	}
	if !strings.Contains(s, "dca:") {
		t.Error("expected dca strategy")
	}

	onCount := strings.Count(s, `"on": binance`)
	if onCount != 2 {
		t.Errorf("expected 2 'on: binance' bindings, got %d", onCount)
	}

	if strings.Contains(s, "PAPER_TRADE") {
		t.Error("should NOT have PAPER_TRADE when all strategies are live")
	}
}

func TestBuildUserYAML_DifferentExchanges_SeparateSessions(t *testing.T) {
	strategies := []StrategyEntry{
		{
			Strategy: "grid2",
			Exchange: "binance",
			Mode:     "paper",
			Config:   rawJSON(`{"symbol":"BTCUSDT"}`),
		},
		{
			Strategy: "grid2",
			Exchange: "okex",
			Mode:     "paper",
			Config:   rawJSON(`{"symbol":"ETHUSDT"}`),
		},
	}
	yaml, err := buildUserYAML("test-user", ModePaper, strategies, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yaml)

	if !strings.Contains(s, "binance:") {
		t.Error("expected binance session")
	}
	if !strings.Contains(s, "okex:") {
		t.Error("expected okex session")
	}
	if !strings.Contains(s, "PAPER_TRADE:") {
		t.Error("expected PAPER_TRADE for paper mode")
	}
	if !strings.Contains(s, "publicOnly: true") {
		t.Error("expected publicOnly when no credentials")
	}
}
