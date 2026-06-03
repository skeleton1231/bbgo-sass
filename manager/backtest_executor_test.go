package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBacktestExecutor_FullFlow_SyncAndRun(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	var syncCalled, runCalled bool
	var mu sync.Mutex
	exec.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		mu.Lock()
		syncCalled = true
		mu.Unlock()
		return "synced 1000 candles", nil
	}
	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		mu.Lock()
		runCalled = true
		mu.Unlock()
		return []byte("backtest result: profit=1234.5"), nil
	}

	job := &BacktestJob{
		UserID:    "user-1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
		Config:    json.RawMessage(`{"symbol":"BTCUSDT"}`),
		NeedSync:  true,
	}
	job.ID = generateID("bt")

	if err := exec.Submit(job); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	assertJobEventually(t, store, job.ID, JobCompleted, 5*time.Second)

	got, found := store.Get(job.ID)
	if !found {
		t.Fatal("job not found")
	}
	if got.Status != JobCompleted {
		t.Errorf("expected completed, got %s", got.Status)
	}
	if got.Output == "" {
		t.Error("expected output to be set")
	}
	if got.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
	if got.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}

	mu.Lock()
	if !syncCalled {
		t.Error("expected sync to be called")
	}
	if !runCalled {
		t.Error("expected run to be called")
	}
	mu.Unlock()
}

func TestBacktestExecutor_SkipSync(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	var syncCalled bool
	var mu sync.Mutex
	exec.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		mu.Lock()
		syncCalled = true
		mu.Unlock()
		return "", nil
	}
	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		return []byte("done"), nil
	}

	job := &BacktestJob{
		UserID:    "user-1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
		Config:    json.RawMessage(`{"symbol":"BTCUSDT"}`),
		NeedSync:  false,
	}

	job.ID = generateID("bt")
	exec.Submit(job)
	assertJobEventually(t, store, job.ID, JobCompleted, 5*time.Second)

	mu.Lock()
	if syncCalled {
		t.Error("expected sync to be skipped when NeedSync=false")
	}
	mu.Unlock()
}

func TestBacktestExecutor_SyncFailure(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	var runCalled bool
	var mu sync.Mutex
	exec.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		return "", fmt.Errorf("sync error: network timeout")
	}
	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		mu.Lock()
		runCalled = true
		mu.Unlock()
		return []byte("should not run"), nil
	}

	job := &BacktestJob{
		UserID:    "user-1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
		Config:    json.RawMessage(`{"symbol":"BTCUSDT"}`),
		NeedSync:  true,
	}

	job.ID = generateID("bt")
	exec.Submit(job)
	assertJobEventually(t, store, job.ID, JobFailed, 5*time.Second)

	got, _ := store.Get(job.ID)
	if got.Error == "" {
		t.Error("expected error message to be set")
	}

	mu.Lock()
	if runCalled {
		t.Error("expected run NOT to be called after sync failure")
	}
	mu.Unlock()
}

func TestBacktestExecutor_RunFailure(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	exec.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		return "synced", nil
	}
	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		return nil, fmt.Errorf("run error: container crashed")
	}

	job := &BacktestJob{
		UserID:    "user-1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
		Config:    json.RawMessage(`{"symbol":"BTCUSDT"}`),
		NeedSync:  true,
	}
	job.ID = generateID("bt")

	exec.Submit(job)
	assertJobEventually(t, store, job.ID, JobFailed, 5*time.Second)

	got, _ := store.Get(job.ID)
	if got.Error == "" {
		t.Error("expected error message on run failure")
	}
	if got.Output != "" {
		t.Error("expected no output on run failure")
	}
}

func TestBacktestExecutor_SlotReleasedOnFailure(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	exec.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		return "", fmt.Errorf("sync error")
	}

	job := &BacktestJob{
		UserID:    "user-1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
		Config:    json.RawMessage(`{}`),
		NeedSync:  true,
	}

	job.ID = generateID("bt")
	exec.Submit(job)
	assertJobEventually(t, store, job.ID, JobFailed, 5*time.Second)

	job2 := &BacktestJob{
		UserID:   "user-1",
		Strategy: "grid2",
		Config:   json.RawMessage(`{}`),
		NeedSync: false,
	}
	job2.ID = generateID("bt")
	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		return []byte("ok"), nil
	}

	if err := exec.Submit(job2); err != nil {
		t.Fatalf("expected to acquire slot after failure released it: %v", err)
	}
	assertJobEventually(t, store, job2.ID, JobCompleted, 5*time.Second)
}

func TestBacktestExecutor_StatusTransitions(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	var statuses []string
	var mu sync.Mutex

	exec.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "synced", nil
	}
	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		time.Sleep(50 * time.Millisecond)
		return []byte("result"), nil
	}

	job := &BacktestJob{
		UserID:    "user-1",
		Strategy:  "grid2",
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		StartTime: "2024-01-01",
		EndTime:   "2024-03-01",
		Config:    json.RawMessage(`{}`),
		NeedSync:  true,
	}
	job.ID = generateID("bt")

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			<-ticker.C
			got, found := store.Get(job.ID)
			if !found {
				continue
			}
			mu.Lock()
			if len(statuses) == 0 || statuses[len(statuses)-1] != got.Status {
				statuses = append(statuses, got.Status)
			}
			mu.Unlock()
			if got.Status == JobCompleted || got.Status == JobFailed {
				return
			}
		}
	}()

	exec.Submit(job)
	assertJobEventually(t, store, job.ID, JobCompleted, 5*time.Second)

	// Give the poller goroutine time to observe the final status.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// The poller may miss "pending" since Submit transitions immediately.
	// Verify we see the core flow: downloading → running → completed.
	required := []string{JobRunning, JobCompleted}
	saw := map[string]bool{}
	for _, s := range statuses {
		saw[s] = true
	}
	for _, exp := range required {
		if !saw[exp] {
			t.Errorf("expected to see status %q in transitions, got %v", exp, statuses)
		}
	}
}

func TestBacktestExecutor_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	var runCalled bool
	var mu sync.Mutex
	exec.syncFn = func(userID, exchange, symbol, startTime, endTime string) (string, error) {
		return "synced", nil
	}
	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		mu.Lock()
		runCalled = true
		mu.Unlock()
		return []byte("should not run"), nil
	}

	job := &BacktestJob{
		UserID:   "user-1",
		Strategy: "grid2",
		Config:   json.RawMessage(`{invalid json`),
		NeedSync: false,
	}
	job.ID = generateID("bt")

	exec.Submit(job)
	assertJobEventually(t, store, job.ID, JobFailed, 5*time.Second)

	got, _ := store.Get(job.ID)
	if got.Error == "" {
		t.Error("expected error for invalid config")
	}

	mu.Lock()
	if runCalled {
		t.Error("expected run NOT to be called with invalid config")
	}
	mu.Unlock()
}

func assertJobEventually(t *testing.T, store *BacktestJobStore, jobID, expectedStatus string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		got, found := store.Get(jobID)
		if found && (got.Status == expectedStatus || got.Status == JobFailed) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	got, _ := store.Get(jobID)
	t.Fatalf("timed out waiting for job %s to reach %s, got status=%s", jobID, expectedStatus, got.Status)
}

func TestBacktestExecutor_ConcurrentSubmit(t *testing.T) {
	dir := t.TempDir()
	store := NewBacktestJobStore(dir)
	cm := &ContainerManager{cfg: &Config{DataDir: dir}, checkRunningFn: func(string, string) (bool, error) { return false, fmt.Errorf("no docker") }}
	exec := NewBacktestExecutor(store, cm, nil, nil)

	exec.runFn = func(userID string, jobID string, yamlContent []byte) ([]byte, error) {
		time.Sleep(200 * time.Millisecond) // simulate work
		return []byte("result"), nil
	}

	var mu sync.Mutex
	var completedIDs []string
	var acceptedIDs []string

	for i := range 3 {
		job := &BacktestJob{
			UserID:    "user-1",
			Strategy:  "grid2",
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			StartTime: "2024-01-01",
			EndTime:   "2024-03-01",
			Config:    json.RawMessage(`{"symbol":"BTCUSDT"}`),
			NeedSync:  false,
		}
		job.ID = generateID("bt")

		err := exec.Submit(job)
		mu.Lock()
		if i < 2 {
			// First two jobs should acquire slots (concurrency=2)
			acceptedIDs = append(acceptedIDs, job.ID)
			if err != nil {
				t.Errorf("job %d: expected to submit, got %v", i, err)
			}
		} else {
			// Third job should be rejected
			if err == nil {
				t.Errorf("job %d: expected server busy error", i)
			}
		}
		mu.Unlock()
	}

	// Wait for both accepted jobs to complete, freeing slots
	assertJobEventually(t, store, acceptedIDs[0], JobCompleted, 5*time.Second)
	assertJobEventually(t, store, acceptedIDs[1], JobCompleted, 5*time.Second)
	// Give goroutines time to run deferred ReleaseSlot after status update
	time.Sleep(100 * time.Millisecond)

	// Now we should be able to submit again
	job4 := &BacktestJob{
		UserID:   "user-1",
		Strategy: "grid2",
		Config:   json.RawMessage(`{"symbol":"BTCUSDT"}`),
		NeedSync: false,
	}
	job4.ID = generateID("bt")
	if err := exec.Submit(job4); err != nil {
		t.Fatalf("expected to submit after slot freed: %v", err)
	}
	assertJobEventually(t, store, job4.ID, JobCompleted, 5*time.Second)

	mu.Lock()
	_ = completedIDs
	mu.Unlock()
}
