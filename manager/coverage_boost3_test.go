package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c9s/bbgo/saas/manager/pool"
)

// --- BacktestExecutor.execute with hooks ---

func TestBacktestExecutor_Execute_WithSync(t *testing.T) {
	jobDir := t.TempDir()
	btStore := NewBacktestJobStore(jobDir)
	job := &BacktestJob{
		ID:        "bt-sync-test",
		UserID:    "u1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-01-31",
		NeedSync:  true,
	}
	btStore.Create(job)

	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:            &Config{},
		store:          store,
		checkRunningFn: func(string) (bool, error) { return true, nil },
		syncBacktestFn: func(uid, ex, sym, start, end string) (string, error) {
			return "synced 500 klines", nil
		},
		runBacktestFn: func(uid, jid string, yaml []byte) ([]byte, error) {
			return []byte("backtest output"), nil
		},
	}

	reportDir := filepath.Join(store.InstanceDir("u1", "live", "grid2-BTCUSDT"), "backtest", job.ID)
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte(`{"totalPnl":50}`), 0o644)

	btStore.AcquireSlot()
	ex := NewBacktestExecutor(btStore, cm, nil, nil, store.Defaults())
	ex.syncFn = cm.syncBacktestFn
	ex.runFn = cm.runBacktestFn
	ex.reportFn = func(uid, jid string) (json.RawMessage, []byte, error) {
		return json.RawMessage(`{"totalPnl":50}`), nil, nil
	}

	ex.execute(job)

	updated, ok := btStore.Get(job.ID)
	if !ok {
		t.Fatal("job not found")
	}
	if updated.Status != JobCompleted {
		t.Errorf("status = %q, want %q", updated.Status, JobCompleted)
	}
}

func TestBacktestExecutor_Execute_SyncFails(t *testing.T) {
	jobDir := t.TempDir()
	btStore := NewBacktestJobStore(jobDir)
	job := &BacktestJob{
		ID:        "bt-sync-fail",
		UserID:    "u1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-01-31",
		NeedSync:  true,
	}
	btStore.Create(job)

	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:            &Config{},
		store:          store,
		checkRunningFn: func(string) (bool, error) { return true, nil },
		syncBacktestFn: func(uid, ex, sym, start, end string) (string, error) {
			return "", fmt.Errorf("sync error")
		},
	}

	btStore.AcquireSlot()
	ex := NewBacktestExecutor(btStore, cm, nil, nil, nil)
	ex.syncFn = cm.syncBacktestFn

	ex.execute(job)

	updated, _ := btStore.Get(job.ID)
	if updated.Status != JobFailed {
		t.Errorf("status = %q, want %q", updated.Status, JobFailed)
	}
}

func TestBacktestExecutor_Execute_BacktestFails(t *testing.T) {
	jobDir := t.TempDir()
	btStore := NewBacktestJobStore(jobDir)
	job := &BacktestJob{
		ID:        "bt-fail",
		UserID:    "u1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-01-31",
		NeedSync:  false,
	}
	btStore.Create(job)

	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:            &Config{},
		store:          store,
		checkRunningFn: func(string) (bool, error) { return true, nil },
		runBacktestFn:  func(uid, jid string, yaml []byte) ([]byte, error) { return nil, fmt.Errorf("backtest crashed") },
	}

	btStore.AcquireSlot()
	ex := NewBacktestExecutor(btStore, cm, nil, nil, store.Defaults())
	ex.runFn = cm.runBacktestFn

	ex.execute(job)

	updated, _ := btStore.Get(job.ID)
	if updated.Status != JobFailed {
		t.Errorf("status = %q, want %q", updated.Status, JobFailed)
	}
}

func TestBacktestExecutor_Execute_ReportReadFails(t *testing.T) {
	jobDir := t.TempDir()
	btStore := NewBacktestJobStore(jobDir)
	job := &BacktestJob{
		ID:        "bt-noreport",
		UserID:    "u1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-01-31",
		NeedSync:  false,
	}
	btStore.Create(job)

	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:            &Config{},
		store:          store,
		checkRunningFn: func(string) (bool, error) { return true, nil },
		runBacktestFn:  func(uid, jid string, yaml []byte) ([]byte, error) { return []byte("ok"), nil },
	}

	btStore.AcquireSlot()
	ex := NewBacktestExecutor(btStore, cm, nil, nil, store.Defaults())
	ex.runFn = cm.runBacktestFn
	ex.reportFn = func(uid, jid string) (json.RawMessage, []byte, error) {
		return nil, nil, fmt.Errorf("no report file")
	}

	ex.execute(job)

	updated, _ := btStore.Get(job.ID)
	if updated.Status != JobCompleted {
		t.Errorf("status = %q, want %q", updated.Status, JobCompleted)
	}
}

func TestBacktestExecutor_Submit_ServerBusy(t *testing.T) {
	jobDir := t.TempDir()
	btStore := NewBacktestJobStore(jobDir)
	btStore.AcquireSlot()
	btStore.AcquireSlot()

	cm := &ContainerManager{cfg: &Config{}}
	ex := NewBacktestExecutor(btStore, cm, nil, nil, nil)

	job := &BacktestJob{ID: "bt-busy", UserID: "u1", Strategy: "grid2"}
	err := ex.Submit(job)
	if err == nil {
		t.Fatal("expected error for server busy")
	}
	if !strings.Contains(err.Error(), "busy") {
		t.Errorf("error = %v", err)
	}
}

// --- BacktestExecutor: syncBacktest/runBacktest with hooks ---

func TestBacktestExecutor_SyncBacktest_WithHook(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	cm := &ContainerManager{cfg: &Config{}, store: store}
	btStore := NewBacktestJobStore(t.TempDir())

	ex := NewBacktestExecutor(btStore, cm, nil, nil, nil)
	ex.syncFn = func(uid, exch, sym, start, end string) (string, error) {
		return "synced", nil
	}

	out, err := ex.syncBacktest("u1", "binance", "BTCUSDT", "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "synced" {
		t.Errorf("got %q", out)
	}
}

func TestBacktestExecutor_RunBacktest_WithHook(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	cm := &ContainerManager{cfg: &Config{}, store: store}
	btStore := NewBacktestJobStore(t.TempDir())

	ex := NewBacktestExecutor(btStore, cm, nil, nil, nil)
	ex.runFn = func(uid, jid string, yaml []byte) ([]byte, error) {
		return []byte("result"), nil
	}

	out, err := ex.runBacktest("u1", "bt-1", []byte("yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "result" {
		t.Errorf("got %q", string(out))
	}
}

// --- container_recovery.go: RecoverUsers dead container recovered ---

func TestRecoverUsers_DeadContainerRecovered(t *testing.T) {
	p := pool.New(5)
	defer p.Release()
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:   &Config{},
		pool:  p,
		store: store,
		checkRunningFn: func(name string) (bool, error) { return false, nil },
		dockerFn: func(args ...string) (string, error) {
			if args[0] == "inspect" {
				return "exited", nil
			}
			if args[0] == "start" {
				return "", nil
			}
			return "", nil
		},
	}
	results := cm.RecoverUsers([]UserMode{{UserID: "u1", Mode: "live"}})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusRunning {
		t.Errorf("status = %q", results[0].Status)
	}
}

// --- container_recovery.go: CheckAndRecover with CreateAndStartInstance ---

func TestCheckAndRecover_DeadNeedsRecreate(t *testing.T) {
	p := pool.New(5)
	defer p.Release()
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:   &Config{DockerNetwork: "net", DataVolume: "vol", BBGOImage: "img", BBGOPort: 8080, BBGOGRPCPort: 9090},
		pool:  p,
		store: store,
		checkRunningFn: func(name string) (bool, error) { return false, nil },
		dockerFn: func(args ...string) (string, error) {
			if args[0] == "inspect" {
				return "dead", nil
			}
			if args[0] == "run" {
				return "container-id", nil
			}
			return "", nil
		},
	}
	instances := []StrategyInstance{
		{InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live", Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT"},
	}
	results := cm.CheckAndRecover(instances)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Alive || !results[0].Restarted {
		t.Errorf("should be alive and restarted: %+v", results[0])
	}
}

// --- container.go: SyncBacktest/RunBacktest with MarketDataAddr ---

func TestSyncBacktest_NoHook_WithMarketDataAddr(t *testing.T) {
	dir := t.TempDir()
	cm := &ContainerManager{
		cfg:   &Config{DataDir: dir, DockerNetwork: "bbgo-net", DataVolume: "bbgo-data", BBGOImage: "bbgo-base:latest", MarketDataAddr: "marketdata:9090"},
		store: nil,
		dockerFn: func(args ...string) (string, error) {
			for _, a := range args {
				if a == "MARKET_DATA_SERVICE_URL=marketdata:9090" {
					return "sync with marketdata", nil
				}
			}
			return "sync without marketdata", nil
		},
	}
	result, err := cm.SyncBacktest("u1", "binance", "BTCUSDT", "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "marketdata") {
		t.Errorf("got %q", result)
	}
}

func TestRunBacktest_NoHook_WithMarketDataAddr(t *testing.T) {
	dir := t.TempDir()
	cm := &ContainerManager{
		cfg:   &Config{DataDir: dir, DockerNetwork: "bbgo-net", DataVolume: "bbgo-data", BBGOImage: "bbgo-base:latest", MarketDataAddr: "marketdata:9090"},
		store: nil,
		dockerFn: func(args ...string) (string, error) {
			for _, a := range args {
				if a == "MARKET_DATA_SERVICE_URL=marketdata:9090" {
					return "backtest with marketdata", nil
				}
			}
			return "backtest without", nil
		},
	}
	result, err := cm.RunBacktest("u1", "bt-1", []byte("strategy: grid2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result), "marketdata") {
		t.Errorf("got %q", string(result))
	}
}

// --- sync.go: SyncCredential / DeleteCredential nil paths ---

func TestSyncCredential_NilSupa(t *testing.T) {
	s := &Syncer{supa: nil}
	s.SyncCredential(ExchangeCredential{UserID: "u1", Exchange: "binance"})
}

func TestSyncDeleteCredential_NilCreds(t *testing.T) {
	s := &Syncer{creds: nil}
	s.DeleteCredential("u1", "binance", false)
}

// --- api.go: RunBacktest HTTP handler ---

func TestAPI_RunBacktest_Handler(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: testUUID, Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }
	api.container.runBacktestFn = func(uid, jid string, yaml []byte) ([]byte, error) {
		return []byte("backtest result output"), nil
	}

	w := doRequest(r, "POST", "/api/backtest", map[string]any{
		"strategy":   "grid2",
		"config":     map[string]any{"gridNumber": 5},
		"exchange":   "binance",
		"start_time": "2024-01-01",
		"end_time":   "2024-01-31",
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_RunBacktest_InvalidBody(t *testing.T) {
	_, r := setupHandlerAPI(t)

	req := httptest.NewRequest("POST", "/api/backtest", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

// --- api.go: SyncBacktestData HTTP handler ---

func TestAPI_SyncBacktestData_Handler(t *testing.T) {
	api, r := setupHandlerAPI(t)
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: testUUID, Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }
	api.container.syncBacktestFn = func(uid, ex, sym, start, end string) (string, error) {
		return "synced 1000 klines", nil
	}

	w := doRequest(r, "POST", "/api/backtest/sync", map[string]any{
		"exchange":   "binance",
		"symbol":     "BTCUSDT",
		"start_time": "2024-01-01",
		"end_time":   "2024-01-31",
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// --- api.go: SubmitBacktest HTTP handler ---

func TestAPI_SubmitBacktest_BadJSON(t *testing.T) {
	_, r := setupHandlerAPI(t)

	req := httptest.NewRequest("POST", "/api/backtest/submit", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Token", "test-token")
	req.Header.Set("X-User-Id", testUUID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

// --- api.go: GetBacktestJob ---

func TestAPI_GetBacktestJob_Missing(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/backtest/jobs/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

// --- backtest_job.go: BacktestJobStore operations ---

func TestBacktestJobStore_AcquireAndRelease(t *testing.T) {
	btStore := NewBacktestJobStore(t.TempDir())
	if !btStore.AcquireSlot() {
		t.Error("should acquire first slot")
	}
	if !btStore.AcquireSlot() {
		t.Error("should acquire second slot")
	}
	if btStore.AcquireSlot() {
		t.Error("should not acquire third slot")
	}
	btStore.ReleaseSlot()
	if !btStore.AcquireSlot() {
		t.Error("should acquire after release")
	}
}

func TestBacktestJobStore_CreateGet(t *testing.T) {
	btStore := NewBacktestJobStore(t.TempDir())
	now := time.Now()
	job := &BacktestJob{
		ID:        "bt-test-1",
		UserID:    "u1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-01-31",
		Status:    JobPending,
		CreatedAt: now,
	}
	btStore.Create(job)

	got, ok := btStore.Get("bt-test-1")
	if !ok {
		t.Fatal("job not found")
	}
	if got.Strategy != "grid2" {
		t.Errorf("strategy = %q", got.Strategy)
	}
}

func TestBacktestJobStore_ListByUser_ThreeUsers(t *testing.T) {
	btStore := NewBacktestJobStore(t.TempDir())
	now := time.Now()
	btStore.Create(&BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", CreatedAt: now})
	btStore.Create(&BacktestJob{ID: "bt-2", UserID: "u1", Strategy: "supertrend", CreatedAt: now})
	btStore.Create(&BacktestJob{ID: "bt-3", UserID: "u2", Strategy: "grid2", CreatedAt: now})

	jobs := btStore.ListByUser("u1")
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs for u1, got %d", len(jobs))
	}
}

func TestBacktestJobStore_UpdateStatus_Pending(t *testing.T) {
	btStore := NewBacktestJobStore(t.TempDir())
	now := time.Now()
	btStore.Create(&BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Status: JobPending, CreatedAt: now})

	btStore.UpdateStatus("bt-1", JobRunning, "running...")

	got, _ := btStore.Get("bt-1")
	if got.Status != JobRunning {
		t.Errorf("status = %q", got.Status)
	}
}

func TestBacktestJobStore_FailJob_FromRunning(t *testing.T) {
	btStore := NewBacktestJobStore(t.TempDir())
	now := time.Now()
	btStore.Create(&BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Status: JobRunning, CreatedAt: now})

	btStore.FailJob("bt-1", "crashed", "runtime error")

	got, _ := btStore.Get("bt-1")
	if got.Status != JobFailed {
		t.Errorf("status = %q", got.Status)
	}
	if got.Error != "runtime error" {
		t.Errorf("error = %q", got.Error)
	}
	if got.Progress != "crashed" {
		t.Errorf("progress = %q", got.Progress)
	}
}

// --- pool.Pool basic operations ---

func TestPool_SubmitAndWait(t *testing.T) {
	p := pool.New(3)
	defer p.Release()
	var count int
	for i := 0; i < 10; i++ {
		p.Submit(func() {
			count++
		})
	}
	p.Wait()
	if count != 10 {
		t.Errorf("expected 10, got %d", count)
	}
}
