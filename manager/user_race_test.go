package main

import (
	"sync"
	"testing"
)

// TestStrategyStore_ConcurrentWrites verifies that concurrent writes to
// StrategyStore don't cause data races.
func TestStrategyStore_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	store := NewStrategyStore(dir, nil)

	var wg sync.WaitGroup
	const goroutines = 20

	// Concurrent writers adding strategies for different users
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := "user-" + string(rune('0'+idx%10))
			entry := StrategyEntry{
				Name:     "test",
				Exchange: "binance",
				Strategy: "grid2",
				Config:   rawJSON(`{"symbol":"BTCUSDT"}`),
			}
			store.AddStrategy(userID, ModeLive, entry, func(string) bool { return false })
		}(i)
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := "user-" + string(rune('0'+idx%10))
			store.ListStrategies(userID, ModeLive)
		}(i)
	}

	wg.Wait()
}

// TestStrategyStore_ConcurrentReadWrite verifies concurrent reads and writes
// to the same user's strategies don't cause data races.
func TestStrategyStore_ConcurrentReadWrite(t *testing.T) {
	dir := t.TempDir()
	store := NewStrategyStore(dir, nil)
	userID := "shared-user"

	// Seed with initial strategy
	store.AddStrategy(userID, ModeLive, StrategyEntry{
		Name: "seed", Exchange: "binance", Strategy: "grid2",
		Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	}, func(string) bool { return false })

	var wg sync.WaitGroup
	const goroutines = 10

	for i := 0; i < goroutines; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			store.ListStrategies(userID, ModeLive)
		}()
		go func(idx int) {
			defer wg.Done()
			store.AddStrategy(userID, ModeLive, StrategyEntry{
				Name:     "concurrent",
				Exchange: "binance",
				Strategy: "grid",
				Config:   rawJSON(`{}`),
			}, func(string) bool { return false })
		}(i)
	}

	wg.Wait()
}
