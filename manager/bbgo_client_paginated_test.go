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
		batchSize := int64(syncPageSize)
		if gid >= 2*batchSize {
			// 3rd page: empty
		} else if gid >= batchSize {
			// 2nd page: partial (fewer than pageSize)
			for i := int64(0); i < batchSize/2; i++ {
				id := gid + i + 1
				trades = append(trades, BBGoTrade{
					GID: id, ID: uint64(id), Symbol: "BTCUSDT", Side: "BUY",
					Price: flexString(fmt.Sprintf("%d", 100+id)), Quantity: "1",
				})
			}
		} else {
			// 1st page: full pageSize
			for i := int64(0); i < batchSize; i++ {
				id := gid + i + 1
				trades = append(trades, BBGoTrade{
					GID: id, ID: uint64(id), Symbol: "BTCUSDT", Side: "BUY",
					Price: flexString(fmt.Sprintf("%d", 100+id)), Quantity: "1",
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

	expectedCount := syncPageSize + syncPageSize/2
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

// TestGetAllTrades_OutOfOrderGID verifies that GetAllTradesFrom uses max GID
// for cursor advancement, not the last element's GID. The bbgo API returns
// trades sorted by traded_at DESC, which may not match GID order.
func TestGetAllTrades_OutOfOrderGID(t *testing.T) {
	var lastCursor int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gidStr := r.URL.Query().Get("gid")
		gid, _ := strconv.ParseInt(gidStr, 10, 64)
		lastCursor = gid

		if gid > 4 {
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: nil})
			return
		}
		// Return exactly syncPageSize trades with GIDs out of order.
		// Last element has GID=1 (lowest), but max is syncPageSize+4.
		// Old code would use cursor=1, missing GIDs 2..syncPageSize+4.
		// Fixed code uses cursor=syncPageSize+4 (max), getting empty next page.
		trades := make([]BBGoTrade, syncPageSize)
		for i := 0; i < syncPageSize; i++ {
			gidVal := int64(i + 5)
			trades[i] = BBGoTrade{
				GID: gidVal, ID: uint64(gidVal), Symbol: "BTCUSDT", Side: "BUY",
				Price: "100", Quantity: "1",
			}
		}
		// Swap last two: put low GID at the end (simulating traded_at DESC sort)
		trades[syncPageSize-1] = BBGoTrade{
			GID: 1, ID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "100", Quantity: "1",
		}
		json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: trades})
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	trades, err := client.GetAllTrades("", "")
	if err != nil {
		t.Fatalf("GetAllTrades: %v", err)
	}
	if len(trades) != syncPageSize {
		t.Fatalf("expected %d trades, got %d", syncPageSize, len(trades))
	}
	// Cursor should be max GID (syncPageSize+3 = 503), not last element's GID (1).
	// If cursor were 1, second call would get gid>1 and return ALL trades again.
	if lastCursor != int64(syncPageSize+3) {
		t.Errorf("second call cursor = %d, want %d (max GID, not last element's GID 1)", lastCursor, syncPageSize+3)
	}
}
