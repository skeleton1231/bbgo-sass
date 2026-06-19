package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLivezHandler(t *testing.T) {
	api := &API{metrics: NewMetrics()}
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()
	api.LivezHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected 'ok', got %q", w.Body.String())
	}
}

func TestReadyzHandler_DuringShutdown(t *testing.T) {
	api := &API{metrics: NewMetrics()}
	api.metrics.MarkNotReady()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	api.ReadyzHandler(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 during shutdown, got %d", w.Code)
	}
}

func TestMetricsHandler_ReportCounters(t *testing.T) {
	api := &API{metrics: NewMetrics()}
	api.metrics.HTTPRequestsTotal.Add(7)
	api.metrics.HTTPErrorsTotal.Add(2)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	api.MetricsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["http_requests_total"].(float64) != 7 {
		t.Errorf("expected 7 requests, got %v", got["http_requests_total"])
	}
	if got["http_errors_total"].(float64) != 2 {
		t.Errorf("expected 2 errors, got %v", got["http_errors_total"])
	}
}

func TestCachedHealth_CachesWithinTTL(t *testing.T) {
	calls := 0
	ch := newCachedHealth(func(context.Context) HealthSnapshot {
		calls++
		return HealthSnapshot{Status: "ok", Users: calls}
	}, time.Second)
	first := ch.Get(context.Background())
	second := ch.Get(context.Background())
	if first.Users != 1 || second.Users != 1 {
		t.Fatalf("expected cached value, got first=%v second=%v", first.Users, second.Users)
	}
}
