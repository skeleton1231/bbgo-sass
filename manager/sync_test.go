package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestSyncer_SyncUserData_PingFails(t *testing.T) {
	users := NewUserContainerManager()
	users.users["user-1"] = &UserContainer{
		UserID:     "user-1",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	cfg := &Config{SupabaseURL: "http://localhost:99999", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	syncer := NewSyncer(users, cfg, cm, nil)

	uc := &UserContainer{UserID: "user-1", Status: StatusRunning}
	// Should not panic when bbgo is unreachable
	syncer.syncUserData(uc)
}

func TestSyncer_SyncOrdersViaAPI(t *testing.T) {
	var supabaseMu sync.Mutex
	var receivedOrders []map[string]interface{}
	var cursorUpdates []map[string]interface{}

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/orders/closed" {
			if r.URL.Query().Get("gid") != "0" {
				t.Errorf("expected gid=0 for initial sync, got %s", r.URL.Query().Get("gid"))
			}
			json.NewEncoder(w).Encode(BBGoOrdersResponse{
				Orders: []BBGoOrder{
					{GID: 10, OrderID: 100, Symbol: "BTCUSDT", Side: "BUY", Type: "LIMIT", Price: "50000", Quantity: "0.1", Status: "FILLED"},
					{GID: 11, OrderID: 101, Symbol: "ETHUSDT", Side: "SELL", Type: "MARKET", Price: "3000", Quantity: "1.5", Status: "FILLED"},
				},
			})
			return
		}
		if r.URL.Path == "/api/ping" {
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
			return
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/rest/v1/sync_orders" {
			var orders []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&orders)
			supabaseMu.Lock()
			receivedOrders = orders
			supabaseMu.Unlock()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(orders)
			return
		}
		if (r.Method == "PATCH" || r.Method == "POST") && r.URL.Path == "/rest/v1/sync_cursors" {
			var cur map[string]interface{}
			json.NewDecoder(r.Body).Decode(&cur)
			supabaseMu.Lock()
			cursorUpdates = append(cursorUpdates, cur)
			supabaseMu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "GET" && r.URL.Path == "/rest/v1/sync_cursors" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	users.users["user-1"] = &UserContainer{UserID: "user-1", Status: StatusRunning}

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}

	syncer := &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{},
	}

	client := NewBBGoClient(bbgoSrv.URL)
	syncer.syncOrdersViaAPI("user-1", client)

	supabaseMu.Lock()
	defer supabaseMu.Unlock()
	if len(receivedOrders) != 2 {
		t.Fatalf("expected 2 orders synced to supabase, got %d", len(receivedOrders))
	}
	if receivedOrders[0]["symbol"] != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %v", receivedOrders[0]["symbol"])
	}
	if receivedOrders[1]["side"] != "SELL" {
		t.Errorf("expected SELL, got %v", receivedOrders[1]["side"])
	}
	if len(cursorUpdates) != 1 {
		t.Fatalf("expected 1 cursor update, got %d", len(cursorUpdates))
	}
	if cursorUpdates[0]["last_gid"] == nil {
		t.Errorf("expected cursor table_name sync_orders, got %v", cursorUpdates[0]["table_name"])
	}
}

func TestSyncer_SyncTradesViaAPI(t *testing.T) {
	var supabaseMu sync.Mutex
	var receivedTrades []map[string]interface{}
	var cursorUpdates []map[string]interface{}

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trades" {
			json.NewEncoder(w).Encode(BBGoTradesResponse{
				Trades: []BBGoTrade{
					{GID: 20, ID: 500, OrderID: 100, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.1", Fee: "0.05", FeeCurrency: "BNB"},
				},
			})
			return
		}
		if r.URL.Path == "/api/ping" {
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
			return
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/rest/v1/sync_trades" {
			var trades []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&trades)
			supabaseMu.Lock()
			receivedTrades = trades
			supabaseMu.Unlock()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(trades)
			return
		}
		if (r.Method == "PATCH" || r.Method == "POST") && r.URL.Path == "/rest/v1/sync_cursors" {
			var cur map[string]interface{}
			json.NewDecoder(r.Body).Decode(&cur)
			supabaseMu.Lock()
			cursorUpdates = append(cursorUpdates, cur)
			supabaseMu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "GET" && r.URL.Path == "/rest/v1/sync_cursors" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	users.users["user-1"] = &UserContainer{UserID: "user-1", Status: StatusRunning}

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}

	syncer := &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{},
	}

	client := NewBBGoClient(bbgoSrv.URL)
	syncer.syncTradesViaAPI("user-1", client)

	supabaseMu.Lock()
	defer supabaseMu.Unlock()
	if len(receivedTrades) != 1 {
		t.Fatalf("expected 1 trade synced to supabase, got %d", len(receivedTrades))
	}
	if receivedTrades[0]["fee_currency"] != "BNB" {
		t.Errorf("expected BNB, got %v", receivedTrades[0]["fee_currency"])
	}
	if len(cursorUpdates) != 1 {
		t.Fatalf("expected 1 cursor update, got %d", len(cursorUpdates))
	}
	if cursorUpdates[0]["last_gid"] == nil {
		t.Errorf("expected cursor table_name sync_trades, got %v", cursorUpdates[0]["table_name"])
	}
}

func TestSyncer_SyncUserData_SkipsStoppedContainer(t *testing.T) {
	syncer := &Syncer{}
	uc := &UserContainer{UserID: "user-1", Status: StatusStopped}
	// Should not panic and should skip
	syncer.syncUserData(uc)
}

func TestSyncer_SyncOrdersViaAPI_EmptyResponse(t *testing.T) {
	var supabaseCalled bool

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: nil})
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/sync_cursors" && r.Method == "GET" {
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		supabaseCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	syncer := &Syncer{
		cfg:    cfg,
		client: &http.Client{},
	}

	client := NewBBGoClient(bbgoSrv.URL)
	syncer.syncOrdersViaAPI("user-1", client)

	if supabaseCalled {
		t.Error("supabase should not be called when orders are empty")
	}
}

func TestSyncer_LoadUsersFromSupabase(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/user_containers" {
			t.Errorf("expected /rest/v1/user_containers, got %s", r.URL.Path)
		}
		if r.Header.Get("apikey") != "test-key" {
			t.Errorf("expected apikey test-key, got %s", r.Header.Get("apikey"))
		}

		json.NewEncoder(w).Encode([]struct {
			UserID     string          `json:"user_id"`
			Status     string          `json:"status"`
			Strategies json.RawMessage `json:"strategies"`
		}{
			{
				UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				Status:     "running",
				Strategies: json.RawMessage(`[{"id":"s1","exchange":"binance","strategy":"grid"}]`),
			},
		})
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test-key"}
	syncer := NewSyncer(NewUserContainerManager(), cfg, &ContainerManager{cfg: cfg}, nil)

	users, err := syncer.LoadUsersFromSupabase()
	if err != nil {
		t.Fatalf("LoadUsersFromSupabase() error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].UserID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("unexpected user ID: %s", users[0].UserID)
	}
	if len(users[0].Strategies) != 1 {
		t.Errorf("expected 1 strategy, got %d", len(users[0].Strategies))
	}
}

func TestSyncer_LoadUsersFromSupabase_Non200(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	syncer := NewSyncer(NewUserContainerManager(), cfg, &ContainerManager{cfg: cfg}, nil)

	users, err := syncer.LoadUsersFromSupabase()
	if err == nil {
		t.Fatal("expected error on non-200 response, got nil")
	}
	if users != nil {
		t.Fatalf("expected nil users on error, got %d", len(users))
	}
}

func TestSyncer_SyncOrdersWithExistingCursor(t *testing.T) {
	var mu sync.Mutex
	var receivedGIDs []string

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/orders/closed" {
			mu.Lock()
			receivedGIDs = append(receivedGIDs, r.URL.Query().Get("gid"))
			mu.Unlock()

			// Return one new order (GID=200 > cursor 150) and one old (GID=100 < cursor)
			json.NewEncoder(w).Encode(BBGoOrdersResponse{
				Orders: []BBGoOrder{
					{GID: 200, OrderID: 200, Symbol: "ETHUSDT", Side: "BUY", Type: "LIMIT", Price: "2500", Quantity: "2.0", Status: "FILLED"},
					{GID: 100, OrderID: 100, Symbol: "ETHUSDT", Side: "SELL", Type: "LIMIT", Price: "2600", Quantity: "1.0", Status: "FILLED"},
				},
			})
			return
		}
	}))
	defer bbgoSrv.Close()

	var upsertedOrders []map[string]interface{}
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/rest/v1/sync_cursors" {
			json.NewEncoder(w).Encode([]struct {
				LastGID int64 `json:"last_gid"`
			}{{LastGID: 150}})
			return
		}
		if r.Method == "POST" && r.URL.Path == "/rest/v1/sync_orders" {
			var rows []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&rows)
			mu.Lock()
			upsertedOrders = append(upsertedOrders, rows...)
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		if (r.Method == "PATCH" || r.Method == "POST") && r.URL.Path == "/rest/v1/sync_cursors" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
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

	mu.Lock()
	defer mu.Unlock()

	// Incremental sync sends gid=0 to get latest orders, not gid=cursor
	if len(receivedGIDs) != 1 || receivedGIDs[0] != "0" {
		t.Errorf("expected gid=0 for incremental sync, got %v", receivedGIDs)
	}

	// Only the order with GID > cursor (150) should be synced
	if len(upsertedOrders) != 1 {
		t.Fatalf("expected 1 new order (GID=200 > cursor 150), got %d", len(upsertedOrders))
	}
	switch v := upsertedOrders[0]["order_id"].(type) {
	case json.Number:
		if v.String() != "200" {
			t.Errorf("expected order_id=200, got %s", v.String())
		}
	case float64:
		if v != 200 {
			t.Errorf("expected order_id=200, got %v", v)
		}
	default:
		t.Errorf("expected order_id=200, got %v (%T)", upsertedOrders[0]["order_id"], upsertedOrders[0]["order_id"])
	}
}
