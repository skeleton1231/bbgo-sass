package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestBBGoClient_GetSessionTrades_MultipleSymbols(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"trades": map[string]any{
				"BTCUSDT": map[string]any{
					"Trades": []map[string]any{
						{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "0.1"},
						{"symbol": "BTCUSDT", "side": "SELL", "price": "51000", "quantity": "0.1"},
					},
				},
				"ETHUSDT": map[string]any{
					"Trades": []map[string]any{
						{"symbol": "ETHUSDT", "side": "BUY", "price": "3000", "quantity": "1.0"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetSessionTrades("binance")
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 3 {
		t.Fatalf("expected 3 trades, got %d", len(trades))
	}
	symbols := map[string]int{}
	for _, tr := range trades {
		symbols[tr.Symbol]++
	}
	if symbols["BTCUSDT"] != 2 {
		t.Errorf("expected 2 BTCUSDT trades, got %d", symbols["BTCUSDT"])
	}
	if symbols["ETHUSDT"] != 1 {
		t.Errorf("expected 1 ETHUSDT trade, got %d", symbols["ETHUSDT"])
	}
}

func TestBBGoClient_GetSessionTrades_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"trades": map[string]any{},
		})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetSessionTrades("binance")
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 0 {
		t.Errorf("expected 0 trades, got %d", len(trades))
	}
}

func TestBBGoTrades_FallbackToSessionTrades(t *testing.T) {
	dbEndpointCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trades" {
			dbEndpointCalled = true
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path == "/api/sessions/binance/trades" {
			json.NewEncoder(w).Encode(map[string]any{
				"trades": map[string]any{
					"BTCUSDT": map[string]any{
						"Trades": []map[string]any{
							{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "0.1"},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &Config{
		SupabaseURL:    "http://localhost:1",
		SupabaseKey:   "test",
		ManagerToken:   "test-token",
		MarketDataAddr: "bbgo-marketdata:9090",
	}
	store := NewStrategyStore("", nil)
	cm := &ContainerManager{cfg: cfg, pool: nil}
	cm.apiURLFn = func(_, _ string) string { return srv.URL }
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: baseURL, client: &http.Client{}}
	}
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.containerRunning = func(_, _ string) bool { return true }

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/trades?exchange=binance&symbol=BTCUSDT", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !dbEndpointCalled {
		t.Error("expected db-backed /api/trades to be tried first")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after fallback, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	trades, ok := resp["trades"].([]any)
	if !ok || len(trades) != 1 {
		t.Errorf("expected 1 trade from session fallback, got %v", resp["trades"])
	}
}

func TestPnL_PaperMode_SkipsSupabase(t *testing.T) {
	supaQueried := false
	supaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		supaQueried = true
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]any{})
	}))
	defer supaSrv.Close()

	containerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trades" {
			json.NewEncoder(w).Encode(map[string]any{
				"trades": []map[string]any{
					{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "0.1", "fee": "0.001", "feeCurrency": "BNB", "tradedAt": "2026-01-01T00:00:00Z"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer containerSrv.Close()

	cfg := &Config{
		SupabaseURL:    "http://localhost:1",
		SupabaseKey:   "test",
		ManagerToken:   "test-token",
		MarketDataAddr: "bbgo-marketdata:9090",
	}
	store := NewStrategyStore("", nil)
	cm := &ContainerManager{cfg: cfg, pool: nil}
	cm.apiURLFn = func(_, _ string) string { return containerSrv.URL }
	proxy := NewBotProxy(cm)
	supa, _ := NewSupabaseClient(supaSrv.URL, "test")
	syncer := NewSyncer(supa)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, syncer, nil, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: baseURL, client: &http.Client{}}
	}
	const pnlUserID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.containerRunning = func(uid, mode string) bool { return uid == pnlUserID && mode == ModePaper }

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/users/"+pnlUserID+"/bbgo/pnl?mode=paper&exchange=binance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if supaQueried {
		t.Error("paper mode should NOT query Supabase for PnL")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
