package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBBGoClient_Ping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ping" {
			t.Errorf("expected /api/ping, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping() returned error: %v", err)
	}
}

func TestBBGoClient_Ping_Unreachable(t *testing.T) {
	client := NewBBGoClient("http://127.0.0.1:1")
	if err := client.Ping(); err == nil {
		t.Fatal("Ping() should return error for unreachable server")
	}
}

func TestBBGoClient_Ping_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	if err := client.Ping(); err == nil {
		t.Fatal("Ping() should return error for non-200 status")
	}
}

func TestBBGoClient_GetTrades(t *testing.T) {
	expectedTrades := []BBGoTrade{
		{GID: 1, ID: 100, OrderID: 200, Exchange: "binance", Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.1", Fee: "0.001", FeeCurrency: "BNB"},
		{GID: 2, ID: 101, OrderID: 201, Exchange: "binance", Symbol: "BTCUSDT", Side: "SELL", Price: "51000", Quantity: "0.1", Fee: "0.001", FeeCurrency: "BNB"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/trades" {
			t.Errorf("expected /api/trades, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("gid") != "5" {
			t.Errorf("expected gid=5, got %s", r.URL.Query().Get("gid"))
		}
		if r.URL.Query().Get("exchange") != "binance" {
			t.Errorf("expected exchange=binance, got %s", r.URL.Query().Get("exchange"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: expectedTrades})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetTrades("binance", "", 5)
	if err != nil {
		t.Fatalf("GetTrades() returned error: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	if trades[0].Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", trades[0].Symbol)
	}
	if trades[1].Side != "SELL" {
		t.Errorf("expected SELL, got %s", trades[1].Side)
	}
}

func TestBBGoClient_GetTrades_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: nil})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetTrades("", "", 0)
	if err != nil {
		t.Fatalf("GetTrades() returned error: %v", err)
	}
	if len(trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(trades))
	}
}

func TestBBGoClient_GetClosedOrders(t *testing.T) {
	expectedOrders := []BBGoOrder{
		{GID: 1, OrderID: 300, Symbol: "ETHUSDT", Side: "BUY", Type: "LIMIT", Price: "3000", Quantity: "1", Status: "FILLED"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/closed" {
			t.Errorf("expected /api/orders/closed, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "ETHUSDT" {
			t.Errorf("expected symbol=ETHUSDT, got %s", r.URL.Query().Get("symbol"))
		}
		json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: expectedOrders})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	orders, err := client.GetClosedOrders("", "ETHUSDT", 0)
	if err != nil {
		t.Fatalf("GetClosedOrders() returned error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].Status != "FILLED" {
		t.Errorf("expected FILLED, got %s", orders[0].Status)
	}
}

func TestBBGoClient_GetSessions(t *testing.T) {
	sessions := []BBGoSession{
		{Name: "binance", ExchangeName: "binance"},
		{Name: "max", ExchangeName: "max"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions" {
			t.Errorf("expected /api/sessions, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(BBGoSessionsResponse{Sessions: sessions})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	result, err := client.GetSessions()
	if err != nil {
		t.Fatalf("GetSessions() returned error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(result))
	}
	if result[0].Name != "binance" {
		t.Errorf("expected binance, got %s", result[0].Name)
	}
}

func TestBBGoClient_GetSessionBalances(t *testing.T) {
	balances := map[string]BBGoBalance{
		"BTC":  {Currency: "BTC", Available: "1.5", Locked: "0.5"},
		"USDT": {Currency: "USDT", Available: "10000", Locked: "0"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions/binance/account/balances" {
			t.Errorf("expected /api/sessions/binance/account/balances, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(BBGoBalancesResponse{Balances: balances})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	result, err := client.GetSessionBalances("binance")
	if err != nil {
		t.Fatalf("GetSessionBalances() returned error: %v", err)
	}
	if result["BTC"].Available != "1.5" {
		t.Errorf("expected BTC available=1.5, got %s", result["BTC"].Available)
	}
}

func TestBBGoClient_GetAssets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/assets" {
			t.Errorf("expected /api/assets, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(BBGoAssetsResponse{Assets: map[string]BBGoAsset{"BTC": {Currency: "BTC", Total: "1.0", Available: "1.0", Locked: "0"}}})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	_, err := client.GetAssets()
	if err != nil {
		t.Fatalf("GetAssets() returned error: %v", err)
	}
}

func TestBBGoClient_GetStrategies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/strategies/single" {
			t.Errorf("expected /api/strategies/single, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(BBGoStrategiesResponse{Strategies: []BBGoStrategyState{{"strategy": "grid"}}})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	strats, err := client.GetStrategies()
	if err != nil {
		t.Fatalf("GetStrategies() returned error: %v", err)
	}
	if len(strats) != 1 || strats[0]["strategy"] != "grid" {
		t.Errorf("unexpected strategies: %v", strats)
	}
}

func TestBBGoClient_GetSessionSymbols(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions/binance/symbols" {
			t.Errorf("expected /api/sessions/binance/symbols, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(BBGoSymbolsResponse{Symbols: []string{"BTCUSDT", "ETHUSDT"}})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	symbols, err := client.GetSessionSymbols("binance")
	if err != nil {
		t.Fatalf("GetSessionSymbols() returned error: %v", err)
	}
	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(symbols))
	}
}

func TestBBGoClient_GetTradingVolume(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/trading-volume" {
			t.Errorf("expected /api/trading-volume, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("period") != "day" {
			t.Errorf("expected period=day, got %s", r.URL.Query().Get("period"))
		}
		json.NewEncoder(w).Encode(BBGoTradingVolumeResponse{TradingVolumes: []interface{}{}})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	_, err := client.GetTradingVolume("day", "")
	if err != nil {
		t.Fatalf("GetTradingVolume() returned error: %v", err)
	}
}

func TestFormatUint(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{18446744073709551615, "18446744073709551615"},
	}
	for _, tt := range tests {
		result := formatUint(tt.input)
		if result != tt.expected {
			t.Errorf("formatUint(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
