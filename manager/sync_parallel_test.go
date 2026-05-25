package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSyncer_SyncAll_Concurrent(t *testing.T) {
	var activeCount atomic.Int32
	var mu sync.Mutex
	syncedUsers := map[string]bool{}

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		activeCount.Add(1)
		defer activeCount.Add(-1)
		time.Sleep(50 * time.Millisecond)

		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: nil})
		case "/api/trades":
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: nil})
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.Method == "POST" && r.URL.Path == "/rest/v1/user_containers" {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if uid, ok := payload["user_id"].(string); ok {
				syncedUsers[uid] = true
			}
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	for i := 0; i < 5; i++ {
		uid := "user-" + string(rune('0'+i))
		users.users[uid] = &UserContainer{
			UserID:     uid,
			Status:     StatusRunning,
			Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
		}
	}

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	syncer := &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{},
	}
	syncer.newBBGoClientFn = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	syncer.SyncAll()

	mu.Lock()
	defer mu.Unlock()
	if len(syncedUsers) != 5 {
		t.Errorf("expected 5 users upserted, got %d", len(syncedUsers))
	}
}

func TestSyncer_SyncAll_SkipsStopped(t *testing.T) {
	var bbgoCalls atomic.Int32

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bbgoCalls.Add(1)
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	users.users["user-stopped"] = &UserContainer{UserID: "user-stopped", Status: StatusStopped}
	users.users["user-running"] = &UserContainer{
		UserID:     "user-running",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	syncer := &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{},
	}
	syncer.newBBGoClientFn = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	syncer.SyncAll()

	if bbgoCalls.Load() == 0 {
		t.Error("expected bbgo calls for running user")
	}
}
