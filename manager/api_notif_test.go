package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func setupTestAPIWithNotifier(t *testing.T) (*API, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	store := NewStrategyStore(tmpDir)
	cfg := &Config{DataDir: tmpDir, ManagerToken: "test-token", SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg}
	proxy := NewBotProxy(cm)
	notifier := NewNotifier(tmpDir, enc)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, notifier, nil, NewBacktestJobStore(tmpDir))
	api.containerRunning = func(string, _ string) bool { return false }
	api.containerStart = func(userID, mode string) error { return nil }
	return api, func() { api.Close() }
}

func TestAPI_CreateNotificationConfig_Telegram(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	body := `{"type":"telegram","token":"123456:ABC-DEF","chat_id":"-1001234","rules":{"trade_events":true,"order_events":false,"container_health":true}}`
	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["type"] != "telegram" {
		t.Errorf("expected type=telegram, got %v", resp["type"])
	}
	if resp["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", resp["enabled"])
	}
	rules, ok := resp["rules"].(map[string]any)
	if !ok {
		t.Fatal("expected rules object")
	}
	if rules["trade_events"] != true {
		t.Errorf("expected trade_events=true, got %v", rules["trade_events"])
	}
}

func TestAPI_CreateNotificationConfig_Slack(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	body := `{"type":"slack","webhook_url":"https://hooks.slack.com/services/XXX","rules":{"trade_events":true,"order_events":true,"container_health":false}}`
	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateNotificationConfig_InvalidType(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	body := `{"type":"email","token":"xxx","chat_id":"123"}`
	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", w.Code)
	}
}

func TestAPI_CreateNotificationConfig_TelegramMissingToken(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	body := `{"type":"telegram","chat_id":"123"}`
	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing telegram token, got %d", w.Code)
	}
}

func TestAPI_CreateNotificationConfig_SlackMissingWebhook(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	body := `{"type":"slack"}`
	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing slack webhook, got %d", w.Code)
	}
}

func TestAPI_ListNotificationConfigs_ReturnsCreated(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	r := testRouter(api)
	body := `{"type":"telegram","token":"123:ABC","chat_id":"-1001","rules":{"trade_events":true}}`
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/notifications/config", nil)
	req2.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w2.Code)
	}

	var configs []map[string]any
	json.NewDecoder(w2.Body).Decode(&configs)
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0]["token"] != nil {
		t.Error("encrypted token should not be in list response")
	}
}

func TestAPI_DeleteNotificationConfig_FullCRUD(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	r := testRouter(api)
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	body := `{"type":"telegram","token":"123:ABC","chat_id":"-1001","rules":{"trade_events":true}}`
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", w.Code)
	}
	var createResp map[string]any
	json.NewDecoder(w.Body).Decode(&createResp)
	notifID, _ := createResp["id"].(string)

	req2 := httptest.NewRequest(http.MethodDelete, "/api/notifications/config/"+notifID, nil)
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/notifications/config", nil)
	req3.Header.Set("X-User-Id", userID)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var configs []map[string]any
	json.NewDecoder(w3.Body).Decode(&configs)
	if len(configs) != 0 {
		t.Errorf("expected 0 configs after delete, got %d", len(configs))
	}
}

func TestAPI_DeleteNotificationConfig_NotFound(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodDelete, "/api/notifications/config/nonexistent", nil)
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_TestNotification_Dispatches(t *testing.T) {
	// Set up a mock Slack webhook server so sendSlack actually succeeds
	slackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer slackSrv.Close()

	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()
	api.notifier.client = slackSrv.Client()
	// Replace transport to route any URL to the mock server
	origTransport := api.notifier.client.Transport
	api.notifier.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(slackSrv.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { api.notifier.client.Transport = origTransport }()

	r := testRouter(api)
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	body := `{"type":"slack","webhook_url":"https://hooks.slack.com/services/FAKE","rules":{"trade_events":true}}`
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/notifications/test", nil)
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestAPI_TestNotification_NoChannels(t *testing.T) {
	api, cleanup := setupTestAPIWithNotifier(t)
	defer cleanup()

	r := testRouter(api)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/test", nil)
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when no channels, got %d", w.Code)
	}
}
