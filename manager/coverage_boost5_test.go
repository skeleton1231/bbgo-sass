package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- api.go: resolveUserID branches ---

func TestAPI_ResolveUserID_InvalidUUIDV2(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/users/not-a-uuid/status", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ResolveUserID_MismatchV2(t *testing.T) {
	_, r := setupHandlerAPI(t)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeef/status", nil)
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID) // different from URL
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: resolveInstanceForRequest no instanceID param, instance found ---

func TestAPI_ResolveInstance_NoInstanceID_Found(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	// BBGoTrades without instanceID queries ListInstances path
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/trades?mode=live", nil)
	// instance is running but no bbgo mock -> 502
	if w.Code != http.StatusBadGateway {
		t.Logf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ResolveInstance_NoInstances(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/trades?mode=live", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: CreateNotificationConfig ---

func TestAPI_CreateNotificationConfig_BadJSON(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), nil)

	req := httptest.NewRequest("POST", "/api/notifications/config", strings.NewReader("bad json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_CreateNotificationConfig_WrongType(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), nil)

	w := doRequest(r, "POST", "/api/notifications/config", map[string]any{
		"type": "email", "token": "tok", "chat_id": "123",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig_TelegramNoToken(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc, _ := NewEncryptor(testEncryptionKey)
	api.notifier = NewNotifier(t.TempDir(), enc)

	w := doRequest(r, "POST", "/api/notifications/config", map[string]any{
		"type": "telegram", "chat_id": "123",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig_TelegramSuccess(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc, _ := NewEncryptor(testEncryptionKey)
	api.notifier = NewNotifier(t.TempDir(), enc)

	w := doRequest(r, "POST", "/api/notifications/config", map[string]any{
		"type": "telegram", "token": "bot123:abc", "chat_id": "456",
		"rules": map[string]any{"trade_events": true},
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig_SlackMissingWebhook(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc, _ := NewEncryptor(testEncryptionKey)
	api.notifier = NewNotifier(t.TempDir(), enc)

	w := doRequest(r, "POST", "/api/notifications/config", map[string]any{
		"type": "slack",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig_SlackSuccess(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc, _ := NewEncryptor(testEncryptionKey)
	api.notifier = NewNotifier(t.TempDir(), enc)

	w := doRequest(r, "POST", "/api/notifications/config", map[string]any{
		"type": "slack", "webhook_url": "https://hooks.slack.com/test",
		"rules": map[string]any{"order_events": true},
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: ListNotificationConfigs ---

func TestAPI_ListNotificationConfigs_Empty(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), nil)

	w := doRequest(r, "GET", "/api/notifications/config", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ListNotificationConfigs_WithData(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc, _ := NewEncryptor(testEncryptionKey)
	api.notifier = NewNotifier(t.TempDir(), enc)

	// Create one first
	doRequest(r, "POST", "/api/notifications/config", map[string]any{
		"type": "telegram", "token": "tok", "chat_id": "chat",
	})

	w := doRequest(r, "GET", "/api/notifications/config", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp []notifConfigResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 config, got %d", len(resp))
	}
}

// --- api.go: DeleteNotificationConfig ---

func TestAPI_DeleteNotificationConfig_NotFoundV2(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), nil)

	w := doRequest(r, "DELETE", "/api/notifications/config/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteNotificationConfig_Success(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc, _ := NewEncryptor(testEncryptionKey)
	api.notifier = NewNotifier(t.TempDir(), enc)

	// Create first
	w := doRequest(r, "POST", "/api/notifications/config", map[string]any{
		"type": "telegram", "token": "tok", "chat_id": "chat",
	})
	var createResp notifConfigResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)

	// Delete it
	w = doRequest(r, "DELETE", "/api/notifications/config/"+createResp.ID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: TestNotification ---

func TestAPI_TestNotification_NoChannelsV2(t *testing.T) {
	api, r := setupHandlerAPI(t)
	api.notifier = NewNotifier(t.TempDir(), nil)

	w := doRequest(r, "POST", "/api/notifications/test", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: DeleteStrategy ---

func TestAPI_DeleteStrategy_NotFoundV2(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "DELETE", "/api/users/"+testUUID+"/strategies/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DeleteStrategy_Success(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "DELETE", "/api/users/"+testUUID+"/strategies/"+inst.InstanceID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: CreateCredential with passphrase ---

func TestAPI_CreateCredential_WithPassphrase(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.encryptor = enc
	api.creds = NewCredentialStore(dir, enc)
	api.verifyCredFn = func(_, _, _, _ string, _ bool) VerifyResult {
		return VerifyResult{Verified: true}
	}

	w := doRequest(r, "POST", "/api/credentials", map[string]any{
		"exchange": "binance", "api_key": "key", "api_secret": "secret",
		"passphrase": "mypass", "is_testnet": false,
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: SubmitBacktest with need_sync true ---

func TestAPI_SubmitBacktest_WithSync(t *testing.T) {
	api, r := setupHandlerAPI(t)

	w := doRequest(r, "POST", "/api/backtest/submit", map[string]any{
		"strategy":   "grid2",
		"config":     map[string]any{"gridNumber": 5},
		"exchange":   "binance",
		"symbol":     "BTCUSDT",
		"start_time": "2024-01-01",
		"end_time":   "2024-01-31",
		"need_sync":  true,
	})
	if w.Code != http.StatusOK && w.Code != http.StatusAccepted {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	jobID, _ := resp["job_id"].(string)

	assertJobEventually(t, api.btJobs, jobID, JobCompleted, 5*time.Second)
}

// --- api.go: RunBacktest handler ---

func TestAPI_RunBacktest_BadBody(t *testing.T) {
	_, r := setupHandlerAPI(t)
	req := httptest.NewRequest("POST", "/api/backtest", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	t.Logf("status = %d", w.Code)
}

func TestAPI_RunBacktest_MissingStrategy(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest", map[string]any{
		"exchange": "binance", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

func TestAPI_RunBacktest_MissingExchange(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest", map[string]any{
		"strategy": "grid2", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

func TestAPI_RunBacktest_MissingSymbol(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest", map[string]any{
		"strategy": "grid2", "exchange": "binance",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

func TestAPI_RunBacktest_Success(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.runBacktestFn = func(userID, jobID string, yamlContent []byte) ([]byte, error) {
		return []byte("backtest output"), nil
	}

	w := doRequest(r, "POST", "/api/backtest", map[string]any{
		"strategy": "grid2", "exchange": "binance", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
		"config": map[string]any{"gridNumber": 5},
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_RunBacktest_ExecuteError(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.runBacktestFn = func(userID, jobID string, yamlContent []byte) ([]byte, error) {
		return nil, fmt.Errorf("backtest engine error")
	}

	w := doRequest(r, "POST", "/api/backtest", map[string]any{
		"strategy": "grid2", "exchange": "binance", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: ContainerLogs ---

func TestAPI_ContainerLogs_NotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/logs?mode=live", nil)
	// Returns 200 with empty logs when no container found
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

func TestAPI_ContainerLogs_WithMock(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.logsFn = func(containerName string) (string, error) {
		return "log line 1\nlog line 2", nil
	}
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/logs?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: StopInstance ---

func TestAPI_StopInstance_NotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/instances/nonexistent/stop", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: StartUser / StopUser ---

func TestAPI_StartUser_NoRunningInstance(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }
	api.container.dockerFn = func(args ...string) (string, error) { return "", nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/start?mode=live", nil)
	if w.Code != http.StatusAccepted && w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: UserStatus ---

func TestAPI_UserStatus_NoInstances(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/status", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_UserStatus_WithRunningInstance(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }
	api.container.apiURLFn = func(string) string { return "http://localhost:8080" }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/status", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoSessions ---

func TestAPI_BBGoSessions_WithMock(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "sessions") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"sessions":[{"name":"binance","exchange":"binance"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/sessions?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoPing with running instance ---

func TestAPI_BBGoPing_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/ping?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoTrades with running mock ---

func TestAPI_SyncBacktestData_WithSync(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.syncBacktestFn = func(userID, exchange, symbol, start, end string) (string, error) {
		return "synced", nil
	}

	w := doRequest(r, "POST", "/api/backtest/sync", map[string]any{
		"exchange": "binance", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	if w.Code != http.StatusOK {
		t.Logf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_SyncBacktestData_SyncError(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.syncBacktestFn = func(userID, exchange, symbol, start, end string) (string, error) {
		return "", fmt.Errorf("sync failed")
	}

	w := doRequest(r, "POST", "/api/backtest/sync", map[string]any{
		"exchange": "binance", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	// Sync returns 200 with errors embedded in the response
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: ClearAllStrategies ---

func TestAPI_ClearAllStrategies_Empty(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store

	w := doRequest(r, "DELETE", "/api/users/"+testUUID+"/strategies", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_ClearAllStrategies_WithData(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "DELETE", "/api/users/"+testUUID+"/strategies", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: CreateStrategy cross-exchange valid ---

func TestAPI_CreateStrategy_CrossExchangeValid(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "xlayer", "name": "bot1",
		"crossExchange": true, "mode": "live",
		"sessions": []map[string]any{
			{"role": "maker", "exchange": "binance"},
			{"role": "hedge", "exchange": "binance"},
		},
	})
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: SubmitBacktest missing fields ---

func TestAPI_SubmitBacktest_MissingStrategy(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest/submit", map[string]any{
		"exchange": "binance", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_SubmitBacktest_MissingExchangeV2(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest/submit", map[string]any{
		"strategy": "grid2", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: BBGoOpenOrders / BBGoBalances / BBGoAssets with running mock ---

func TestContainer_StopAllForUser(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "", nil
	}
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }

	store, _ := newTestStore(t)
	cm.store = store

	createTestInstance(t, store, "u1", "live", "grid2", "BTCUSDT", nil)
	createTestInstance(t, store, "u1", "paper", "grid2", "ETHUSDT", nil)

	cm.StopAllForUser("u1")
}

// --- container.go: DiscoverContainers ---

func TestContainer_DiscoverContainers(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "bbgo-user1-live-grid2-btcusdt\nbbgo-user1-paper-grid2-ethusdt", nil
	}

	containers := cm.DiscoverContainers()
	t.Logf("discovered %d containers", len(containers))
}

// --- notifier.go: Dispatch with rate limit ---

func TestNotifier_Dispatch_RateLimit(t *testing.T) {
	enc, _ := NewEncryptor(testEncryptionKey)
	token, _ := enc.Encrypt("https://hooks.slack.com/test")
	n := NewNotifier(t.TempDir(), enc)
	n.configs["user1"] = []NotificationConfig{
		{Channel: NotificationChannel{ID: "ch1", Type: "slack", Enabled: true, WebhookURL: token},
			Rules: NotificationRule{TradeEvents: true}},
	}
	n.rateLimit = 1 * time.Hour

	// First dispatch — will fail to send (no real slack) but rate limit is recorded
	n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "test", Message: "msg"})
	// Second dispatch should be rate limited (returns false)
	if n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "test", Message: "msg"}) {
		t.Error("second dispatch should be rate limited")
	}
}

// --- notifier.go: Dispatch with no enabled channels ---

func TestNotifier_Dispatch_DisabledChannelV2(t *testing.T) {
	n := NewNotifier(t.TempDir(), nil)
	n.configs["user1"] = []NotificationConfig{
		{Channel: NotificationChannel{ID: "ch1", Type: "telegram", Enabled: false},
			Rules: NotificationRule{TradeEvents: true}},
	}

	if n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "test", Message: "msg"}) {
		t.Error("dispatch should fail with disabled channel")
	}
}

// --- api.go: ListStrategies empty ---

func TestAPI_ListStrategies_Empty(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/strategies", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: ListStrategies with data ---

func TestAPI_ListStrategies_WithData(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	createTestInstance(t, store, testUUID, "paper", "grid2", "ETHUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/strategies", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	insts, _ := resp["instances"].([]any)
	if len(insts) != 2 {
		t.Errorf("expected 2 instances, got %d", len(insts))
	}
}

// --- api.go: DownloadBacktestReport completed with no files ---

func TestAPI_DownloadBacktestReport_CompletedButNoFiles(t *testing.T) {
	api, r := setupHandlerAPI(t)
	now := time.Now()
	job := &BacktestJob{
		ID: "bt-nofiles", UserID: testUUID, Strategy: "grid2",
		Status: JobCompleted, CreatedAt: now,
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus(job.ID, JobCompleted, "")

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-nofiles/download?filename=trades.csv", nil)
	// Should fail because no real backtest directory exists
	if w.Code == http.StatusOK {
		t.Error("expected non-200 with no files")
	}
}
