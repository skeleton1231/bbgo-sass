package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAPI_StartUser_IdempotentWhenStarting(t *testing.T) {
	store, _ := newTestStore(t)
	writeTestStrategies(t, store, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid", Config: rawJSON(`{}`)},
	})

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	defer api.Close()
	api.containerRunning = func(_, _ string) bool { return false }

	var startCalls int64
	unblock := make(chan struct{})
	api.containerStart = func(userID, mode string) error {
		atomic.AddInt64(&startCalls, 1)
		<-unblock
		return nil
	}

	r := testRouter(api)

	// First request triggers start
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("first request: expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Second request while starting — should be idempotent (returns starting, no new containerStart)
	req2 := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusAccepted {
		t.Fatalf("second request: expected 202, got %d: %s", w2.Code, w2.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	calls := atomic.LoadInt64(&startCalls)
	close(unblock)

	if calls != 1 {
		t.Fatalf("expected exactly 1 containerStart, got %d", calls)
	}
}

func TestAPI_StartUser_ConcurrentRequests(t *testing.T) {
	store, _ := newTestStore(t)
	writeTestStrategies(t, store, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid", Config: rawJSON(`{}`)},
	})

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return false }
	defer api.Close()

	var startCalls int64
	unblock := make(chan struct{})
	api.containerStart = func(userID, mode string) error {
		atomic.AddInt64(&startCalls, 1)
		<-unblock
		return nil
	}

	r := testRouter(api)
	var wg sync.WaitGroup
	type result struct{ code int }
	results := make(chan result, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			results <- result{code: w.Code}
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	close(unblock)

	for i := 0; i < 2; i++ {
		r := <-results
		if r.code != http.StatusAccepted {
			t.Errorf("request %d: expected 202, got %d", i, r.code)
		}
	}

	if calls := atomic.LoadInt64(&startCalls); calls != 1 {
		t.Fatalf("expected exactly 1 containerStart call, got %d", calls)
	}
}

func TestAPI_CreateStrategy_RunningUser_SetsStarting(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	store, _ := newTestStore(t)
	writeTestStrategies(t, store, userID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid", Config: rawJSON(`{}`)},
	})

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }
	defer api.Close()

	unblock := make(chan struct{})
	var startCalls int64
	api.containerStart = func(userID, mode string) error {
		atomic.AddInt64(&startCalls, 1)
		<-unblock
		return nil
	}

	r := testRouter(api)

	// Two rapid strategy creates on a running user
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := strings.NewReader(`{"name":"Test","exchange":"binance","strategy":"grid","config":{}}`)
			req := httptest.NewRequest("POST", "/api/users/"+userID+"/strategies", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	calls := atomic.LoadInt64(&startCalls)
	close(unblock)

	// Only 1 containerStart should fire — the second create sees status=starting
	if calls != 1 {
		t.Fatalf("expected exactly 1 containerStart for rapid strategy creates, got %d", calls)
	}
}
