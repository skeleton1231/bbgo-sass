package main

import (
	"sync"
	"testing"
)

func TestGetReturnsSnapshot(t *testing.T) {
	m := NewUserContainerManager()
	m.AddStrategy("user-1", StrategyEntry{
		ID:       "s1",
		Strategy: "grid2",
		Exchange: "binance",
		Config:   rawJSON(`{"symbol":"BTCUSDT"}`),
	})

	uc1, ok := m.Get("user-1")
	if !ok {
		t.Fatal("expected to find user-1")
	}

	// Mutate the returned value — should NOT affect the stored entry
	uc1.Status = StatusRunning

	uc2, _ := m.Get("user-1")
	if uc2.Status == StatusRunning {
		t.Error("Get() should return a snapshot, but mutating it affected the stored entry")
	}
}

func TestListUsersReturnsSnapshots(t *testing.T) {
	m := NewUserContainerManager()
	m.AddStrategy("user-1", StrategyEntry{ID: "s1", Strategy: "grid2", Exchange: "binance", Config: rawJSON(`{}`)})

	users := m.ListUsers()
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	// Mutate the returned value
	users[0].Status = StatusRunning

	// Verify internal state is unaffected
	users2 := m.ListUsers()
	if users2[0].Status == StatusRunning {
		t.Error("ListUsers() should return snapshots, but mutating it affected stored entries")
	}
}

func TestConcurrentAccessNoRace(t *testing.T) {
	m := NewUserContainerManager()
	m.AddStrategy("user-1", StrategyEntry{ID: "s1", Strategy: "grid2", Exchange: "binance", Config: rawJSON(`{}`)})

	var wg sync.WaitGroup
	const goroutines = 20

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uc, ok := m.Get("user-1")
			if ok {
				_ = uc.Status
			}
			users := m.ListUsers()
			for _, u := range users {
				_ = u.Status
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.UpdateStatus("user-1", StatusRunning)
			m.UpdateStatus("user-1", StatusStopped)
		}()
	}

	wg.Wait()
}

func TestAddStrategyReturnsSnapshot(t *testing.T) {
	m := NewUserContainerManager()

	uc, created := m.AddStrategy("user-1", StrategyEntry{
		ID:       "s1",
		Strategy: "grid2",
		Exchange: "binance",
		Config:   rawJSON(`{}`),
	})

	if !created {
		t.Fatal("expected user to be created")
	}

	// Mutate returned value — should not affect stored state
	uc.Status = StatusRunning

	uc2, _ := m.Get("user-1")
	if uc2.Status == StatusRunning {
		t.Error("AddStrategy() should return a snapshot, not a pointer to internal state")
	}
}
