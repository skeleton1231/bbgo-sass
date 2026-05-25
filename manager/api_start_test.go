package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAPI_StartUser_AlreadyRunning(t *testing.T) {
	api, _ := setupTestAPIWithMockCM(nil, true)
	r := testRouter(api)

	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for already running, got %d: %s", w.Code, w.Body.String())
	}

	var resp UserContainer
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != StatusRunning {
		t.Errorf("expected running status, got %s", resp.Status)
	}
}

func TestAPI_StartUser_AcceptedAsync(t *testing.T) {
	api, _ := setupTestAPIWithMockCM(nil, false)
	r := testRouter(api)

	start := time.Now()
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	elapsed := time.Since(start)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted for async start, got %d: %s", w.Code, w.Body.String())
	}

	if elapsed > 2*time.Second {
		t.Fatalf("StartUser should return immediately, took %v", elapsed)
	}

	var resp UserContainer
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != StatusStarting {
		t.Errorf("expected starting status, got %s", resp.Status)
	}
}

func TestAPI_StartUserAsync_NoStrategies(t *testing.T) {
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status: StatusStopped,
	}
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg}
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

func TestAPI_StartUser_BackgroundHealthCheck(t *testing.T) {
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	var mu sync.Mutex
	var pingCalls int
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		pingCalls++
		mu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}
	api.containerStart = func(_ *UserContainer) error { return nil }

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for status to become running")
		case <-ticker.C:
			uc, _ := users.Get("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			if uc.Status == StatusRunning {
				mu.Lock()
				calls := pingCalls
				mu.Unlock()
				if calls == 0 {
					t.Error("expected at least one ping call during health check")
				}
				return
			}
		}
	}
}

func TestAPI_StartUser_UserNotFound(t *testing.T) {
	api, _ := setupTestAPI(nil)
	r := testRouter(api)

	req := httptest.NewRequest("POST", "/api/users/00000000-0000-0000-0000-000000000000/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown user, got %d", w.Code)
	}
}

// setupTestAPIWithMockCM creates a test API with a mocked container manager.
// If isRunning is true, the mock reports the container as already running.
func setupTestAPIWithMockCM(bbgoHandler http.HandlerFunc, isRunning bool) (*API, *httptest.Server) {
	users := NewUserContainerManager()
	status := StatusStopped
	if isRunning {
		status = StatusRunning
	}
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     status,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	bbgoSrv := httptest.NewServer(bbgoHandler)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_ string) bool { return isRunning }
	api.containerStart = func(_ *UserContainer) error { return nil }
	if bbgoHandler != nil {
		api.newBBGoClient = func(_ string) *BBGoClient {
			return NewBBGoClient(bbgoSrv.URL)
		}
	}
	return api, bbgoSrv
}
