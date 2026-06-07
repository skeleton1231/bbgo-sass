package main

import (
	"path/filepath"
	"testing"
)

func TestNotifier_FilePath(t *testing.T) {
	n := NewNotifier("/data", nil)
	if n.filePath("u1") != filepath.Join("/data", "u1", "notifications.json") {
		t.Errorf("filePath = %q", n.filePath("u1"))
	}
}

func TestNotifier_CreateAndList(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))

	cfg := NotificationConfig{
		Channel: NotificationChannel{Type: "telegram", ChatID: "123", Enabled: true},
		Rules:   NotificationRule{TradeEvents: true},
	}
	if err := n.Create("u1", cfg); err != nil {
		t.Fatal(err)
	}

	configs := n.List("u1")
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Channel.Type != "telegram" {
		t.Errorf("type = %q", configs[0].Channel.Type)
	}
	if configs[0].Channel.ID == "" {
		t.Error("ID should be auto-generated")
	}
}

func TestNotifier_ListEmpty(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))

	if configs := n.List("u1"); len(configs) != 0 {
		t.Errorf("expected empty, got %d", len(configs))
	}
}

func TestNotifier_Delete(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))

	n.Create("u1", NotificationConfig{Channel: NotificationChannel{ID: "n1", Type: "telegram"}})
	n.Create("u1", NotificationConfig{Channel: NotificationChannel{ID: "n2", Type: "slack"}})

	if err := n.Delete("u1", "n1"); err != nil {
		t.Fatal(err)
	}
	configs := n.List("u1")
	if len(configs) != 1 || configs[0].Channel.ID != "n2" {
		t.Errorf("after delete: %v", configs)
	}
}

func TestNotifier_DeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))

	if err := n.Delete("u1", "nonexistent"); err == nil {
		t.Error("expected error for missing config")
	}
}

func TestNotifier_LoadUser(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))
	n.Create("u1", NotificationConfig{
		Channel: NotificationChannel{Type: "telegram", Enabled: true},
	})

	n2 := NewNotifier(dir, testEncryptor(t))
	n2.LoadUser("u1")
	if configs := n2.List("u1"); len(configs) != 1 {
		t.Errorf("LoadUser: expected 1, got %d", len(configs))
	}
}

func TestNotifier_EncryptToken(t *testing.T) {
	n := NewNotifier(t.TempDir(), testEncryptor(t))
	enc, err := n.EncryptToken("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if enc == "" || enc == "secret123" {
		t.Errorf("token should be encrypted")
	}
}

func TestNotifier_RuleEnabled(t *testing.T) {
	n := NewNotifier(t.TempDir(), nil)
	tests := []struct {
		ruleType string
		rules    NotificationRule
		want     bool
	}{
		{"trade", NotificationRule{TradeEvents: true}, true},
		{"trade", NotificationRule{TradeEvents: false}, false},
		{"order", NotificationRule{OrderEvents: true}, true},
		{"container", NotificationRule{ContainerHealth: true}, true},
		{"backtest", NotificationRule{}, true},
		{"test", NotificationRule{}, true},
		{"unknown", NotificationRule{}, false},
	}
	for _, tt := range tests {
		if got := n.ruleEnabled(tt.rules, tt.ruleType); got != tt.want {
			t.Errorf("ruleEnabled(%+v, %q) = %v, want %v", tt.rules, tt.ruleType, got, tt.want)
		}
	}
}

func TestNotifier_Dispatch_NoConfigs(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))
	n.LoadUser("u1")
	if n.Dispatch("u1", NotificationEvent{Type: "trade"}) {
		t.Error("should not send with no configs")
	}
}

func TestNotifier_Dispatch_DisabledChannel(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))
	n.Create("u1", NotificationConfig{
		Channel: NotificationChannel{Type: "telegram", Enabled: false},
		Rules:   NotificationRule{TradeEvents: true},
	})
	if n.Dispatch("u1", NotificationEvent{Type: "trade"}) {
		t.Error("disabled channel should not send")
	}
}

func TestNotifier_Dispatch_RuleNotEnabled(t *testing.T) {
	dir := t.TempDir()
	n := NewNotifier(dir, testEncryptor(t))
	n.Create("u1", NotificationConfig{
		Channel: NotificationChannel{Type: "telegram", Enabled: true, TokenEnc: "x", ChatID: "123"},
		Rules:   NotificationRule{TradeEvents: false},
	})
	if n.Dispatch("u1", NotificationEvent{Type: "trade"}) {
		t.Error("rule not enabled should not send")
	}
}
