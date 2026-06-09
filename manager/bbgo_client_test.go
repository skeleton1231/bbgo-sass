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

