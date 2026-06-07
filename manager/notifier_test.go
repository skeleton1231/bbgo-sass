package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatch_ConcurrentCreate(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	dir := t.TempDir()
	enc := newTestEncryptor(t)
	n := NewNotifier(dir, enc)
	n.rateLimit = 0

	webhook, _ := enc.Encrypt(srv.URL)
	n.configs["user1"] = []NotificationConfig{
		{
			Channel: NotificationChannel{
				Type:       "slack",
				WebhookURL: webhook,
				Enabled:    true,
			},
			Rules: NotificationRule{TradeEvents: true},
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "t", Message: "m"})
		}()
	}
	wg.Wait()

	if calls.Load() == 0 {
		t.Error("expected at least one notification to be sent")
	}
}

func TestDispatch_RateLimitsEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	dir := t.TempDir()
	enc := newTestEncryptor(t)
	n := NewNotifier(dir, enc)
	n.rateLimit = 1 * time.Hour

	webhook, _ := enc.Encrypt(srv.URL)
	n.configs["user1"] = []NotificationConfig{
		{
			Channel: NotificationChannel{
				Type:       "slack",
				WebhookURL: webhook,
				Enabled:    true,
			},
			Rules: NotificationRule{TradeEvents: true},
		},
	}

	sent1 := n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "t1", Message: "m1"})
	sent2 := n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "t2", Message: "m2"})

	if !sent1 {
		t.Error("first dispatch should succeed")
	}
	if sent2 {
		t.Error("second dispatch within rate limit window should be suppressed")
	}
}

func TestDispatch_NoConfigs(t *testing.T) {
	dir := t.TempDir()
	enc := newTestEncryptor(t)
	n := NewNotifier(dir, enc)

	sent := n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "t", Message: "m"})
	if sent {
		t.Error("should return false when no configs exist")
	}
}

func TestDispatch_DisabledChannel(t *testing.T) {
	dir := t.TempDir()
	enc := newTestEncryptor(t)
	n := NewNotifier(dir, enc)
	n.rateLimit = 0

	n.configs["user1"] = []NotificationConfig{
		{
			Channel: NotificationChannel{
				Type:    "slack",
				Enabled: false,
			},
			Rules: NotificationRule{TradeEvents: true},
		},
	}

	sent := n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "t", Message: "m"})
	if sent {
		t.Error("should not send to disabled channel")
	}
}

func TestRuleEnabled(t *testing.T) {
	n := &Notifier{}
	tests := []struct {
		ruleType  string
		eventType string
		expected  bool
	}{
		{"trade", "trade", true},
		{"order", "order", true},
		{"container", "container", true},
		{"trade", "order", false},
		{"", "unknown", false},
	}
	for _, tt := range tests {
		rules := NotificationRule{}
		switch tt.ruleType {
		case "trade":
			rules.TradeEvents = true
		case "order":
			rules.OrderEvents = true
		case "container":
			rules.ContainerHealth = true
		}
		if got := n.ruleEnabled(rules, tt.eventType); got != tt.expected {
			t.Errorf("ruleEnabled(%v, %q) = %v, want %v", rules, tt.eventType, got, tt.expected)
		}
	}
	if !n.ruleEnabled(NotificationRule{}, "backtest") {
		t.Error("backtest events should always be enabled")
	}
	if !n.ruleEnabled(NotificationRule{}, "test") {
		t.Error("test events should always be enabled")
	}
}

func TestSendTelegram_ConnectionError(t *testing.T) {
	client := &http.Client{}
	err := sendTelegram(client, "token", "chat", "t", "m")
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestSendSlack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := srv.Client()
	err := sendSlack(client, srv.URL, "Title", "Message")
	if err != nil {
		t.Fatalf("sendSlack: %v", err)
	}
}

func TestSendSlack_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	client := srv.Client()
	err := sendSlack(client, srv.URL, "t", "m")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestNotifier_DecryptToken_Empty(t *testing.T) {
	dir := t.TempDir()
	enc := newTestEncryptor(t)
	n := NewNotifier(dir, enc)
	_, err := n.decryptToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestNotifier_Dispatch_SlackRuleNotEnabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	dir := t.TempDir()
	enc := newTestEncryptor(t)
	n := NewNotifier(dir, enc)
	n.rateLimit = 0

	webhook, _ := enc.Encrypt(srv.URL)
	n.configs["user1"] = []NotificationConfig{
		{
			Channel: NotificationChannel{
				Type:       "slack",
				WebhookURL: webhook,
				Enabled:    true,
			},
			Rules: NotificationRule{OrderEvents: true},
		},
	}

	sent := n.Dispatch("user1", NotificationEvent{Type: "trade", Title: "t", Message: "m"})
	if sent {
		t.Error("should not send when rule not enabled for event type")
	}
}

func TestNotifier_EncryptDecrypt(t *testing.T) {
	dir := t.TempDir()
	enc := newTestEncryptor(t)
	n := NewNotifier(dir, enc)

	plain := "my-secret-token"
	encrypted, err := n.EncryptToken(plain)
	if err != nil {
		t.Fatalf("EncryptToken: %v", err)
	}
	if encrypted == plain {
		t.Error("encrypted should differ from plain")
	}

	decrypted, err := n.decryptToken(encrypted)
	if err != nil {
		t.Fatalf("decryptToken: %v", err)
	}
	if decrypted != plain {
		t.Errorf("decrypt = %q, want %q", decrypted, plain)
	}
}

func newTestEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	enc, err := NewEncryptor("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=") // 32 bytes base64
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}
	return enc
}
