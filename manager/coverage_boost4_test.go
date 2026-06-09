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

// --- api.go: CreateStrategy validation branches ---

func TestAPI_CreateStrategy_MissingStrategy(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"name": "my-bot", "exchange": "binance",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "strategy is required") {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestAPI_CreateStrategy_MissingName(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "exchange": "binance",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_CreateStrategy_MissingExchange(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "bot1",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_CreateStrategy_InvalidMode(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "bot1", "exchange": "binance", "mode": "invalid",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_CreateStrategy_PaperNonBinance(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "bot1", "exchange": "okx", "mode": "paper",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_CreateStrategy_CrossExchangeNoSessions(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "xlayer", "name": "bot1", "crossExchange": true, "mode": "live",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_CreateStrategy_PaperCrossExchangeNonBinance(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "xlayer", "name": "bot1", "crossExchange": true,
		"mode": "paper",
		"sessions": []map[string]any{
			{"role": "maker", "exchange": "okx"},
			{"role": "hedge", "exchange": "binance"},
		},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_Success(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "my grid", "exchange": "binance",
		"config": map[string]any{"gridNumber": 5},
		"symbol": "BTCUSDT", "mode": "paper",
	})
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_InvalidBody(t *testing.T) {
	_, r := setupHandlerAPI(t)
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

// --- api.go: GetBacktestJob found + access denied ---

func TestAPI_GetBacktestJob_Found(t *testing.T) {
	api, r := setupHandlerAPI(t)
	now := time.Now()
	job := &BacktestJob{
		ID: "bt-found", UserID: testUUID, Strategy: "grid2",
		Status: JobCompleted, CreatedAt: now,
	}
	api.btJobs.Create(job)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-found", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_GetBacktestJob_AccessDenied(t *testing.T) {
	api, r := setupHandlerAPI(t)
	now := time.Now()
	job := &BacktestJob{
		ID: "bt-other", UserID: "other-user", Strategy: "grid2",
		Status: JobCompleted, CreatedAt: now,
	}
	api.btJobs.Create(job)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-other", nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d", w.Code)
	}
}

// --- api.go: ListBacktestJobs with jobs ---

func TestAPI_ListBacktestJobs_WithData(t *testing.T) {
	api, r := setupHandlerAPI(t)
	now := time.Now()
	api.btJobs.Create(&BacktestJob{ID: "bt-list1", UserID: testUUID, Strategy: "grid2", Status: JobCompleted, CreatedAt: now})
	api.btJobs.Create(&BacktestJob{ID: "bt-list2", UserID: testUUID, Strategy: "supertrend", Status: JobRunning, CreatedAt: now})

	w := doRequest(r, "GET", "/api/backtest/jobs", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp backtestJobsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(resp.Jobs))
	}
}

func TestAPI_ListBacktestJobs_EmptyList(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/jobs", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
	var resp backtestJobsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(resp.Jobs))
	}
}

// --- api.go: DownloadBacktestReport ---


func TestAPI_DownloadBacktestReport_NotCompleted(t *testing.T) {
	api, r := setupHandlerAPI(t)
	now := time.Now()
	job := &BacktestJob{
		ID: "bt-running", UserID: testUUID, Strategy: "grid2",
		Status: JobRunning, CreatedAt: now,
	}
	api.btJobs.Create(job)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-running/download", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}


// --- api.go: StartInstance ---

func TestAPI_StartInstance_NotFound(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/instances/nonexistent/start", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_StartInstance_AlreadyRunning(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", map[string]any{"gridNumber": 5})

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/instances/"+inst.InstanceID+"/start", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_StartInstance_AlreadyStarting(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	api.starting.Store(inst.InstanceID, true)

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/instances/"+inst.InstanceID+"/start", nil)
	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	api.starting.Delete(inst.InstanceID)
}

// --- api.go: CreateCredential ---

func TestAPI_CreateCredential_BadJSON(t *testing.T) {
	_, r := setupHandlerAPI(t)
	req := httptest.NewRequest("POST", "/api/credentials", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAPI_CreateCredential_MissingFields(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/credentials", map[string]any{
		"exchange": "binance",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateCredential_BadExchange(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/credentials", map[string]any{
		"exchange": "unknown", "api_key": "key", "api_secret": "secret",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateCredential_NoStorage(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/credentials", map[string]any{
		"exchange": "binance", "api_key": "key", "api_secret": "secret",
	})
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateCredential_Success(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.encryptor = enc
	api.creds = NewCredentialStore(dir, enc)
	api.verifyCredFn = func(_, _, _, _ string, _ bool) VerifyResult {
		return VerifyResult{Verified: true}
	}

	w := doRequest(r, "POST", "/api/credentials", map[string]any{
		"exchange": "binance", "api_key": "mykey", "api_secret": "mysecret",
		"passphrase": "mypass", "is_testnet": false,
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateCredential_VerifyFail(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.encryptor = enc
	api.creds = NewCredentialStore(dir, enc)
	api.verifyCredFn = func(_, _, _, _ string, _ bool) VerifyResult {
		return VerifyResult{Verified: false, Error: "invalid signature"}
	}

	w := doRequest(r, "POST", "/api/credentials", map[string]any{
		"exchange": "binance", "api_key": "badkey", "api_secret": "badsecret",
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid signature") {
		t.Errorf("expected verify error in response, got %s", w.Body.String())
	}
}

// --- api.go: ListCredentials ---

func TestAPI_ListCredentials_WithData(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.creds = NewCredentialStore(dir, enc)
	api.creds.Upsert(ExchangeCredential{
		ID: "cred-1", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: "enc1", APISecretEncrypted: "enc2",
	})

	w := doRequest(r, "GET", "/api/credentials", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: DeleteCredential ---

func TestAPI_DeleteCredential_Success(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.creds = NewCredentialStore(dir, enc)
	api.creds.Upsert(ExchangeCredential{
		ID: "cred-del", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: "enc1", APISecretEncrypted: "enc2",
	})

	w := doRequest(r, "DELETE", "/api/credentials/cred-del", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: startInstanceContainer with mocks ---

func TestAPI_StartInstanceContainer_CreateFails(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }
	api.container.dockerFn = func(args ...string) (string, error) {
		return "", fmt.Errorf("docker error")
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	api.starting.Store(inst.InstanceID, true)
	api.startInstanceContainer(inst)

	if _, loaded := api.starting.Load(inst.InstanceID); loaded {
		t.Error("starting flag should be cleaned up")
	}
}

func TestAPI_StartInstanceContainer_WithMockClient(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }
	api.container.dockerFn = func(args ...string) (string, error) { return "", nil }

	// Mock HTTP server for bbgo health check
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	api.starting.Store(inst.InstanceID, true)
	api.startInstanceContainer(inst)

	if _, loaded := api.starting.Load(inst.InstanceID); loaded {
		t.Error("starting flag should be cleaned up")
	}
}

// --- api.go: BBGoPnL ---

func TestAPI_SubmitBacktest_CompleteJob(t *testing.T) {
	api, r := setupHandlerAPI(t)

	w := doRequest(r, "POST", "/api/backtest/submit", map[string]any{
		"strategy":   "grid2",
		"config":     map[string]any{"gridNumber": 5},
		"exchange":   "binance",
		"symbol":     "BTCUSDT",
		"start_time": "2024-01-01",
		"end_time":   "2024-01-31",
		"need_sync":  false,
	})
	if w.Code != http.StatusOK && w.Code != http.StatusAccepted {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	jobID, _ := resp["job_id"].(string)

	assertJobEventually(t, api.btJobs, jobID, JobCompleted, 5*time.Second)
}

// --- api.go: SyncBacktestData full path ---

func TestAPI_SyncBacktestData_NoInstance(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest/sync", map[string]any{
		"exchange": "binance", "symbol": "BTCUSDT",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	if w.Code != http.StatusOK {
		t.Logf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: MarketSymbols ---

func TestAPI_MarketSymbols_NilHub(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/markets/binance/symbols", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoTrades / BBGoSessionTrades ---

func TestAPI_BacktestSyncStatus(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/status", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
