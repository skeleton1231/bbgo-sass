package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pb "github.com/c9s/bbgo/saas/manager/pb"
)

// ==================== Phase 1: uploadToStorage ====================

func mockStorageServer(t *testing.T) *StorageClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return NewStorageClient(srv.URL, "test-key")
}

func setupBacktestExecutorWithStorage(t *testing.T) (*BacktestExecutor, string) {
	t.Helper()
	store, dir := newTestStore(t)
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }

	storage := mockStorageServer(t)
	btJobs := NewBacktestJobStore(t.TempDir())
	ex := NewBacktestExecutor(btJobs, cm, nil, storage, store.Defaults())
	return ex, dir
}

func TestUploadToStorage_AllFiles(t *testing.T) {
	ex, _ := setupBacktestExecutorWithStorage(t)
	store := ex.container.store
	ex.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-upload")
	os.MkdirAll(reportDir, 0o755)

	os.WriteFile(filepath.Join(reportDir, "trades.tsv"), []byte("time,price\n1,100"), 0644)
	os.WriteFile(filepath.Join(reportDir, "orders.tsv"), []byte("time,side\n1,BUY"), 0644)
	os.WriteFile(filepath.Join(reportDir, "BTCUSDT-1h.tsv"), []byte("time,open\n1,100"), 0644)

	ex.uploadToStorage(testUUID, "bt-upload", json.RawMessage(`{"profit":100}`), []byte("time\tequity\n1\t1000"))
}

func TestUploadToStorage_NoEquityCurve(t *testing.T) {
	ex, _ := setupBacktestExecutorWithStorage(t)
	store := ex.container.store
	ex.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-noeq")
	os.MkdirAll(reportDir, 0o755)

	ex.uploadToStorage(testUUID, "bt-noeq", json.RawMessage(`{}`), nil)
}

func TestUploadToStorage_StorageError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	store, dir := newTestStore(t)
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }

	storage := NewStorageClient(srv.URL, "key")
	btJobs := NewBacktestJobStore(t.TempDir())
	ex := NewBacktestExecutor(btJobs, cm, nil, storage, store.Defaults())

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-err")
	os.MkdirAll(reportDir, 0o755)

	ex.uploadToStorage(testUUID, "bt-err", json.RawMessage(`{}`), []byte("data"))
}

func TestUploadToStorage_KlineFileSkipsNamed(t *testing.T) {
	ex, _ := setupBacktestExecutorWithStorage(t)
	store := ex.container.store
	ex.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-skip")
	os.MkdirAll(reportDir, 0o755)

	os.WriteFile(filepath.Join(reportDir, "trades.tsv"), []byte("t"), 0644)
	os.WriteFile(filepath.Join(reportDir, "orders.tsv"), []byte("t"), 0644)
	os.WriteFile(filepath.Join(reportDir, "equity_curve.tsv"), []byte("t"), 0644)
	os.WriteFile(filepath.Join(reportDir, "BTCUSDT-1h.tsv"), []byte("kline-data"), 0644)

	ex.uploadToStorage(testUUID, "bt-skip", json.RawMessage(`{}`), nil)
}

func TestUploadToStorage_MissingTradeFiles(t *testing.T) {
	ex, _ := setupBacktestExecutorWithStorage(t)
	store := ex.container.store
	ex.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-nofiles")
	os.MkdirAll(reportDir, 0o755)

	ex.uploadToStorage(testUUID, "bt-nofiles", json.RawMessage(`{}`), []byte("eq"))
}

// ==================== Phase 2: upsertToSupabase ====================

func TestInstanceStore_upsertToSupabase_ViaHook(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)

	called := false
	store.supabaseUpsertFn = func(inst *StrategyInstance) {
		called = true
		if inst.InstanceID != "inst-hook" {
			t.Errorf("expected inst-hook, got %s", inst.InstanceID)
		}
	}

	inst := &StrategyInstance{
		InstanceID: "inst-hook",
		UserID:     testUUID,
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     json.RawMessage(`{"symbol":"BTCUSDT"}`),
	}
	store.upsertToSupabase(inst)

	if !called {
		t.Error("expected hook to be called")
	}
}

func TestInstanceStore_upsertToSupabase_NilConfig(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)

	var gotConfig string
	store.supabaseUpsertFn = func(inst *StrategyInstance) {
		gotConfig = string(inst.Config)
	}

	store.upsertToSupabase(&StrategyInstance{
		InstanceID: "inst-nil-cfg",
		UserID:     testUUID,
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     nil,
	})

	if gotConfig != "" {
		t.Errorf("expected empty config, got %q", gotConfig)
	}
}

func TestInstanceStore_upsertToSupabase_WithMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "strategy_instances") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]any{{"instance_id": "inst-mock"}})
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sb, err := NewSupabaseClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("create supabase client: %v", err)
	}

	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	store.SetSupabase(sb)

	inst := &StrategyInstance{
		InstanceID: "inst-mock",
		UserID:     testUUID,
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     json.RawMessage(`{"symbol":"BTCUSDT"}`),
		Name:       "test",
	}
	store.upsertToSupabase(inst)
}

func TestInstanceStore_upsertToSupabase_NullConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	sb, err := NewSupabaseClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("create supabase client: %v", err)
	}

	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	store.SetSupabase(sb)

	// Test with null config
	store.upsertToSupabase(&StrategyInstance{
		InstanceID: "inst-null",
		UserID:     testUUID,
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     json.RawMessage(`null`),
	})
}

func TestInstanceStore_upsertToSupabase_WithSessions(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)

	var captured *StrategyInstance
	store.supabaseUpsertFn = func(inst *StrategyInstance) {
		captured = inst
	}

	store.upsertToSupabase(&StrategyInstance{
		InstanceID: "inst-sessions",
		UserID:     testUUID,
		Mode:       ModeLive,
		Strategy:   "xmaker",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     json.RawMessage(`{}`),
		Sessions: []SessionRoleConfig{
			{Name: "binance", Exchange: "binance"},
			{Name: "bybit", Exchange: "bybit"},
		},
		CrossExchange: true,
	})

	if captured == nil {
		t.Fatal("expected hook to be called")
	}
	if len(captured.Sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(captured.Sessions))
	}
}

// ==================== Phase 3: MarketKlines & MarketTicker ====================

func TestMarketKlines_Success(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryKlinesFn = func(ctx context.Context, req *pb.QueryKLinesRequest) (*pb.QueryKLinesResponse, error) {
		return &pb.QueryKLinesResponse{
			Klines: []*pb.KLine{
				{Symbol: "BTCUSDT", Open: "100", High: "110", Low: "90", Close: "105",
					Volume: "50", QuoteVolume: "5000", StartTime: 1700000000, Closed: true},
			},
		}, nil
	}

	w := doRequest(r, "GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_NoHub(t *testing.T) {
	_, r := setupHandlerAPI(t)

	w := doRequest(r, "GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestMarketKlines_InvalidParams(t *testing.T) {
	_, r := setupHandlerAPI(t)

	w := doRequest(r, "GET", "/api/markets/binance/klines", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestMarketKlines_EmptyResult(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryKlinesFn = func(ctx context.Context, req *pb.QueryKLinesRequest) (*pb.QueryKLinesResponse, error) {
		return &pb.QueryKLinesResponse{Klines: nil}, nil
	}

	w := doRequest(r, "GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_ResponseError(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryKlinesFn = func(ctx context.Context, req *pb.QueryKLinesRequest) (*pb.QueryKLinesResponse, error) {
		return &pb.QueryKLinesResponse{
			Error: &pb.Error{ErrorMessage: "upstream error"},
		}, nil
	}

	w := doRequest(r, "GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestMarketKlines_QueryError(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryKlinesFn = func(ctx context.Context, req *pb.QueryKLinesRequest) (*pb.QueryKLinesResponse, error) {
		return nil, context.DeadlineExceeded
	}

	w := doRequest(r, "GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestMarketKlines_CustomParams(t *testing.T) {
	api, r := setupHandlerAPI(t)

	var capturedReq *pb.QueryKLinesRequest
	api.queryKlinesFn = func(ctx context.Context, req *pb.QueryKLinesRequest) (*pb.QueryKLinesResponse, error) {
		capturedReq = req
		return &pb.QueryKLinesResponse{}, nil
	}

	w := doRequest(r, "GET", "/api/markets/binance/klines?symbol=ETHUSDT&interval=15m&limit=100&start_time=1700000000&end_time=1700086400", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if capturedReq.Interval != "15m" {
		t.Errorf("interval = %q", capturedReq.Interval)
	}
	if capturedReq.Limit != 100 {
		t.Errorf("limit = %d", capturedReq.Limit)
	}
	if capturedReq.StartTime != 1700000000 {
		t.Errorf("start_time = %d", capturedReq.StartTime)
	}
	if capturedReq.EndTime != 1700086400 {
		t.Errorf("end_time = %d", capturedReq.EndTime)
	}
}

func TestMarketKlines_DefaultInterval(t *testing.T) {
	api, r := setupHandlerAPI(t)

	var capturedReq *pb.QueryKLinesRequest
	api.queryKlinesFn = func(ctx context.Context, req *pb.QueryKLinesRequest) (*pb.QueryKLinesResponse, error) {
		capturedReq = req
		return &pb.QueryKLinesResponse{}, nil
	}

	doRequest(r, "GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	if capturedReq.Interval != "1h" {
		t.Errorf("default interval = %q, want 1h", capturedReq.Interval)
	}
}

func TestMarketTicker_Success(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryTickerFn = func(ctx context.Context, req *pb.QueryTickerRequest) (*pb.QueryTickerResponse, error) {
		return &pb.QueryTickerResponse{
			Ticker: &pb.Ticker{Symbol: "BTCUSDT", Open: 100, High: 110, Low: 90, Close: 105, Volume: 5000},
		}, nil
	}

	w := doRequest(r, "GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestMarketTicker_InvalidParams(t *testing.T) {
	_, r := setupHandlerAPI(t)

	w := doRequest(r, "GET", "/api/markets/binance/ticker", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestMarketTicker_NoHub(t *testing.T) {
	_, r := setupHandlerAPI(t)

	w := doRequest(r, "GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestMarketTicker_TickerNotFound(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryTickerFn = func(ctx context.Context, req *pb.QueryTickerRequest) (*pb.QueryTickerResponse, error) {
		return &pb.QueryTickerResponse{Ticker: nil}, nil
	}

	w := doRequest(r, "GET", "/api/markets/binance/ticker?symbol=NOTEXIST", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMarketTicker_ResponseError(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryTickerFn = func(ctx context.Context, req *pb.QueryTickerRequest) (*pb.QueryTickerResponse, error) {
		return &pb.QueryTickerResponse{
			Error: &pb.Error{ErrorMessage: "exchange error"},
		}, nil
	}

	w := doRequest(r, "GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestMarketTicker_QueryError(t *testing.T) {
	api, r := setupHandlerAPI(t)

	api.queryTickerFn = func(ctx context.Context, req *pb.QueryTickerRequest) (*pb.QueryTickerResponse, error) {
		return nil, context.DeadlineExceeded
	}

	w := doRequest(r, "GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

// ==================== Phase 5: uploadLocalToStorage ====================

func TestUploadLocalToStorage_Success(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	storage := mockStorageServer(t)
	api.storage = storage

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-local")
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte(`{"profit":100}`), 0644)

	job := &BacktestJob{ID: "bt-local", UserID: testUUID}
	if !api.uploadLocalToStorage(job, "summary.json") {
		t.Error("expected true")
	}
}

func TestUploadLocalToStorage_StorageError(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	api.storage = NewStorageClient(srv.URL, "key")

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-locerr")
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte(`{}`), 0644)

	job := &BacktestJob{ID: "bt-locerr", UserID: testUUID}
	if api.uploadLocalToStorage(job, "summary.json") {
		t.Error("expected false on storage error")
	}
}

func TestUploadLocalToStorage_FileNotFound(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }
	api.storage = mockStorageServer(t)

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-nofile")
	os.MkdirAll(reportDir, 0o755)

	job := &BacktestJob{ID: "bt-nofile", UserID: testUUID}
	if api.uploadLocalToStorage(job, "summary.json") {
		t.Error("expected false when file missing")
	}
}

func TestUploadLocalToStorage_TradesFile(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }
	api.storage = mockStorageServer(t)

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-trades")
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "trades.tsv"), []byte("time,price\n1,100"), 0644)

	job := &BacktestJob{ID: "bt-trades", UserID: testUUID}
	if !api.uploadLocalToStorage(job, "trades.tsv") {
		t.Error("expected true for trades.tsv upload")
	}
}

func TestUploadLocalToStorage_EquityCurve(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }
	api.storage = mockStorageServer(t)

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	reportDir := filepath.Join(store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID), "backtest", "bt-eq")
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "equity_curve.tsv"), []byte("time\tequity\n1\t1000"), 0644)

	job := &BacktestJob{ID: "bt-eq", UserID: testUUID}
	if !api.uploadLocalToStorage(job, "equity_curve.tsv") {
		t.Error("expected true for equity_curve.tsv")
	}
}

// ==================== Phase 4: dockerLong via hook ====================

func TestDockerLong_ViaHook(t *testing.T) {
	cm := testContainerManager(t)
	cm.dockerFn = func(args ...string) (string, error) {
		return "long-op-result", nil
	}
	out, err := cm.dockerLong("exec", "container", "bbgo", "backtest")
	if err != nil {
		t.Fatal(err)
	}
	if out != "long-op-result" {
		t.Errorf("got %q", out)
	}
}

// ==================== Bonus: BacktestExecutor execute paths ====================

func TestBacktestExecutor_Execute_WithSyncAndStorage(t *testing.T) {
	store, _ := newTestStore(t)
	dir := store.dataDir
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	btJobs := NewBacktestJobStore(t.TempDir())
	storage := mockStorageServer(t)
	ex := NewBacktestExecutor(btJobs, cm, nil, storage, store.Defaults())

	syncCalled := false
	ex.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		syncCalled = true
		return "synced", nil
	}
	ex.runFn = func(userID, jobID string, yamlContent []byte) ([]byte, error) {
		return []byte("ok"), nil
	}
	ex.reportFn = func(userID, jobID string) (json.RawMessage, []byte, error) {
		return json.RawMessage(`{"profit":100}`), []byte("equity"), nil
	}

	job := &BacktestJob{
		ID:        "bt-exec-sync",
		UserID:    testUUID,
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-06-01",
		NeedSync:  true,
		Config:    json.RawMessage(`{"symbol":"BTCUSDT"}`),
	}
	ex.store.Create(job)

	if !ex.store.AcquireSlot() {
		t.Fatal("expected slot available")
	}

	go ex.execute(job)
	time.Sleep(500 * time.Millisecond)

	got, ok := ex.store.Get(job.ID)
	if !ok {
		t.Fatal("job not found")
	}
	if got.Status != JobCompleted {
		t.Errorf("status = %q, want completed", got.Status)
	}
	if !syncCalled {
		t.Error("expected sync to be called")
	}
}

func TestBacktestExecutor_Execute_SyncFailsV2(t *testing.T) {
	store, _ := newTestStore(t)
	dir := store.dataDir
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	btJobs := NewBacktestJobStore(t.TempDir())
	ex := NewBacktestExecutor(btJobs, cm, nil, nil, store.Defaults())

	ex.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		return "", context.DeadlineExceeded
	}

	job := &BacktestJob{
		ID:        "bt-sync-fail",
		UserID:    testUUID,
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-06-01",
		NeedSync:  true,
		Config:    json.RawMessage(`{}`),
	}
	ex.store.Create(job)

	if !ex.store.AcquireSlot() {
		t.Fatal("expected slot available")
	}

	go ex.execute(job)
	time.Sleep(500 * time.Millisecond)

	got, ok := ex.store.Get(job.ID)
	if !ok {
		t.Fatal("job not found")
	}
	if got.Status != JobFailed {
		t.Errorf("status = %q, want failed", got.Status)
	}
}

func TestBacktestExecutor_Execute_RunFailsV2(t *testing.T) {
	store, _ := newTestStore(t)
	dir := store.dataDir
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	btJobs := NewBacktestJobStore(t.TempDir())
	ex := NewBacktestExecutor(btJobs, cm, nil, nil, store.Defaults())

	ex.runFn = func(userID, jobID string, yamlContent []byte) ([]byte, error) {
		return nil, context.DeadlineExceeded
	}

	job := &BacktestJob{
		ID:        "bt-run-fail",
		UserID:    testUUID,
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-06-01",
		NeedSync:  false,
		Config:    json.RawMessage(`{}`),
	}
	ex.store.Create(job)

	if !ex.store.AcquireSlot() {
		t.Fatal("expected slot available")
	}

	go ex.execute(job)
	time.Sleep(500 * time.Millisecond)

	got, ok := ex.store.Get(job.ID)
	if !ok {
		t.Fatal("job not found")
	}
	if got.Status != JobFailed {
		t.Errorf("status = %q, want failed", got.Status)
	}
}
