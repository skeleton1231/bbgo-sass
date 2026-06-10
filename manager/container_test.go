package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var errFake = errors.New("fake error")

func TestNewContainerManager(t *testing.T) {
	cfg := &Config{ManagerToken: "test", DataDir: t.TempDir()}
	cm := NewContainerManager(cfg, nil, nil, nil)
	if cm == nil || cm.cfg != cfg {
		t.Error("NewContainerManager failed")
	}
}

func TestContainerManager_InstanceContainerName(t *testing.T) {
	cfg := &Config{ManagerToken: "test"}
	cm := NewContainerManager(cfg, nil, nil, nil)
	name := cm.InstanceContainerName("user1", "live", "grid2-btcusdt")
	if name != "bbgo-user1-live-grid2-btcusdt" {
		t.Errorf("name = %q", name)
	}
}

func TestContainerManager_InstanceAPIURL(t *testing.T) {
	cfg := &Config{ManagerToken: "test", BBGOPort: 8080}
	cm := NewContainerManager(cfg, nil, nil, nil)
	url := cm.InstanceAPIURL("user1", "live", "grid2-btcusdt")
	if url != "http://bbgo-user1-live-grid2-btcusdt:8080" {
		t.Errorf("url = %q", url)
	}
}

func TestContainerManager_InstanceAPIURL_CustomFn(t *testing.T) {
	cfg := &Config{ManagerToken: "test"}
	cm := NewContainerManager(cfg, nil, nil, nil)
	cm.apiURLFn = func(name string) string { return "http://custom:9999" }
	url := cm.InstanceAPIURL("user1", "live", "grid2-btcusdt")
	if url != "http://custom:9999" {
		t.Errorf("url = %q", url)
	}
}

func TestContainerManager_InstanceGRPCAddr(t *testing.T) {
	cfg := &Config{ManagerToken: "test", BBGOGRPCPort: 9090}
	cm := NewContainerManager(cfg, nil, nil, nil)
	addr := cm.InstanceGRPCAddr("user1", "live", "grid2-btcusdt")
	if addr != "bbgo-user1-live-grid2-btcusdt:9090" {
		t.Errorf("addr = %q", addr)
	}
}

func TestContainerManager_IsInstanceRunning_True(t *testing.T) {
	cm := &ContainerManager{
		checkRunningFn: func(name string) (bool, error) { return true, nil },
	}
	if !cm.IsInstanceRunning("u1", "live", "inst1") {
		t.Error("expected running")
	}
}

func TestContainerManager_IsInstanceRunning_False(t *testing.T) {
	cm := &ContainerManager{
		checkRunningFn: func(name string) (bool, error) { return false, nil },
	}
	if cm.IsInstanceRunning("u1", "live", "inst1") {
		t.Error("expected not running")
	}
}

func TestContainerManager_InstanceLogs(t *testing.T) {
	cm := &ContainerManager{
		logsFn: func(name string) (string, error) { return "log output", nil },
	}
	out, err := cm.InstanceLogs("u1", "live", "inst1", "100")
	if err != nil || out != "log output" {
		t.Errorf("logs = %q, err = %v", out, err)
	}
}

func TestContainerManager_Docker_UsesHook(t *testing.T) {
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) { return "hooked", nil },
	}
	out, err := cm.docker("ps")
	if err != nil || out != "hooked" {
		t.Errorf("docker = %q, err = %v", out, err)
	}
}

func TestContainerManager_DockerLong_UsesHook(t *testing.T) {
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) { return "long hooked", nil },
	}
	out, err := cm.dockerLong("run", "foo")
	if err != nil || out != "long hooked" {
		t.Errorf("dockerLong = %q, err = %v", out, err)
	}
}

func TestContainerManager_EnsureNetwork_AlreadyExists(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{DockerNetwork: "test-net"},
		dockerFn: func(args ...string) (string, error) {
			return "network test-net already exists", nil
		},
	}
	if err := cm.EnsureNetwork(); err != nil {
		t.Errorf("expected nil for already exists, got %v", err)
	}
}

func TestContainerManager_EnsureNetwork_Error(t *testing.T) {
	cm := &ContainerManager{
		cfg: &Config{DockerNetwork: "test-net"},
		dockerFn: func(args ...string) (string, error) {
			return "some error", errFake
		},
	}
	if err := cm.EnsureNetwork(); err == nil {
		t.Error("expected error")
	}
}

func TestContainerManager_StopInstance(t *testing.T) {
	var calls [][]string
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) {
			calls = append(calls, args)
			return "", nil
		},
	}
	cm.StopInstance("u1", "live", "inst1")
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

func TestContainerManager_DiscoverContainers(t *testing.T) {
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) {
			return "bbgo-user1-live-inst1\nbbgo-user2-paper-inst2\nbbgo-marketdata-live-main\n", nil
		},
	}
	instances := cm.DiscoverContainers()
	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d: %+v", len(instances), instances)
	}
	if instances[0].UserID != "user1" || instances[0].Mode != "live" {
		t.Errorf("instance 0 = %+v", instances[0])
	}
	if instances[1].UserID != "user2" || instances[1].Mode != "paper" {
		t.Errorf("instance 1 = %+v", instances[1])
	}
}

func TestContainerManager_DiscoverContainers_Empty(t *testing.T) {
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) { return "", nil },
	}
	if instances := cm.DiscoverContainers(); len(instances) != 0 {
		t.Errorf("expected 0, got %d", len(instances))
	}
}

func TestContainerManager_DiscoverContainers_Error(t *testing.T) {
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) { return "", errFake },
	}
	if instances := cm.DiscoverContainers(); instances != nil {
		t.Errorf("expected nil on error, got %v", instances)
	}
}

func TestContainerManager_StopAllForUser(t *testing.T) {
	store, _ := newTestStore(t)
	var calls [][]string
	cm := &ContainerManager{
		cfg:   &Config{},
		store: store,
		dockerFn: func(args ...string) (string, error) {
			calls = append(calls, args)
			return "", nil
		},
	}
	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	cm.StopAllForUser(testUUID)
	if len(calls) == 0 {
		t.Error("expected docker calls")
	}
}

func TestContainerManager_StartAllForUser(t *testing.T) {
	store, _ := newTestStore(t)
	cm := &ContainerManager{
		cfg:   &Config{ManagerToken: "test"},
		store: store,
		dockerFn: func(args ...string) (string, error) {
			if args[0] == "run" {
				return "container-id", nil
			}
			return "", nil
		},
	}
	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	errs := cm.StartAllForUser(testUUID, "live")
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestExchangeEnvPrefix(t *testing.T) {
	tests := []struct {
		exchange string
		prefix   string
	}{
		{"binance", "BINANCE"}, {"okex", "OKEX"}, {"kucoin", "KUCOIN"},
		{"bybit", "BYBIT"}, {"bitget", "BITGET"}, {"max", "MAX"},
		{"coinbase", "COINBASE"}, {"bitfinex", "BITFINEX"}, {"unknown", "EXCHANGE"},
	}
	for _, tt := range tests {
		if got := exchangeEnvPrefix(tt.exchange); got != tt.prefix {
			t.Errorf("exchangeEnvPrefix(%q) = %q, want %q", tt.exchange, got, tt.prefix)
		}
	}
}

func TestContainerManager_InstanceEnvArgs_PaperMode(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{}}
	inst := &StrategyInstance{
		UserID: "u1", Mode: ModePaper, Exchange: "binance", InstanceID: "inst1",
	}
	args := cm.instanceEnvArgs(inst)
	if !hasEnv(args, "PAPER_TRADE=1") {
		t.Error("paper mode should have PAPER_TRADE=1")
	}
	if !hasEnv(args, "DB_DRIVER=postgresql") {
		t.Error("paper mode should have DB_DRIVER=postgresql")
	}
	if !hasEnv(args, "SUPABASE_TABLE_PREFIX=paper_") {
		t.Error("paper mode should have SUPABASE_TABLE_PREFIX=paper_")
	}
}


func TestContainerManager_InstanceEnvArgs_LiveMode(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{
		SupabaseURL: "https://supa.example.com", SupabaseKey: "test-key",
		SupabaseDBURL: "postgresql://test:test@localhost:5432/postgres",
	}}
	inst := &StrategyInstance{
		UserID: "u1", Mode: ModeLive, Exchange: "binance", InstanceID: "inst1",
	}
	args := cm.instanceEnvArgs(inst)
	if hasEnv(args, "PAPER_TRADE=1") {
		t.Error("live mode should NOT have PAPER_TRADE=1")
	}
	if !hasEnv(args, "DB_DRIVER=postgresql") {
		t.Error("live mode should have DB_DRIVER=postgresql")
	}
	if !hasEnv(args, "BBGO_USER_ID=u1") {
		t.Error("live mode should have BBGO_USER_ID")
	}
	if hasEnv(args, "SUPABASE_TABLE_PREFIX=paper_") {
		t.Error("live mode should NOT have SUPABASE_TABLE_PREFIX")
	}
}


func TestContainerManager_InstanceEnvArgs_MarketDataAddr(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{MarketDataAddr: "marketdata:9090"}}
	inst := &StrategyInstance{
		UserID: "u1", Mode: ModeLive, Exchange: "binance", InstanceID: "inst1",
	}
	args := cm.instanceEnvArgs(inst)
	if !hasEnv(args, "MARKET_DATA_SERVICE_URL=marketdata:9090") {
		t.Error("should have MARKET_DATA_SERVICE_URL")
	}
}

func TestContainerManager_InstanceResourceArgs(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{
		InstanceResources: ContainerResources{
			Memory: "256m", MemorySwap: "512m", CPUs: "0.5",
			PidsLimit: 100, LogMaxSize: "10m", LogMaxFile: 3,
		},
	}}
	args := cm.instanceResourceArgs()
	for _, want := range []string{"--memory", "256m", "--memory-swap", "512m",
		"--cpus", "0.5", "--pids-limit", "100", "max-size=10m", "max-file=3"} {
		found := false
		for _, a := range args {
			if a == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing %q in resource args: %v", want, args)
		}
	}
}

func TestContainerManager_InstanceResourceArgs_Empty(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{}}
	if args := cm.instanceResourceArgs(); len(args) != 0 {
		t.Errorf("expected empty, got %v", args)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	os.WriteFile(src, []byte("hello world"), 0o644)
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("copyFile = %q", string(data))
	}
}

func TestCopyFile_SrcNotFound(t *testing.T) {
	dir := t.TempDir()
	if err := copyFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst")); err == nil {
		t.Error("expected error for missing source")
	}
}

func TestContainerManager_FindRunningInstance_NoStore(t *testing.T) {
	cm := &ContainerManager{}
	_, err := cm.FindRunningInstance("u1")
	if err == nil {
		t.Error("expected error with no store")
	}
}

func TestContainerManager_FindRunningInstance_NoneRunning(t *testing.T) {
	store, _ := newTestStore(t)
	cm := &ContainerManager{
		store: store,
		checkRunningFn: func(name string) (bool, error) { return false, nil },
	}
	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	_, err := cm.FindRunningInstance(testUUID)
	if err == nil {
		t.Error("expected error when no instance running")
	}
}

func TestContainerManager_FindRunningInstance_Found(t *testing.T) {
	store, _ := newTestStore(t)
	cm := &ContainerManager{
		store: store,
		checkRunningFn: func(name string) (bool, error) { return true, nil },
	}
	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	inst, err := cm.FindRunningInstance(testUUID)
	if err != nil {
		t.Fatal(err)
	}
	if inst.Strategy != "grid2" {
		t.Errorf("strategy = %q", inst.Strategy)
	}
}

func TestContainerDir(t *testing.T) {
	dir := ContainerDir("user1", "live", "grid2-btcusdt")
	if dir != "/data/user1/live/grid2-btcusdt" {
		t.Errorf("ContainerDir = %q", dir)
	}
}

func hasEnv(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}
