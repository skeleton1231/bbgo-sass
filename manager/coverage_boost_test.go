package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestComputeInstanceID_AllVariants(t *testing.T) {
	tests := []struct {
		strategy string
		symbol   string
		config   string
		want     string
	}{
		{"grid2", "BTCUSDT", `{"gridNumber":5,"upperPrice":"60000","lowerPrice":"40000"}`, "grid2-BTCUSDT-size-5-60000-40000"},
		{"grid2", "BTCUSDT", `{"gridNumber":5}`, "grid2-BTCUSDT-size-5--"},
		{"grid", "ETHUSDT", `{"gridNumber":3,"upperPrice":"3000","lowerPrice":"2000"}`, "grid-ETHUSDT-3-3000-2000"},
		{"bollgrid", "BTCUSDT", `{"gridNumber":4,"upperPrice":"50000","lowerPrice":"30000","interval":"1h"}`, "bollgrid:BTCUSDT:1h"},
		{"emacross", "BTCUSDT", `{"interval":"5m","fastWindow":12,"slowWindow":26}`, "emacross:BTCUSDT:5m:12-26"},
		{"emacross", "BTCUSDT", `{"interval":"1h"}`, "emacross:BTCUSDT:1h:0-0"},
		{"supertrend", "BTCUSDT", `{"interval":"15m"}`, "supertrend:BTCUSDT"},
		{"supertrend", "BTCUSDT", `{}`, "supertrend:BTCUSDT"},
		{"bollmaker", "BTCUSDT", `{"interval":"1h"}`, "bollmaker:BTCUSDT"},
		{"dca", "BTCUSDT", `{"investmentInterval":"1h"}`, "dca:BTCUSDT"},
		{"dca2", "BTCUSDT", `{"interval":"4h"}`, "dca2-BTCUSDT"},
		{"dca3", "BTCUSDT", `{}`, "dca3-BTCUSDT"},
		{"autobuy", "BTCUSDT", `{"schedule":"daily"}`, "autobuy:BTCUSDT"},
		{"unknown", "BTCUSDT", `{}`, "unknown:BTCUSDT"},
		{"grid2", "BTCUSDT", `null`, "grid2-BTCUSDT-size-0--"},
	}
	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			got := computeInstanceID(tt.strategy, tt.symbol, json.RawMessage(tt.config))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchContainerTrades_FallbackToSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/trades") {
			w.WriteHeader(500)
		} else if p == "/api/sessions" {
			json.NewEncoder(w).Encode(BBGoSessionsResponse{Sessions: []BBGoSession{{Name: "binance"}}})
		} else if p == "/api/sessions/binance/trades" {
			json.NewEncoder(w).Encode(BBGoSessionTradesResponse{Trades: map[string]bbgoSessionTradeSlice{
				"binance": {Trades: []BBGoTrade{{GID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", TradedAt: time.Now().Format(time.RFC3339)}}},
			}})
		} else if strings.HasPrefix(p, "/api/trade-position-summary") {
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()

	client := &BBGoClient{baseURL: srv.URL, client: srv.Client(), ctx: context.Background()}
	api, _ := setupHandlerAPI(t)
	trades, err := api.fetchContainerTrades(client, "", "BTCUSDT", "", nil, nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

func TestFetchContainerTrades_PrimarySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: []BBGoTrade{
			{GID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", TradedAt: time.Now().Format(time.RFC3339)},
		}})
	}))
	defer srv.Close()

	client := &BBGoClient{baseURL: srv.URL, client: srv.Client(), ctx: context.Background()}
	api, _ := setupHandlerAPI(t)
	trades, err := api.fetchContainerTrades(client, "binance", "BTCUSDT", "", nil, nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

func TestFetchContainerTrades_AllFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	client := &BBGoClient{baseURL: srv.URL, client: srv.Client(), ctx: context.Background()}
	api, _ := setupHandlerAPI(t)
	_, err := api.fetchContainerTrades(client, "", "BTCUSDT", "", nil, nil, "", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

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

