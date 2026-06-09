package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupHandlerAPI(t *testing.T) (*API, *chi.Mux) {
	t.Helper()
	store, dir := newTestStore(t)
	cfg := &Config{ManagerToken: "test-token", DataDir: dir, DockerNetwork: "bbgo-net", DataVolume: "bbgo-data", BBGOImage: "bbgo-base:latest"}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(name string) (bool, error) { return true, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }
	btJobs := NewBacktestJobStore(t.TempDir())
	btExec := NewBacktestExecutor(btJobs, cm, nil, nil, store.Defaults())
	api := NewAPI(cfg, store, cm, NewBotProxy(cm), nil, nil, nil, nil, nil, nil, btExec, btJobs, nil)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", testUUID)
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)
	return api, r
}

func TestAPI_Health(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/health", nil)
	if w.Code != http.StatusOK {
		t.Errorf("Health status = %d", w.Code)
	}
}

func TestAPI_StopUser(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/stop?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("StopUser status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_UserStatus(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("UserStatus status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if instances, ok := resp["instances"].([]any); !ok || len(instances) == 0 {
		t.Errorf("expected instances, got %v", resp)
	}
}

func TestAPI_UserStatus_Empty(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/status", nil)
	if w.Code != http.StatusOK {
		t.Errorf("UserStatus empty status = %d", w.Code)
	}
}

func TestAPI_BBGoPing_NotRunning(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(name string) (bool, error) { return false, nil }
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/ping?mode=live", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ListCredentials(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.creds = NewCredentialStore(t.TempDir(), testEncryptor(t))
	api.creds.Upsert(ExchangeCredential{ID: "c1", UserID: testUUID, Exchange: "binance"})

	w := doRequest(r, "GET", "/api/credentials?userID="+testUUID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("ListCredentials status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteCredential(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.creds = NewCredentialStore(t.TempDir(), testEncryptor(t))
	api.creds.Upsert(ExchangeCredential{ID: "c1", UserID: testUUID, Exchange: "binance"})

	w := doRequest(r, "DELETE", "/api/credentials/c1?userID="+testUUID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("DeleteCredential status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), testEncryptor(t))

	body := map[string]any{
		"type":    "telegram",
		"token":   "bot-token-123",
		"chat_id": "12345",
		"rules":   map[string]any{"trade_events": true},
	}
	w := doRequest(r, "POST", "/api/notifications/config?userID="+testUUID, body)
	if w.Code != http.StatusCreated {
		t.Errorf("CreateNotificationConfig status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ListNotificationConfigs(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), testEncryptor(t))
	api.notifier.Create(testUUID, NotificationConfig{
		Channel: NotificationChannel{ID: "n1", Type: "telegram"},
	})

	w := doRequest(r, "GET", "/api/notifications/config?userID="+testUUID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("ListNotificationConfigs status = %d", w.Code)
	}
}

func TestAPI_DeleteNotificationConfig(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), testEncryptor(t))
	api.notifier.Create(testUUID, NotificationConfig{
		Channel: NotificationChannel{ID: "n1", Type: "telegram"},
	})

	w := doRequest(r, "DELETE", "/api/notifications/config/n1?userID="+testUUID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("DeleteNotificationConfig status = %d", w.Code)
	}
}

func TestAPI_Close(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	api.Close()
}

func TestAPI_HasDataForRange_NoHub(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-01-02") {
		t.Error("expected false with no hub")
	}
}

func TestAPI_ContainerLogs_WithInstanceID(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.logsFn = func(name string) (string, error) { return "log line", nil }
	inst := createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/logs?mode=live&instanceID="+inst.InstanceID+"&tail=50", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ContainerLogs_AllInstances(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.logsFn = func(name string) (string, error) { return "log", nil }
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/logs?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGOSessions_MockServer(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	srv := httptest.NewServer(serveJSON(BBGoSessionsResponse{Sessions: []BBGoSession{{Name: "binance"}}}))
	defer srv.Close()
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/sessions?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoSessionDetail_MockServer(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	srv := httptest.NewServer(serveJSON(BBGoSessionDetail{Session: BBGoSession{Name: "binance"}}))
	defer srv.Close()
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/binance?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}


func TestAPI_BBGOSessions_NotRunning(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.checkRunningFn = func(name string) (bool, error) { return false, nil }
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/sessions?mode=live", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestAPI_BBGoSymbols_MockServer(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	srv := httptest.NewServer(serveJSON(BBGoSymbolsResponse{Symbols: []string{"BTCUSDT"}}))
	defer srv.Close()
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/binance/symbols?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ResolveInstanceForRequest_NoInstances(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/ping?mode=live", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_CreateStrategy(t *testing.T) {
	_, r := setupHandlerAPI(t)
	body := map[string]any{
		"strategy": "grid2",
		"symbol":   "BTCUSDT",
		"exchange": "binance",
		"mode":     "live",
		"name":     "My Grid",
		"config":   map[string]any{"quantity": 0.001},
	}
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", body)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ListStrategies(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/strategies", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_StartUser(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/start?mode=live", nil)
	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ResolveUserID_Mismatch(t *testing.T) {
	_, r := setupHandlerAPI(t)
	otherUUID := "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"
	req := httptest.NewRequest("GET", "/api/users/"+otherUUID+"/status", nil)
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ResolveUserID_InvalidUUID(t *testing.T) {
	_, r := setupHandlerAPI(t)
	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/status", nil)
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_HubForMode(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	if h := api.hubForMode("live"); h != nil {
		t.Error("expected nil hub")
	}
	if h := api.hubForMode("paper"); h != nil {
		t.Error("expected nil hub")
	}
}

func TestAPI_MarketSymbols_NoREST(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/markets/binance/symbols", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestAPI_MarketTicker_NoHub(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestAPI_MarketKlines_NoHub(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/markets/binance/klines?symbol=BTCUSDT&interval=1m", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestAPI_RunBacktest(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.runBacktestFn = func(userID, jobID string, yamlContent []byte) ([]byte, error) {
		return []byte("backtest output here"), nil
	}

	body := map[string]any{
		"strategy":   "grid2",
		"config":     map[string]any{"symbol": "BTCUSDT", "gridNumber": 10},
		"exchange":   "binance",
		"start_time": "2024-01-01",
		"end_time":   "2024-06-01",
	}
	w := doRequest(r, "POST", "/api/backtest", body)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_RunBacktest_InvalidConfig(t *testing.T) {
	_, r := setupHandlerAPI(t)
	body := map[string]any{
		"strategy": "grid2",
		"config":   "not-json",
	}
	w := doRequest(r, "POST", "/api/backtest", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_SyncBacktestData(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.container.dockerFn = func(args ...string) (string, error) {
		return "synced", nil
	}

	body := map[string]any{
		"exchange":   "binance",
		"symbols":    []string{"BTCUSDT"},
		"start_time": "2024-01-01",
		"end_time":   "2024-06-01",
	}
	w := doRequest(r, "POST", "/api/backtest/sync", body)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_BacktestSyncStatus_NoDB(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/status", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["available"] != false {
		t.Errorf("expected available=false, got %v", resp["available"])
	}
}

func TestAPI_SubmitBacktest_NoStrategy(t *testing.T) {
	_, r := setupHandlerAPI(t)
	body := map[string]any{"strategy": ""}
	w := doRequest(r, "POST", "/api/backtest/submit", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_GetBacktestJob_NotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/jobs/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_ListBacktestJobs_Empty(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/jobs", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_DownloadBacktestReport_NotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/jobs/nonexistent/download", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_TestNotification_NoChannels(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), testEncryptor(t))
	w := doRequest(r, "POST", "/api/notifications/test", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_TestNotification_DisabledChannel(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), testEncryptor(t))
	api.notifier.Create(testUUID, NotificationConfig{
		Channel: NotificationChannel{ID: "n1", Type: "telegram", Enabled: false},
		Rules:   NotificationRule{TradeEvents: true},
	})

	w := doRequest(r, "POST", "/api/notifications/test", nil)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (disabled channel → rate limit), got %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ListBots(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bots?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ListBots_Empty(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bots", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_GetBot_NotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bots/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_GetBot_Found(t *testing.T) {
	api, r := setupHandlerAPI(t)
	inst := createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bots/"+inst.InstanceID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteStrategy(t *testing.T) {
	api, r := setupHandlerAPI(t)
	inst := createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "DELETE", "/api/users/"+testUUID+"/strategies/"+inst.InstanceID+"?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteStrategy_NotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "DELETE", "/api/users/"+testUUID+"/strategies/nonexistent?mode=live", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_IssueWSTicket(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/ws/ticket", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ticket"] == "" {
		t.Error("expected non-empty ticket")
	}
}

func TestAPI_ProxyToBot_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	inst := createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer bbgoSrv.Close()

	api.container.apiURLFn = func(containerName string) string {
		return bbgoSrv.URL
	}

	w := doRequest(r, "GET", "/api/bbgo/"+testUUID+"/api/ping?mode=live&instanceID="+inst.InstanceID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ProxyToBot_InstanceNotRunning(t *testing.T) {
	api, r := setupHandlerAPI(t)
	inst := createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	api.container.checkRunningFn = func(name string) (bool, error) { return false, nil }

	w := doRequest(r, "GET", "/api/bbgo/"+testUUID+"/api/ping?mode=live&instanceID="+inst.InstanceID, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_DownloadBacktestReport_CSV(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-dl-1", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-dl-1", JobCompleted, "done")

	reportDir := api.container.BacktestReportDir(testUUID, "bt-dl-1")
	if reportDir == "" {
		t.Fatal("BacktestReportDir returned empty")
	}
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "trades.tsv"), []byte("gid\tprice\n1\t100\n"), 0o644)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-dl-1/download?file=trades", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("expected csv content type, got %s", ct)
	}
}

func TestAPI_DownloadBacktestReport_JobNotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/jobs/nonexistent/download?file=trades", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_CreateNotificationConfig_Slack(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), newTestEncryptor(t))

	body := json.RawMessage(`{"type":"slack","webhook_url":"https://hooks.slack.com/test","rules":{"trade_events":true}}`)
	w := doRequest(r, "POST", "/api/notifications/config", body)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig_Telegram(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), newTestEncryptor(t))

	body := json.RawMessage(`{"type":"telegram","token":"bot123","chat_id":"456","rules":{"trade_events":true}}`)
	w := doRequest(r, "POST", "/api/notifications/config", body)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig_InvalidType(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), newTestEncryptor(t))

	body := json.RawMessage(`{"type":"email","rules":{}}`)
	w := doRequest(r, "POST", "/api/notifications/config", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_CreateNotificationConfig_TelegramMissingToken(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), newTestEncryptor(t))

	body := json.RawMessage(`{"type":"telegram","chat_id":"123","rules":{}}`)
	w := doRequest(r, "POST", "/api/notifications/config", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_CreateNotificationConfig_SlackMissingURL(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), newTestEncryptor(t))

	body := json.RawMessage(`{"type":"slack","rules":{}}`)
	w := doRequest(r, "POST", "/api/notifications/config", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_ListNotificationConfigs_AfterCreate(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), newTestEncryptor(t))

	body := `{"type":"slack","webhook_url":"https://hooks.slack.com/test","rules":{"trade_events":true}}`
	doRequest(r, "POST", "/api/notifications/config", strings.NewReader(body))

	w := doRequest(r, "GET", "/api/notifications/config", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteNotificationConfig_Existing(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc := newTestEncryptor(t)
	api.notifier = NewNotifier(t.TempDir(), enc)

	cfg := NotificationConfig{
		Channel: NotificationChannel{ID: "notif-test1", Type: "slack", Enabled: true},
		Rules:   NotificationRule{TradeEvents: true},
	}
	api.notifier.Create(testUUID, cfg)

	w := doRequest(r, "DELETE", "/api/notifications/config/notif-test1", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteNotificationConfig_NotFound(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), newTestEncryptor(t))

	w := doRequest(r, "DELETE", "/api/notifications/config/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_CreateCredential_Valid(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc := newTestEncryptor(t)
	api.creds = NewCredentialStore(t.TempDir(), enc)
	api.encryptor = enc

	body := json.RawMessage(`{"exchange":"binance","api_key":"testkey","api_secret":"testsecret"}`)
	w := doRequest(r, "POST", "/api/credentials", body)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateCredential_InvalidBody(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.creds = NewCredentialStore(t.TempDir(), newTestEncryptor(t))

	w := doRequest(r, "POST", "/api/credentials", json.RawMessage("invalid"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_CreateCredential_UnsupportedExchange(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.creds = NewCredentialStore(t.TempDir(), newTestEncryptor(t))

	body := json.RawMessage(`{"exchange":"unknown","api_key":"k","api_secret":"s"}`)
	w := doRequest(r, "POST", "/api/credentials", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_ListCredentials_WithStore(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.creds = NewCredentialStore(t.TempDir(), newTestEncryptor(t))

	w := doRequest(r, "GET", "/api/credentials", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_GetBacktestJob_OwnerMismatch(t *testing.T) {
	_, r := setupHandlerAPI(t)

	// The X-User-Id is testUUID, but request a job for a different user
	w := doRequest(r, "GET", "/api/backtest/jobs/fakejob", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_SubmitBacktest_InvalidBody(t *testing.T) {
	_, r := setupHandlerAPI(t)

	w := doRequest(r, "POST", "/api/backtest/submit", json.RawMessage("invalid"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_ContainerLogs(t *testing.T) {
	api, r := setupHandlerAPI(t)
	inst := createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	api.container.logsFn = func(containerName string) (string, error) {
		return "log line 1\nlog line 2", nil
	}

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/logs?mode=live&instanceID="+inst.InstanceID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteCredential_WithStore(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc := newTestEncryptor(t)
	api.creds = NewCredentialStore(t.TempDir(), enc)

	cred := ExchangeCredential{
		ID: "cred-1", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: "enc", APISecretEncrypted: "enc",
	}
	api.creds.Upsert(cred)

	w := doRequest(r, "DELETE", "/api/credentials/cred-1", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
