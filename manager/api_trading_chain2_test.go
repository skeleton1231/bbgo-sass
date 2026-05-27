package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/c9s/bbgo/saas/manager/pool"
)

// --- Sync chain: credential verification during data sync ---

func TestSyncer_MarkCredentialsVerified(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "user-1", "binance", "key", "secret")

	users := NewUserContainerManager()
	users.users["user-1"] = &UserContainer{
		Mode:   ModeLive,
		UserID: "user-1",
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live"},
		},
	}

	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	syncer := NewSyncerWithCreds(users, cfg, cm, creds, pool.New(1))

	syncer.markCredentialsVerified(users.users["user-1"])

	stored, err := creds.List("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(stored))
	}
	if !stored[0].IsVerified {
		t.Error("credential should be marked as verified after sync with matching strategy exchange")
	}
}

func TestSyncer_MarkCredentialsVerified_CrossExchange(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "user-1", "binance", "bin-key", "bin-secret")
	insertTestCredential(t, creds, "user-1", "bybit", "byb-key", "byb-secret")

	users := NewUserContainerManager()
	users.users["user-1"] = &UserContainer{
		Mode:   ModeLive,
		UserID: "user-1",
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT"},
				},
			},
		},
	}

	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	syncer := NewSyncerWithCreds(users, cfg, cm, creds, pool.New(1))

	syncer.markCredentialsVerified(users.users["user-1"])

	stored, _ := creds.List("user-1")
	for _, c := range stored {
		if !c.IsVerified {
			t.Errorf("credential for %s should be verified after cross-exchange sync", c.Exchange)
		}
	}
}

func TestSyncer_MarkCredentialsVerified_UnusedExchange(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "user-1", "binance", "key", "secret")
	insertTestCredential(t, creds, "user-1", "okex", "unused-key", "unused-secret")

	users := NewUserContainerManager()
	users.users["user-1"] = &UserContainer{
		Mode:   ModeLive,
		UserID: "user-1",
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live"},
		},
	}

	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	syncer := NewSyncerWithCreds(users, cfg, cm, creds, pool.New(1))

	syncer.markCredentialsVerified(users.users["user-1"])

	stored, _ := creds.List("user-1")
	for _, c := range stored {
		if c.Exchange == "binance" && !c.IsVerified {
			t.Error("binance credential should be verified")
		}
		if c.Exchange == "okex" && c.IsVerified {
			t.Error("okex credential should NOT be verified (no strategy uses it)")
		}
	}
}

// --- SyncCredential to Supabase ---

func TestSyncer_SyncCredential(t *testing.T) {
	var mu sync.Mutex
	var received map[string]interface{}

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/rest/v1/exchange_credentials" {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			mu.Lock()
			received = payload
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test-key"}
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	syncer := NewSyncerWithCreds(NewUserContainerManager(), cfg, &ContainerManager{cfg: cfg}, NewCredentialStore(dir, enc), nil)

	keyEnc, _ := enc.Encrypt("my-key")
	secretEnc, _ := enc.Encrypt("my-secret")
	syncer.SyncCredential(ExchangeCredential{
		UserID:             "user-1",
		Exchange:           "binance",
		APIKeyEncrypted:    keyEnc,
		APISecretEncrypted: secretEnc,
	})

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("expected credential to be synced to Supabase")
	}
	if received["exchange"] != "binance" {
		t.Errorf("expected exchange=binance, got %v", received["exchange"])
	}
	if received["user_id"] != "user-1" {
		t.Errorf("expected user_id=user-1, got %v", received["user_id"])
	}
}

func TestSyncer_DeleteCredential(t *testing.T) {
	var mu sync.Mutex
	var deletedMethod, deletedPath string

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deletedMethod = r.Method
		deletedPath = r.URL.Path
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer supabaseSrv.Close()

	cfg := &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "test-key"}
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	syncer := NewSyncerWithCreds(NewUserContainerManager(), cfg, &ContainerManager{cfg: cfg}, NewCredentialStore(dir, enc), nil)

	syncer.DeleteCredential("user-1", "binance")

	mu.Lock()
	defer mu.Unlock()
	if deletedMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", deletedMethod)
	}
	if deletedPath != "/rest/v1/exchange_credentials" {
		t.Errorf("expected path /rest/v1/exchange_credentials, got %s", deletedPath)
	}
}

// --- Full sync chain: running container → ping → sync orders + trades ---

func TestSyncer_SyncUserData_FullChain(t *testing.T) {
	var supabaseMu sync.Mutex
	var syncedOrders []map[string]interface{}
	var syncedTrades []map[string]interface{}

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(BBGoOrdersResponse{
				Orders: []BBGoOrder{
					{GID: 1, OrderID: 100, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.1", Status: "FILLED"},
				},
			})
		case "/api/trades":
			json.NewEncoder(w).Encode(BBGoTradesResponse{
				Trades: []BBGoTrade{
					{GID: 1, ID: 500, OrderID: 100, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.1", Fee: "0.05"},
				},
			})
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/rest/v1/sync_cursors":
			json.NewEncoder(w).Encode([]interface{}{})
		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_orders":
			var rows []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&rows)
			supabaseMu.Lock()
			syncedOrders = rows
			supabaseMu.Unlock()
			w.WriteHeader(http.StatusOK)
		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_trades":
			var rows []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&rows)
			supabaseMu.Lock()
			syncedTrades = rows
			supabaseMu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer supabaseSrv.Close()

	users := NewUserContainerManager()
	users.users["user-1"] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "user-1",
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

	syncer.syncUserData(users.users["user-1"])

	supabaseMu.Lock()
	defer supabaseMu.Unlock()

	if len(syncedOrders) != 1 {
		t.Fatalf("expected 1 order synced, got %d", len(syncedOrders))
	}
	if syncedOrders[0]["symbol"] != "BTCUSDT" {
		t.Errorf("expected BTCUSDT order, got %v", syncedOrders[0]["symbol"])
	}
	if len(syncedTrades) != 1 {
		t.Fatalf("expected 1 trade synced, got %d", len(syncedTrades))
	}
	if syncedTrades[0]["price"] != "50000" {
		t.Errorf("expected trade price 50000, got %v", syncedTrades[0]["price"])
	}
}

// --- Session names: mixed single + cross-exchange strategies ---

func TestExtractSessionNames_MixedSingleAndCrossExchange(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2"},
			{
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance"},
					{Name: "hedge", Exchange: "bybit"},
				},
			},
		},
	}
	sessions := extractSessionNames(uc)

	// Single-exchange uses exchange name, cross-exchange uses session Name field
	seen := map[string]bool{}
	for _, s := range sessions {
		seen[s] = true
	}
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions (binance + maker + hedge), got %d: %v", len(sessions), sessions)
	}
	if !seen["binance"] || !seen["maker"] || !seen["hedge"] {
		t.Errorf("expected binance+maker+hedge, got %v", sessions)
	}
}

func TestExtractSessionNames_CrossExchangeDeduplicates(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance"},
					{Name: "hedge", Exchange: "bybit"},
				},
			},
			{Exchange: "binance", Strategy: "dca"},
		},
	}
	sessions := extractSessionNames(uc)

	binanceCount := 0
	for _, s := range sessions {
		if s == "binance" {
			binanceCount++
		}
	}
	if binanceCount != 1 {
		t.Errorf("expected binance deduplicated to 1, got %d", binanceCount)
	}
}
