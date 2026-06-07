package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- api.go: BBGoPnL with paper mode (skips syncer) ---

func TestAPI_BBGoPnL_PaperMode(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"trades":[{"id":1,"symbol":"BTCUSDT","side":"buy","price":"50000","quantity":"0.1","fee":"0.001"}]}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "paper", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/pnl?instanceID="+inst.InstanceID+"&mode=paper", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoClosedOrders with running mock ---

func TestAPI_BBGoClosedOrders_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	// Use the session-specific closed-orders path
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/closed-orders?instanceID="+inst.InstanceID+"&mode=live&session=binance", nil)
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: BBGoTradingVolume with running mock ---

func TestAPI_BBGoTradingVolume_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/trading-volume?instanceID="+inst.InstanceID+"&mode=live", nil)
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: BBGoTradeMarkers with running mock ---

func TestAPI_BBGoTradeMarkers_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/trade-markers?instanceID="+inst.InstanceID+"&mode=live&session=binance", nil)
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: MarketSymbols with hub ---

func TestAPI_MarketSymbols_WithHub(t *testing.T) {
	api, r := setupHandlerAPI(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"symbols":["BTCUSDT","ETHUSDT"]}`))
	}))
	defer srv.Close()

	// MarketDataHub needs a real gRPC conn for MarketSymbols
	// Without a real conn, it returns 502 — just exercise the path
	api.hub = &MarketDataHub{}
	w := doRequest(r, "GET", "/api/markets/binance/symbols", nil)
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: CreateCredential with syncer ---

func TestAPI_CreateCredential_WithSyncer(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.encryptor = enc
	api.creds = NewCredentialStore(dir, enc)
	api.verifyCredFn = func(_, _, _, _ string, _ bool) VerifyResult {
		return VerifyResult{Verified: true}
	}
	api.syncer = NewSyncer(nil)

	w := doRequest(r, "POST", "/api/credentials", map[string]any{
		"exchange": "binance", "api_key": "key", "api_secret": "secret",
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: DeleteCredential with syncer ---

func TestAPI_DeleteCredential_WithSyncer(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.creds = NewCredentialStore(dir, enc)
	api.syncer = NewSyncer(nil)
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	api.creds.Upsert(ExchangeCredential{
		ID: "cred-del2", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: "enc1", APISecretEncrypted: "enc2",
	})

	w := doRequest(r, "DELETE", "/api/credentials/cred-del2", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: CreateCredential with verified testnet ---

func TestAPI_CreateCredential_VerifiedTestnet(t *testing.T) {
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
		"is_testnet": true,
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: TestNotification with configured channel ---

func TestAPI_TestNotification_WithChannel(t *testing.T) {
	api, r := setupHandlerAPI(t)
	enc, _ := NewEncryptor(testEncryptionKey)
	api.notifier = NewNotifier(t.TempDir(), enc)

	token, _ := enc.Encrypt("https://hooks.slack.com/test")
	api.notifier.Create(testUUID, NotificationConfig{
		Channel: NotificationChannel{ID: "ch1", Type: "slack", Enabled: true, WebhookURL: token},
		Rules:   NotificationRule{TradeEvents: true},
	})

	w := doRequest(r, "POST", "/api/notifications/test", nil)
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: ListCredentials with data ---

func TestAPI_ListCredentials_WithDataV2(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	api.creds = NewCredentialStore(dir, enc)

	api.creds.Upsert(ExchangeCredential{
		ID: "cred-1", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: "enc1", APISecretEncrypted: "enc2", IsVerified: true,
	})

	w := doRequest(r, "GET", "/api/credentials", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp []credentialResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 credential, got %d", len(resp))
	}
}

// --- api.go: DownloadBacktestReport with allowed filename ---

func TestAPI_DownloadBacktestReport_AllowedFile(t *testing.T) {
	api, r := setupHandlerAPI(t)
	now := time.Now()
	job := &BacktestJob{
		ID: "bt-allowed", UserID: testUUID, Strategy: "grid2",
		Status: JobCompleted, CreatedAt: now,
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus(job.ID, JobCompleted, "")

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-allowed/download?filename=equity.csv", nil)
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: ListBots with running containers ---

func TestAPI_ListBots_RunningContainers(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"strategies":[{"strategyInstanceID":"inst-1","strategy":"grid2","on":["binance"],"grid2":{"symbol":"BTCUSDT","gridNumber":5}}]}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bots?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: GetBot with running container ---

func TestAPI_GetBot_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"strategies":[{"strategyInstanceID":"inst-1","strategy":"grid2","on":["binance"],"grid2":{"symbol":"BTCUSDT","gridNumber":5}}]}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bots/"+inst.InstanceID+"?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: SubmitBacktest bad JSON ---

func TestAPI_SubmitBacktest_BadBody(t *testing.T) {
	_, r := setupHandlerAPI(t)
	req := httptest.NewRequest("POST", "/api/backtest/submit", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

// --- api.go: SubmitBacktest missing symbol ---

func TestAPI_SubmitBacktest_MissingSymbolV2(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest/submit", map[string]any{
		"strategy": "grid2", "exchange": "binance",
		"start_time": "2024-01-01", "end_time": "2024-01-31",
	})
	// Submit doesn't validate all fields — returns 202
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: SubmitBacktest missing start time ---

func TestAPI_SubmitBacktest_MissingStartTime(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "POST", "/api/backtest/submit", map[string]any{
		"strategy": "grid2", "exchange": "binance", "symbol": "BTCUSDT",
	})
	t.Logf("status = %d, body = %s", w.Code, w.Body.String())
}

// --- api.go: CreateStrategy with symbol ---

func TestAPI_CreateStrategy_WithSymbol(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "my grid", "exchange": "binance",
		"config": map[string]any{"gridNumber": 5},
		"symbol": "BTCUSDT", "mode": "paper",
	})
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: StopUser ---

func TestAPI_StopUser_WithInstances(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.dockerFn = func(args ...string) (string, error) { return "", nil }
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/stop?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoSessionTrades with running mock ---

func TestAPI_BBGoSessionTrades_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"trades":[{"id":1}]}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/trades?instanceID="+inst.InstanceID+"&mode=live&session=binance", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoSessionOpenOrders with running mock ---

func TestAPI_BBGoSessionOpenOrders_Running(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"session":{"name":"binance"},"orders":[]}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/open-orders?instanceID="+inst.InstanceID+"&mode=live&session=binance", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: BBGoStrategies with running mock ---

func TestAPI_BBGoStrategies_RunningV2(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"strategies":[]}`))
	}))
	defer srv.Close()

	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return NewBBGoClient(srv.URL)
	}

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/strategies?instanceID="+inst.InstanceID+"&mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
