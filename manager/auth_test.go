package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSharedSecretAuth_HealthAndWSSkip(t *testing.T) {
	called := false
	handler := SharedSecretAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/api/health", "/api/ws"} {
		called = false
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if !called {
			t.Errorf("%s: handler was not called", path)
		}
		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSharedSecretAuth_RequiresToken(t *testing.T) {
	handler := SharedSecretAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/users/123/start", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}
}

func TestSharedSecretAuth_WSTicketEndpointRequiresAuth(t *testing.T) {
	handler := SharedSecretAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/ws/ticket", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("/api/ws/ticket should require auth, got %d", w.Code)
	}
}

func TestUserRateLimit_AllowsWithinBurst(t *testing.T) {
	handler := UserRateLimit(time.Second, 3)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/users/test/start", nil)
		req.Header.Set("X-User-Id", "11111111-1111-1111-1111-111111111111")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestUserRateLimit_BlocksAfterBurst(t *testing.T) {
	handler := UserRateLimit(time.Second, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	userID := "22222222-2222-2222-2222-222222222222"
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/users/test/start", nil)
		req.Header.Set("X-User-Id", userID)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("POST", "/api/users/test/start", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after burst, got %d", w.Code)
	}
}

func TestUserRateLimit_NoUserIDPasses(t *testing.T) {
	handler := UserRateLimit(time.Second, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/users/test/start", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("request without user ID should pass through, got %d", w.Code)
	}
}
