package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/c9s/bbgo/saas/manager/pool"
)

func TestSyncer_DispatchesTradeNotification(t *testing.T) {
	var notifSent atomic.Int32
	notifSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notifSent.Add(1)
		w.WriteHeader(200)
	}))
	defer notifSrv.Close()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: nil})
		case "/api/trades":
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: []BBGoTrade{
				{ID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.001", GID: 1},
			}})
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if r.Method == "POST" {
			w.Write([]byte("[]"))
		}
	}))
	defer supabaseSrv.Close()

	dir := t.TempDir()
	enc := newTestEncryptor(t)
	notifier := NewNotifier(dir, enc)
	notifier.rateLimit = 0
	webhook, _ := enc.Encrypt(notifSrv.URL)
	notifier.configs["user-1"] = []NotificationConfig{
		{Channel: NotificationChannel{ID: "ch1", Type: "slack", WebhookURL: webhook, Enabled: true}, Rules: NotificationRule{TradeEvents: true}},
	}

	syncer := NewSyncer(&UserContainerManager{}, &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "key"}, &ContainerManager{}, pool.New(5))
	syncer.notifier = notifier
	syncer.newBBGoClientFn = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	uc := &UserContainer{UserID: "user-1", Mode: ModePaper, Status: StatusRunning}
	syncer.syncUserData(uc)

	if notifSent.Load() == 0 {
		t.Error("expected trade notification to be dispatched")
	}
}

func TestSyncer_DispatchesOrderNotification_Incremental(t *testing.T) {
	var notifSent atomic.Int32
	notifSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notifSent.Add(1)
		w.WriteHeader(200)
	}))
	defer notifSrv.Close()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: []BBGoOrder{
				{OrderID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", GID: 20},
				{OrderID: 2, Symbol: "ETHUSDT", Side: "SELL", Price: "3000", GID: 15},
				{OrderID: 3, Symbol: "BTCUSDT", Side: "BUY", Price: "49000", GID: 5},
			}})
		case "/api/trades":
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: nil})
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if r.Method == "POST" {
			w.Write([]byte("[]"))
		}
		// Return cursor=10 for sync_orders → incremental path
		if r.Method == "GET" && r.URL.Path == "/rest/v1/sync_cursors" {
			w.Write([]byte(`[{"last_gid":10}]`))
		}
	}))
	defer supabaseSrv.Close()

	dir := t.TempDir()
	enc := newTestEncryptor(t)
	notifier := NewNotifier(dir, enc)
	notifier.rateLimit = 0
	webhook, _ := enc.Encrypt(notifSrv.URL)
	notifier.configs["user-1"] = []NotificationConfig{
		{Channel: NotificationChannel{ID: "ch1", Type: "slack", WebhookURL: webhook, Enabled: true}, Rules: NotificationRule{OrderEvents: true}},
	}

	syncer := NewSyncer(&UserContainerManager{}, &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "key"}, &ContainerManager{}, pool.New(5))
	syncer.notifier = notifier
	syncer.newBBGoClientFn = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	uc := &UserContainer{UserID: "user-1", Mode: ModePaper, Status: StatusRunning}
	syncer.syncUserData(uc)

	if notifSent.Load() == 0 {
		t.Error("expected order notification to be dispatched for incremental sync")
	}
}

func TestSyncer_NoDispatchWithoutNotifier(t *testing.T) {
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(BBGoOrdersResponse{Orders: []BBGoOrder{
				{OrderID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", GID: 10},
			}})
		case "/api/trades":
			json.NewEncoder(w).Encode(BBGoTradesResponse{Trades: []BBGoTrade{
				{ID: 1, Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.001", GID: 1},
			}})
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if r.Method == "POST" {
			w.Write([]byte("[]"))
		}
		if r.Method == "GET" {
			w.Write([]byte("[]"))
		}
	}))
	defer supabaseSrv.Close()

	syncer := NewSyncer(&UserContainerManager{}, &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "key"}, &ContainerManager{}, pool.New(5))
	syncer.newBBGoClientFn = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	uc := &UserContainer{UserID: "user-1", Mode: ModePaper, Status: StatusRunning}
	syncer.syncUserData(uc) // should not panic
}
