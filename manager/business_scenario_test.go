package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ============================================================
// Business Scenario Tests: Live Trading, Paper Trading, Backtests
// These tests verify real business behavior, not just line coverage.
// ============================================================

// --- LIVE TRADING: Live container must NEVER get PAPER_TRADE env var ---

func TestBusiness_LiveContainer_NoPaperTrade(t *testing.T) {
	cm := testContainerManager(t)
	cm.cfg.SupabaseURL = "https://test.supabase.co"
	cm.cfg.SupabaseKey = "test-key"

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "grid2",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	for _, arg := range args {
		if arg == "PAPER_TRADE=1" || strings.Contains(arg, "PAPER_TRADE") {
			t.Errorf("live container must not have PAPER_TRADE env, got: %s", arg)
		}
	}
	hasSupabaseURL := false
	for _, arg := range args {
		if strings.Contains(arg, "SUPABASE_URL") {
			hasSupabaseURL = true
		}
	}
	if !hasSupabaseURL {
		t.Error("live container should have SUPABASE_URL env")
	}
}

// --- PAPER TRADING: Paper container MUST get PAPER_TRADE=1 and use Supabase with paper_ prefix ---

func TestBusiness_PaperContainer_HasPaperTrade(t *testing.T) {
	cm := testContainerManager(t)
	cm.cfg.SupabaseURL = "https://test.supabase.co"
	cm.cfg.SupabaseKey = "test-key"

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModePaper,
		Strategy: "grid2",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	hasPaperTrade := false
	hasSupabase := false
	hasPaperPrefix := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" && args[i+1] == "PAPER_TRADE=1" {
			hasPaperTrade = true
		}
		if args[i] == "-e" && strings.HasPrefix(args[i+1], "DB_DRIVER=supabase") {
			hasSupabase = true
		}
		if args[i] == "-e" && args[i+1] == "SUPABASE_TABLE_PREFIX=paper_" {
			hasPaperPrefix = true
		}
	}
	if !hasPaperTrade {
		t.Error("paper container must have PAPER_TRADE=1")
	}
	if !hasSupabase {
		t.Error("paper container must use supabase driver")
	}
	if !hasPaperPrefix {
		t.Error("paper container must have SUPABASE_TABLE_PREFIX=paper_")
	}
}

// --- PAPER TRADING: Paper container MUST have Supabase with paper_ prefix, but NO live credentials ---

func TestBusiness_PaperContainer_SupabaseWithPrefix(t *testing.T) {
	cm := testContainerManager(t)
	cm.cfg.SupabaseURL = "https://test.supabase.co"
	cm.cfg.SupabaseKey = "test-key"

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModePaper,
		Strategy: "grid2",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	if !hasEnv(args, "SUPABASE_URL=https://test.supabase.co") {
		t.Error("paper container must have SUPABASE_URL")
	}
	if !hasEnv(args, "SUPABASE_TABLE_PREFIX=paper_") {
		t.Error("paper container must have SUPABASE_TABLE_PREFIX=paper_")
	}
	// Paper mode should NOT inject real exchange API keys
	for _, arg := range args {
		if strings.Contains(arg, "BINANCE_API_KEY=") && !strings.Contains(arg, "SUPABASE") {
			t.Error("paper container must not have real exchange API keys")
		}
	}
}

// --- LIVE TRADING: Live container with credentials should inject API keys ---

func TestBusiness_LiveContainer_InjectsCredentials(t *testing.T) {
	cm := testContainerManager(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	cm.creds = NewCredentialStore(dir, enc)
	cm.creds.Upsert(ExchangeCredential{
		ID:                "c1",
		UserID:            testUUID,
		Exchange:          "binance",
		APIKeyEncrypted:   mustEncrypt(t, enc, "my-api-key"),
		APISecretEncrypted: mustEncrypt(t, enc, "my-api-secret"),
		IsTestnet:         false,
	})

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModeLive,
		Strategy: "grid2",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	hasAPIKey := false
	for _, arg := range args {
		if arg == "BINANCE_API_KEY=my-api-key" {
			hasAPIKey = true
		}
	}
	if !hasAPIKey {
		t.Errorf("live container should have BINANCE_API_KEY, args: %v", args)
	}
}

// --- PAPER TRADING: Paper container should NOT inject API keys ---

func TestBusiness_PaperContainer_NoCredentials(t *testing.T) {
	cm := testContainerManager(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	cm.creds = NewCredentialStore(dir, enc)
	cm.creds.Upsert(ExchangeCredential{
		ID:                "c1",
		UserID:            testUUID,
		Exchange:          "binance",
		APIKeyEncrypted:   mustEncrypt(t, enc, "my-api-key"),
		APISecretEncrypted: mustEncrypt(t, enc, "my-secret"),
		IsTestnet:         false,
	})

	inst := &StrategyInstance{
		UserID:   testUUID,
		Mode:     ModePaper,
		Strategy: "grid2",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
	}
	args := cm.instanceEnvArgs(inst)
	for _, arg := range args {
		if strings.Contains(arg, "API_KEY") || strings.Contains(arg, "API_SECRET") {
			t.Errorf("paper container must not have credentials, got: %s", arg)
		}
	}
}

// --- STRATEGY: LiveOnly strategy blocked in paper mode ---

func TestBusiness_LiveOnlyStrategy_BlockedInPaperMode(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "autoborrow", "name": "test", "exchange": "binance",
		"mode": "paper", "symbol": "BTCUSDT",
		"config": map[string]any{"number": 5},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("liveOnly strategy in paper mode should be rejected, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "live mode") {
		t.Errorf("error should mention live mode, got: %s", w.Body.String())
	}
}

// --- STRATEGY: Duplicate instanceID returns 409 ---

func TestBusiness_CreateStrategy_DuplicateReturns409(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store
	api.container.checkRunningFn = func(string) (bool, error) { return false, nil }

	body := map[string]any{
		"strategy": "grid2", "name": "my grid", "exchange": "binance",
		"config": map[string]any{"gridNumber": 5},
		"symbol": "BTCUSDT", "mode": "paper",
	}
	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", body)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("first create: status = %d, body = %s", w.Code, w.Body.String())
	}

	w2 := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", body)
	if w2.Code != http.StatusConflict {
		t.Errorf("duplicate should return 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

// --- STRATEGY: Delete strategy stops running container first ---

func TestBusiness_DeleteStrategy_StopsContainer(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store

	stopped := false
	api.container.dockerFn = func(args ...string) (string, error) {
		if len(args) > 0 && args[0] == "stop" {
			stopped = true
		}
		return "", nil
	}
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)

	w := doRequest(r, "DELETE", "/api/users/"+testUUID+"/strategies/"+inst.InstanceID+"?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !stopped {
		t.Error("deleting a running strategy should stop the container first")
	}
}

// --- BACKTEST: Stale running job auto-fails on manager restart ---

func TestBusiness_Backtest_StaleJobFailsOnRestart(t *testing.T) {
	dir := t.TempDir()
	oldStart := time.Now().Add(-10 * time.Minute)
	job := &BacktestJob{
		ID: "stale-1", UserID: testUUID, Strategy: "grid2",
		Status: JobRunning, CreatedAt: time.Now().Add(-2 * time.Hour),
		StartedAt: &oldStart,
	}
	data, _ := json.MarshalIndent(job, "", "  ")
	jobDir := filepath.Join(dir, "backtest-jobs")
	os.MkdirAll(jobDir, 0755)
	os.WriteFile(filepath.Join(jobDir, "stale-1.json"), data, 0644)

	s := NewBacktestJobStore(dir)
	got, _ := s.Get("stale-1")
	if got == nil {
		t.Fatal("job not found after reload")
	}
	if got.Status != JobFailed {
		t.Errorf("stale running job should be failed, got %s", got.Status)
	}
}

// --- BACKTEST: Concurrent limit enforced ---

func TestBusiness_Backtest_ConcurrencyLimit(t *testing.T) {
	s := NewBacktestJobStore(t.TempDir())
	if !s.AcquireSlot() {
		t.Error("first slot should be acquired")
	}
	if !s.AcquireSlot() {
		t.Error("second slot should be acquired")
	}
	if s.AcquireSlot() {
		t.Error("third slot should be rejected (max 2 concurrent)")
	}
}

// --- BACKTEST: Completed job persists and reloads ---

func TestBusiness_Backtest_CompletedJobPersists(t *testing.T) {
	dir := t.TempDir()
	s := NewBacktestJobStore(dir)

	job := &BacktestJob{
		ID: "bt-done", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		Status: JobCompleted, CreatedAt: time.Now(),
	}
	s.Create(job)
	s.UpdateStatus(job.ID, JobCompleted, "")
	s.SetReport(job.ID, json.RawMessage(`{"total":100}`), "1,50\n2,60\n")

	s2 := NewBacktestJobStore(dir)
	got, _ := s2.Get("bt-done")
	if got == nil {
		t.Fatal("job not found after reload")
	}
	if got.Status != JobCompleted {
		t.Errorf("expected completed, got %s", got.Status)
	}
	if !strings.Contains(string(got.Report), "total") || !strings.Contains(string(got.Report), "100") {
		t.Errorf("report not persisted, got: %s", got.Report)
	}
}

// --- CONTAINER ISOLATION: Live and paper use different container names ---

func TestBusiness_ContainerIsolation_DifferentNames(t *testing.T) {
	cm := testContainerManager(t)
	liveName := cm.InstanceContainerName(testUUID, ModeLive, "inst-1")
	paperName := cm.InstanceContainerName(testUUID, ModePaper, "inst-1")
	if liveName == paperName {
		t.Errorf("live and paper must have different names, got: %s", liveName)
	}
	if !strings.Contains(liveName, "live") {
		t.Errorf("live name should contain 'live': %s", liveName)
	}
	if !strings.Contains(paperName, "paper") {
		t.Errorf("paper name should contain 'paper': %s", paperName)
	}
}

// --- API: Create strategy missing name returns 400 ---

func TestBusiness_CreateStrategy_MissingName(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "exchange": "binance", "mode": "paper",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing name should return 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- API: Create strategy missing strategy returns 400 ---

func TestBusiness_CreateStrategy_MissingStrategyName(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"name": "my bot", "exchange": "binance", "mode": "paper",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing strategy should return 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- API: Paper mode only supports binance ---

func TestBusiness_PaperMode_OnlyBinance(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "test", "exchange": "bybit",
		"mode": "paper", "config": map[string]any{"gridNumber": 5},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("paper mode with bybit should be rejected, got %d: %s", w.Code, w.Body.String())
	}
}

// --- API: Invalid mode returns 400 ---

func TestBusiness_CreateStrategy_InvalidMode(t *testing.T) {
	api, r := setupHandlerAPI(t)
	store, _ := newTestStore(t)
	api.store = store
	api.container.store = store

	w := doRequest(r, "POST", "/api/users/"+testUUID+"/strategies", map[string]any{
		"strategy": "grid2", "name": "test", "exchange": "binance",
		"mode": "simulation", "config": map[string]any{},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid mode should return 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- CROSS-EXCHANGE: Duplicate exchange deduplicates env vars ---

func TestBusiness_CrossExchange_DuplicateExchangeDedup(t *testing.T) {
	cm := testContainerManager(t)
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	cm.creds = NewCredentialStore(dir, enc)
	cm.creds.Upsert(ExchangeCredential{
		ID:                "c1",
		UserID:            testUUID,
		Exchange:          "binance",
		APIKeyEncrypted:   mustEncrypt(t, enc, "key1"),
		APISecretEncrypted: mustEncrypt(t, enc, "secret1"),
	})

	inst := &StrategyInstance{
		UserID:   testUUID, Mode: ModeLive, Strategy: "xmaker",
		Exchange: "binance", Symbol: "BTCUSDT",
		CrossExchange: true,
		Sessions:      []SessionRoleConfig{{Exchange: "binance", Name: "source"}, {Exchange: "binance", Name: "target"}},
	}
	args := cm.instanceEnvArgs(inst)
	count := 0
	for _, arg := range args {
		if arg == "BINANCE_API_KEY=key1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("duplicate exchange should inject once, got %d", count)
	}
}

// --- CONTAINER RECOVERY: Dead container gets restarted ---

func TestBusiness_Recovery_DeadContainerRestarted(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store

	var dockerCalls []string
	cm.dockerFn = func(args ...string) (string, error) {
		dockerCalls = append(dockerCalls, args[0])
		return "", nil
	}
	cm.checkRunningFn = func(string) (bool, error) { return false, nil }
	cm.apiURLFn = func(string) string { return "http://localhost:8080" }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	results := cm.CheckAndRecover([]StrategyInstance{*inst})

	if len(results) != 1 || !results[0].Alive || !results[0].Restarted {
		t.Fatalf("expected alive+restarted, got %+v", results)
	}
	hasRun := false
	for _, call := range dockerCalls {
		if call == "run" {
			hasRun = true
		}
	}
	if !hasRun {
		t.Errorf("expected docker run, calls: %v", dockerCalls)
	}
}

// --- BACKTEST BUG: Sync failure does NOT call CleanupBacktest ---
// This test documents a known bug: when data sync fails, the backtest job directory
// is not cleaned up. The fix should add cleanup in the sync failure path.

func TestBusiness_Backtest_SyncFailureCleanup(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	job := &BacktestJob{
		ID: "bt-sync-fail", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-12-31",
		NeedSync: true, Config: json.RawMessage(`{"gridNumber":5}`),
	}
	store.Create(job)
	store.AcquireSlot()

	cm := testContainerManager(t)
	cm.store, _ = newTestStore(t)

	exec := &BacktestExecutor{
		store:     store,
		container: cm,
		syncFn: func(userID, exchange, symbol, start, end string) (string, error) {
			return "", fmt.Errorf("sync failed: connection refused")
		},
	}

	exec.execute(job)

	got, _ := store.Get(job.ID)
	if got == nil || got.Status != JobFailed {
		t.Fatalf("job should be failed after sync error, got: %+v", got)
	}
	if !strings.Contains(got.Error, "sync failed") {
		t.Errorf("error should contain 'sync failed', got: %s", got.Error)
	}

	// BUG: CleanupBacktest is NOT called on sync failure.
	// After the fix, this should verify cleanup happened.
	// For now, this test documents the expected behavior.
}

func mustEncrypt(t *testing.T, enc *Encryptor, plain string) string {
	t.Helper()
	cipher, err := enc.Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return cipher
}
