package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupBacktestTestAPI(t *testing.T) (*API, *BacktestJobStore, *chi.Mux) {
	t.Helper()
	store := NewStrategyStore("")

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      t.TempDir(),
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

	btJobs := NewBacktestJobStore(t.TempDir())
	btExec := NewBacktestExecutor(btJobs, cm, nil)

	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, btExec, btJobs)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	return api, btJobs, r
}

func TestAPI_SubmitBacktest(t *testing.T) {
	_, _, r := setupBacktestTestAPI(t)

	body := map[string]any{
		"strategy":   "grid2",
		"exchange":   "binance",
		"symbol":     "BTCUSDT",
		"start_time": "2024-01-01",
		"end_time":   "2024-03-01",
		"config": map[string]any{
			"symbol":     "BTCUSDT",
			"gridNumber": 10,
		},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["job_id"] == nil || resp["job_id"] == "" {
		t.Error("expected job_id in response")
	}
	if resp["status"] != JobPending {
		t.Errorf("expected pending status, got %v", resp["status"])
	}
}

func TestAPI_SubmitBacktest_MissingStrategy(t *testing.T) {
	_, _, r := setupBacktestTestAPI(t)

	body := map[string]any{
		"exchange": "binance",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing strategy, got %d", w.Code)
	}
}

func TestAPI_SubmitBacktest_DefaultsSymbol(t *testing.T) {
	_, store, r := setupBacktestTestAPI(t)

	body := map[string]any{
		"strategy": "grid2",
		"exchange": "binance",
		"config": map[string]any{
			"symbol": "ETHUSDT",
		},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	jobID := resp["job_id"].(string)

	job, found := store.Get(jobID)
	if !found {
		t.Fatal("expected job to be created in store")
	}
	if job.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol from config, got %s", job.Symbol)
	}
}

func TestAPI_SubmitBacktest_DefaultsFallback(t *testing.T) {
	_, store, r := setupBacktestTestAPI(t)

	body := map[string]any{
		"strategy": "grid2",
		"config":   map[string]any{},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	jobID := resp["job_id"].(string)

	job, _ := store.Get(jobID)
	if job.Exchange != "binance" {
		t.Errorf("expected default exchange binance, got %s", job.Exchange)
	}
	if job.Symbol != "BTCUSDT" {
		t.Errorf("expected default symbol BTCUSDT, got %s", job.Symbol)
	}
	if job.StartTime != "2024-01-01" {
		t.Errorf("expected default start_time, got %s", job.StartTime)
	}
	if job.EndTime != "2024-06-01" {
		t.Errorf("expected default end_time, got %s", job.EndTime)
	}
}

func TestAPI_SubmitBacktest_ServerBusy(t *testing.T) {
	_, store, r := setupBacktestTestAPI(t)

	store.AcquireSlot()
	store.AcquireSlot()
	defer store.ReleaseSlot()
	defer store.ReleaseSlot()

	body := map[string]any{
		"strategy": "grid2",
		"config":   map[string]any{},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when busy, got %d", w.Code)
	}
}

func TestAPI_SubmitBacktest_InvalidBody(t *testing.T) {
	_, _, r := setupBacktestTestAPI(t)

	req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestAPI_SubmitBacktest_NoAuth(t *testing.T) {
	api, _, _ := setupBacktestTestAPI(t)

	r := chi.NewRouter()
	r.Post("/api/backtest/submit", api.SubmitBacktest)

	body := map[string]any{"strategy": "grid2"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user id, got %d", w.Code)
	}
}

func TestAPI_GetBacktestJob(t *testing.T) {
	_, store, r := setupBacktestTestAPI(t)

	store.Create(&BacktestJob{
			ID:       "bt-test-1",
		UserID:    "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Config:    json.RawMessage(`{}`),
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
	})

	req := httptest.NewRequest("GET", "/api/backtest/jobs/bt-test-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var job BacktestJob
	json.NewDecoder(w.Body).Decode(&job)
	if job.ID != "bt-test-1" {
		t.Errorf("expected job ID bt-test-1, got %s", job.ID)
	}
	if job.Status != JobPending {
		t.Errorf("expected pending, got %s", job.Status)
	}
}

func TestAPI_GetBacktestJob_NotFound(t *testing.T) {
	_, _, r := setupBacktestTestAPI(t)

	req := httptest.NewRequest("GET", "/api/backtest/jobs/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPI_GetBacktestJob_OtherUser(t *testing.T) {
	_, store, r := setupBacktestTestAPI(t)

		store.Create(&BacktestJob{
			ID:       "bt-other-user",
		UserID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
		Strategy: "grid2",
		Config:   json.RawMessage(`{}`),
	})

	req := httptest.NewRequest("GET", "/api/backtest/jobs/bt-other-user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for other user's job, got %d", w.Code)
	}
}

func TestAPI_ListBacktestJobs(t *testing.T) {
	_, store, r := setupBacktestTestAPI(t)

	store.Create(&BacktestJob{
		ID:       "bt-user-1",
		UserID:   "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Strategy: "grid2",
		Config:   json.RawMessage(`{}`),
	})
	store.Create(&BacktestJob{
		ID:       "bt-user-2",
		UserID:   "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Strategy: "dca",
		Config:   json.RawMessage(`{}`),
	})
	store.Create(&BacktestJob{
		ID:       "bt-other",
		UserID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
		Strategy: "grid2",
		Config:   json.RawMessage(`{}`),
	})


	req := httptest.NewRequest("GET", "/api/backtest/jobs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	jobs := resp["jobs"].([]any)
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs for user, got %d", len(jobs))
	}
}

func TestAPI_ListBacktestJobs_Empty(t *testing.T) {
	_, _, r := setupBacktestTestAPI(t)

	req := httptest.NewRequest("GET", "/api/backtest/jobs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	jobs := resp["jobs"].([]any)
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestAPI_HasDataForRange_AlwaysSyncs(t *testing.T) {
	cfg := &Config{DataDir: t.TempDir()}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	store := NewStrategyStore("")
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)

	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-03-01") {
		t.Error("expected false — sync should always be forced")
	}
}

func TestAPI_SubmitBacktest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		body     map[string]any
		wantCode int
	}{
		{
			name:     "start_after_end_time",
			body:     map[string]any{"strategy": "grid2", "start_time": "2025-01-01", "end_time": "2024-01-01"},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "empty_config",
			body:     map[string]any{"strategy": "grid2", "config": map[string]any{}},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "nil_config",
			body:     map[string]any{"strategy": "grid2"},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "very_long_date_range",
			body:     map[string]any{"strategy": "grid2", "start_time": "2020-01-01", "end_time": "2026-01-01"},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "single_day_range",
			body:     map[string]any{"strategy": "grid2", "start_time": "2024-06-01", "end_time": "2024-06-01"},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "supported_exchange_bybit",
			body:     map[string]any{"strategy": "grid2", "exchange": "bybit", "symbol": "BTCUSDT"},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "supported_exchange_okex",
			body:     map[string]any{"strategy": "grid2", "exchange": "okex", "symbol": "BTCUSDT"},
			wantCode: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, r := setupBacktestTestAPI(t)
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/backtest/submit", bytes.NewReader(b))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected %d, got %d: %s", tt.wantCode, w.Code, w.Body.String())
			}

			if tt.wantCode == http.StatusAccepted {
				var resp map[string]any
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["job_id"] == nil || resp["job_id"] == "" {
					t.Error("expected job_id in accepted response")
				}
			}
		})
	}
}

func TestAPI_GetBacktestJob_InvalidID(t *testing.T) {
	_, _, r := setupBacktestTestAPI(t)

	for _, id := range []string{"", "nonexistent", "../../../etc/passwd"} {
		req := httptest.NewRequest("GET", "/api/backtest/jobs/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for job id %q, got %d", id, w.Code)
		}
	}
}
