package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Sync chain: incremental order sync ---

func TestSyncChain_OrdersIncremental(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	// Supabase mock: tracks cursor and upserted orders
	var savedCursor int64
	var upsertedOrders []map[string]interface{}

	supaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && len(r.URL.Query()["table_name"]) > 0 && r.URL.Query()["table_name"][0] == "sync_orders":
			w.WriteHeader(http.StatusOK)
			cursorVal := 0
			if savedCursor > 0 {
				cursorVal = int(savedCursor)
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{{"last_gid": cursorVal}})

		case r.Method == "GET" && len(r.URL.Query()["table_name"]) > 0 && r.URL.Query()["table_name"][0] == "sync_trades":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{{"last_gid": 0}})

		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_orders":
			var orders []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&orders)
			upsertedOrders = append(upsertedOrders, orders...)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_cursors":
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if gid, ok := payload["last_gid"].(float64); ok {
				savedCursor = int64(gid)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	}))
	defer supaSrv.Close()

	// bbgo mock: returns orders with GIDs > cursor
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ping" {
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
			return
		}
		if r.URL.Path == "/api/orders/closed" {
			json.NewEncoder(w).Encode(BBGoOrdersResponse{
				Orders: []BBGoOrder{
					{GID: 101, OrderID: 1001, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Status: "FILLED"},
					{GID: 102, OrderID: 1002, Symbol: "ETHUSDT", Side: "SELL", Price: "3000", Status: "FILLED"},
				},
			})
			return
		}
		if r.URL.Path == "/api/trades" {
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: []BBGoTrade{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	users := NewUserContainerManager()
	users.users[userID+":"+ModeLive] = &UserContainer{UserID: userID, Status: StatusRunning, Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance"}}}

	cfg := &Config{SupabaseURL: supaSrv.URL, SupabaseKey: "test-key"}
	cm := &ContainerManager{cfg: cfg}
	cm2 := &ContainerManager{cfg: cfg}
	_ = cm2

	syncer := NewSyncer(users, cfg, cm, nil)
	syncer.newBBGoClientFn = func(baseURL string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	// First sync: cursor=0 → fullSyncOrders → paginates and saves
	savedCursor = 100
	syncer.SyncUser(userID, ModeLive)

	if len(upsertedOrders) < 2 {
		t.Fatalf("expected at least 2 upserted orders, got %d", len(upsertedOrders))
	}
}

// --- Sync chain: bbgo client non-200 error ---

func TestBBGoClient_Non200ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	err := client.Ping()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// --- BBGo client malformed JSON ---

func TestBBGoClient_MalformedJSONReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json at all`))
	}))
	defer srv.Close()

	client := NewBBGoClient(srv.URL)
	err := client.Ping()
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// --- Credential round-trip: encrypt → store → retrieve → decrypt ---

func TestCredentialRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	store := NewCredentialStore(tmpDir, enc)
	userID := "user-rt-test"

	apiKey, apiSecret, passphrase := "myApiKey123", "mySecret456", "myPass789"

	encKey, err := enc.Encrypt(apiKey)
	if err != nil {
		t.Fatal(err)
	}
	encSecret, err := enc.Encrypt(apiSecret)
	if err != nil {
		t.Fatal(err)
	}
	encPass, err := enc.Encrypt(passphrase)
	if err != nil {
		t.Fatal(err)
	}

	cred := ExchangeCredential{
		ID:                  "cred-1",
		UserID:              userID,
		Exchange:            "binance",
		APIKeyEncrypted:     encKey,
		APISecretEncrypted:  encSecret,
		PassphraseEncrypted: encPass,
	}

	if err := store.Upsert(cred); err != nil {
		t.Fatal(err)
	}

	gotKey, gotSecret, gotPass, err := store.GetDecrypted(userID, "binance")
	if err != nil {
		t.Fatal(err)
	}

	if gotKey != apiKey {
		t.Errorf("apiKey: expected %q, got %q", apiKey, gotKey)
	}
	if gotSecret != apiSecret {
		t.Errorf("apiSecret: expected %q, got %q", apiSecret, gotSecret)
	}
	if gotPass != passphrase {
		t.Errorf("passphrase: expected %q, got %q", passphrase, gotPass)
	}
}

// --- Credential round-trip without passphrase ---

func TestCredentialRoundTrip_NoPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	store := NewCredentialStore(tmpDir, enc)
	userID := "user-no-pass"

	encKey, _ := enc.Encrypt("key1")
	encSecret, _ := enc.Encrypt("secret1")

	cred := ExchangeCredential{
		ID:                 "cred-2",
		UserID:             userID,
		Exchange:           "binance",
		APIKeyEncrypted:    encKey,
		APISecretEncrypted: encSecret,
	}

	if err := store.Upsert(cred); err != nil {
		t.Fatal(err)
	}

	gotKey, gotSecret, gotPass, err := store.GetDecrypted(userID, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != "key1" || gotSecret != "secret1" {
		t.Errorf("expected key1/secret1, got %s/%s", gotKey, gotSecret)
	}
	if gotPass != "" {
		t.Errorf("expected empty passphrase, got %q", gotPass)
	}
}

// --- Wrong encryption key fails to decrypt ---

func TestCredential_WrongKeyFails(t *testing.T) {
	tmpDir := t.TempDir()
	enc1, _ := NewEncryptor(testEncryptionKey)
	enc2, _ := NewEncryptor("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=") // different valid 32-byte key

	store := NewCredentialStore(tmpDir, enc1)
	encKey, _ := enc1.Encrypt("sensitive-key")
	encSecret, _ := enc1.Encrypt("sensitive-secret")

	cred := ExchangeCredential{
		ID: "cred-3", UserID: "user-wrong", Exchange: "binance",
		APIKeyEncrypted: encKey, APISecretEncrypted: encSecret,
	}
	store.Upsert(cred)

	// Try reading with wrong key
	badStore := NewCredentialStore(tmpDir, enc2)
	_, _, _, err := badStore.GetDecrypted("user-wrong", "binance")
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}

// --- Credential not found ---

func TestCredential_GetDecrypted_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	store := NewCredentialStore(tmpDir, enc)

	_, _, _, err := store.GetDecrypted("nonexistent-user", "binance")
	if err == nil {
		t.Error("expected error for nonexistent credentials")
	}
}

// --- Sync skips non-running users ---

func TestSyncChain_SkipsNonRunningUsers(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	pingCalled := false

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pingCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer bbgoSrv.Close()

	users := NewUserContainerManager()
	users.users[userID+":"+ModeLive] = &UserContainer{UserID: userID, Status: StatusStopped}

	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}

	syncer := NewSyncer(users, cfg, cm, nil)
	syncer.newBBGoClientFn = func(baseURL string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	syncer.syncUserData(users.users[userID+":"+ModeLive])
	if pingCalled {
		t.Error("sync should not call bbgo API for stopped user")
	}
}

// --- Sync ping failure aborts sync ---

func TestSyncChain_PingFailure_AbortsSync(t *testing.T) {
	userID := "user-ping-fail"
	pingCalled := false
	ordersCalled := false

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ping" {
			pingCalled = true
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/api/orders/closed" {
			ordersCalled = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer bbgoSrv.Close()

	users := NewUserContainerManager()
	users.users[userID+":"+ModeLive] = &UserContainer{UserID: userID, Status: StatusRunning}

	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}

	syncer := NewSyncer(users, cfg, cm, nil)
	syncer.newBBGoClientFn = func(baseURL string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	syncer.syncUserData(users.users[userID+":"+ModeLive])
	if !pingCalled {
		t.Error("expected ping to be called")
	}
	if ordersCalled {
		t.Error("orders should not be called after ping failure")
	}
}

// --- Upsert replaces credential for same exchange ---

func TestCredential_UpsertReplaces(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	store := NewCredentialStore(tmpDir, enc)
	userID := "user-upsert"

	for i, key := range []string{"key-v1", "key-v2"} {
		encKey, _ := enc.Encrypt(key)
		encSecret, _ := enc.Encrypt("secret")
		cred := ExchangeCredential{
			ID: fmt.Sprintf("cred-%d", i+1), UserID: userID, Exchange: "binance",
			APIKeyEncrypted: encKey, APISecretEncrypted: encSecret,
		}
		if err := store.Upsert(cred); err != nil {
			t.Fatal(err)
		}
	}

	gotKey, _, _, err := store.GetDecrypted(userID, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != "key-v2" {
		t.Errorf("expected key-v2 after upsert, got %q", gotKey)
	}

	list, _ := store.List(userID)
	if len(list) != 1 {
		t.Errorf("expected 1 credential after upsert, got %d", len(list))
	}
}

// --- Sync trade cursor advances ---

func TestSyncChain_TradeCursorAdvances(t *testing.T) {
	userID := "user-trade-cursor"
	var savedCursor int64

	supaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && len(r.URL.Query()["table_name"]) > 0 && r.URL.Query()["table_name"][0] == "sync_trades":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{{"last_gid": savedCursor}})

		case r.Method == "GET" && len(r.URL.Query()["table_name"]) > 0 && r.URL.Query()["table_name"][0] == "sync_orders":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{{"last_gid": 999}})

		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_trades":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		case r.Method == "POST" && r.URL.Path == "/rest/v1/sync_cursors":
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if gid, ok := payload["last_gid"].(float64); ok {
				if int64(gid) > savedCursor {
					savedCursor = int64(gid)
				}
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	}))
	defer supaSrv.Close()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: []BBGoOrder{}})
		case "/api/trades":
			json.NewEncoder(w).Encode(BBGoTradesResponse{
				Trades: []BBGoTrade{
					{GID: 10, ID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000"},
					{GID: 20, ID: 2, Symbol: "ETHUSDT", Side: "SELL", Price: "3000"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer bbgoSrv.Close()

	users := NewUserContainerManager()
	users.users[userID+":"+ModeLive] = &UserContainer{UserID: userID, Status: StatusRunning}

	cfg := &Config{SupabaseURL: supaSrv.URL, SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}

	syncer := NewSyncer(users, cfg, cm, nil)
	syncer.newBBGoClientFn = func(baseURL string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	syncer.syncUserData(users.users[userID+":"+ModeLive])

	time.Sleep(50 * time.Millisecond)
	if savedCursor < 20 {
		t.Errorf("expected cursor >= 20, got %d", savedCursor)
	}
}

// --- Supabase request rejection ---

func TestSyncChain_SupabaseRejection_AbortsSync(t *testing.T) {
	userID := "user-supa-fail"

	supaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("database unavailable"))
	}))
	defer supaSrv.Close()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ping" {
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
			return
		}
		if r.URL.Path == "/api/orders/closed" {
			json.NewEncoder(w).Encode(BBGoOrdersResponse{
				Orders: []BBGoOrder{
					{GID: 1, OrderID: 1, Symbol: "BTCUSDT", Side: "BUY", Status: "FILLED"},
				},
			})
			return
		}
		if r.URL.Path == "/api/trades" {
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: []BBGoTrade{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	users := NewUserContainerManager()
	users.users[userID+":"+ModeLive] = &UserContainer{UserID: userID, Status: StatusRunning}

	cfg := &Config{SupabaseURL: supaSrv.URL, SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}

	syncer := NewSyncer(users, cfg, cm, nil)
	syncer.newBBGoClientFn = func(baseURL string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	// Should not panic or hang on Supabase rejection
	syncer.syncUserData(users.users[userID+":"+ModeLive])
}
