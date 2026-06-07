package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- sync.go: Syncer.MarkCredentialsVerified no creds found ---

func TestSyncer_MarkCredentialsVerified_NoCredsV2(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	s := NewSyncerWithCreds(nil, creds)
	s.MarkCredentialsVerified(testUUID, ModeLive, []StrategyEntry{
		{Exchange: "binance"},
	})
}

// --- sync.go: Syncer.MarkCredentialsVerified skips exchange mismatch ---

func TestSyncer_MarkCredentialsVerified_ExchangeMismatch(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	creds.Upsert(ExchangeCredential{
		ID: "c1", UserID: testUUID, Exchange: "bybit",
		APIKeyEncrypted: "k", APISecretEncrypted: "s",
		IsTestnet: false, IsVerified: false,
	})
	s := NewSyncerWithCreds(nil, creds)
	s.MarkCredentialsVerified(testUUID, ModeLive, []StrategyEntry{
		{Exchange: "binance"},
	})
	list, _ := creds.List(testUUID)
	if len(list) > 0 && list[0].IsVerified {
		t.Error("bybit credential should not be verified for binance strategy")
	}
}

// --- sync.go: Syncer.MarkCredentialsVerified with testnet cred but live mode ---

func TestSyncer_MarkCredentialsVerified_TestnetMismatch(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	creds.Upsert(ExchangeCredential{
		ID: "c1", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: "k", APISecretEncrypted: "s",
		IsTestnet: true, IsVerified: false,
	})
	s := NewSyncerWithCreds(nil, creds)
	s.MarkCredentialsVerified(testUUID, ModeLive, []StrategyEntry{
		{Exchange: "binance"},
	})
	list, _ := creds.List(testUUID)
	if len(list) > 0 && list[0].IsVerified {
		t.Error("testnet credential should not be verified for live mode")
	}
}

// --- container.go: cleanupBackups with fewer than keepNewest ---

func TestCleanupBackups_FewerThanKeep(t *testing.T) {
	dir := t.TempDir()
	f, _ := os.Create(filepath.Join(dir, "bk-1.yaml"))
	f.Close()

	cleanupBackups(dir, "bk-", 5)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected 1 file kept, got %d", len(entries))
	}
}

// --- container.go: cleanupBackups bad dir ---

func TestCleanupBackups_BadDir(t *testing.T) {
	cleanupBackups("/nonexistent/dir", "bk-", 2)
}

// --- container.go: InstanceContainerName ---

func TestInstanceContainerName(t *testing.T) {
	cm := testContainerManager(t)
	name := cm.InstanceContainerName(testUUID, "live", "inst-123")
	if name == "" {
		t.Error("expected non-empty container name")
	}
}

// --- container.go: StartAllForUser no instances ---

func TestStartAllForUser_NoInstances(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	errs := cm.StartAllForUser(testUUID, ModeLive)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

// --- container.go: StartAllForUser with mock docker ---

func TestStartAllForUser_WithMockDocker(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }
	cm.apiURLFn = func(string) string { return "http://localhost:8080" }
	cm.checkRunningFn = func(string) (bool, error) { return true, nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	errs := cm.StartAllForUser(testUUID, ModeLive)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

// --- container.go: StopAllForUser with instances ---

func TestStopAllForUser_WithInstances(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }

	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	cm.StopAllForUser(testUUID)
}

// --- container.go: StopAllForUser no instances ---

func TestStopAllForUser_NoInstances(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	cm.StopAllForUser(testUUID)
}

// --- config.go: getEnv ---

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_MGR_GETENV", "42")
	defer os.Unsetenv("TEST_MGR_GETENV")
	if v := getEnv("TEST_MGR_GETENV", "fallback"); v != "42" {
		t.Errorf("got %q", v)
	}
	if v := getEnv("TEST_MGR_MISSING", "fallback"); v != "fallback" {
		t.Errorf("got %q", v)
	}
}

// --- config.go: getEnvInt ---

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_MGR_INT", "99")
	defer os.Unsetenv("TEST_MGR_INT")
	if v := getEnvInt("TEST_MGR_INT", 0); v != 99 {
		t.Errorf("got %d", v)
	}
	if v := getEnvInt("TEST_MGR_MISSING_INT", 42); v != 42 {
		t.Errorf("got %d", v)
	}
	os.Setenv("TEST_MGR_BADINT", "abc")
	defer os.Unsetenv("TEST_MGR_BADINT")
	if v := getEnvInt("TEST_MGR_BADINT", 7); v != 7 {
		t.Errorf("got %d, want fallback for bad int", v)
	}
}

// --- config.go: getEnvSlice ---

func TestGetEnvSlice(t *testing.T) {
	os.Setenv("TEST_MGR_SLICE", "a,b,c")
	defer os.Unsetenv("TEST_MGR_SLICE")
	v := getEnvSlice("TEST_MGR_SLICE", nil)
	if len(v) != 3 || v[0] != "a" {
		t.Errorf("got %v", v)
	}
	v2 := getEnvSlice("TEST_MGR_MISSING_SLICE", []string{"x"})
	if len(v2) != 1 || v2[0] != "x" {
		t.Errorf("got %v", v2)
	}
}

// --- config.go: LoadConfig with valid env ---

func TestLoadConfig_Valid(t *testing.T) {
	os.Setenv("SUPABASE_URL", "https://test.supabase.co")
	os.Setenv("SUPABASE_SERVICE_KEY", "test-key")
	os.Setenv("MANAGER_TOKEN", "test-token")
	os.Setenv("ENCRYPTION_KEY", testEncryptionKey)
	defer func() {
		os.Unsetenv("SUPABASE_URL")
		os.Unsetenv("SUPABASE_SERVICE_KEY")
		os.Unsetenv("MANAGER_TOKEN")
		os.Unsetenv("ENCRYPTION_KEY")
	}()
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8090 {
		t.Errorf("default port = %d", cfg.Port)
	}
	if cfg.SupabaseURL != "https://test.supabase.co" {
		t.Errorf("supabase URL = %q", cfg.SupabaseURL)
	}
}

// --- container_recovery.go: CheckAndRecover empty list ---

func TestCheckAndRecover_Empty(t *testing.T) {
	cm := testContainerManager(t)
	results := cm.CheckAndRecover(nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// --- container_recovery.go: CheckAndRecover dead container, recovery via recreate ---

func TestCheckAndRecover_DeadRecreated(t *testing.T) {
	cm := testContainerManager(t)
	store, _ := newTestStore(t)
	cm.store = store
	cm.checkRunningFn = func(string) (bool, error) { return false, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }
	cm.apiURLFn = func(string) string { return "http://localhost:8080" }

	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	results := cm.CheckAndRecover([]StrategyInstance{*inst})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Restarted {
		t.Error("expected restarted")
	}
}

// --- backtest_job.go: AcquireSlot / ReleaseSlot ---

func TestBacktestJobStore_Slot(t *testing.T) {
	s := NewBacktestJobStore(t.TempDir())
	if !s.AcquireSlot() {
		t.Error("expected slot acquired")
	}
	if !s.AcquireSlot() {
		t.Error("expected second slot acquired")
	}
	if s.AcquireSlot() {
		t.Error("expected no slot available")
	}
	s.ReleaseSlot()
	if !s.AcquireSlot() {
		t.Error("expected slot after release")
	}
	s.ReleaseSlot()
	s.ReleaseSlot()
}

// --- api.go: hasDataForRange small DB ---

func TestAPI_HasDataForRange_SmallDB(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	cm := testContainerManager(t)
	dir := t.TempDir()
	cm.cfg.BacktestSharedDir = dir
	api.container = cm
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "backtest.db"), []byte("tiny"), 0644)
	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-12-31") {
		t.Error("expected false for small DB")
	}
}

// --- api.go: hasDataForRange valid DB with bad date ---

func TestAPI_HasDataForRange_BadDate(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	cm := testContainerManager(t)
	dir := t.TempDir()
	cm.cfg.BacktestSharedDir = dir
	api.container = cm
	os.MkdirAll(dir, 0755)
	data := make([]byte, 2048)
	os.WriteFile(filepath.Join(dir, "backtest.db"), data, 0644)
	if api.hasDataForRange("binance", "BTCUSDT", "not-a-date", "also-bad") {
		t.Error("expected false for bad dates")
	}
}

// --- api.go: uploadLocalToStorage disallowed file ---

func TestAPI_UploadLocalToStorage_Disallowed(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	job := &BacktestJob{ID: "j1", UserID: testUUID}
	if api.uploadLocalToStorage(job, "evil.exe") {
		t.Error("expected false for disallowed file")
	}
}

// --- api.go: uploadLocalToStorage nil storage ---

func TestAPI_UploadLocalToStorage_NoStorage(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	job := &BacktestJob{ID: "j1", UserID: testUUID}
	if api.uploadLocalToStorage(job, "summary.json") {
		t.Error("expected false with no storage client")
	}
}

// --- backtest_job.go: stale running jobs auto-fail on reload ---

func TestBacktestJobStore_StaleAutoFail(t *testing.T) {
	dir := t.TempDir()
	s := NewBacktestJobStore(dir)

	// Create a job, set it running, then write a stale version directly
	oldStart := time.Now().Add(-10 * time.Minute)
	job := &BacktestJob{
		ID:        "stale-job",
		UserID:    testUUID,
		Strategy:  "grid2",
		Status:    JobRunning,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		StartedAt: &oldStart,
	}
	data, _ := json.MarshalIndent(job, "", "  ")
	os.WriteFile(filepath.Join(dir, "backtest-jobs", "stale-job.json"), data, 0644)

	// Re-open the store — stale running job should be auto-failed
	s2 := NewBacktestJobStore(dir)
	got, _ := s2.Get("stale-job")
	if got == nil {
		t.Fatal("job not found")
	}
	if got.Status != JobFailed {
		t.Errorf("stale job should be failed, got %s", got.Status)
	}
	_ = s // keep reference for pool release
}
