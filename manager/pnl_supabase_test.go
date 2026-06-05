package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

const pnlTestUserID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

type pnlTestSetup struct {
	router     chi.Router
	bbgoCalled bool
}

func setupPnLTest(t *testing.T, containerRunning bool, supabaseHandler, bbgoHandler http.HandlerFunc) *pnlTestSetup {
	t.Helper()

	supabaseSrv := httptest.NewServer(supabaseHandler)
	t.Cleanup(supabaseSrv.Close)

	setup := &pnlTestSetup{}
	var bbgoURL string
	if bbgoHandler != nil {
		bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setup.bbgoCalled = true
			bbgoHandler(w, r)
		}))
		t.Cleanup(bbgoSrv.Close)
		bbgoURL = bbgoSrv.URL
	}

	store := NewStrategyStore("", nil)

	cfg := &Config{
		SupabaseURL:  supabaseSrv.URL,
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg}
	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "test")
	if err != nil {
		t.Fatal(err)
	}
	syncer := NewSyncer(supaClient)
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, syncer, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return containerRunning }
	if bbgoURL != "" {
		api.newBBGoClient = func(_ string) *BBGoClient { return NewBBGoClient(bbgoURL) }
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)
	setup.router = r
	return setup
}

func TestBBGoPnL_UsesSupabaseFirst(t *testing.T) {
	setup := setupPnLTest(t, true,
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/rest/v1/trades" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]any{
					{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "1.0", "fee": "25", "traded_at": "2024-01-01T00:00:00Z"},
					{"symbol": "BTCUSDT", "side": "SELL", "price": "55000", "quantity": "1.0", "fee": "27.5", "traded_at": "2024-01-02T00:00:00Z"},
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]any{})
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	)

	req := httptest.NewRequest("GET", "/api/users/"+pnlTestUserID+"/bbgo/pnl", nil)
	w := httptest.NewRecorder()
	setup.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if setup.bbgoCalled {
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
	setup := setupPnLTest(t, true,
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/rest/v1/trades" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]any{})
				return
			}
			w.WriteHeader(http.StatusOK)
		},
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/trades" {
				json.NewEncoder(w).Encode(BBGoTradesResponse{
					Trades: []BBGoTrade{
						{GID: 1, ID: 1, Symbol: "ETHUSDT", Side: "BUY", Price: "3000", Quantity: "2", Fee: "6", TradedAt: "2024-01-01T00:00:00Z"},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	)

	req := httptest.NewRequest("GET", "/api/users/"+pnlTestUserID+"/bbgo/pnl", nil)
	w := httptest.NewRecorder()
	setup.router.ServeHTTP(w, req)

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
	setup := setupPnLTest(t, false,
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/rest/v1/trades" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]any{
					{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "0.5", "fee": "12.5", "traded_at": "2024-01-01T00:00:00Z"},
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]any{})
		},
		nil,
	)

	req := httptest.NewRequest("GET", "/api/users/"+pnlTestUserID+"/bbgo/pnl", nil)
	w := httptest.NewRecorder()
	setup.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (PnL from Supabase when container stopped), got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncer_GetTradesForPnL(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/trades" {
			t.Errorf("expected /rest/v1/trades, got %s", r.URL.Path)
		}
		if r.Header.Get("apikey") != "test-key" {
			t.Errorf("expected apikey test-key, got %s", r.Header.Get("apikey"))
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"symbol": "BTCUSDT", "side": "BUY", "price": "42000", "quantity": "0.1", "fee": "2.1", "traded_at": "2024-01-15T10:00:00Z"},
			{"symbol": "ETHUSDT", "side": "SELL", "price": "2500", "quantity": "4", "fee": "5", "traded_at": "2024-01-15T11:00:00Z"},
		})
	}))
	defer supabaseSrv.Close()

	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	syncer := NewSyncer(supaClient)

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

	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	syncer := NewSyncer(supaClient)

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
		json.NewEncoder(w).Encode([]any{})
	}))
	defer supabaseSrv.Close()

	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	syncer := NewSyncer(supaClient)

	trades, err := syncer.GetTradesForPnL("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}
}

func TestSyncer_GetTradesForPnL_Pagination(t *testing.T) {
	callCount := 0
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path != "/rest/v1/trades" {
			t.Errorf("expected /rest/v1/trades, got %s", r.URL.Path)
		}

		offset := r.URL.Query().Get("offset")
		limit := r.URL.Query().Get("limit")
		if limit != fmt.Sprintf("%d", pnlPageSize) {
			t.Errorf("expected limit=%d, got %s", pnlPageSize, limit)
		}

		var page []map[string]any
		if offset == "0" {
			for i := 0; i < pnlPageSize; i++ {
				page = append(page, map[string]any{
					"symbol": "BTCUSDT", "side": "BUY",
					"price": "50000", "quantity": "1", "fee": "25",
					"traded_at": fmt.Sprintf("2024-01-%02dT00:00:00Z", (i%28)+1),
				})
			}
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(page)
	}))
	defer supabaseSrv.Close()

	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	syncer := NewSyncer(supaClient)

	trades, err := syncer.GetTradesForPnL("user-1")
	if err != nil {
		t.Fatalf("GetTradesForPnL() error: %v", err)
	}
	if len(trades) != pnlPageSize {
		t.Errorf("expected %d trades, got %d", pnlPageSize, len(trades))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (full page + empty), got %d", callCount)
	}
}

// --- Paper mode PnL: skips Supabase, queries bbgo container directly ---

func TestBBGoPnL_PaperMode_SkipsSupabase(t *testing.T) {
	supabaseCalled := false
	setup := setupPnLTest(t, true,
		func(w http.ResponseWriter, r *http.Request) {
			supabaseCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]any{})
		},
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/trades" {
				json.NewEncoder(w).Encode(map[string]any{
					"trades": []map[string]any{
						{"gid": 1, "symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "1", "fee": "25", "tradedAt": "2024-01-01"},
						{"gid": 2, "symbol": "BTCUSDT", "side": "SELL", "price": "55000", "quantity": "1", "fee": "27", "tradedAt": "2024-01-02"},
					},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{})
		},
	)

	req := httptest.NewRequest("GET", "/api/users/"+pnlTestUserID+"/bbgo/pnl?mode=paper", nil)
	w := httptest.NewRecorder()
	setup.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if supabaseCalled {
		t.Error("paper mode should NOT call Supabase")
	}
	if !setup.bbgoCalled {
		t.Error("paper mode should call bbgo container")
	}

	var report PnLReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("decode pnl report: %v", err)
	}
	if len(report.Symbols) != 1 {
		t.Errorf("expected 1 symbol, got %d", len(report.Symbols))
	}
}

func TestBBGoPnL_PaperMode_ContainerNotRunning(t *testing.T) {
	setup := setupPnLTest(t, false,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]any{})
		},
		nil,
	)

	req := httptest.NewRequest("GET", "/api/users/"+pnlTestUserID+"/bbgo/pnl?mode=paper", nil)
	w := httptest.NewRecorder()
	setup.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("paper mode with stopped container = %d, want 503", w.Code)
	}
}
