package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/c9s/bbgo/saas/manager/pool"
)

func TestAPI_CreateStrategy(t *testing.T) {
	api := setupStoppedTestAPI(t)
	r := testRouter(api)

	body := map[string]any{
		"name":     "My Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config": map[string]any{
			"symbol":     "BTCUSDT",
			"gridNumber": 10,
			"upperPrice": 70000,
			"lowerPrice": 50000,
			"quantity":   0.001,
		},
		"mode": "paper",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["user_id"] != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected user ID in response, got %v", resp["user_id"])
	}
	if resp["status"] != "created" {
		t.Errorf("expected status 'created', got %v", resp["status"])
	}
}

func TestAPI_CreateStrategy_MissingStrategy(t *testing.T) {
	api, _ := setupTestAPI(t, nil)
	r := testRouter(api)

	body := map[string]any{"exchange": "binance"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing strategy, got %d", w.Code)
	}
}

func TestAPI_CreateStrategy_MissingExchange(t *testing.T) {
	api, _ := setupTestAPI(t, nil)
	r := testRouter(api)

	body := map[string]any{"strategy": "grid2"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing exchange, got %d", w.Code)
	}
}

func TestAPI_CreateStrategy_CrossExchange(t *testing.T) {
	api := setupStoppedTestAPI(t)
	r := testRouter(api)

	body := map[string]any{
		"name":          "XMaker",
		"strategy":      "xmaker",
		"crossExchange": true,
		"config":        map[string]any{"symbol": "BTCUSDT"},
		"sessions": []map[string]any{
			{"name": "maker", "exchange": "binance", "envVarPrefix": "BINANCE", "futures": false},
			{"name": "hedge", "exchange": "bybit", "envVarPrefix": "BYBIT", "futures": true},
		},
		"mode": "live",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "created" {
		t.Errorf("expected status 'created', got %v", resp["status"])
	}
}

func TestAPI_CreateStrategy_CrossExchangeMissingSessions(t *testing.T) {
	api, _ := setupTestAPI(t, nil)
	r := testRouter(api)

	body := map[string]any{"strategy": "xmaker", "crossExchange": true}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing sessions, got %d", w.Code)
	}
}

func TestAPI_ListStrategies(t *testing.T) {
	bbgoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]any{
				"strategies": []map[string]any{
					{"strategyInstanceID": "s1", "strategy": "grid", "symbol": "BTCUSDT"},
				},
			})
			return
		}
		if r.URL.Path == "/api/ping" {
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	api, _ := setupTestAPI(t, bbgoHandler)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	containers, _ := resp["containers"].(map[string]any)
	liveContainer, _ := containers["live"].(map[string]any)
	strategies, _ := liveContainer["strategies"].([]any)
	if len(strategies) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(strategies))
	}
	firstStrategy, _ := strategies[0].(map[string]any)
	if firstStrategy["strategy"] != "grid" {
		t.Errorf("expected grid strategy, got %v", firstStrategy["strategy"])
	}
}

func TestAPI_ListStrategies_UserNotFound(t *testing.T) {
	api, _ := setupTestAPI(t, nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/00000000-0000-0000-0000-000000000000/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	containers, _ := resp["containers"].(map[string]any)
	if len(containers) != 0 {
		t.Errorf("expected empty containers for unknown user, got %v", containers)
	}
}

func TestAPI_DeleteStrategy(t *testing.T) {
	api, bbgoSrv := setupTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]any{
				"strategies": []map[string]any{
					{"strategyInstanceID": "s1", "strategy": "grid", "symbol": "BTCUSDT", "session": "binance", "on": []any{"binance"}},
				},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)

	req := httptest.NewRequest("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	strats, _ := api.strategies.ListStrategies("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive)
	if len(strats) != 0 {
		t.Errorf("expected 0 strategies after delete, got %d", len(strats))
	}
}

func TestAPI_DeleteStrategy_NotFound(t *testing.T) {
	api, bbgoSrv := setupTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]any{
				"strategies": []map[string]any{
					{"strategyInstanceID": "s1", "strategy": "grid", "symbol": "BTCUSDT", "session": "binance", "on": []any{"binance"}},
				},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)

	req := httptest.NewRequest("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPI_StopUser(t *testing.T) {
	api, _ := setupTestAPI(t, nil)
	r := testRouter(api)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "stopped" {
		t.Errorf("expected stopped status, got %s", resp["status"])
	}
}

func TestAPI_StartUser_NoStrategies(t *testing.T) {
	store, _ := newTestStore(t)
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return false }
	r := testRouter(api)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for no strategies, got %d", w.Code)
	}
}

func TestAPI_UserStatus(t *testing.T) {
	api, _ := setupTestAPI(t, nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	containers, _ := resp["containers"].(map[string]any)
	liveContainer, _ := containers["live"].(map[string]any)
	if liveContainer["user_id"] != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected user ID, got %v", liveContainer["user_id"])
	}
	if liveContainer["status"] != StatusRunning {
		t.Errorf("expected running status, got %v", liveContainer["status"])
	}
}

func TestAPI_Health(t *testing.T) {
	api, _ := setupTestAPI(t, nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected ok, got %v", resp["status"])
	}
	if resp["users"] != float64(1) {
		t.Errorf("expected 1 user, got %v", resp["users"])
	}
	if resp["running"] != float64(1) {
		t.Errorf("expected 1 running, got %v", resp["running"])
	}
}

func TestAPI_BBGoStrategies(t *testing.T) {
	api, bbgoSrv := setupTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]any{
				"strategies": []map[string]any{
					{"strategy": "grid2", "symbol": "BTCUSDT"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	strategies := resp["strategies"].([]any)
	if len(strategies) != 1 {
		t.Errorf("expected 1 strategy, got %d", len(strategies))
	}
}

func setupStoppedTestAPI(t *testing.T) *API {
	t.Helper()
	store, _ := newTestStore(t)
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      t.TempDir(),
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return false }
	return api
}

func TestContainerManager_RecoverUsers_Concurrent(t *testing.T) {
	cfg := &Config{ManagerToken: "test-token", DataDir: t.TempDir()}
	p := pool.New(5)
	defer p.Release()
	cm := &ContainerManager{cfg: cfg, pool: p}

	um1 := UserMode{UserID: "user-1", Mode: ModeLive}
	um2 := UserMode{UserID: "user-2", Mode: ModeLive}

	results := cm.RecoverUsers([]UserMode{um1, um2})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Both should return a result with a valid status
	for _, r := range results {
		if r.UserID == "" {
			t.Error("expected non-empty user ID in result")
		}
		if r.Status == "" {
			t.Error("expected non-empty status in result")
		}
	}
}

func TestAPI_CreateCredential_InvalidExchange(t *testing.T) {
	api := setupStoppedTestAPI(t)
	r := testRouter(api)

	body := map[string]any{
		"exchange":   "fakeexchange",
		"api_key":    "key123",
		"api_secret": "secret456",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/credentials", bytes.NewReader(b))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported exchange, got %d: %s", w.Code, w.Body.String())
	}
}
