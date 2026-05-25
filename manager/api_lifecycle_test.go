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

	body := map[string]interface{}{
		"name":     "My Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config": map[string]interface{}{
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

	var resp UserContainer
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.UserID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected user ID in response, got %s", resp.UserID)
	}
	if len(resp.Strategies) != 2 {
		t.Errorf("expected 2 strategies (1 existing + 1 new), got %d", len(resp.Strategies))
	}
	found := false
	for _, s := range resp.Strategies {
		if s.Strategy == "grid2" && s.Name == "My Grid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find new grid2 strategy in response")
	}
}

func TestAPI_CreateStrategy_MissingStrategy(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	body := map[string]interface{}{"exchange": "binance"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing strategy, got %d", w.Code)
	}
}

func TestAPI_CreateStrategy_MissingExchange(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	body := map[string]interface{}{"strategy": "grid2"}
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

	body := map[string]interface{}{
		"name":          "XMaker",
		"strategy":      "xmaker",
		"crossExchange": true,
		"config":        map[string]interface{}{"symbol": "BTCUSDT"},
		"sessions": []map[string]interface{}{
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

	var resp UserContainer
	json.NewDecoder(w.Body).Decode(&resp)
	found := false
	for _, s := range resp.Strategies {
		if s.Strategy == "xmaker" && s.CrossExchange {
			found = true
			if len(s.Sessions) != 2 {
				t.Errorf("expected 2 sessions, got %d", len(s.Sessions))
			}
			break
		}
	}
	if !found {
		t.Error("expected to find cross-exchange xmaker strategy")
	}
}

func TestAPI_CreateStrategy_CrossExchangeMissingSessions(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	body := map[string]interface{}{"strategy": "xmaker", "crossExchange": true}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing sessions, got %d", w.Code)
	}
}

func TestAPI_ListStrategies(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp UserContainer
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Strategies) != 1 {
		t.Errorf("expected 1 strategy, got %d", len(resp.Strategies))
	}
	if resp.Strategies[0].Strategy != "grid" {
		t.Errorf("expected grid strategy, got %s", resp.Strategies[0].Strategy)
	}
}

func TestAPI_ListStrategies_UserNotFound(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/00000000-0000-0000-0000-000000000000/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != StatusStopped {
		t.Errorf("expected stopped status for unknown user, got %v", resp["status"])
	}
}

func TestAPI_DeleteStrategy(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	req := httptest.NewRequest("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	uc, _ := api.users.Get("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if len(uc.Strategies) != 0 {
		t.Errorf("expected 0 strategies after delete, got %d", len(uc.Strategies))
	}
}

func TestAPI_DeleteStrategy_NotFound(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	req := httptest.NewRequest("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPI_StopUser(t *testing.T) {
	api, _ := setupTestAPI(nil)
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

	uc, _ := api.users.Get("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if uc.Status != StatusStopped {
		t.Errorf("expected user status stopped, got %s", uc.Status)
	}
}

func TestAPI_StartUser_NoStrategies(t *testing.T) {
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status: StatusStopped,
	}
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	r := testRouter(api)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for no strategies, got %d", w.Code)
	}
}

func TestAPI_UserStatus(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp UserContainer
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.UserID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected user ID, got %s", resp.UserID)
	}
	if resp.Status != StatusRunning {
		t.Errorf("expected running status, got %s", resp.Status)
	}
}

func TestAPI_Health(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
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
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"strategies": []map[string]interface{}{
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

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	strategies := resp["strategies"].([]interface{})
	if len(strategies) != 1 {
		t.Errorf("expected 1 strategy, got %d", len(strategies))
	}
}

func setupStoppedTestAPI(t *testing.T) *API {
	t.Helper()
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      t.TempDir(),
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	return api
}

func TestContainerManager_RecoverUsers_Concurrent(t *testing.T) {
	users := NewUserContainerManager()
	uc1 := &UserContainer{UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Status: StatusStopped, Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}}}
	uc2 := &UserContainer{UserID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", Status: StatusStopped, Strategies: []StrategyEntry{{ID: "s2", Exchange: "binance", Strategy: "grid"}}}
	users.Restore([]*UserContainer{uc1, uc2})

	cfg := &Config{ManagerToken: "test-token", DataDir: t.TempDir()}
	p := pool.New(5)
	defer p.Release()
	cm := &ContainerManager{cfg: cfg, pool: p}

	results := cm.RecoverUsers([]*UserContainer{uc1, uc2})

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

	// Verify original UserContainer structs were NOT mutated
	orig1, _ := users.Get(uc1.UserID)
	if orig1.Status != StatusStopped {
		t.Errorf("original user status should not be mutated by RecoverUsers, got %s", orig1.Status)
	}
}

func TestAPI_CreateCredential_InvalidExchange(t *testing.T) {
	api := setupStoppedTestAPI(t)
	r := testRouter(api)

	body := map[string]interface{}{
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
