package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- ProxyToBot validation ---

func TestProxyToBot_InvalidUUID(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach backend for invalid UUID")
	})
	defer cleanup()

	req := chiReq2("GET", "/api/bbgo/bad-uuid/sessions", "", map[string]string{"userID": "bad-uuid"})
	req.Header.Set("X-User-Id", "bad-uuid")
	w := httptest.NewRecorder()
	api.ProxyToBot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid UUID: status = %d, want 400", w.Code)
	}
}

func TestProxyToBot_UserIDMismatch(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach backend for mismatched user")
	})
	defer cleanup()

	req := chiReq2("GET", "/api/bbgo/"+proxyUID+"/sessions", "", map[string]string{"userID": proxyUID})
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000002")
	w := httptest.NewRecorder()
	api.ProxyToBot(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("mismatch: status = %d, want 403", w.Code)
	}
}

func TestProxyToBot_UserNotFound(t *testing.T) {
	users := NewUserContainerManager()
	api := &API{
		users:     users,
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	uid := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000099"
	req := chiReq2("GET", "/api/bbgo/"+uid+"/sessions", "", map[string]string{"userID": uid})
	req.Header.Set("X-User-Id", uid)
	w := httptest.NewRecorder()
	api.ProxyToBot(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("not found: status = %d, want 404", w.Code)
	}
}

// --- Session account proxy ---

func TestBBGoSessionAccount_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance/account" {
			w.Write([]byte(`{"account":{"balances":{"USDT":{"available":"5000"}}}}`))
		}
	})
	defer cleanup()

	req := chiReq2("GET", "/api/users/"+proxyUID+"/bbgo/sessions/binance/account", "", map[string]string{"userID": proxyUID, "session": "binance"})
	req.Header.Set("X-User-Id", proxyUID)
	w := httptest.NewRecorder()
	api.BBGoSessionAccount(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// --- Session symbols proxy ---

func TestBBGoSessionSymbols_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance/symbols" {
			w.Write([]byte(`{"symbols":["BTCUSDT","ETHUSDT"]}`))
		}
	})
	defer cleanup()

	req := chiReq2("GET", "/api/users/"+proxyUID+"/bbgo/sessions/binance/symbols", "", map[string]string{"userID": proxyUID, "session": "binance"})
	req.Header.Set("X-User-Id", proxyUID)
	w := httptest.NewRecorder()
	api.BBGoSessionSymbols(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// --- Assets proxy ---

func TestBBGoAssets_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/assets" {
			w.Write([]byte(`{"assets":{"USDT":{"currency":"USDT","total":"5000"}}}`))
		}
	})
	defer cleanup()

	req := chiReq("GET", "/api/users/"+proxyUID+"/bbgo/assets", "", "userID", proxyUID)
	req.Header.Set("X-User-Id", proxyUID)
	w := httptest.NewRecorder()
	api.BBGoAssets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// --- Strategies proxy ---

func TestBBGoStrategies_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			w.Write([]byte(`{"strategies":[{"ID":"grid2","Symbol":"BTCUSDT"}]}`))
		}
	})
	defer cleanup()

	req := chiReq("GET", "/api/users/"+proxyUID+"/bbgo/strategies", "", "userID", proxyUID)
	req.Header.Set("X-User-Id", proxyUID)
	w := httptest.NewRecorder()
	api.BBGoStrategies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// --- Closed orders proxy ---

func TestBBGoClosedOrders_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/orders/closed" {
			w.Write([]byte(`{"orders":[{"orderID":42,"symbol":"BTCUSDT","side":"SELL"}]}`))
		}
	})
	defer cleanup()

	req := chiReq("GET", "/api/users/"+proxyUID+"/bbgo/orders/closed?exchange=binance&symbol=BTCUSDT", "", "userID", proxyUID)
	req.Header.Set("X-User-Id", proxyUID)
	w := httptest.NewRecorder()
	api.BBGoClosedOrders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// --- Notification CRUD ---

func TestNotifier_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	n := NewNotifier(tmpDir, enc)

	cfg := NotificationConfig{
		Channel: NotificationChannel{ID: "n1", Type: "test", Enabled: true},
		Rules:   NotificationRule{TradeEvents: true},
	}
	if err := n.Create("u1", cfg); err != nil {
		t.Fatal(err)
	}

	list := n.List("u1")
	if len(list) != 1 {
		t.Fatalf("expected 1 config, got %d", len(list))
	}
	if list[0].Channel.ID != "n1" {
		t.Errorf("expected ID n1, got %s", list[0].Channel.ID)
	}

	if err := n.Delete("u1", "n1"); err != nil {
		t.Fatal(err)
	}
	if len(n.List("u1")) != 0 {
		t.Error("expected empty list after delete")
	}
}

func TestNotifier_DeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	n := NewNotifier(tmpDir, enc)

	err := n.Delete("u1", "nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent config")
	}
}

func TestNotifier_DispatchDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	n := NewNotifier(tmpDir, enc)
	n.configs["u1"] = []NotificationConfig{{
		Channel: NotificationChannel{ID: "n1", Type: "telegram", Enabled: false},
		Rules:   NotificationRule{TradeEvents: true},
	}}

	sent := n.Dispatch("u1", NotificationEvent{Type: "trade", Title: "test", Message: "msg"})
	if sent {
		t.Error("should not dispatch to disabled channel")
	}
}

func TestNotifier_DispatchNoConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	n := NewNotifier(tmpDir, enc)

	sent := n.Dispatch("u1", NotificationEvent{Type: "trade", Title: "test", Message: "msg"})
	if sent {
		t.Error("should not dispatch with no configs")
	}
}

func TestNotifier_DispatchRateLimit(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	n := NewNotifier(tmpDir, enc)
	n.rateLimit = 999 * time.Hour
	n.configs["u1"] = []NotificationConfig{{
		Channel: NotificationChannel{ID: "n1", Type: "test", Enabled: true},
		Rules:   NotificationRule{TradeEvents: true},
	}}

	sent := n.Dispatch("u1", NotificationEvent{Type: "trade", Title: "t", Message: "m"})
	if !sent {
		t.Fatal("first dispatch should succeed")
	}

	sent = n.Dispatch("u1", NotificationEvent{Type: "trade", Title: "t2", Message: "m2"})
	if sent {
		t.Error("second dispatch within rate limit window should be dropped")
	}
}

func TestNotifier_LoadAndPersist(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)

	n1 := NewNotifier(tmpDir, enc)
	n1.configs["u1"] = []NotificationConfig{{
		Channel: NotificationChannel{ID: "n1", Type: "telegram", Enabled: true},
		Rules:   NotificationRule{TradeEvents: true},
	}}
	if err := n1.saveAll("u1", n1.configs["u1"]); err != nil {
		t.Fatal(err)
	}

	n2 := NewNotifier(tmpDir, enc)
	configs, err := n2.loadAll("u1")
	if err != nil {
		t.Fatal(err)
	}
	n2.configs["u1"] = configs
	list := n2.List("u1")
	if len(list) != 1 {
		t.Fatalf("expected 1 config after reload, got %d", len(list))
	}
	if list[0].Channel.ID != "n1" {
		t.Errorf("expected ID n1, got %s", list[0].Channel.ID)
	}
}

// --- Notification API handlers ---

func TestNotificationAPI_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	notifier := NewNotifier(tmpDir, enc)

	users := NewUserContainerManager()
	api := &API{
		users:     users,
		notifier:  notifier,
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeee000050"

	// Create
	credBody := `{"type":"slack","webhook_url":"https://hooks.slack.com/test","rules":{"trade_events":true,"order_events":false,"container_health":true}}`
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(credBody))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	api.CreateNotificationConfig(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201: %s", w.Code, w.Body.String())
	}

	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected non-empty id")
	}

	// List
	req2 := httptest.NewRequest(http.MethodGet, "/api/notifications/config", nil)
	req2.Header.Set("X-User-Id", userID)
	w2 := httptest.NewRecorder()
	api.ListNotificationConfigs(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list: status = %d, want 200", w2.Code)
	}

	var listed []map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&listed)
	if len(listed) != 1 {
		t.Fatalf("expected 1 config, got %d", len(listed))
	}

	// Delete
	req3 := chiReq("DELETE", "/api/notifications/config/"+id, "", "id", id)
	req3.Header.Set("X-User-Id", userID)
	w3 := httptest.NewRecorder()
	api.DeleteNotificationConfig(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("delete: status = %d, want 200: %s", w3.Code, w3.Body.String())
	}

	// Verify empty
	if len(api.notifier.List(userID)) != 0 {
		t.Error("expected empty list after delete")
	}
}

func TestNotificationAPI_CreateInvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	notifier := NewNotifier(tmpDir, enc)

	api := &API{
		users:     NewUserContainerManager(),
		notifier:  notifier,
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	body := `{"type":"email","token":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000050")
	w := httptest.NewRecorder()
	api.CreateNotificationConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid type: status = %d, want 400", w.Code)
	}
}

func TestNotificationAPI_TestNotification_NoConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	notifier := NewNotifier(tmpDir, enc)

	api := &API{
		users:     NewUserContainerManager(),
		notifier:  notifier,
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/test", nil)
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000050")
	w := httptest.NewRecorder()
	api.TestNotification(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("no configs: status = %d, want 400", w.Code)
	}
}

// --- Trades proxy ---

func TestBBGoTrades_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trades" {
			w.Write([]byte(`{"trades":[{"id":1,"symbol":"BTCUSDT","side":"BUY"}]}`))
		}
	})
	defer cleanup()

	req := chiReq("GET", "/api/users/"+proxyUID+"/bbgo/trades?exchange=binance", "", "userID", proxyUID)
	req.Header.Set("X-User-Id", proxyUID)
	w := httptest.NewRecorder()
	api.BBGoTrades(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}
