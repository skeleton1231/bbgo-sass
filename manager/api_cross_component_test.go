package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestIssueWSTicket_RequiresAuth verifies the ticket endpoint requires X-User-Id.
func TestIssueWSTicket_RequiresAuth(t *testing.T) {
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, NewUserContainerManager(), cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	defer api.Close()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/ws/ticket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without X-User-Id, got %d", w.Code)
	}
}

// TestIssueWSTicket_ReturnsTicket verifies a valid auth header returns a ticket.
func TestIssueWSTicket_ReturnsTicket(t *testing.T) {
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, NewUserContainerManager(), cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	defer api.Close()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/ws/ticket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ticket"] == "" {
		t.Error("expected non-empty ticket")
	}
}

// TestStartUser_NoStrategies_Returns400 verifies you can't start without strategies.
func TestStartUser_NoStrategies_Returns400(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for start with no strategies, got %d: %s", w.Code, w.Body.String())
	}
}

// TestStopUser_SetsStatusStopped verifies the stop endpoint stops the container
// and updates status.
func TestStopUser_SetsStatusStopped(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	var stopCalled bool
	api.containerStop = func(_, _ string) {
		stopCalled = true
	}

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !stopCalled {
		t.Error("expected containerStop to be called")
	}
	uc, _ := users.Get("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive)
	if uc.Status != StatusStopped {
		t.Errorf("expected status stopped, got %s", uc.Status)
	}
}

// TestStartUser_AlreadyRunning_Returns200 verifies that starting an already-running
// container returns 200 without spawning a new start goroutine.
func TestStartUser_AlreadyRunning_Returns200(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid", Config: rawJSON(`{}`),
	})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, StatusRunning)

	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }

	var startCalls int
	api.containerStart = func(_ *UserContainer) error {
		startCalls++
		return nil
	}

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for already running, got %d: %s", w.Code, w.Body.String())
	}
	time.Sleep(100 * time.Millisecond)
	if startCalls != 0 {
		t.Errorf("expected no containerStart call for already-running container, got %d", startCalls)
	}
}

// TestCreateStrategy_StartingContainer_NoExtraStart verifies that creating a
// strategy while the container is in StatusStarting adds the strategy but
// does NOT launch an additional start goroutine.
func TestCreateStrategy_StartingContainer_NoExtraStart(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	users := NewUserContainerManager()
	users.AddStrategy(userID, ModePaper, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid", Config: rawJSON(`{}`), Mode: "paper",
	})
	users.UpdateStatus(userID, ModePaper, StatusStarting)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	var startCalls int
	api.containerStart = func(_ *UserContainer) error {
		startCalls++
		return nil
	}

	r := testRouter(api)
	body := `{"name":"Grid2","exchange":"binance","strategy":"grid2","config":{"symbol":"ETHUSDT"},"mode":"paper"}`
	req := httptest.NewRequest("POST", "/api/users/"+userID+"/strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	time.Sleep(200 * time.Millisecond)
	if startCalls != 0 {
		t.Errorf("expected no containerStart when status is starting, got %d calls", startCalls)
	}

	uc, _ := users.Get(userID, ModePaper)
	if len(uc.Strategies) != 2 {
		t.Fatalf("expected 2 strategies, got %d", len(uc.Strategies))
	}
	if uc.Status != StatusStarting {
		t.Errorf("expected status starting, got %s", uc.Status)
	}
}

// TestUserStatus_UnknownUser_ReturnsStopped verifies the status endpoint returns
// stopped for a user with no container.
func TestUserStatus_UnknownUser_ReturnsStopped(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, NewUserContainerManager(), cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	containers, ok := resp["containers"].(map[string]interface{})
	if !ok || len(containers) != 0 {
		t.Errorf("expected empty containers for unknown user, got %v", resp)
	}
}

// TestListStrategies_UnknownUser_ReturnsEmpty verifies list strategies returns
// empty for a user with no container.
func TestListStrategies_UnknownUser_ReturnsEmpty(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, NewUserContainerManager(), cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	containers, ok := resp["containers"].(map[string]interface{})
	if !ok || len(containers) != 0 {
		t.Errorf("expected empty containers for unknown user, got %v", resp)
	}
}

// TestCreateCredential_UnsupportedExchange verifies credential creation rejects
// unknown exchanges.
func TestCreateCredential_UnsupportedExchange(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, NewUserContainerManager(), cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	body := `{"exchange":"unknown_ex","api_key":"k","api_secret":"s"}`
	req := httptest.NewRequest("POST", "/api/credentials", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported exchange, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateCredential_MissingFields verifies credential creation requires all fields.
func TestCreateCredential_MissingFields(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, NewUserContainerManager(), cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	tests := []struct {
		name string
		body string
	}{
		{"missing exchange", `{"api_key":"k","api_secret":"s"}`},
		{"missing api_key", `{"exchange":"binance","api_secret":"s"}`},
		{"missing api_secret", `{"exchange":"binance","api_key":"k"}`},
		{"empty body", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			req := httptest.NewRequest("POST", "/api/credentials", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestCreateStrategy_CrossExchange_RequiresSessions verifies cross-exchange
// strategy creation requires session configs.
func TestCreateStrategy_CrossExchange_RequiresSessions(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	body := `{"name":"XMaker","strategy":"xmaker","crossExchange":true,"config":{},"mode":"paper"}`
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for cross-exchange without sessions, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteStrategy_NotFound verifies deleting a non-existent strategy returns 404.
func TestDeleteStrategy_NotFound(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing strategy, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHealthEndpoint_ReturnsCounts verifies the health endpoint returns user counts.
func TestHealthEndpoint_ReturnsCounts(t *testing.T) {
	cfg := &Config{ManagerToken: "test-token"}
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1"}},
	}
	users.users["11111111-2222-3333-4444-555555555555:"+ModeLive] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "11111111-2222-3333-4444-555555555555",
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s2"}},
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["users"].(float64) != 2 {
		t.Errorf("expected 2 users, got %v", resp["users"])
	}
	if resp["running"].(float64) != 1 {
		t.Errorf("expected 1 running, got %v", resp["running"])
	}
}

// TestBacktestSync_TooManySymbols verifies symbol count is capped at 10.
func TestBacktestSync_TooManySymbols(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, NewUserContainerManager(), cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	symbols := make([]string, 11)
	for i := range symbols {
		symbols[i] = "SYM" + string(rune('A'+i)) + "USDT"
	}
	body, _ := json.Marshal(map[string]interface{}{
		"exchange":   "binance",
		"symbols":    symbols,
		"start_time": "2024-01-01",
		"end_time":   "2025-12-31",
	})
	req := httptest.NewRequest("POST", "/api/backtest/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for >10 symbols, got %d: %s", w.Code, w.Body.String())
	}
}
