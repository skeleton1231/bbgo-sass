package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type StrategyDefaultsCache struct {
	sc       *SupabaseClient
	mu       sync.RWMutex
	defaults map[string]map[string]any
}

func NewStrategyDefaultsCache(sc *SupabaseClient) *StrategyDefaultsCache {
	return &StrategyDefaultsCache{
		sc:       sc,
		defaults: make(map[string]map[string]any),
	}
}

func (r *StrategyDefaultsCache) Load() error {
	rows, _, err := r.sc.client.From("strategy_registry").
		Select("id,defaults", "", false).
		Eq("enabled", "true").
		Execute()
	if err != nil {
		return err
	}

	var entries []struct {
		ID       string          `json:"id"`
		Defaults json.RawMessage `json:"defaults"`
	}
	if err := json.Unmarshal(rows, &entries); err != nil {
		return err
	}

	m := make(map[string]map[string]any, len(entries))
	for _, e := range entries {
		if len(e.Defaults) == 0 || string(e.Defaults) == "{}" {
			m[e.ID] = nil
			continue
		}
		var defs map[string]any
		if err := json.Unmarshal(e.Defaults, &defs); err != nil {
			log.Printf("strategy_registry: skipping %s: %v", e.ID, err)
			continue
		}
		m[e.ID] = defs
	}

	r.mu.Lock()
	r.defaults = m
	r.mu.Unlock()
	return nil
}

func (r *StrategyDefaultsCache) RefreshLoop(done <-chan struct{}) {
	if err := r.Load(); err != nil {
		log.Printf("strategy_registry: initial load failed: %v", err)
	} else {
		r.mu.RLock()
		n := len(r.defaults)
		r.mu.RUnlock()
		log.Printf("strategy_registry: loaded %d strategies", n)
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := r.Load(); err != nil {
				log.Printf("strategy_registry: refresh failed: %v", err)
			}
		}
	}
}

func (r *StrategyDefaultsCache) GetDefaults(strategyID string) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaults[strategyID]
}
