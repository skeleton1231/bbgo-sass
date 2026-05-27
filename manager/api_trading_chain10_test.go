package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// chain10Setup creates a full API with mock bbgo server, mock container ops,
// and a chi router with auth middleware pre-configured.
func chain10Setup(t *testing.T) (*API, *chi.Mux, *httptest.Server) {
	t.Helper()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/sessions":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sessions": []map[string]interface{}{{"name": "binance", "exchange": "binance"}},
			})
		case "/api/sessions/binance":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"session": map[string]interface{}{"name": "binance", "exchange": "binance"},
			})
		case "/api/sessions/binance/trades":
			json.NewEncoder(w).Encode(map[string]interface{}{"trades": []map[string]interface{}{}})
		case "/api/sessions/binance/open-orders":
			json.NewEncoder(w).Encode(map[string]interface{}{"orders": []map[string]interface{}{}})
		case "/api/sessions/binance/account":
			json.NewEncoder(w).Encode(map[string]interface{}{"account": map[string]interface{}{}})
		case "/api/sessions/binance/account/balances":
			json.NewEncoder(w).Encode(map[string]interface{}{"balances": map[string]interface{}{}})
		case "/api/sessions/binance/symbols":
			json.NewEncoder(w).Encode(map[string]interface{}{"symbols": []string{"BTCUSDT"}})
		case "/api/assets":
			json.NewEncoder(w).Encode(map[string]interface{}{"assets": map[string]interface{}{}})
		case "/api/strategies/single":
			json.NewEncoder(w).Encode(map[string]interface{}{"strategies": []map[string]interface{}{}})
		case "/api/trades":
			json.NewEncoder(w).Encode(map[string]interface{}{"trades": []map[string]interface{}{}})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(map[string]interface{}{"orders": []map[string]interface{}{}})
		case "/api/trading-volume":
			json.NewEncoder(w).Encode(map[string]interface{}{"tradingVolumes": []map[string]interface{}{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	users := NewUserContainerManager()
	cfg := &Config{
		SupabaseURL:        "http://localhost:54321",
		SupabaseKey:        "test",
		ManagerToken:       "test-token",
		MarketDataAddr:     "http://market:50051",
		MarketDataRESTAddr: "localhost:9090",
	}
	cm := &ContainerManager{cfg: cfg}
	enc, _ := NewEncryptor(testEncryptionKey)

	creds := NewCredentialStore(t.TempDir(), enc)
	api := NewAPI(cfg, users, cm, &BotProxy{}, creds, enc, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(baseURL)
	}
	api.containerStart = func(_ *UserContainer) error { return nil }
	api.containerStop = func(_, _ string) {}
	api.containerRunning = func(_, _ string) bool { return true }

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000100")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	return api, r, bbgoSrv
}

const chain10UID = "aaaaaaaa-bbbb-cccc-dddd-eeeeee000100"

// --- Health endpoint ---

func TestTradingChain_Health(t *testing.T) {
	api, r, _ := chain10Setup(t)
	_ = api

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("health = %d, want 200", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["running"].(float64) != 1 {
		t.Errorf("running = %v, want 1", resp["running"])
	}
}

// --- ListStrategies ---

func TestTradingChain_ListStrategies_UserNotFound(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("GET", "/api/users/"+chain10UID+"/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	containers, _ := resp["containers"].(map[string]interface{})
	if len(containers) != 0 {
		t.Errorf("unknown user containers = %v, want empty", containers)
	}
}

func TestTradingChain_ListStrategies_WithStrategies(t *testing.T) {
	api, r, _ := chain10Setup(t)

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)

	req := httptest.NewRequest("GET", "/api/users/"+chain10UID+"/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	containers, _ := resp["containers"].(map[string]interface{})
	liveContainer, _ := containers["live"].(map[string]interface{})
	strats, _ := liveContainer["strategies"].([]interface{})
	if len(strats) != 1 {
		t.Errorf("strategies count = %d, want 1", len(strats))
	}
}

// --- DeleteStrategy ---

func TestTradingChain_DeleteStrategy_NotFound(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("DELETE", "/api/users/"+chain10UID+"/strategies/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("delete nonexistent = %d, want 404", w.Code)
	}
}

func TestTradingChain_DeleteStrategy_LastStrategy_StopsContainer(t *testing.T) {
	api, r, _ := chain10Setup(t)
	stopped := false
	api.containerStop = func(_, _ string) { stopped = true }

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})

	req := httptest.NewRequest("DELETE", "/api/users/"+chain10UID+"/strategies/s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if !stopped {
		t.Error("expected container to be stopped when last strategy deleted")
	}
}

func TestTradingChain_DeleteStrategy_RemainingStrategies_Restarts(t *testing.T) {
	api, r, _ := chain10Setup(t)
	_ = false
	api.containerStart = func(_ *UserContainer) error { return nil }

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s2", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"ETHUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)

	req := httptest.NewRequest("DELETE", "/api/users/"+chain10UID+"/strategies/s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	// Restart is async (go api.startUserContainer), so check status changed
	uc2, _ := api.users.Get(chain10UID, ModeLive)
	if uc2.Status != StatusStarting {
		t.Errorf("status = %s, want starting (restart triggered)", uc2.Status)
	}
}

// --- UserStatus ---

func TestTradingChain_UserStatus_UnknownUser(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("GET", "/api/users/"+chain10UID+"/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	containers, _ := resp["containers"].(map[string]interface{})
	if len(containers) != 0 {
		t.Errorf("unknown user should have empty containers, got %v", containers)
	}
}

func TestTradingChain_UserStatus_RunningUser(t *testing.T) {
	api, r, _ := chain10Setup(t)

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)

	req := httptest.NewRequest("GET", "/api/users/"+chain10UID+"/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	containers, _ := resp["containers"].(map[string]interface{})
	liveContainer, _ := containers["live"].(map[string]interface{})
	if liveContainer == nil {
		t.Fatal("expected live container in response")
	}
	if liveContainer["status"] != StatusRunning {
		t.Errorf("status = %v, want running", liveContainer["status"])
	}
}

// --- ProxyToBot ---

func TestTradingChain_ProxyToBot_UserNotFound(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("GET", "/api/bbgo/"+chain10UID+"/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("proxy for unknown user = %d, want 404", w.Code)
	}
}

func TestTradingChain_ProxyToBot_UserMismatch(t *testing.T) {
	_, r, _ := chain10Setup(t)

	// Router middleware sets X-User-Id to chain10UID, but URL uses different ID
	req := httptest.NewRequest("GET", "/api/bbgo/aaaaaaaa-bbbb-cccc-dddd-eeeeee000199/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("proxy user mismatch = %d, want 403", w.Code)
	}
}

// --- BacktestSyncStatus ---

func TestTradingChain_BacktestSyncStatus_NoFile(t *testing.T) {
	api, r, _ := chain10Setup(t)
	api.container.cfg.DataDir = t.TempDir()

	req := httptest.NewRequest("GET", "/api/backtest/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["available"].(bool) {
		t.Error("should report unavailable when file missing")
	}
}

func TestTradingChain_BacktestSyncStatus_WithFile(t *testing.T) {
	api, r, _ := chain10Setup(t)
	dir := t.TempDir()
	api.container.cfg.DataDir = dir

	os.MkdirAll(dir+"/backtest-shared", 0755)
	f, _ := os.Create(filepath.Join(dir, "backtest-shared", "backtest.db"))
	f.WriteString("sqlite")
	f.Close()

	req := httptest.NewRequest("GET", "/api/backtest/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp["available"].(bool) {
		t.Error("should report available when file exists")
	}
	if resp["size"].(float64) < 1 {
		t.Error("should report non-zero size")
	}
}

// --- RunBacktest (legacy) ---

func TestTradingChain_RunBacktest_InvalidBody(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("POST", "/api/backtest", strings.NewReader("not-json"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("bad body = %d, want 400", w.Code)
	}
}

func TestTradingChain_RunBacktest_Success(t *testing.T) {
	api, r, _ := chain10Setup(t)
	api.container.cfg.DataDir = t.TempDir()

	api.container.runBacktestFn = func(userID string, yamlContent []byte) ([]byte, error) {
		return []byte("BACKTEST RESULT: profit=100"), nil
	}

	body, _ := json.Marshal(map[string]interface{}{
		"strategy":   "grid2",
		"config":     map[string]interface{}{"symbol": "BTCUSDT", "quantity": 0.001},
		"exchange":   "binance",
		"start_time": "2024-01-01",
		"end_time":   "2024-12-31",
	})
	req := httptest.NewRequest("POST", "/api/backtest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("backtest = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp["output"].(string), "BACKTEST RESULT") {
		t.Error("expected backtest output in response")
	}
}

// --- SyncBacktestData ---

func TestTradingChain_SyncBacktestData_Defaults(t *testing.T) {
	api, r, _ := chain10Setup(t)
	api.container.cfg.DataDir = t.TempDir()

	api.container.syncBacktestFn = func(exchange, symbol, start, end string) (string, error) {
		return "synced " + symbol, nil
	}

	body, _ := json.Marshal(map[string]interface{}{
		"exchange": "binance",
	})
	req := httptest.NewRequest("POST", "/api/backtest/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("sync = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	results := resp["synced"].([]interface{})
	// Default symbols: BTCUSDT, ETHUSDT
	if len(results) != 2 {
		t.Errorf("synced symbols = %d, want 2", len(results))
	}
}

func TestTradingChain_SyncBacktestData_TooManySymbols(t *testing.T) {
	_, r, _ := chain10Setup(t)

	symbols := make([]string, 11)
	for i := range symbols {
		symbols[i] = "SYM" + strings.Repeat("A", 5)
	}
	body, _ := json.Marshal(map[string]interface{}{"symbols": symbols})
	req := httptest.NewRequest("POST", "/api/backtest/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("too many symbols = %d, want 400", w.Code)
	}
}

// --- CreateCredential ---

func TestTradingChain_CreateCredential_Success(t *testing.T) {
	_, r, _ := chain10Setup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"exchange":   "binance",
		"api_key":    "testkey123",
		"api_secret": "testsecret456",
	})
	req := httptest.NewRequest("POST", "/api/credentials", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create credential = %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["exchange"] != "binance" {
		t.Errorf("exchange = %v, want binance", resp["exchange"])
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected credential ID")
	}
}

func TestTradingChain_CreateCredential_MissingFields(t *testing.T) {
	_, r, _ := chain10Setup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"exchange": "binance",
	})
	req := httptest.NewRequest("POST", "/api/credentials", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing fields = %d, want 400", w.Code)
	}
}

func TestTradingChain_CreateCredential_UnsupportedExchange(t *testing.T) {
	_, r, _ := chain10Setup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"exchange":   "unknown_exchange",
		"api_key":    "k",
		"api_secret": "s",
	})
	req := httptest.NewRequest("POST", "/api/credentials", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unsupported exchange = %d, want 400", w.Code)
	}
}

// --- ListCredentials ---

func TestTradingChain_ListCredentials(t *testing.T) {
	api, r, _ := chain10Setup(t)

	// Create a credential first
	api.creds.Upsert(ExchangeCredential{
		ID:                 "cred-test1",
		UserID:             chain10UID,
		Exchange:           "binance",
		APIKeyEncrypted:    "enc",
		APISecretEncrypted: "enc",
	})

	req := httptest.NewRequest("GET", "/api/credentials", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list = %d", w.Code)
	}
	var resp []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("credentials count = %d, want 1", len(resp))
	}
	if resp[0]["exchange"] != "binance" {
		t.Errorf("exchange = %v, want binance", resp[0]["exchange"])
	}
}

// --- DeleteCredential ---

func TestTradingChain_DeleteCredential_NotFound(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("DELETE", "/api/credentials/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("delete nonexistent = %d, want 404", w.Code)
	}
}

// --- BBGo proxy endpoints (ping, sessions, trades, etc.) ---

func TestTradingChain_BBGoPing_UserNotFound(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("GET", "/api/users/"+chain10UID+"/bbgo/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// User has no container → 404 from userFromURL
	if w.Code != http.StatusNotFound {
		t.Errorf("ping for missing container = %d, want 404", w.Code)
	}
}

func TestTradingChain_BBGoProxy_Endpoints(t *testing.T) {
	api, r, bbgoSrv := chain10Setup(t)
	_ = api

	// Add a running user so bbgoClientForUser works
	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)
	api.container.apiURLFn = func(userID, mode string) string { return bbgoSrv.URL }

	endpoints := []struct {
		path string
		key  string
	}{
		{"/api/users/" + chain10UID + "/bbgo/ping", ""},
		{"/api/users/" + chain10UID + "/bbgo/sessions", "sessions"},
		{"/api/users/" + chain10UID + "/bbgo/session/binance", "session"},
		{"/api/users/" + chain10UID + "/bbgo/session/binance/trades", "trades"},
		{"/api/users/" + chain10UID + "/bbgo/session/binance/open-orders", "orders"},
		{"/api/users/" + chain10UID + "/bbgo/session/binance/account", "account"},
		{"/api/users/" + chain10UID + "/bbgo/session/binance/balances", "balances"},
		{"/api/users/" + chain10UID + "/bbgo/session/binance/symbols", "symbols"},
		{"/api/users/" + chain10UID + "/bbgo/assets", "assets"},
		{"/api/users/" + chain10UID + "/bbgo/strategies", "strategies"},
		{"/api/users/" + chain10UID + "/bbgo/trades", "trades"},
		{"/api/users/" + chain10UID + "/bbgo/orders/closed", "orders"},
		{"/api/users/" + chain10UID + "/bbgo/trading-volume", "tradingVolumes"},
	}

	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", ep.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("%s = %d, want 200; body: %s", ep.path, w.Code, w.Body.String())
			}
		})
	}
}

// --- StopUser ---

func TestTradingChain_StopUser_Success(t *testing.T) {
	api, r, _ := chain10Setup(t)
	stopped := false
	api.containerStop = func(_, _ string) { stopped = true }

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)

	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("stop = %d, want 200", w.Code)
	}
	if !stopped {
		t.Error("expected container stop to be called")
	}
	uc2, _ := api.users.Get(chain10UID, ModeLive)
	if uc2.Status != StatusStopped {
		t.Errorf("status = %s, want stopped", uc2.Status)
	}
}

// --- CreateStrategy validation chain ---

func TestTradingChain_CreateStrategy_MissingStrategy(t *testing.T) {
	_, r, _ := chain10Setup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"exchange": "binance",
		"name":     "my-bot",
		"mode":     "live",
	})
	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/strategies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing strategy = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestTradingChain_CreateStrategy_LiveNoCreds_Rejected(t *testing.T) {
	api, r, _ := chain10Setup(t)

	// Add running user with no credentials
	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s0", Exchange: "binance", Strategy: "grid2", Mode: "paper", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)

	body, _ := json.Marshal(map[string]interface{}{
		"strategy": "grid2",
		"exchange": "binance",
		"name":     "live-bot",
		"mode":     "live",
		"config":   map[string]interface{}{"symbol": "BTCUSDT", "quantity": 0.001},
	})
	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/strategies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("live without creds = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestTradingChain_CreateStrategy_LiveWithCreds_Accepted(t *testing.T) {
	api, r, _ := chain10Setup(t)

	// Store a credential with properly encrypted values
	keyEnc, _ := api.encryptor.Encrypt("testkey")
	secretEnc, _ := api.encryptor.Encrypt("testsecret")
	api.creds.Upsert(ExchangeCredential{
		ID:                 "cred-1",
		UserID:             chain10UID,
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"strategy": "grid2",
		"exchange": "binance",
		"name":     "live-bot",
		"mode":     "live",
		"config":   map[string]interface{}{"symbol": "BTCUSDT", "quantity": 0.001},
	})
	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/strategies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("live with creds = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestTradingChain_CreateStrategy_PaperNoCreds_Accepted(t *testing.T) {
	_, r, _ := chain10Setup(t)

	body, _ := json.Marshal(map[string]interface{}{
		"strategy": "grid2",
		"exchange": "binance",
		"name":     "paper-bot",
		"mode":     "paper",
		"config":   map[string]interface{}{"symbol": "BTCUSDT", "quantity": 0.001},
	})
	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/strategies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("paper no creds = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestTradingChain_CreateStrategy_LiveOnlyPaper_Rejected(t *testing.T) {
	_, r, _ := chain10Setup(t)

	// sentinel is liveOnly — paper mode should be rejected
	body, _ := json.Marshal(map[string]interface{}{
		"strategy": "sentinel",
		"exchange": "binance",
		"name":     "sentinel-bot",
		"mode":     "paper",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
	})
	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/strategies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("liveOnly paper = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestTradingChain_CreateStrategy_ModeConflict(t *testing.T) {
	api, r, _ := chain10Setup(t)

	// Existing live strategy
	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})

	keyEnc, _ := api.encryptor.Encrypt("testkey")
	secretEnc, _ := api.encryptor.Encrypt("testsecret")
	api.creds.Upsert(ExchangeCredential{
		ID:                 "cred-1",
		UserID:             chain10UID,
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
	})

	// Add paper strategy — goes to separate container, allowed
	body, _ := json.Marshal(map[string]interface{}{
		"strategy": "grid2",
		"exchange": "binance",
		"name":     "paper-bot",
		"mode":     "paper",
		"config":   map[string]interface{}{"symbol": "ETHUSDT"},
	})
	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/strategies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("mixed mode = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

// --- Full chain: credential → strategy → YAML → env ---

func TestTradingChain_FullLiveChain_YAMLAndEnv(t *testing.T) {
	api, _, _ := chain10Setup(t)

	// Step 1: Store credential
	api.creds.Upsert(ExchangeCredential{
		ID:                 "cred-live1",
		UserID:             chain10UID,
		Exchange:           "binance",
		APIKeyEncrypted:    "encryptedKey",
		APISecretEncrypted: "encryptedSecret",
	})

	// Step 2: Add live strategy
	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID:       "strat-live",
		Exchange: "binance",
		Strategy: "grid2",
		Mode:     "live",
		Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
	})

	// Step 3: Generate YAML
	uc, _ := api.users.Get(chain10UID, ModeLive)
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool {
		creds, _ := api.creds.List(chain10UID)
		for _, c := range creds {
			if c.Exchange == exchange {
				return true
			}
		}
		return false
	})
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("live YAML should not contain PAPER_TRADE")
	}
	if !strings.Contains(yaml, "grid2") {
		t.Error("YAML should contain strategy grid2")
	}
	if !strings.Contains(yaml, "binance") {
		t.Error("YAML should contain exchange binance")
	}

	// Step 4: Check env args
	args := api.container.envArgs(uc)
	for _, arg := range args {
		if strings.HasPrefix(arg, "PAPER_TRADE=") {
			t.Errorf("live mode should not set PAPER_TRADE env, got: %s", arg)
		}
	}
	foundDB := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "DB_DRIVER=") {
			foundDB = true
		}
	}
	if !foundDB {
		t.Error("env should include DB_DRIVER")
	}
}

func TestTradingChain_FullPaperChain_YAMLAndEnv(t *testing.T) {
	api, _, _ := chain10Setup(t)

	// No credentials stored — paper container
	api.users.AddStrategy(chain10UID, ModePaper, StrategyEntry{
		ID:       "strat-paper",
		Exchange: "binance",
		Strategy: "grid2",
		Mode:     "paper",
		Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
	})

	uc, _ := api.users.Get(chain10UID, ModePaper)

	// Paper YAML with no creds
	yamlBytes, err := buildUserYAML(uc, func(_ string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("paper YAML should contain PAPER_TRADE")
	}

	// Paper env args
	args := api.container.envArgs(uc)
	foundPaper := false
	for _, arg := range args {
		if arg == "PAPER_TRADE=1" {
			foundPaper = true
		}
	}
	if !foundPaper {
		t.Error("paper mode env should include PAPER_TRADE=1")
	}
}

// --- refreshContainerStatus ---

func TestTradingChain_RefreshStatus_RunningContainerDied(t *testing.T) {
	api, _, _ := chain10Setup(t)
	api.container.checkRunningFn = func(_, _ string) (bool, error) { return false, nil }

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)
	uc, _ := api.users.Get(chain10UID, ModeLive)

	api.refreshContainerStatus(uc)

	uc, _ = api.users.Get(chain10UID, ModeLive)
	if uc.Status != StatusStopped {
		t.Errorf("status after container died = %s, want stopped", uc.Status)
	}
}

func TestTradingChain_RefreshStatus_StoppedContainer_NoChange(t *testing.T) {
	api, _, _ := chain10Setup(t)
	api.containerRunning = func(_, _ string) bool { return true }

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "paper", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusStopped)
	uc, _ := api.users.Get(chain10UID, ModeLive)

	api.refreshContainerStatus(uc)

	uc, _ = api.users.Get(chain10UID, ModeLive)
	if uc.Status != StatusStopped {
		t.Errorf("stopped container status changed to %s", uc.Status)
	}
}

// --- Backtest job concurrent limit ---

func TestTradingChain_SyncBacktestData_ConcurrencyLimit(t *testing.T) {
	api, r, _ := chain10Setup(t)

	// Fill the semaphore (cap 2)
	api.btSyncSem <- struct{}{}
	api.btSyncSem <- struct{}{}
	defer func() { <-api.btSyncSem; <-api.btSyncSem }()

	body, _ := json.Marshal(map[string]interface{}{
		"exchange": "binance",
		"symbols":  []string{"BTCUSDT"},
	})
	req := httptest.NewRequest("POST", "/api/backtest/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("concurrent sync = %d, want 429", w.Code)
	}
}

// --- WS ticket store ---

func TestTradingChain_WSTicket_IssueRedeem(t *testing.T) {
	store := NewWSTicketStore()

	ticket := store.Issue(chain10UID)
	if ticket == "" {
		t.Fatal("expected non-empty ticket")
	}

	if _, ok := store.Redeem(ticket); !ok {
		t.Error("first redeem should succeed")
	}
	if _, ok := store.Redeem(ticket); ok {
		t.Error("second redeem of same ticket should fail")
	}
}

func TestTradingChain_WSTicket_SingleUse(t *testing.T) {
	store := NewWSTicketStore()

	ticket := store.Issue(chain10UID)

	if _, ok := store.Redeem(ticket); !ok {
		t.Error("ticket should be valid")
	}
}

// --- Notification config ---

func TestTradingChain_Notification_CreateListDelete(t *testing.T) {
	api, r, _ := chain10Setup(t)
	api.notifier = NewNotifier(t.TempDir(), api.encryptor)

	// Create — use slack so we don't need encryption for token
	body, _ := json.Marshal(map[string]interface{}{
		"type":        "slack",
		"webhook_url": "https://hooks.slack.com/test",
		"rules":       map[string]interface{}{"trade_events": true},
	})
	req := httptest.NewRequest("POST", "/api/notifications/config", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create notification = %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	notifID := createResp["id"].(string)

	// List
	req = httptest.NewRequest("GET", "/api/notifications/config", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list notifications = %d", w.Code)
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/notifications/config/"+notifID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("delete notification = %d, want 200", w.Code)
	}
}

// --- Auth edge cases ---

func TestTradingChain_Auth_MissingToken(t *testing.T) {
	api, _, _ := chain10Setup(t)

	// Create a router with auth middleware but DON'T set X-Manager-Token in request
	rAuth := chi.NewRouter()
	rAuth.Use(SharedSecretAuth("test-token"))
	api.RegisterRoutes(rAuth)

	// Health is exempt from auth
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	rAuth.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("health without token = %d, want 200", w.Code)
	}

	// Other endpoints require auth
	req = httptest.NewRequest("GET", "/api/users/"+chain10UID+"/strategies", nil)
	w = httptest.NewRecorder()
	rAuth.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("strategies without token = %d, want 401", w.Code)
	}
}

// --- Container logs ---

func TestTradingChain_ContainerLogs(t *testing.T) {
	api, r, _ := chain10Setup(t)
	api.container.logsFn = func(userID string, tail string) (string, error) {
		return "line1\nline2\n", nil
	}

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})

	req := httptest.NewRequest("GET", "/api/users/"+chain10UID+"/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("logs = %d, want 200", w.Code)
	}
}

// --- StartUser already running ---

func TestTradingChain_StartUser_AlreadyRunning(t *testing.T) {
	api, r, _ := chain10Setup(t)

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})
	api.users.UpdateStatus(chain10UID, ModeLive, StatusRunning)

	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("already running = %d, want 200", w.Code)
	}
}

func TestTradingChain_StartUser_NoStrategies(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("start without strategies = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestTradingChain_StartUser_StartsAsync(t *testing.T) {
	api, r, _ := chain10Setup(t)

	// Mock bbgo container that responds to ping
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(baseURL)
	}
	api.containerStart = func(_ *UserContainer) error { return nil }
	api.containerRunning = func(_, _ string) bool { return false }
	api.container.apiURLFn = func(userID, mode string) string { return bbgoSrv.URL }

	api.users.AddStrategy(chain10UID, ModeLive, StrategyEntry{
		ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "paper", Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	})

	req := httptest.NewRequest("POST", "/api/users/"+chain10UID+"/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("start = %d, want 202; body: %s", w.Code, w.Body.String())
	}

	// Wait for async goroutine
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		uc, _ := api.users.Get(chain10UID, ModeLive)
		if uc.Status == StatusRunning {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	uc, _ := api.users.Get(chain10UID, ModeLive)
	if uc.Status != StatusRunning {
		t.Errorf("status = %s, want running after async start", uc.Status)
	}
}

// --- resolveUserID edge cases ---

func TestTradingChain_ResolveUserID_InvalidUUID(t *testing.T) {
	_, r, _ := chain10Setup(t)

	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid UUID = %d, want 400", w.Code)
	}
}

func TestTradingChain_ResolveUserID_Mismatch(t *testing.T) {
	_, r, _ := chain10Setup(t)

	// Middleware sets X-User-Id to chain10UID, URL uses different valid UUID
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeee000199/strategies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("user mismatch = %d, want 403", w.Code)
	}
}
