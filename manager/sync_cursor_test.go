package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

// makeOrderBatch creates n orders with GIDs from start to start+n-1 in DESC order.
func makeOrderBatch(startGID, n int) []BBGoOrder {
	orders := make([]BBGoOrder, n)
	for i := 0; i < n; i++ {
		gid := uint64(startGID + n - 1 - i) // DESC: highest first
		orders[i] = BBGoOrder{
			GID:              gid,
			OrderID:          gid,
			Symbol:           "BTCUSDT",
			Side:             "BUY",
			Type:             "LIMIT",
			Price:            "50000",
			Quantity:         "0.1",
			ExecutedQuantity: "0.1",
			Status:           "FILLED",
			CreationTime:     "2024-01-01",
		}
	}
	return orders
}

// TestSyncer_OrderSync_IncrementalGetsNewOrders verifies that incremental sync
// fetches NEW orders (GID > saved cursor), not old ones.
//
// The bbgo REST API uses DESC ordering: gid=0 returns newest first,
// gid=N returns orders with gid < N (going backward).
// The sync must account for this by fetching without cursor and filtering.
func TestSyncer_OrderSync_IncrementalGetsNewOrders(t *testing.T) {
	var bbgoMu sync.Mutex
	var receivedGIDs []int64

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/closed" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		gidStr := r.URL.Query().Get("gid")
		gid, _ := strconv.ParseInt(gidStr, 10, 64)

		bbgoMu.Lock()
		receivedGIDs = append(receivedGIDs, gid)
		bbgoMu.Unlock()

		// Simulate bbgo DESC behavior:
		// gid=0 → return all orders (newest first)
		// gid=N → return orders with GID < N (backward from cursor)
		if gid == 0 {
			orders := makeOrderBatch(201, 10)
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: orders})
		} else {
			var filtered []BBGoOrder
			for _, o := range makeOrderBatch(1, int(gid-1)) {
				if int64(o.GID) < gid {
					filtered = append(filtered, o)
				}
			}
			if len(filtered) > 500 {
				filtered = filtered[:500]
			}
			if filtered == nil {
				filtered = []BBGoOrder{}
			}
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: filtered})
		}
	}))
	defer bbgoSrv.Close()

	var supabaseMu sync.Mutex
	var syncedOrderIDs []string
	savedCursor := int64(200) // simulate: GIDs 1-200 already synced

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/rest/v1/sync_cursors":
			json.NewEncoder(w).Encode([]struct {
				LastGID int64 `json:"last_gid"`
			}{{LastGID: savedCursor}})

		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_orders":
			var rows []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&rows)
			supabaseMu.Lock()
			for _, row := range rows {
				switch v := row["order_id"].(type) {
				case json.Number:
					syncedOrderIDs = append(syncedOrderIDs, v.String())
				case float64:
					syncedOrderIDs = append(syncedOrderIDs, strconv.FormatInt(int64(v), 10))
				case string:
					syncedOrderIDs = append(syncedOrderIDs, v)
				}
			}
			supabaseMu.Unlock()
			w.WriteHeader(http.StatusOK)

		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_cursors":
			var cur map[string]interface{}
			json.NewDecoder(r.Body).Decode(&cur)
			if cur["last_gid"] != nil {
				supabaseMu.Lock()
				if f, ok := cur["last_gid"].(float64); ok {
					savedCursor = int64(f)
				}
				supabaseMu.Unlock()
			}
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	syncer := &Syncer{
		cfg:       cfg,
		container: &ContainerManager{cfg: cfg},
		client:    &http.Client{},
	}

	client := NewBBGoClient(bbgoSrv.URL)
	syncer.syncOrdersViaAPI("user-1", client)

	supabaseMu.Lock()
	defer supabaseMu.Unlock()

	if len(syncedOrderIDs) != 10 {
		t.Fatalf("expected 10 new orders synced, got %d (order IDs: %v)", len(syncedOrderIDs), syncedOrderIDs)
	}

	for _, id := range syncedOrderIDs {
		gid, _ := strconv.ParseInt(id, 10, 64)
		if gid <= 200 {
			t.Errorf("synced old order GID %d (should only sync GID > 200)", gid)
		}
	}

	if savedCursor != 210 {
		t.Errorf("expected cursor 210, got %d", savedCursor)
	}
}
