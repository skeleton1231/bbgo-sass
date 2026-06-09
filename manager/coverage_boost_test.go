package main

import "testing"

func TestInstanceLogs_MockLogsFn(t *testing.T) {
	cm := &ContainerManager{
		logsFn: func(containerName string) (string, error) {
			return "line 1\nline 2", nil
		},
		cfg: &Config{},
	}
	logs, err := cm.InstanceLogs("u1", "live", "grid2-btcusdt", "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs != "line 1\nline 2" {
		t.Errorf("got %q", logs)
	}
}

func TestInstanceLogs_FallbackDocker(t *testing.T) {
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) {
			return "docker logs output", nil
		},
		cfg: &Config{},
	}
	logs, err := cm.InstanceLogs("u1", "live", "grid2-btcusdt", "50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs != "docker logs output" {
		t.Errorf("got %q", logs)
	}
}

func TestCreateAndGet_Instance(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	inst := &StrategyInstance{
		InstanceID: "grid2-BTCUSDT", UserID: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
	}
	if err := store.CreateInstance(inst, func(ex string) bool { return true }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := store.GetInstance("u1", "live", "grid2-BTCUSDT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Strategy != "grid2" {
		t.Errorf("got strategy %q", got.Strategy)
	}
}

func TestListInstances_Empty(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	list, err := store.ListAllInstances("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty, got %d", len(list))
	}
}

func TestListInstances_MultipleUsers(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	store.CreateInstance(&StrategyInstance{InstanceID: "a1", UserID: "u1", Mode: "live", Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT"}, func(string) bool { return true })
	store.CreateInstance(&StrategyInstance{InstanceID: "a2", UserID: "u2", Mode: "live", Strategy: "grid2", Exchange: "binance", Symbol: "ETHUSDT"}, func(string) bool { return true })

	u1, _ := store.ListAllInstances("u1")
	if len(u1) != 1 {
		t.Errorf("u1: expected 1, got %d", len(u1))
	}
	u2, _ := store.ListAllInstances("u2")
	if len(u2) != 1 {
		t.Errorf("u2: expected 1, got %d", len(u2))
	}
}

func TestRemoveInstance(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	store.CreateInstance(&StrategyInstance{InstanceID: "r1", UserID: "u1", Mode: "live", Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT"}, func(string) bool { return true })

	if err := store.RemoveInstance("u1", "live", "r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.GetInstance("u1", "live", "r1"); err == nil {
		t.Error("expected error for removed instance")
	}
}

func TestDockerLong_Mock(t *testing.T) {
	cm := &ContainerManager{
		dockerFn: func(args ...string) (string, error) {
			return "output", nil
		},
		cfg: &Config{},
	}
	out, err := cm.dockerLong("ps", "-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "output" {
		t.Errorf("got %q", out)
	}
}

func TestRunBacktest_MockFn(t *testing.T) {
	cm := &ContainerManager{
		runBacktestFn: func(userID, jobID string, yamlContent []byte) ([]byte, error) {
			return []byte("backtest output"), nil
		},
		cfg: &Config{},
	}
	result, err := cm.RunBacktest("u1", "bt-123", []byte("strategy: grid2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != "backtest output" {
		t.Errorf("got %q", string(result))
	}
}

func TestRunBacktest_InvalidJobID(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{}}
	_, err := cm.RunBacktest("u1", "../etc/passwd", nil)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}



func TestSyncBacktest_MockFn(t *testing.T) {
	cm := &ContainerManager{
		syncBacktestFn: func(userID, exchange, symbol, start, end string) (string, error) {
			return "synced " + symbol, nil
		},
		cfg: &Config{},
	}
	result, err := cm.SyncBacktest("u1", "binance", "BTCUSDT", "2024-01-01", "2024-01-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "synced BTCUSDT" {
		t.Errorf("got %q", result)
	}
}

