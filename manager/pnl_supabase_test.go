package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestBBGoPnL_UsesSupabaseFirst(t *testing.T) {
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/sync_trades" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "1.0", "fee": "25", "traded_at": "2024-01-01T00:00:00Z"},
				{"symbol": "BTCUSDT", "side": "SELL", "price": "55000", "quantity": "1.0", "fee": "27.5", "traded_at": "2024-01-02T00:00:00Z"},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	bbgoCalled := false
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bbgoCalled = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bbgoSrv.Close()

	users := NewUserContainerManager()
	users.users[userID] = &UserContainer{
		UserID:     userID,
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{
		SupabaseURL:  supabaseSrv.URL,
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	syncer := &Syncer{users: users, cfg: cfg, container: cm, client: &http.Client{}}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, syncer, nil, nil, nil, nil)
	api.newBBGoClient = func(_ string) *BBGoClient { return NewBBGoClient(bbgoSrv.URL) }

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/pnl", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if bbgoCalled {
		t.Error("bbgo container should NOT be called when Supabase has trades")
	}

	var report PnLReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("decode pnl report: %v", err)
	}
	if report.TotalTrades != 2 {
		t.Errorf("expected 2 total trades, got %d", report.TotalTrades)
	}
	if len(report.Symbols) != 1 || report.Symbols[0].Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT symbol, got %v", report.Symbols)
	}
	if report.Symbols[0].RealizedPnL <= 0 {
		t.Errorf("expected positive realized PnL, got %f", report.Symbols[0].RealizedPnL)
	}
}

func TestBBGoPnL_FallsBackToContainer(t *testing.T) {
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/sync_trades" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer supabaseSrv.Close()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trades" {
			json.NewEncoder(w).Encode(BBGoTradesResponse{
				Trades: []BBGoTrade{
					{GID: 1, ID: 1, Symbol: "ETHUSDT", Side: "BUY", Price: "3000", Quantity: "2", Fee: "6", TradedAt: "2024-01-01T00:00:00Z"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	users := NewUserContainerManager()
	users.users[userID] = &UserContainer{
		UserID:     userID,
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{
		SupabaseURL:  supabaseSrv.URL,
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	syncer := &Syncer{users: users, cfg: cfg, container: cm, client: &http.Client{}}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, syncer, nil, nil, nil, nil)
	api.newBBGoClient = func(_ string) *BBGoClient { return NewBBGoClient(bbgoSrv.URL) }

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/pnl", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var report PnLReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("decode pnl report: %v", err)
	}
	if report.TotalTrades != 1 {
		t.Errorf("expected 1 total trade from container fallback, got %d", report.TotalTrades)
	}
}

func TestBBGoPnL_WorksWhenContainerStopped(t *testing.T) {
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/sync_trades" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "0.5", "fee": "12.5", "traded_at": "2024-01-01T00:00:00Z"},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	users.users[userID] = &UserContainer{
		UserID:     userID,
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{
		SupabaseURL:  supabaseSrv.URL,
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	syncer := &Syncer{users: users, cfg: cfg, container: cm, client: &http.Client{}}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, syncer, nil, nil, nil, nil)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/pnl", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (PnL from Supabase when container stopped), got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncer_GetTradesForPnL(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/sync_trades" {
			t.Errorf("expected /rest/v1/sync_trades, got %s", r.URL.Path)
		}
		if r.Header.Get("apikey") != "test-key" {
			t.Errorf("expected apikey test-key, got %s", r.Header.Get("apikey"))
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"symbol": "BTCUSDT", "side": "BUY", "price": "42000", "quantity": "0.1", "fee": "2.1", "traded_at": "2024-01-15T10:00:00Z"},
			{"symbol": "ETHUSDT", "side": "SELL", "price": "2500", "quantity": "4", "fee": "5", "traded_at": "2024-01-15T11:00:00Z"},
		})
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test-key"}
	syncer := NewSyncer(NewUserContainerManager(), cfg, &ContainerManager{cfg: cfg}, nil)

	trades, err := syncer.GetTradesForPnL("user-1")
	if err != nil {
		t.Fatalf("GetTradesForPnL() error: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	if trades[0].Symbol != "BTCUSDT" {
		t.Errorf("trades[0].Symbol = %s, want BTCUSDT", trades[0].Symbol)
	}
	if trades[1].Side != "SELL" {
		t.Errorf("trades[1].Side = %s, want SELL", trades[1].Side)
	}
	if trades[0].Price != "42000" {
		t.Errorf("trades[0].Price = %s, want 42000", trades[0].Price)
	}
	if trades[0].TradedAt != "2024-01-15T10:00:00Z" {
		t.Errorf("trades[0].TradedAt = %s, want 2024-01-15T10:00:00Z", trades[0].TradedAt)
	}
}

func TestSyncer_GetTradesForPnL_ServerError(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test-key"}
	syncer := NewSyncer(NewUserContainerManager(), cfg, &ContainerManager{cfg: cfg}, nil)

	trades, err := syncer.GetTradesForPnL("user-1")
	if err == nil {
		t.Fatal("expected error on 500 response, got nil")
	}
	if trades != nil {
		t.Fatalf("expected nil trades on error, got %d", len(trades))
	}
}

func TestSyncer_GetTradesForPnL_Empty(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test-key"}
	syncer := NewSyncer(NewUserContainerManager(), cfg, &ContainerManager{cfg: cfg}, nil)

	trades, err := syncer.GetTradesForPnL("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}
}
