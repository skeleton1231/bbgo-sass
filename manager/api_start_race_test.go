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
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", StrategyEntry{
		ID:       "s1",
		Exchange: "binance",
		Strategy: "grid",
		Config:   rawJSON(`{}`),
	})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", StatusStarting)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_ string) bool { return false }

	// Block startContainer so the goroutine stays alive — if the fix is wrong,
	// the goroutine would call this and we'd detect it.
	var startCalls int64
	unblock := make(chan struct{})
	api.containerStart = func(_ *UserContainer) error {
		atomic.AddInt64(&startCalls, 1)
		<-unblock
		return nil
	}

	r := testRouter(api)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for already-starting user, got %d: %s", w.Code, w.Body.String())
	}

	// Give any stray goroutine time to call containerStart
	time.Sleep(100 * time.Millisecond)
	calls := atomic.LoadInt64(&startCalls)
	close(unblock)

	if calls != 0 {
		t.Fatalf("containerStart should NOT be called when status is already starting, got %d calls", calls)
	}
}

func TestAPI_StartUser_ConcurrentRequests(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", StrategyEntry{
		ID:       "s1",
		Exchange: "binance",
		Strategy: "grid",
		Config:   rawJSON(`{}`),
	})

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_ string) bool { return false }

	var startCalls int64
	unblock := make(chan struct{})
	api.containerStart = func(_ *UserContainer) error {
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
	users := NewUserContainerManager()
	users.AddStrategy(userID, StrategyEntry{
		ID:       "s1",
		Exchange: "binance",
		Strategy: "grid",
		Config:   rawJSON(`{}`),
	})
	users.UpdateStatus(userID, StatusRunning)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_ string) bool { return false }

	unblock := make(chan struct{})
	var startCalls int64
	api.containerStart = func(_ *UserContainer) error {
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
			body := strings.NewReader(`{"exchange":"binance","strategy":"grid","config":{}}`)
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
