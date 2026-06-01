package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupMarketDataAPI(hub *MarketDataHub) (*API, *chi.Mux) {
	return setupMarketDataAPIWithTestnet(hub, nil)
}

func setupMarketDataAPIWithTestnet(hub *MarketDataHub, testnetHub *MarketDataHub) (*API, *chi.Mux) {
	cfg := &Config{
		SupabaseURL:               "http://localhost:1",
		SupabaseKey:               "test",
		ManagerToken:              "test-token",
		MarketDataAddr:            "bbgo-marketdata:9090",
		MarketDataRESTAddr:        "bbgo-marketdata:8080",
		MarketDataTestnetAddr:     "bbgo-marketdata-testnet:9090",
		MarketDataTestnetRESTAddr: "bbgo-marketdata-testnet:8080",
	}
	store := NewStrategyStore("")
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, hub, testnetHub, nil, nil, nil)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)
	return api, r
}

func TestMarketTicker_MissingParams(t *testing.T) {
	_, router := setupMarketDataAPI(nil)

	tests := []struct {
		name string
		url  string
	}{
		{"no symbol", "/api/markets/binance/ticker"},
		{"empty symbol", "/api/markets/binance/ticker?symbol="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestMarketTicker_NoHub(t *testing.T) {
	_, router := setupMarketDataAPI(nil)
	req := httptest.NewRequest("GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketTicker_NilConn(t *testing.T) {
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
	}
	_, router := setupMarketDataAPI(hub)
	req := httptest.NewRequest("GET", "/api/markets/binance/ticker?symbol=BTCUSDT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_MissingParams(t *testing.T) {
	_, router := setupMarketDataAPI(nil)

	tests := []struct {
		name string
		url  string
	}{
		{"no symbol", "/api/markets/binance/klines"},
		{"empty symbol", "/api/markets/binance/klines?symbol="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestMarketKlines_NoHub(t *testing.T) {
	_, router := setupMarketDataAPI(nil)
	req := httptest.NewRequest("GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_DefaultInterval(t *testing.T) {
	_, router := setupMarketDataAPI(nil)
	req := httptest.NewRequest("GET", "/api/markets/binance/klines?symbol=BTCUSDT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (past param validation), got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_LimitClamping(t *testing.T) {
	_, router := setupMarketDataAPI(nil)
	req := httptest.NewRequest("GET", "/api/markets/binance/klines?symbol=BTCUSDT&limit=2000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_NegativeLimit(t *testing.T) {
	_, router := setupMarketDataAPI(nil)
	req := httptest.NewRequest("GET", "/api/markets/binance/klines?symbol=BTCUSDT&limit=-5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMarketKlines_InvalidTimeParams(t *testing.T) {
	_, router := setupMarketDataAPI(nil)
	req := httptest.NewRequest("GET", "/api/markets/binance/klines?symbol=BTCUSDT&start_time=abc&end_time=xyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}
