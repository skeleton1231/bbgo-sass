package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestSyncer_IncrementalOrderSync_Overflow verifies that when there are more
// than 500 new orders since last sync, the incremental sync does not miss orders.
//
// Scenario: cursor=150, 600 new orders exist (GIDs 151-750).
// API with gid=0 returns latest 500 (GIDs 251-750 in DESC).
// Orders 151-250 would be missed without overflow handling.
func TestSyncer_IncrementalOrderSync_Overflow(t *testing.T) {
	var bbgoMu sync.Mutex
	var bbgoRequests []string

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/closed" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		gidStr := r.URL.Query().Get("gid")

		bbgoMu.Lock()
		bbgoRequests = append(bbgoRequests, "gid="+gidStr)
		bbgoMu.Unlock()

		// gid=0 returns LATEST 500 (DESC): GIDs 750 down to 251
		if gidStr == "0" {
			orders := makeOrderBatch(251, 500)
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: orders})
			return
		}

		// gid=N returns orders with GID < N, DESC, limit 500
		gid := parseInt(gidStr)
		if gid <= 251 {
			// Remaining orders: 151-250
			orders := makeOrderBatch(151, int(gid-151))
			if len(orders) > 500 {
				orders = orders[:500]
			}
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: orders})
			return
		}
		orders := makeOrderBatch(1, int(gid-1))
		if len(orders) > 500 {
			orders = orders[:500]
		}
		json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: orders})
	}))
	defer bbgoSrv.Close()

	var supabaseMu sync.Mutex
	savedCursor := int64(150)
	var totalUpserted int

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/rest/v1/sync_cursors":
			supabaseMu.Lock()
			cur := savedCursor
			supabaseMu.Unlock()
			json.NewEncoder(w).Encode([]struct {
				LastGID int64 `json:"last_gid"`
			}{{LastGID: cur}})

		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_orders":
			var rows []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&rows)
			supabaseMu.Lock()
			totalUpserted += len(rows)
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

	if savedCursor != 750 {
		t.Errorf("cursor should be 750 (all 600 new orders), got %d", savedCursor)
	}
	if totalUpserted != 600 {
		t.Errorf("expected 600 orders synced (GIDs 151-750), got %d", totalUpserted)
	}
}

func parseInt(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
