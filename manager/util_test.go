package main

import (
	"strings"
	"sync"
	"testing"
)

func TestGenerateID_HasPrefix(t *testing.T) {
	id := generateID("strat")
	if !strings.HasPrefix(id, "strat-") {
		t.Errorf("expected prefix 'strat-', got %q", id)
	}
}

func TestGenerateID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := generateID("test")
		if seen[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestGenerateID_Concurrent(t *testing.T) {
	seen := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := generateID("par")
			mu.Lock()
			if seen[id] {
				t.Errorf("duplicate ID under concurrency: %s", id)
			}
			seen[id] = true
			mu.Unlock()
		}()
	}
	wg.Wait()
}
