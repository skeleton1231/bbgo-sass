package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
