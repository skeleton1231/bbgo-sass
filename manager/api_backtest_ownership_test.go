package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestGetBacktestJob_OwnershipEnforced verifies that a user cannot access
// another user's backtest job.
func TestGetBacktestJob_OwnershipEnforced(t *testing.T) {
	btJobs := NewBacktestJobStore(t.TempDir())

	const ownerID = "11111111-1111-1111-1111-111111111111"
	const attackerID = "22222222-2222-2222-2222-222222222222"

	job := &BacktestJob{
		ID:     "bt-owner-job",
		UserID: ownerID,
		Status: JobCompleted,
	}
	btJobs.Create(job)

	api := &API{
		btJobs: btJobs,
	}

	withChiParam := func(r *http.Request, key, val string) *http.Request {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add(key, val)
		return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	}

	t.Run("owner can access", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/backtest/jobs/bt-owner-job", nil)
		r = withChiParam(r, "jobID", "bt-owner-job")
		r.Header.Set("X-User-Id", ownerID)
		w := httptest.NewRecorder()

		api.GetBacktestJob(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("non-owner gets 403", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/backtest/jobs/bt-owner-job", nil)
		r = withChiParam(r, "jobID", "bt-owner-job")
		r.Header.Set("X-User-Id", attackerID)
		w := httptest.NewRecorder()

		api.GetBacktestJob(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing auth gets 401", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/backtest/jobs/bt-owner-job", nil)
		r = withChiParam(r, "jobID", "bt-owner-job")
		w := httptest.NewRecorder()

		api.GetBacktestJob(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("nonexistent job gets 404", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/backtest/jobs/bt-nonexistent", nil)
		r = withChiParam(r, "jobID", "bt-nonexistent")
		r.Header.Set("X-User-Id", ownerID)
		w := httptest.NewRecorder()

		api.GetBacktestJob(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestListBacktestJobs_UserScoped verifies that ListByUser only returns
// jobs belonging to the requesting user.
func TestListBacktestJobs_UserScoped(t *testing.T) {
	btJobs := NewBacktestJobStore(t.TempDir())

	btJobs.Create(&BacktestJob{ID: "j1", UserID: "user-a", Status: JobCompleted})
	btJobs.Create(&BacktestJob{ID: "j2", UserID: "user-b", Status: JobRunning})
	btJobs.Create(&BacktestJob{ID: "j3", UserID: "user-a", Status: JobPending})

	jobsA := btJobs.ListByUser("user-a")
	if len(jobsA) != 2 {
		t.Fatalf("expected 2 jobs for user-a, got %d", len(jobsA))
	}
	ids := map[string]bool{}
	for _, j := range jobsA {
		ids[j.ID] = true
	}
	if !ids["j1"] || !ids["j3"] {
		t.Errorf("expected j1 and j3, got %v", ids)
	}

	jobsB := btJobs.ListByUser("user-b")
	if len(jobsB) != 1 || jobsB[0].ID != "j2" {
		t.Errorf("expected 1 job (j2) for user-b, got %v", jobsB)
	}

	jobsC := btJobs.ListByUser("user-c")
	if len(jobsC) != 0 {
		t.Errorf("expected 0 jobs for unknown user, got %d", len(jobsC))
	}
}
