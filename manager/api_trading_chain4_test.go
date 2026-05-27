package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c9s/bbgo/saas/manager/pool"
)

// --- PnL edge cases (pnl.go) ---

func TestCalculatePnL_NakedShort(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "SELL", Price: "50000", Quantity: "1", Fee: "0.01", TradedAt: "2024-01-01"},
	}
	report := calculatePnL(trades)
	if len(report.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(report.Symbols))
	}
	if report.Symbols[0].RealizedPnL != 0 {
		t.Errorf("naked short with no prior buy should have 0 realized PnL (costBasis=sellPrice), got %v", report.Symbols[0].RealizedPnL)
	}
}

func TestCalculatePnL_PartialFIFOMatch(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "40000", Quantity: "1", Fee: "0.01", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "50000", Quantity: "0.5", Fee: "0.005", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)
	sym := report.Symbols[0]
	expected := 0.5*50000 - 0.5*40000 // 5000
	if sym.RealizedPnL != expected {
		t.Errorf("partial FIFO: realized = %v, want %v", sym.RealizedPnL, expected)
	}
	if sym.OpenPosition != 0.5 {
		t.Errorf("open position = %v, want 0.5", sym.OpenPosition)
	}
}

func TestCalculatePnL_FeesAreAbsolute(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "40000", Quantity: "1", Fee: "-0.5", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "41000", Quantity: "1", Fee: "0.3", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)
	if report.TotalFees != 0.8 {
		t.Errorf("total fees = %v, want 0.8 (abs of -0.5 + abs of 0.3)", report.TotalFees)
	}
}

func TestCalculatePnL_SkipZeroPriceOrQuantity(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "0", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "ETHUSDT", Side: "BUY", Price: "3000", Quantity: "0", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "50000", Quantity: "1", Fee: "0.1", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)
	if len(report.Symbols) != 1 {
		t.Fatalf("expected 1 symbol (zero-price/qty trades skipped), got %d", len(report.Symbols))
	}
}

func TestCalculatePnL_WinRate(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "40000", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-02"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "55000", Quantity: "1", Fee: "0", TradedAt: "2024-01-03"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "45000", Quantity: "1", Fee: "0", TradedAt: "2024-01-04"},
	}
	report := calculatePnL(trades)
	if report.WinRate != 50.0 {
		t.Errorf("win rate = %v, want 50.0 (1 win + 1 loss)", report.WinRate)
	}
}

// --- fmtFloat edge cases (pnl.go) ---

func TestFmtFloat_NaN(t *testing.T) {
	if got := fmtFloat(math.NaN(), 2); got != "0" {
		t.Errorf("fmtFloat(NaN) = %q, want %q", got, "0")
	}
}

func TestFmtFloat_Inf(t *testing.T) {
	if got := fmtFloat(math.Inf(1), 2); got != "0" {
		t.Errorf("fmtFloat(+Inf) = %q, want %q", got, "0")
	}
	if got := fmtFloat(math.Inf(-1), 2); got != "0" {
		t.Errorf("fmtFloat(-Inf) = %q, want %q", got, "0")
	}
}

func TestFmtFloat_Normal(t *testing.T) {
	if got := fmtFloat(123.4567, 2); got != "123.46" {
		t.Errorf("fmtFloat(123.4567, 2) = %q, want %q", got, "123.46")
	}
}

// --- Sync cursor advance (sync.go) ---

func TestSyncTradesViaAPI_CursorAdvance(t *testing.T) {
	callCount := 0
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Write([]byte(`{"trades":[{"id":1,"orderID":10,"symbol":"BTCUSDT","side":"BUY","price":"40000","quantity":"1","fee":"0.01","feeCurrency":"USDT","quoteQuantity":"40000","tradedAt":"2024-01-01","gid":100}]}`))
		} else {
			w.Write([]byte(`{"trades":[]}`))
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "sync_cursors") {
			w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	p := pool.New(5)
	users := NewUserContainerManager()
	users.AddStrategy("sync-cursor-u1", ModeLive, StrategyEntry{Exchange: "binance", Strategy: "grid2"})
	users.UpdateStatus("sync-cursor-u1", ModeLive, StatusRunning)

	s := &Syncer{
		users:     users,
		cfg:       &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		pool:      p,
		client:    supabaseSrv.Client(),
		newBBGoClientFn: func(_ string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
	}
	client := s.bbgoClient("sync-cursor-u1", ModeLive).WithContext(context.Background())
	s.syncTradesViaAPI("sync-cursor-u1", client)
	if callCount != 1 {
		t.Errorf("expected 1 bbgo API call (single trade, no pagination), got %d", callCount)
	}
}

// --- SyncAll skips stopped containers ---

func TestSyncAll_SkipsStoppedContainers(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("sync-all-u1", ModeLive, StrategyEntry{Exchange: "binance", Strategy: "grid2"})
	users.AddStrategy("sync-all-u2", ModeLive, StrategyEntry{Exchange: "binance", Strategy: "grid2"})
	users.UpdateStatus("sync-all-u1", ModeLive, StatusRunning)
	// sync-all-u2 stays stopped

	bbgoCalled := false
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bbgoCalled = true
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	p := pool.New(5)
	s := &Syncer{
		users:     users,
		cfg:       &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		pool:      p,
		client:    supabaseSrv.Client(),
		newBBGoClientFn: func(_ string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
	}
	s.SyncAll()
	if bbgoCalled {
		t.Error("bbgo should not be called — stopped containers should be skipped")
	}
}

// --- upsertOrders error path ---

func TestUpsertOrders_SupabaseError(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`internal server error`))
	}))
	defer supabaseSrv.Close()

	s := &Syncer{
		cfg:    &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		client: supabaseSrv.Client(),
	}

	err := s.upsertOrders("u1", []BBGoOrder{
		{OrderID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1"},
	})
	if err == nil {
		t.Error("expected error from supabase 500")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %v, want status 500", err)
	}
}

// --- Concurrent strategy add/remove ---

func TestConcurrentAddRemoveStrategies(t *testing.T) {
	m := NewUserContainerManager()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			m.AddStrategy("concurrent-u1", ModeLive, StrategyEntry{
				ID:       generateID("strat"),
				Exchange: "binance",
				Strategy: "grid2",
				Mode:     "paper",
			})
		}(i)
		go func(i int) {
			defer wg.Done()
			uc, ok := m.Get("concurrent-u1", ModeLive)
			if ok && len(uc.Strategies) > 0 {
				m.RemoveStrategy("concurrent-u1", uc.Strategies[0].ID)
			}
		}(i)
	}
	wg.Wait()
	// Should not panic or deadlock
}

// --- API contract validation (CreateStrategy) ---

func TestCreateStrategy_EmptyName(t *testing.T) {
	api := &API{users: NewUserContainerManager(), wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"","exchange":"binance","strategy":"grid2","mode":"paper","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty name: status = %d, want 400", w.Code)
	}
}

func TestCreateStrategy_EmptyStrategy(t *testing.T) {
	api := &API{users: NewUserContainerManager(), wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"test","exchange":"binance","strategy":"","mode":"paper","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty strategy: status = %d, want 400", w.Code)
	}
}

func TestCreateStrategy_NoExchange(t *testing.T) {
	api := &API{users: NewUserContainerManager(), wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"test","exchange":"","strategy":"grid2","mode":"paper","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("no exchange: status = %d, want 400", w.Code)
	}
}

func TestCreateStrategy_CrossExchangeWithoutSessions(t *testing.T) {
	api := &API{users: NewUserContainerManager(), wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"test","strategy":"xmaker","crossExchange":true,"sessions":[],"config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("cross-exchange without sessions: status = %d, want 400", w.Code)
	}
}

func TestCreateStrategy_InvalidMode(t *testing.T) {
	api := &API{users: NewUserContainerManager(), wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"test","exchange":"binance","strategy":"grid2","mode":"invalid","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid mode: status = %d, want 400", w.Code)
	}
}

func TestCreateStrategy_LiveOnlyWithPaper(t *testing.T) {
	api := &API{users: NewUserContainerManager(), wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"test","exchange":"binance","strategy":"bollmaker","mode":"paper","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("liveOnly with paper: status = %d, want 400", w.Code)
	}
}

// --- WSTicket expiry and single-use ---

func TestWSTicket_Expiry(t *testing.T) {
	ts := NewWSTicketStore()
	defer ts.Close()

	ticket := ts.Issue("u1")
	// Manually expire it
	ts.mu.Lock()
	ts.tickets[ticket].expiresAt = time.Now().Add(-time.Second)
	ts.mu.Unlock()

	_, ok := ts.Redeem(ticket)
	if ok {
		t.Error("expired ticket should not redeem")
	}
}

func TestWSTicket_SingleUse(t *testing.T) {
	ts := NewWSTicketStore()
	defer ts.Close()

	ticket := ts.Issue("u1")
	userID, ok := ts.Redeem(ticket)
	if !ok || userID != "u1" {
		t.Errorf("first redeem: ok=%v userID=%q", ok, userID)
	}
	_, ok = ts.Redeem(ticket)
	if ok {
		t.Error("second redeem of same ticket should fail")
	}
}

// --- UserContainer clone safety ---

func TestCloneUserContainer_Isolation(t *testing.T) {
	original := &UserContainer{
		Mode:   ModeLive,
		UserID: "u1",
		Status: StatusStopped,
		Strategies: []StrategyEntry{
			{ID: "s1", Strategy: "grid2", Exchange: "binance"},
		},
	}
	cloned := cloneUserContainer(original)
	cloned.Strategies[0].Strategy = "xmaker"
	if original.Strategies[0].Strategy != "grid2" {
		t.Error("clone mutation leaked to original")
	}
}

// --- normalizeStrategyConfig ---

func TestNormalizeStrategyConfig_FieldRename(t *testing.T) {
	strategy, params := normalizeStrategyConfig("dca", map[string]interface{}{"interval": "1h"})
	if strategy != "dca" {
		t.Errorf("strategy = %q, want %q", strategy, "dca")
	}
	if _, has := params["investmentInterval"]; !has {
		t.Error("expected investmentInterval key after field rename")
	}
	if _, has := params["interval"]; has {
		t.Error("interval key should be removed after rename")
	}
}

// --- BuildUserYAML paper mode ---

func TestBuildUserYAML_PaperMode(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModePaper,
		UserID: "u1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "paper", Config: json.RawMessage(`{"symbol":"BTCUSDT"}`)},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)
	if !strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Errorf("paper mode YAML should contain PAPER_TRADE, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "publicOnly: true") {
		t.Errorf("no credentials should set publicOnly: true, got:\n%s", yamlStr)
	}
}

func TestBuildUserYAML_LiveMode(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "u1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "live", Config: json.RawMessage(`{"symbol":"BTCUSDT"}`)},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)
	if strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Errorf("live mode YAML should not contain PAPER_TRADE, got:\n%s", yamlStr)
	}
	if strings.Contains(yamlStr, "publicOnly") {
		t.Errorf("with credentials should not set publicOnly, got:\n%s", yamlStr)
	}
}
