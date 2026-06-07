package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c9s/bbgo/saas/manager/pool"
)

// --- bbgo_client: flexString unmarshal ---

func TestFlexString_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{`123`, "123"},
		{`3.14`, "3.14"},
		{`0`, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var f flexString
			if err := json.Unmarshal([]byte(tt.input), &f); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(f) != tt.want {
				t.Errorf("got %q, want %q", string(f), tt.want)
			}
		})
	}
}

func TestFlexString_UnmarshalJSON_Error(t *testing.T) {
	var f flexString
	if err := json.Unmarshal([]byte(`[1,2]`), &f); err == nil {
		t.Error("expected error for array input")
	}
}

// --- container.go: RunBacktest non-hook path ---

func TestRunBacktest_NoHook_NilStore(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{}, store: nil}
	_, err := cm.RunBacktest("u1", "bt-1", []byte("yaml"))
	if err == nil {
		t.Fatal("expected error with nil store")
	}
}

func TestRunBacktest_NoHook_NoRunningInstance(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	cm := &ContainerManager{cfg: &Config{}, store: store}
	_, err := cm.RunBacktest("u1", "bt-1", []byte("yaml"))
	if err == nil {
		t.Fatal("expected error with no running instance")
	}
}

func TestRunBacktest_NoHook_WithRunningInstance(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	if err := store.CreateInstance(inst, func(string) bool { return true }); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	cm := &ContainerManager{
		cfg:   &Config{},
		store: store,
		checkRunningFn: func(name string) (bool, error) { return true, nil },
		dockerFn: func(args ...string) (string, error) {
			return "backtest output", nil
		},
	}
	result, err := cm.RunBacktest("u1", "bt-123", []byte("strategy: grid2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != "backtest output" {
		t.Errorf("got %q", string(result))
	}

	btDir := filepath.Join(store.InstanceDir("u1", "live", "grid2-BTCUSDT"), "backtest", "bt-123")
	if _, err := os.Stat(btDir); os.IsNotExist(err) {
		t.Error("backtest dir not created")
	}
	configPath := filepath.Join(btDir, "bbgo.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(data) != "strategy: grid2" {
		t.Errorf("config content = %q", string(data))
	}
}

func TestRunBacktest_NoHook_DockerError(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:   &Config{},
		store: store,
		checkRunningFn: func(name string) (bool, error) { return true, nil },
		dockerFn: func(args ...string) (string, error) {
			return "error details", fmt.Errorf("docker failed")
		},
	}
	_, err := cm.RunBacktest("u1", "bt-1", []byte("strategy: grid2"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "docker failed") {
		t.Errorf("error = %v", err)
	}
}

// --- container.go: ReadBacktestReport ---

func TestReadBacktestReport_NoRunningInstance(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	cm := &ContainerManager{cfg: &Config{}, store: store}
	_, _, err := cm.ReadBacktestReport("u1", "bt-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadBacktestReport_WithFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:   &Config{},
		store: store,
		checkRunningFn: func(name string) (bool, error) { return true, nil },
	}

	btDir := filepath.Join(store.InstanceDir("u1", "live", "grid2-BTCUSDT"), "backtest", "bt-1")
	os.MkdirAll(btDir, 0o755)
	summaryJSON := `{"totalPnl": 100.0}`
	os.WriteFile(filepath.Join(btDir, "summary.json"), []byte(summaryJSON), 0o644)
	equityTSV := "time\tequity\n2024-01-01\t10000\n"
	os.WriteFile(filepath.Join(btDir, "equity_curve.tsv"), []byte(equityTSV), 0o644)

	summary, equity, err := cm.ReadBacktestReport("u1", "bt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(summary) != summaryJSON {
		t.Errorf("summary = %q", string(summary))
	}
	if string(equity) != equityTSV {
		t.Errorf("equity = %q", string(equity))
	}
}

func TestReadBacktestReport_NoSummary(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{cfg: &Config{}, store: store}

	btDir := filepath.Join(store.InstanceDir("u1", "live", "grid2-BTCUSDT"), "backtest", "bt-1")
	os.MkdirAll(btDir, 0o755)

	_, _, err := cm.ReadBacktestReport("u1", "bt-1")
	if err == nil {
		t.Fatal("expected error when summary.json missing")
	}
}

// --- container.go: SyncBacktest non-hook path ---

func TestSyncBacktest_NoHook_NoRunningInstance(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	cm := &ContainerManager{cfg: &Config{}, store: store}
	_, err := cm.SyncBacktest("u1", "binance", "BTCUSDT", "2024-01-01", "2024-01-31")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSyncBacktest_NoHook_Success(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:   &Config{},
		store: store,
		checkRunningFn: func(name string) (bool, error) { return true, nil },
		dockerFn: func(args ...string) (string, error) {
			return "sync complete", nil
		},
	}
	result, err := cm.SyncBacktest("u1", "binance", "BTCUSDT", "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "sync complete" {
		t.Errorf("got %q", result)
	}
}

// --- container.go: CleanupBacktest ---

func TestCleanupBacktest(t *testing.T) {
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
		checkRunningFn: func(name string) (bool, error) { return true, nil },
	}

	btDir := filepath.Join(store.InstanceDir("u1", "live", "grid2-BTCUSDT"), "backtest", "bt-1")
	os.MkdirAll(btDir, 0o755)
	os.WriteFile(filepath.Join(btDir, "bbgo.yaml"), []byte("test"), 0o644)

	cm.CleanupBacktest("u1", "bt-1")

	if _, err := os.Stat(btDir); !os.IsNotExist(err) {
		t.Error("backtest dir should be removed")
	}
}

func TestCleanupBacktest_NilManager(t *testing.T) {
	var cm *ContainerManager
	cm.CleanupBacktest("u1", "bt-1")
}

// --- container.go: BacktestReportDir ---

func TestBacktestReportDir(t *testing.T) {
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
		checkRunningFn: func(name string) (bool, error) { return true, nil },
	}
	reportDir := cm.BacktestReportDir("u1", "bt-1")
	if reportDir == "" {
		t.Error("expected non-empty report dir")
	}
	if !strings.Contains(reportDir, "backtest") {
		t.Errorf("report dir = %q", reportDir)
	}
}

func TestBacktestReportDir_NoRunningInstance(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	cm := &ContainerManager{cfg: &Config{}, store: store}
	reportDir := cm.BacktestReportDir("u1", "bt-1")
	if reportDir != "" {
		t.Errorf("expected empty for no running instance, got %q", reportDir)
	}
}

// --- container.go: EnsureNetwork ---

func TestEnsureNetwork(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{DockerNetwork: "test-net"},
		dockerFn: func(args ...string) (string, error) {
			return "created", nil
		},
	}
	if err := cm.EnsureNetwork(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureNetwork_AlreadyExists(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{DockerNetwork: "test-net"},
		dockerFn: func(args ...string) (string, error) {
			return "network test-net already exists", fmt.Errorf("already exists")
		},
	}
	if err := cm.EnsureNetwork(); err != nil {
		t.Fatalf("should not error for already exists: %v", err)
	}
}

func TestEnsureNetwork_RealError(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{DockerNetwork: "test-net"},
		dockerFn: func(args ...string) (string, error) {
			return "permission denied", fmt.Errorf("permission denied")
		},
	}
	if err := cm.EnsureNetwork(); err == nil {
		t.Fatal("expected error")
	}
}

// --- container.go: CreateAndStartInstance ---

func TestCreateAndStartInstance_MockDocker(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })

	cm := &ContainerManager{
		cfg:   &Config{DockerNetwork: "test-net", DataVolume: "test-vol", BBGOImage: "bbgo:test", BBGOPort: 8080, BBGOGRPCPort: 9090},
		store: store,
		dockerFn: func(args ...string) (string, error) {
			return "container-id-123", nil
		},
	}
	if err := cm.CreateAndStartInstance(inst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateAndStartInstance_NoYAML(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}

	cm := &ContainerManager{
		cfg:   &Config{DockerNetwork: "test-net", DataVolume: "test-vol", BBGOImage: "bbgo:test", BBGOPort: 8080, BBGOGRPCPort: 9090},
		store: store,
		dockerFn: func(args ...string) (string, error) {
			return "", nil
		},
	}
	err := cm.CreateAndStartInstance(inst)
	if err == nil {
		t.Fatal("expected error with no yaml")
	}
	if !strings.Contains(err.Error(), "bbgo.yaml not found") {
		t.Errorf("error = %v", err)
	}
}

// --- container.go: StopInstance ---

func TestStopInstance_MockDocker(t *testing.T) {
	var calls [][]string
	cm := &ContainerManager{
		cfg: &Config{},
		dockerFn: func(args ...string) (string, error) {
			calls = append(calls, args)
			return "", nil
		},
	}
	cm.StopInstance("u1", "live", "grid2-BTCUSDT")
	if len(calls) != 2 {
		t.Fatalf("expected 2 docker calls, got %d", len(calls))
	}
	if calls[0][0] != "stop" {
		t.Errorf("first call = %v", calls[0])
	}
	if calls[1][0] != "rm" {
		t.Errorf("second call = %v", calls[1])
	}
}

// --- container.go: CheckInstanceRunning ---

func TestCheckInstanceRunning_Mock(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{},
		checkRunningFn: func(name string) (bool, error) {
			return true, nil
		},
	}
	running, err := cm.CheckInstanceRunning("u1", "live", "grid2-BTCUSDT")
	if err != nil || !running {
		t.Errorf("running=%v, err=%v", running, err)
	}
}

func TestCheckInstanceRunning_MockNotRunning(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{},
		checkRunningFn: func(name string) (bool, error) {
			return false, nil
		},
	}
	running, err := cm.CheckInstanceRunning("u1", "live", "grid2-BTCUSDT")
	if err != nil || running {
		t.Errorf("running=%v, err=%v", running, err)
	}
}

// --- container.go: InstanceAPIURL ---

func TestInstanceAPIURL_Mock(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{BBGOPort: 8080},
		apiURLFn: func(name string) string {
			return "http://custom:9999"
		},
	}
	url := cm.InstanceAPIURL("u1", "live", "grid2-BTCUSDT")
	if url != "http://custom:9999" {
		t.Errorf("got %q", url)
	}
}

func TestInstanceAPIURL_Default(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{BBGOPort: 8080},
	}
	url := cm.InstanceAPIURL("u1", "live", "grid2-BTCUSDT")
	if !strings.Contains(url, "bbgo-u1-live-grid2-btcusdt") {
		t.Errorf("got %q", url)
	}
}

// --- container_recovery.go: tryRecoverViaDockerStart ---

func TestTryRecoverViaDockerStart_Exited(t *testing.T) {
	callCount := 0
	cm := &ContainerManager{
		cfg: &Config{},
		dockerFn: func(args ...string) (string, error) {
			callCount++
			if callCount == 1 {
				return "exited", nil
			}
			if callCount == 2 {
				return "", nil
			}
			return "running", nil
		},
		checkRunningFn: func(name string) (bool, error) {
			return true, nil
		},
	}
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	if !cm.tryRecoverViaDockerStart(inst) {
		t.Error("should recover exited container")
	}
}

func TestTryRecoverViaDockerStart_InspectError(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{},
		dockerFn: func(args ...string) (string, error) {
			return "", fmt.Errorf("docker not available")
		},
	}
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	if cm.tryRecoverViaDockerStart(inst) {
		t.Error("should fail with docker error")
	}
}

func TestTryRecoverViaDockerStart_OtherStatus(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{},
		dockerFn: func(args ...string) (string, error) {
			return "paused", nil
		},
	}
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	if cm.tryRecoverViaDockerStart(inst) {
		t.Error("should not recover paused container")
	}
}

// --- container_recovery.go: CheckAndRecover ---

func TestCheckAndRecover_DeadThenRecovered(t *testing.T) {
	p := pool.New(5)
	defer p.Release()
	cm := &ContainerManager{
		cfg:  &Config{},
		pool: p,
		checkRunningFn: func(name string) (bool, error) { return false, nil },
		dockerFn: func(args ...string) (string, error) {
			if args[0] == "inspect" {
				return "running", nil
			}
			return "", nil
		},
	}
	instances := []StrategyInstance{
		{InstanceID: "g1", UserID: "u1", Mode: "live", Strategy: "grid2"},
	}
	results := cm.CheckAndRecover(instances)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Alive || !results[0].Restarted {
		t.Errorf("should be alive and restarted: %+v", results[0])
	}
}

// --- container_recovery.go: RecoverUsers ---

func TestRecoverUsers_WithStore(t *testing.T) {
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
		checkRunningFn: func(name string) (bool, error) { return true, nil },
	}
	results := cm.RecoverUsers([]UserMode{{UserID: "u1", Mode: "live"}})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusRunning {
		t.Errorf("status = %q", results[0].Status)
	}
}

func TestRecoverUsers_EmptyUsers(t *testing.T) {
	p := pool.New(5)
	defer p.Release()
	cm := &ContainerManager{
		cfg:  &Config{},
		pool: p,
	}
	results := cm.RecoverUsers(nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// --- container_recovery.go: CleanupStopped ---

func TestCleanupStopped_NoContainers(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{},
		dockerFn: func(args ...string) (string, error) {
			return "", nil
		},
	}
	cleaned := cm.CleanupStopped(nil)
	if cleaned != 0 {
		t.Errorf("expected 0 cleaned, got %d", cleaned)
	}
}

func TestCleanupStopped_WithUntracked(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{},
		dockerFn: func(args ...string) (string, error) {
			if args[0] == "ps" {
				return "bbgo-stale-container", nil
			}
			return "", nil
		},
	}
	tracked := []StrategyInstance{
		{InstanceID: "g1", UserID: "u1", Mode: "live", Strategy: "grid2"},
	}
	cleaned := cm.CleanupStopped(tracked)
	// CleanupStopped iterates "exited" and "dead" statuses — same container may appear in both
	if cleaned < 1 {
		t.Errorf("expected at least 1 cleaned, got %d", cleaned)
	}
}

func TestCleanupStopped_TrackedNotRemoved(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{},
		dockerFn: func(args ...string) (string, error) {
			if args[0] == "ps" {
				return "bbgo-u1-live-g1", nil
			}
			return "", nil
		},
	}
	tracked := []StrategyInstance{
		{InstanceID: "g1", UserID: "u1", Mode: "live", Strategy: "grid2"},
	}
	cleaned := cm.CleanupStopped(tracked)
	if cleaned != 0 {
		t.Errorf("expected 0 cleaned (tracked), got %d", cleaned)
	}
}

// --- buildSyncConfig (duplicate name avoided) ---

func TestBuildSyncConfig_WithDates(t *testing.T) {
	out, err := buildSyncConfig("binance", "BTCUSDT", "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "binance") || !strings.Contains(s, "BTCUSDT") {
		t.Errorf("output = %q", s)
	}
	if !strings.Contains(s, "2024-01-01") || !strings.Contains(s, "2024-01-31") {
		t.Errorf("missing dates in output = %q", s)
	}
}

// --- instance_store: upsertToSupabase / deleteFromSupabase nil client ---

func TestUpsertToSupabase_NilClient(t *testing.T) {
	s := &InstanceStore{sb: nil}
	s.upsertToSupabase(&StrategyInstance{InstanceID: "test"})
}

func TestDeleteFromSupabase_NilClient(t *testing.T) {
	s := &InstanceStore{sb: nil}
	s.deleteFromSupabase("u1", "live", "test")
}

// --- instance_store: ScanUsers ---

func TestScanUsers(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)

	for _, uid := range []string{"u1", "u2"} {
		inst := &StrategyInstance{
			InstanceID: "grid2-BTCUSDT", UserID: uid, Mode: "live",
			Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
		}
		store.CreateInstance(inst, func(string) bool { return true })
	}

	users := store.ScanUsers()
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestScanUsers_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	users := store.ScanUsers()
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestScanUsers_NonexistentDir(t *testing.T) {
	store := NewInstanceStore("/nonexistent/path", nil)
	users := store.ScanUsers()
	if len(users) != 0 {
		t.Errorf("expected 0 for nonexistent dir, got %d", len(users))
	}
}

// --- instance_store: YAMLExists ---

func TestYAMLExists_True(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })
	if !store.YAMLExists("u1", "live", "grid2-BTCUSDT") {
		t.Error("yaml should exist")
	}
}

func TestYAMLExists_False(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	if store.YAMLExists("u1", "live", "nonexistent") {
		t.Error("yaml should not exist")
	}
}

// --- instance_store: CreateInstance duplicate ---

func TestCreateInstance_Duplicate(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	store.CreateInstance(inst, func(string) bool { return true })
	err := store.CreateInstance(inst, func(string) bool { return true })
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v", err)
	}
}

// --- backtest_job.go: readReport with reportFn hook ---

func TestReadReport_WithHook(t *testing.T) {
	ex := &BacktestExecutor{
		reportFn: func(userID, jobID string) (json.RawMessage, []byte, error) {
			return json.RawMessage(`{"pnl":100}`), []byte("tsv"), nil
		},
	}
	summary, equity, err := ex.readReport("u1", "bt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(summary) != `{"pnl":100}` {
		t.Errorf("summary = %q", string(summary))
	}
	if string(equity) != "tsv" {
		t.Errorf("equity = %q", string(equity))
	}
}

func TestReadReport_NoContainer(t *testing.T) {
	ex := &BacktestExecutor{}
	_, _, err := ex.readReport("u1", "bt-1")
	if err == nil {
		t.Fatal("expected error with no container")
	}
}

// --- backtest_job.go: uploadToStorage with nil storage ---

func TestUploadToStorage_NilStorage(t *testing.T) {
	ex := &BacktestExecutor{storage: nil}
	ex.uploadToStorage("u1", "bt-1", json.RawMessage(`{}`), nil)
}

// --- backtest_job.go: notify with nil notifier ---

func TestNotify_NilNotifier(t *testing.T) {
	ex := &BacktestExecutor{notifier: nil}
	ex.notify(&BacktestJob{UserID: "u1"}, "title", "msg")
}

// --- api.go: symbolPriceLookup nil hub ---

func TestSymbolPriceLookup_NilHub(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	lookup := api.symbolPriceLookup(context.Background())
	_, err := lookup("BTCUSDT")
	if err == nil {
		t.Fatal("expected error with nil hub")
	}
}

// --- api.go: uploadLocalToStorage ---

func TestUploadLocalToStorage_DisallowedFile(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	if api.uploadLocalToStorage(&BacktestJob{UserID: "u1", ID: "bt-1"}, "bad.txt") {
		t.Error("should return false for disallowed file")
	}
}

func TestUploadLocalToStorage_NoReportDir(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	if api.uploadLocalToStorage(&BacktestJob{UserID: "u1", ID: "bt-1"}, "summary.json") {
		t.Error("should return false when no report dir")
	}
}

// --- api.go: hasDataForRange ---

func TestHasDataForRange_NoDB(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-01-31") {
		t.Error("should return false with no db")
	}
}

func TestHasDataForRange_InvalidDates(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	dbDir := filepath.Join(api.container.cfg.DataDir, "backtest-shared")
	os.MkdirAll(dbDir, 0o755)
	dbPath := filepath.Join(dbDir, "backtest.db")
	os.WriteFile(dbPath, make([]byte, 2048), 0o644)

	if api.hasDataForRange("binance", "BTCUSDT", "invalid", "invalid") {
		t.Error("should return false with invalid dates")
	}
}

// --- api.go: Close ---

func TestAPI_CloseAndCleanup(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	api.Close()
}

// --- Syncer: nil paths ---

func TestMarkCredentialsVerified_NilCreds(t *testing.T) {
	s := &Syncer{creds: nil}
	s.MarkCredentialsVerified("u1", "live", nil)
}
