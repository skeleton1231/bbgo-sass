package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestProxy(backendURL string, clientTimeout time.Duration) *BotProxy {
	return &BotProxy{
		cm:          &ContainerManager{cfg: &Config{BBGOPort: 0}},
		resolveAddr: func(_, _ string) string { return backendURL },
		client:      &http.Client{Timeout: clientTimeout},
	}
}

func TestProxyToBot_BackendDown(t *testing.T) {
	proxy := newTestProxy("http://127.0.0.1:1", 500*time.Millisecond)

	r := httptest.NewRequest("GET", "/api/bbgo/user-1/api/ping", nil)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		proxy.ProxyToBot(w, r, "user-1", ModeLive)
		close(done)
	}()

	select {
	case <-done:
		if w.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", w.Code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("proxy blocked too long on unreachable backend")
	}
}

func TestProxyToBot_CancelledRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(10 * time.Second):
			w.Write([]byte(`{}`))
		}
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL, 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequest("GET", "/api/bbgo/user-1/api/ping", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		w := httptest.NewRecorder()
		proxy.ProxyToBot(w, r, "user-1", ModeLive)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("proxy did not return after context cancellation")
	}
}

func TestProxyToBot_StripAuthHeaders(t *testing.T) {
	var receivedHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Write([]byte(`{}`))
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL, 5*time.Second)

	r := httptest.NewRequest("GET", "/api/bbgo/user-1/api/ping", nil)
	r.Header.Set("X-Manager-Token", "secret-token")
	r.Header.Set("X-User-Id", "user-1")
	r.Header.Set("Authorization", "Bearer tok")

	w := httptest.NewRecorder()
	proxy.ProxyToBot(w, r, "user-1", ModeLive)

	if receivedHeaders.Get("X-Manager-Token") != "" {
		t.Error("X-Manager-Token should be stripped")
	}
	if receivedHeaders.Get("X-User-Id") != "" {
		t.Error("X-User-Id should be stripped")
	}
	if receivedHeaders.Get("Authorization") != "Bearer tok" {
		t.Error("Authorization should be preserved")
	}
}

func TestProxyToBot_ResponseHeaderPassthrough(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL, 5*time.Second)

	r := httptest.NewRequest("GET", "/api/bbgo/user-1/api/ping", nil)
	w := httptest.NewRecorder()
	proxy.ProxyToBot(w, r, "user-1", ModeLive)

	if w.Header().Get("X-Custom") != "value" {
		t.Errorf("expected X-Custom=value, got %q", w.Header().Get("X-Custom"))
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProxyToBot_LargeBody(t *testing.T) {
	largeBody := strings.Repeat(`{"data":"padding"},`, 10000)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer backend.Close()

	proxy := newTestProxy(backend.URL, 5*time.Second)

	r := httptest.NewRequest("GET", "/api/bbgo/user-1/api/data", nil)
	w := httptest.NewRecorder()
	proxy.ProxyToBot(w, r, "user-1", ModeLive)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if len(w.Body.String()) < 1000 {
		t.Error("body seems truncated")
	}
}
