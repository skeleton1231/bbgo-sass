package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
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

	strategies := []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	}

	syncer := NewSyncerWithCreds(nil, creds)

	syncer.MarkCredentialsVerified("user-1", ModeLive, strategies)

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

	strategies := []StrategyEntry{
		{
			Strategy:      "xmaker",
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
				{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT"},
			},
		},
	}

	syncer := NewSyncerWithCreds(nil, creds)

	syncer.MarkCredentialsVerified("user-1", ModeLive, strategies)

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

	strategies := []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	}

	syncer := NewSyncerWithCreds(nil, creds)

	syncer.MarkCredentialsVerified("user-1", ModeLive, strategies)

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

	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	syncer := NewSyncerWithCreds(supaClient, NewCredentialStore(dir, enc))

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

	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	syncer := NewSyncerWithCreds(supaClient, NewCredentialStore(dir, enc))

	syncer.DeleteCredential("user-1", "binance", false)

	mu.Lock()
	defer mu.Unlock()
	if deletedMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", deletedMethod)
	}
	if deletedPath != "/rest/v1/exchange_credentials" {
		t.Errorf("expected path /rest/v1/exchange_credentials, got %s", deletedPath)
	}
}

// --- Session names: mixed single + cross-exchange strategies ---

func TestExtractSessionNames_MixedSingleAndCrossExchange(t *testing.T) {
	strategies := []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
		{
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance"},
				{Name: "hedge", Exchange: "bybit"},
			},
		},
	}
	sessions := extractSessionNames(strategies)

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
	strategies := []StrategyEntry{
		{
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance"},
				{Name: "hedge", Exchange: "bybit"},
			},
		},
		{Exchange: "binance", Strategy: "dca"},
	}
	sessions := extractSessionNames(strategies)

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
