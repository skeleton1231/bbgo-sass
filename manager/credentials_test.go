package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func testEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc, err := NewEncryptor(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func TestCredentialStore_UpsertAndList(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	cred := ExchangeCredential{
		ID:       "c1",
		UserID:   "u1",
		Exchange: "binance",
	}
	if err := cs.Upsert(cred); err != nil {
		t.Fatal(err)
	}

	creds, err := cs.List("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 || creds[0].Exchange != "binance" {
		t.Errorf("List = %v", creds)
	}
}

func TestCredentialStore_UpsertReplaces(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance"})
	cs.Upsert(ExchangeCredential{ID: "c2", UserID: "u1", Exchange: "binance"})

	creds, _ := cs.List("u1")
	if len(creds) != 1 {
		t.Errorf("upsert should replace, got %d creds", len(creds))
	}
}

func TestCredentialStore_UpsertSeparatesModes(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance", IsTestnet: false})
	cs.Upsert(ExchangeCredential{ID: "c2", UserID: "u1", Exchange: "binance", IsTestnet: true})

	creds, _ := cs.List("u1")
	if len(creds) != 2 {
		t.Errorf("live and testnet should be separate, got %d", len(creds))
	}
}

func TestCredentialStore_ListEmpty(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	creds, err := cs.List("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 0 {
		t.Errorf("expected empty, got %d", len(creds))
	}
}

func TestCredentialStore_Update(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance"})
	err := cs.Update("u1", ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance", IsVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	creds, _ := cs.List("u1")
	if !creds[0].IsVerified {
		t.Error("expected verified")
	}
}

func TestCredentialStore_UpdateNotFound(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	err := cs.Update("u1", ExchangeCredential{ID: "c99"})
	if err == nil {
		t.Error("expected error for missing credential")
	}
}

func TestCredentialStore_Delete(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance"})
	if err := cs.Delete("u1", "c1"); err != nil {
		t.Fatal(err)
	}
	creds, _ := cs.List("u1")
	if len(creds) != 0 {
		t.Errorf("expected empty after delete, got %d", len(creds))
	}
}

func TestCredentialStore_DeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	err := cs.Delete("u1", "c99")
	if err == nil {
		t.Error("expected error for missing credential")
	}
}

func TestCredentialStore_GetDecrypted(t *testing.T) {
	dir := t.TempDir()
	enc := testEncryptor(t)
	cs := NewCredentialStore(dir, enc)

	apiKey, _ := enc.Encrypt("mykey")
	apiSecret, _ := enc.Encrypt("mysecret")
	cs.Upsert(ExchangeCredential{
		ID: "c1", UserID: "u1", Exchange: "binance",
		APIKeyEncrypted: apiKey, APISecretEncrypted: apiSecret,
	})

	k, s, p, err := cs.GetDecrypted("u1", "binance")
	if err != nil {
		t.Fatal(err)
	}
	if k != "mykey" || s != "mysecret" || p != "" {
		t.Errorf("got key=%q secret=%q pass=%q", k, s, p)
	}
}

func TestCredentialStore_GetDecryptedWithPassphrase(t *testing.T) {
	dir := t.TempDir()
	enc := testEncryptor(t)
	cs := NewCredentialStore(dir, enc)

	apiKey, _ := enc.Encrypt("mykey")
	apiSecret, _ := enc.Encrypt("mysecret")
	passphrase, _ := enc.Encrypt("mypass")
	cs.Upsert(ExchangeCredential{
		ID: "c1", UserID: "u1", Exchange: "kucoin",
		APIKeyEncrypted: apiKey, APISecretEncrypted: apiSecret,
		PassphraseEncrypted: passphrase,
	})

	_, _, p, _, err := cs.GetDecryptedWithMeta("u1", "kucoin")
	if err != nil {
		t.Fatal(err)
	}
	if p != "mypass" {
		t.Errorf("passphrase = %q, want mypass", p)
	}
}

func TestCredentialStore_GetDecryptedNotFound(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	_, _, _, err := cs.GetDecrypted("u1", "binance")
	if err == nil {
		t.Error("expected error for missing exchange")
	}
}

func TestCredentialStore_GetDecryptedByMode(t *testing.T) {
	dir := t.TempDir()
	enc := testEncryptor(t)
	cs := NewCredentialStore(dir, enc)

	liveKey, _ := enc.Encrypt("live-key")
	liveSecret, _ := enc.Encrypt("live-secret")
	testKey, _ := enc.Encrypt("test-key")
	testSecret, _ := enc.Encrypt("test-secret")

	cs.Upsert(ExchangeCredential{
		ID: "c1", UserID: "u1", Exchange: "binance", IsTestnet: false,
		APIKeyEncrypted: liveKey, APISecretEncrypted: liveSecret,
	})
	cs.Upsert(ExchangeCredential{
		ID: "c2", UserID: "u1", Exchange: "binance", IsTestnet: true,
		APIKeyEncrypted: testKey, APISecretEncrypted: testSecret,
	})

	k, _, _, err := cs.GetDecryptedByMode("u1", "binance", false)
	if err != nil || k != "live-key" {
		t.Errorf("live key = %q, err = %v", k, err)
	}
	k, _, _, err = cs.GetDecryptedByMode("u1", "binance", true)
	if err != nil || k != "test-key" {
		t.Errorf("test key = %q, err = %v", k, err)
	}
}

func TestCredentialStore_GetByModeCRUD(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))

	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance", IsTestnet: false})

	cred, err := cs.GetByMode("u1", "binance", false)
	if err != nil || cred.ID != "c1" {
		t.Errorf("cred = %v, err = %v", cred, err)
	}
	_, err = cs.GetByMode("u1", "binance", true)
	if err == nil {
		t.Error("expected error for missing testnet cred")
	}
}

func TestCredentialStore_FilePath(t *testing.T) {
	cs := NewCredentialStore("/data", nil)
	expected := filepath.Join("/data", "user1", "credentials.json")
	if cs.filePath("user1") != expected {
		t.Errorf("filePath = %q, want %q", cs.filePath("user1"), expected)
	}
}

func TestCredentialStore_LoadAll_BadJSON(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "u1")
	os.MkdirAll(userDir, 0o755)
	os.WriteFile(filepath.Join(userDir, "credentials.json"), []byte("bad json"), 0o600)

	cs := NewCredentialStore(dir, testEncryptor(t))
	_, err := cs.loadAll("u1")
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestModeLabel(t *testing.T) {
	if modeLabel(false) != "live" {
		t.Errorf("modeLabel(false) = %q", modeLabel(false))
	}
	if modeLabel(true) != "testnet" {
		t.Errorf("modeLabel(true) = %q", modeLabel(true))
	}
}
