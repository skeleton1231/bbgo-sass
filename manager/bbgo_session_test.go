package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBBGoClient_GetSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions/binance" {
			t.Errorf("expected /api/sessions/binance, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session": map[string]interface{}{
				"name":         "binance",
				"exchange": "binance",
			},
		})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	session, err := client.GetSession("binance")
	if err != nil {
		t.Fatal(err)
	}
	if session.ExchangeName != "binance" {
		t.Errorf("expected exchangeName=binance, got %s", session.ExchangeName)
	}
}

func TestBBGoClient_GetSessionTrades(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"trades": []map[string]interface{}{
				{"symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "0.1"},
			},
		})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetSessionTrades("binance")
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", trades[0].Symbol)
	}
}

func TestBBGoClient_GetSessionOpenOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"orders": []map[string]interface{}{
				{"id": "o1", "symbol": "BTCUSDT", "side": "BUY", "price": "49000", "quantity": "0.5"},
			},
		})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	orders, err := client.GetSessionOpenOrders("binance")
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
}

func TestBBGoClient_GetSessionAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"account": map[string]interface{}{
				"balances": map[string]interface{}{
					"USDT": map[string]string{"available": "10000", "locked": "0"},
				},
			},
		})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	account, err := client.GetSessionAccount("binance")
	if err != nil {
		t.Fatal(err)
	}
	if account == nil {
		t.Fatal("expected account data, got nil")
	}
}
