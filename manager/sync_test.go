package main

import (
	"testing"
)

func TestPluralS(t *testing.T) {
	if pluralS(1) != "" {
		t.Errorf("pluralS(1) = %q, want empty", pluralS(1))
	}
	if pluralS(0) != "s" {
		t.Errorf("pluralS(0) = %q, want s", pluralS(0))
	}
	if pluralS(2) != "s" {
		t.Errorf("pluralS(2) = %q, want s", pluralS(2))
	}
}

func TestNewSyncer(t *testing.T) {
	s := NewSyncer(nil)
	if s == nil {
		t.Fatal("NewSyncer returned nil")
	}
}

func TestNewSyncerWithCreds(t *testing.T) {
	dir := t.TempDir()
	cs := NewCredentialStore(dir, testEncryptor(t))
	s := NewSyncerWithCreds(nil, cs)
	if s == nil || s.creds != cs {
		t.Fatal("NewSyncerWithCreds failed")
	}
}

func TestSyncer_SetNotifier(t *testing.T) {
	s := NewSyncer(nil)
	n := NewNotifier(t.TempDir(), testEncryptor(t))
	s.SetNotifier(n)
	if s.notifier != n {
		t.Error("notifier not set")
	}
}

func TestSyncer_SyncCredential_NilSupa(t *testing.T) {
	s := NewSyncer(nil)
	s.SyncCredential(ExchangeCredential{UserID: "u1", Exchange: "binance"})
}

func TestSyncer_DeleteCredential_NilCreds(t *testing.T) {
	s := NewSyncer(nil)
	s.DeleteCredential("u1", "binance", false)
}

func TestSyncer_MarkCredentialsVerified_NilCreds(t *testing.T) {
	s := NewSyncer(nil)
	s.MarkCredentialsVerified("u1", "live", nil)
}

func TestSyncer_MarkCredentialsVerified(t *testing.T) {
	dir := t.TempDir()
	enc := testEncryptor(t)
	cs := NewCredentialStore(dir, enc)
	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance", IsTestnet: false})

	s := NewSyncerWithCreds(nil, cs)
	s.MarkCredentialsVerified("u1", "live", []StrategyEntry{
		{Exchange: "binance"},
	})

	creds, _ := cs.List("u1")
	if !creds[0].IsVerified {
		t.Error("expected credential to be verified")
	}
}

func TestSyncer_MarkCredentialsVerified_WrongMode(t *testing.T) {
	dir := t.TempDir()
	enc := testEncryptor(t)
	cs := NewCredentialStore(dir, enc)
	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance", IsTestnet: false})

	s := NewSyncerWithCreds(nil, cs)
	s.MarkCredentialsVerified("u1", "paper", []StrategyEntry{
		{Exchange: "binance"},
	})

	creds, _ := cs.List("u1")
	if creds[0].IsVerified {
		t.Error("should not verify live cred when mode=paper")
	}
}

func TestSyncer_MarkCredentialsVerified_AlreadyVerified(t *testing.T) {
	dir := t.TempDir()
	enc := testEncryptor(t)
	cs := NewCredentialStore(dir, enc)
	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance", IsTestnet: false, IsVerified: true})

	s := NewSyncerWithCreds(nil, cs)
	s.MarkCredentialsVerified("u1", "live", []StrategyEntry{
		{Exchange: "binance"},
	})

	creds, _ := cs.List("u1")
	if !creds[0].IsVerified {
		t.Error("should remain verified")
	}
}

func TestSyncer_MarkCredentialsVerified_CrossExchange(t *testing.T) {
	dir := t.TempDir()
	enc := testEncryptor(t)
	cs := NewCredentialStore(dir, enc)
	cs.Upsert(ExchangeCredential{ID: "c1", UserID: "u1", Exchange: "binance", IsTestnet: false})

	s := NewSyncerWithCreds(nil, cs)
	s.MarkCredentialsVerified("u1", "live", []StrategyEntry{
		{CrossExchange: true, Sessions: []SessionRoleConfig{{Exchange: "binance"}}},
	})

	creds, _ := cs.List("u1")
	if !creds[0].IsVerified {
		t.Error("expected verified via cross-exchange session")
	}
}
