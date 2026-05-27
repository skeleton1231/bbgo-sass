package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/c9s/bbgo/saas/manager/pool"
)

var errTestSyncFail = errors.New("data sync failed: network error")

// --- PnL edge cases ---

func TestCalculatePnL_OnlyBuys_NoRealizedPnL(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "10"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "48000", Quantity: "0.5", Fee: "5"},
	}
	report := calculatePnL(trades)
	if report.TotalRealizedPnL != 0 {
		t.Errorf("expected 0 realized PnL with no sells, got %f", report.TotalRealizedPnL)
	}
	if report.TotalTrades != 2 {
		t.Errorf("expected 2 total trades, got %d", report.TotalTrades)
	}
	if len(report.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(report.Symbols))
	}
	sym := report.Symbols[0]
	if sym.OpenPosition != 1.5 {
		t.Errorf("expected open position 1.5, got %f", sym.OpenPosition)
	}
}

func TestCalculatePnL_SellBeforeBuy(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "ETHUSDT", Side: "SELL", Price: "3000", Quantity: "1", Fee: "3"},
		{Symbol: "ETHUSDT", Side: "BUY", Price: "2800", Quantity: "1", Fee: "3"},
	}
	report := calculatePnL(trades)
	if len(report.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(report.Symbols))
	}
	sym := report.Symbols[0]
	if sym.RealizedPnL != 0 {
		t.Errorf("expected 0 realized PnL for naked sell, got %f", sym.RealizedPnL)
	}
}

func TestCalculatePnL_FIFO_MatchedTrade(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "10", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "10", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)
	if report.TotalRealizedPnL != 5000 {
		t.Errorf("expected realized PnL 5000, got %f", report.TotalRealizedPnL)
	}
	if report.TotalFees != 20 {
		t.Errorf("expected 20 total fees, got %f", report.TotalFees)
	}
	if report.WinRate != 100 {
		t.Errorf("expected 100%% win rate, got %f", report.WinRate)
	}
}

func TestCalculatePnL_MultipleSymbols(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "10", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "10", TradedAt: "2024-01-02"},
		{Symbol: "ETHUSDT", Side: "BUY", Price: "3000", Quantity: "2", Fee: "5", TradedAt: "2024-01-01"},
		{Symbol: "ETHUSDT", Side: "SELL", Price: "2800", Quantity: "2", Fee: "5", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)
	// BTC: +5000, ETH: -400 → total +4600
	if report.TotalRealizedPnL != 4600 {
		t.Errorf("expected realized PnL 4600, got %f", report.TotalRealizedPnL)
	}
	if report.TotalTrades != 4 {
		t.Errorf("expected 4 trades, got %d", report.TotalTrades)
	}
	if len(report.Symbols) != 2 {
		t.Errorf("expected 2 symbols, got %d", len(report.Symbols))
	}
}

// --- Backtest notification dispatch ---

func TestBacktestExecutor_NotificationOnComplete(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor("0123456789abcdef0123456789abcdef")
	notifier := NewNotifier(tmpDir, enc)
	notifier.configs["u1"] = []NotificationConfig{{
		Channel: NotificationChannel{ID: "n1", Type: "test", Enabled: true},
		Rules:   NotificationRule{TradeEvents: true, OrderEvents: true, ContainerHealth: true},
	}}

	store := NewBacktestJobStore(tmpDir)
	exec := NewBacktestExecutor(store, nil, notifier)
	exec.syncFn = func(exchange, symbol, startTime, endTime string) (string, error) {
		return "synced", nil
	}
	exec.runFn = func(userID string, yamlContent []byte) ([]byte, error) {
		return []byte("backtest output"), nil
	}

	job := &BacktestJob{
		ID: "bt-notif", UserID: "u1", Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-06-01", NeedSync: true,
		Config: json.RawMessage(`{"symbol":"BTCUSDT"}`),
	}
	if err := exec.Submit(job); err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)

	updated, ok := store.Get(job.ID)
	if !ok {
		t.Fatal("job not found")
	}
	if updated.Status != JobCompleted {
		t.Errorf("expected completed, got %s (error: %s)", updated.Status, updated.Error)
	}
}

func TestBacktestExecutor_NotificationOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor("0123456789abcdef0123456789abcdef")
	notifier := NewNotifier(tmpDir, enc)
	notifier.configs["u1"] = []NotificationConfig{{
		Channel: NotificationChannel{ID: "n1", Type: "test", Enabled: true},
		Rules:   NotificationRule{TradeEvents: true, OrderEvents: true, ContainerHealth: true},
	}}

	store := NewBacktestJobStore(tmpDir)
	exec := NewBacktestExecutor(store, nil, notifier)
	exec.syncFn = func(exchange, symbol, startTime, endTime string) (string, error) {
		return "", errTestSyncFail
	}

	job := &BacktestJob{
		ID: "bt-notif-fail", UserID: "u1", Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-06-01", NeedSync: true,
		Config: json.RawMessage(`{}`),
	}
	exec.Submit(job)
	time.Sleep(300 * time.Millisecond)

	updated, _ := store.Get(job.ID)
	if updated.Status != JobFailed {
		t.Errorf("expected failed, got %s", updated.Status)
	}
}

// --- Credential → Live strategy full chain ---

func setupTestAPIWithCreds(t *testing.T) (*API, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(tmpDir, enc)
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}
	cfg := &Config{DataDir: tmpDir, ManagerToken: "test-token", SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, NewBacktestJobStore(tmpDir))
	api.containerRunning = func(string, _ string) bool { return false }
	api.containerStart = func(*UserContainer) error { return nil }
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`ok`))
		}))
		t.Cleanup(srv.Close)
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}
	return api, func() { api.Close() }
}

func TestAPI_CredentialThenLiveStrategy_Chain(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	r := testRouter(api)

	// Step 1: Create credential
	credBody := `{"exchange":"binance","api_key":"testkey","api_secret":"testsecret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/credentials", strings.NewReader(credBody))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create credential: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: Create live strategy — should succeed
	stratBody := `{"name":"My Grid","exchange":"binance","strategy":"grid2","config":{},"mode":"live"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", strings.NewReader(stratBody))
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("create live strategy: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var uc map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&uc)
	strategies, ok := uc["strategies"].([]interface{})
	if !ok || len(strategies) != 2 {
		t.Fatalf("expected 2 strategies (existing + new), got %v", uc["strategies"])
	}
}

// --- PnL Supabase fallback when container stopped ---

func TestAPI_PnL_SupabaseFallback_WhenStopped(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.users.users[userID+":"+ModeLive].Status = StatusStopped

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/rest/v1/sync_trades" {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "1", "fee": "10", "traded_at": "2024-01-01"},
				{"symbol": "BTCUSDT", "side": "SELL", "price": "55000", "quantity": "1", "fee": "10", "traded_at": "2024-01-02"},
			})
			return
		}
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	syncerCfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "key"}
	api.syncer = NewSyncer(api.users, syncerCfg, api.container, pool.New(5))

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID+"/bbgo/pnl", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var report PnLReport
	json.NewDecoder(w.Body).Decode(&report)
	if report.TotalRealizedPnL != 5000 {
		t.Errorf("expected realized PnL 5000, got %f", report.TotalRealizedPnL)
	}
}
