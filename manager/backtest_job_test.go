package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBacktestJobStore_CreateAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	job := &BacktestJob{
		ID:        "bt-test-1",
		UserID:    "user-1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
		Config:    json.RawMessage(`{"symbol":"BTCUSDT"}`),
		NeedSync:  true,
	}

	store.Create(job)

	got, found := store.Get("bt-test-1")
	if !found {
		t.Fatal("expected to find job")
	}
	if got.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", got.UserID)
	}
	if got.Status != JobPending {
		t.Errorf("expected pending, got %s", got.Status)
	}
	if got.NeedSync != true {
		t.Error("expected need_sync=true")
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestBacktestJobStore_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	_, found := store.Get("nonexistent")
	if found {
		t.Error("expected not found")
	}
}

func TestBacktestJobStore_UpdateStatus(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	job := &BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)}
	store.Create(job)

	updated := store.UpdateStatus("bt-1", JobDownloading, "syncing...")
	if updated == nil {
		t.Fatal("expected updated job")
	}
	if updated.Status != JobDownloading {
		t.Errorf("expected downloading, got %s", updated.Status)
	}
	if updated.Progress != "syncing..." {
		t.Errorf("expected syncing..., got %s", updated.Progress)
	}
	if updated.StartedAt == nil {
		t.Error("expected started_at to be set")
	}

	updated2 := store.UpdateStatus("bt-1", JobRunning, "running backtest...")
	if updated2.Status != JobRunning {
		t.Errorf("expected running, got %s", updated2.Status)
	}

	updated3 := store.UpdateStatus("bt-1", JobCompleted, "done")
	if updated3.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestBacktestJobStore_SetOutput(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	job := &BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)}
	store.Create(job)

	store.SetOutput("bt-1", "result data here")
	got, _ := store.Get("bt-1")
	if got.Output != "result data here" {
		t.Errorf("expected output, got %s", got.Output)
	}
}

func TestBacktestJobStore_SetError(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	job := &BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)}
	store.Create(job)

	store.SetError("bt-1", "something failed")
	got, _ := store.Get("bt-1")
	if got.Error != "something failed" {
		t.Errorf("expected error, got %s", got.Error)
	}
}

func TestBacktestJobStore_ListByUser(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	store.Create(&BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)})
	store.Create(&BacktestJob{ID: "bt-2", UserID: "u2", Strategy: "grid2", Config: json.RawMessage(`{}`)})
	store.Create(&BacktestJob{ID: "bt-3", UserID: "u1", Strategy: "dca", Config: json.RawMessage(`{}`)})

	u1Jobs := store.ListByUser("u1")
	if len(u1Jobs) != 2 {
		t.Errorf("expected 2 jobs for u1, got %d", len(u1Jobs))
	}

	u2Jobs := store.ListByUser("u2")
	if len(u2Jobs) != 1 {
		t.Errorf("expected 1 job for u2, got %d", len(u2Jobs))
	}
}

func TestBacktestJobStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	store1 := NewBacktestJobStore(dir)
	store1.Create(&BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)})
	store1.UpdateStatus("bt-1", JobCompleted, "done")
	store1.SetOutput("bt-1", "test output")

	store2 := NewBacktestJobStore(dir)
	got, found := store2.Get("bt-1")
	if !found {
		t.Fatal("expected job to survive restart")
	}
	if got.Status != JobCompleted {
		t.Errorf("expected completed, got %s", got.Status)
	}
	if got.Output != "test output" {
		t.Errorf("expected output, got %s", got.Output)
	}
}

func TestBacktestJobStore_StaleJobsReset(t *testing.T) {
	dir := t.TempDir()

	store1 := NewBacktestJobStore(dir)
	store1.Create(&BacktestJob{ID: "bt-1", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)})

	jobPath := filepath.Join(dir, "backtest-jobs", "bt-2.json")
	staleJob := &BacktestJob{
		ID:        "bt-2",
		UserID:    "u1",
		Strategy:  "grid2",
		Config:    json.RawMessage(`{}`),
		Status:    JobRunning,
		CreatedAt: time.Now(),
	}
	data, _ := json.MarshalIndent(staleJob, "", "  ")
	os.WriteFile(jobPath, data, 0o644)

	store2 := NewBacktestJobStore(dir)
	got, found := store2.Get("bt-2")
	if !found {
		t.Fatal("expected stale job to be loaded")
	}
	if got.Status != JobPending {
		t.Errorf("expected stale running job to be reset to pending, got %s", got.Status)
	}
}

func TestBacktestJobStore_Prune(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	store.Create(&BacktestJob{ID: "bt-old", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)})
	store.UpdateStatus("bt-old", JobCompleted, "done")
	store.mu.Lock()
	j := store.jobs["bt-old"]
	past := time.Now().Add(-48 * time.Hour)
	j.CompletedAt = &past
	store.persist(j)
	store.mu.Unlock()

	store.Create(&BacktestJob{ID: "bt-new", UserID: "u1", Strategy: "grid2", Config: json.RawMessage(`{}`)})
	store.UpdateStatus("bt-new", JobCompleted, "done")

	store.Prune(24 * time.Hour)

	_, found := store.Get("bt-old")
	if found {
		t.Error("expected old job to be pruned")
	}
	_, found = store.Get("bt-new")
	if !found {
		t.Error("expected new job to be kept")
	}
}

func TestBacktestJobStore_Semaphore(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)

	if !store.AcquireSlot() {
		t.Error("expected to acquire slot")
	}
	if store.AcquireSlot() {
		t.Error("expected slot to be full (concurrency=1)")
	}
	store.ReleaseSlot()
	if !store.AcquireSlot() {
		t.Error("expected to acquire slot after release")
	}
	store.ReleaseSlot()
}
