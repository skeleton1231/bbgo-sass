package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestGetAllTrades_Pagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		gidStr := r.URL.Query().Get("gid")
		gid, _ := strconv.ParseInt(gidStr, 10, 64)

		var trades []BBGoTrade
		batchSize := int64(tradesPageSize)
		if gid >= 2*batchSize {
			// 3rd page: empty
		} else if gid >= batchSize {
			// 2nd page: partial (fewer than pageSize)
			for i := int64(0); i < batchSize/2; i++ {
				id := gid + i + 1
				trades = append(trades, BBGoTrade{
					GID: id, ID: uint64(id), Symbol: "BTCUSDT", Side: "BUY",
					Price: fmt.Sprintf("%d", 100+id), Quantity: "1",
				})
			}
		} else {
			// 1st page: full pageSize
			for i := int64(0); i < batchSize; i++ {
				id := gid + i + 1
				trades = append(trades, BBGoTrade{
					GID: id, ID: uint64(id), Symbol: "BTCUSDT", Side: "BUY",
					Price: fmt.Sprintf("%d", 100+id), Quantity: "1",
				})
			}
		}
		json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: trades})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetAllTrades("", "")
	if err != nil {
		t.Fatalf("GetAllTrades: %v", err)
	}

	expectedCount := tradesPageSize + tradesPageSize/2
	if len(trades) != expectedCount {
		t.Errorf("expected %d trades, got %d", expectedCount, len(trades))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
	// Verify GIDs are ordered
	for i, tr := range trades {
		expected := int64(i + 1)
		if tr.GID != expected {
			t.Errorf("trades[%d].GID = %d, want %d", i, tr.GID, expected)
		}
	}
}

func TestGetAllTrades_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: nil})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetAllTrades("", "")
	if err != nil {
		t.Fatalf("GetAllTrades: %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("expected 0 trades, got %d", len(trades))
	}
}
