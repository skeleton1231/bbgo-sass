package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c9s/bbgo/saas/manager/pool"
)

// --- Sync pipeline tests ---

func setupSyncTest(t *testing.T) (*Syncer, *httptest.Server, *httptest.Server) {
	t.Helper()

	var upsertedOrders []map[string]interface{}
	var upsertedTrades []map[string]interface{}
	var savedCursors map[string]int64

	savedCursors = map[string]int64{}

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: []BBGoOrder{
				{GID: 10, OrderID: 1001, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.1", Status: "FILLED", Type: "LIMIT", ExecutedQuantity: "0.1", CreationTime: "2025-01-01T00:00:00Z"},
				{GID: 8, OrderID: 1002, Symbol: "ETHUSDT", Side: "SELL", Price: "3000", Quantity: "1.0", Status: "FILLED", Type: "LIMIT", ExecutedQuantity: "1.0", CreationTime: "2025-01-01T00:01:00Z"},
				{GID: 3, OrderID: 1003, Symbol: "BTCUSDT", Side: "BUY", Price: "49000", Quantity: "0.05", Status: "FILLED", Type: "LIMIT", ExecutedQuantity: "0.05", CreationTime: "2025-01-01T00:02:00Z"},
			}})
		case "/api/trades":
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: []BBGoTrade{
				{GID: 20, ID: 2001, OrderID: 1001, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.1", Fee: "0.0001", FeeCurrency: "BNB", QuoteQuantity: "5000", TradedAt: "2025-01-01T00:00:01Z"},
				{GID: 15, ID: 2002, OrderID: 1002, Symbol: "ETHUSDT", Side: "SELL", Price: "3000", Quantity: "1.0", Fee: "0.001", FeeCurrency: "BNB", QuoteQuantity: "3000", TradedAt: "2025-01-01T00:01:01Z"},
			}})
		}
	}))

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "sync_cursors"):
			var rows []map[string]interface{}
			tableName := strings.TrimPrefix(r.URL.Query().Get("table_name"), "eq.")
			if gid, ok := savedCursors[tableName]; ok {
				rows = append(rows, map[string]interface{}{"last_gid": gid})
			}
			if rows == nil {
				rows = []map[string]interface{}{}
			}
			json.NewEncoder(w).Encode(rows)

		case r.Method == "POST" && strings.Contains(r.URL.Path, "sync_cursors"):
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			tableName, _ := payload["table_name"].(string)
			gid, _ := payload["last_gid"].(float64)
			savedCursors[tableName] = int64(gid)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)

		case r.Method == "POST" && strings.Contains(r.URL.Path, "sync_orders"):
			var rows []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&rows)
			upsertedOrders = append(upsertedOrders, rows...)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(rows)

		case r.Method == "POST" && strings.Contains(r.URL.Path, "sync_trades"):
			var rows []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&rows)
			upsertedTrades = append(upsertedTrades, rows...)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(rows)

		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]interface{}{})
		}
	}))

	users := NewUserContainerManager()
	syncUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000090"
	users.users[syncUID] = &UserContainer{
		Mode:       ModeLive,
		UserID:     syncUID,
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid2"}},
	}

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	p := pool.New(5)
	cm := &ContainerManager{cfg: cfg}

	syncer := &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{},
		pool:      p,
		newBBGoClientFn: func(_ string) *BBGoClient {
			return NewBBGoClient(bbgoSrv.URL)
		},
	}

	return syncer, bbgoSrv, supabaseSrv
}

func TestSync_IncrementalOrders_OnlyNew(t *testing.T) {
	syncer, bbgoSrv, supaSrv := setupSyncTest(t)
	defer bbgoSrv.Close()
	defer supaSrv.Close()
	defer syncer.pool.Release()

	syncUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000090"

	// Set cursor to 5 — only orders with GID > 5 should sync (GID 8 and 10)
	syncer.updateCursor(syncUID, "sync_orders", 5)

	client := syncer.bbgoClient(syncUID, ModeLive)
	syncer.syncOrdersViaAPI(syncUID, client)

	cursor := syncer.getCursor(syncUID, "sync_orders")
	if cursor != 10 {
		t.Errorf("cursor = %d, want 10", cursor)
	}
}

func TestSync_FullSyncOrders_Paginated(t *testing.T) {
	syncer, bbgoSrv, supaSrv := setupSyncTest(t)
	defer bbgoSrv.Close()
	defer supaSrv.Close()
	defer syncer.pool.Release()

	syncUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000090"

	// Cursor = 0 triggers full sync
	client := syncer.bbgoClient(syncUID, ModeLive)
	syncer.syncOrdersViaAPI(syncUID, client)

	cursor := syncer.getCursor(syncUID, "sync_orders")
	if cursor != 10 {
		t.Errorf("cursor after full sync = %d, want 10 (max GID)", cursor)
	}
}

func TestSync_TradesWithCursor(t *testing.T) {
	syncer, bbgoSrv, supaSrv := setupSyncTest(t)
	defer bbgoSrv.Close()
	defer supaSrv.Close()
	defer syncer.pool.Release()

	syncUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000090"

	// Set trade cursor to 10 — only trades with GID > 10 should be fetched
	syncer.updateCursor(syncUID, "sync_trades", 10)

	client := syncer.bbgoClient(syncUID, ModeLive)
	syncer.syncTradesViaAPI(syncUID, client)

	cursor := syncer.getCursor(syncUID, "sync_trades")
	if cursor != 20 {
		t.Errorf("trade cursor = %d, want 20", cursor)
	}
}

func TestSync_StoppedContainer_SkipsSync(t *testing.T) {
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("bbgo should not be called for stopped container")
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	stoppedUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000091"
	users.users[stoppedUID] = &UserContainer{
		Mode:       ModeLive,
		UserID:     stoppedUID,
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid2"}},
	}

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	p := pool.New(5)
	defer p.Release()

	syncer := &Syncer{
		users:     users,
		cfg:       cfg,
		container: &ContainerManager{cfg: cfg},
		client:    &http.Client{},
		pool:      p,
		newBBGoClientFn: func(_ string) *BBGoClient {
			return NewBBGoClient(bbgoSrv.URL)
		},
	}

	syncer.syncUserData(users.users[stoppedUID])
	// No panic and no bbgo calls = success
}

func TestSync_UnreachableBBGo_SkipsData(t *testing.T) {
	// bbgo server that returns error on ping
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ping" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Data endpoints should NOT be called
		t.Errorf("data endpoint called despite ping failure: %s", r.URL.Path)
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	uid := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000092"
	users.users[uid] = &UserContainer{
		Mode:       ModeLive,
		UserID:     uid,
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid2"}},
	}

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test"}
	p := pool.New(5)
	defer p.Release()

	syncer := &Syncer{
		users:     users,
		cfg:       cfg,
		container: &ContainerManager{cfg: cfg},
		client:    &http.Client{},
		pool:      p,
		newBBGoClientFn: func(_ string) *BBGoClient {
			return NewBBGoClient(bbgoSrv.URL)
		},
	}

	syncer.syncUserData(users.users[uid])
}

// --- Complete trading chain YAML + env tests ---

func TestTradingChain_LiveGrid2_CompleteYAML(t *testing.T) {
	uid := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000093"
	users := NewUserContainerManager()
	users.AddStrategy(uid, ModeLive, StrategyEntry{
		ID:       "strat-1",
		Exchange: "binance",
		Strategy: "grid2",
		Mode:     "live",
		Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
	})

	uc, _ := users.Get(uid, ModeLive)
	yamlBytes, err := buildUserYAML(uc, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("live YAML should not contain PAPER_TRADE")
	}
	if !strings.Contains(yaml, "grid2") {
		t.Error("live YAML should contain strategy name grid2")
	}
	if !strings.Contains(yaml, "binance") {
		t.Error("live YAML should contain exchange binance")
	}

	cm := &ContainerManager{cfg: &Config{MarketDataAddr: "http://market:50051"}}
	args := cm.envArgs(uc)
	for _, arg := range args {
		if strings.HasPrefix(arg, "PAPER_TRADE=") {
			t.Errorf("live mode should not set PAPER_TRADE env, got: %s", arg)
		}
	}
}

func TestTradingChain_PaperGrid2_CompleteYAML(t *testing.T) {
	uid := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000094"
	users := NewUserContainerManager()
	users.AddStrategy(uid, ModePaper, StrategyEntry{
		ID:       "strat-1",
		Exchange: "binance",
		Strategy: "grid2",
		Mode:     "paper",
		Config:   rawJSON(`{"symbol":"BTCUSDT","quantity":0.001}`),
	})

	uc, _ := users.Get(uid, ModePaper)
	yamlBytes, err := buildUserYAML(uc, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("paper YAML should contain PAPER_TRADE environment variable")
	}
	if !strings.Contains(yaml, "grid2") {
		t.Error("paper YAML should contain strategy name grid2")
	}

	cm := &ContainerManager{cfg: &Config{MarketDataAddr: "http://market:50051"}}
	args := cm.envArgs(uc)
	found := false
	for _, arg := range args {
		if arg == "PAPER_TRADE=1" {
			found = true
		}
	}
	if !found {
		t.Error("paper mode envArgs should include PAPER_TRADE=1")
	}
}

func TestTradingChain_CrossExchangeYAML(t *testing.T) {
	uid := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000095"
	users := NewUserContainerManager()
	users.AddStrategy(uid, ModeLive, StrategyEntry{
		ID:            "xstrat-1",
		Strategy:      "xmaker",
		Mode:          "paper",
		Config:        rawJSON(`{"symbol":"BTCUSDT"}`),
		CrossExchange: true,
		Sessions: []SessionRoleConfig{
			{Exchange: "binance", Name: "binance"},
			{Exchange: "bybit", Name: "bybit"},
		},
	})

	uc, _ := users.Get(uid, ModeLive)
	yamlBytes, err := buildUserYAML(uc, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "binance") {
		t.Error("cross-exchange YAML should contain binance session")
	}
	if !strings.Contains(yaml, "bybit") {
		t.Error("cross-exchange YAML should contain bybit session")
	}
	if !strings.Contains(yaml, "xmaker") {
		t.Error("cross-exchange YAML should contain xmaker strategy")
	}
}

func TestTradingChain_LegacyAlias_Normalized(t *testing.T) {
	uid := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000096"
	users := NewUserContainerManager()
	users.AddStrategy(uid, ModeLive, StrategyEntry{
		ID:       "strat-1",
		Exchange: "binance",
		Strategy: "sentinel_anomaly",
		Mode:     "live",
		Config:   rawJSON(`{"symbol":"BTCUSDT"}`),
	})

	uc, _ := users.Get(uid, ModeLive)
	yamlBytes, err := buildUserYAML(uc, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "sentinel") {
		t.Error("legacy alias sentinel_anomaly should be normalized to sentinel in YAML")
	}
	if strings.Contains(yaml, "sentinel_anomaly") {
		t.Error("sentinel_anomaly should not appear as-is in YAML, should be normalized")
	}
}
