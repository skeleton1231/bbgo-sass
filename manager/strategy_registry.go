package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type StrategyDefaultsCache struct {
	sc              *SupabaseClient
	mu              sync.RWMutex
	defaults        map[string]map[string]any
	liveOnly        map[string]bool
	requiresFutures map[string]bool
}

func NewStrategyDefaultsCache(sc *SupabaseClient) *StrategyDefaultsCache {
	return &StrategyDefaultsCache{
		sc:              sc,
		defaults:        make(map[string]map[string]any),
		liveOnly:        make(map[string]bool),
		requiresFutures: make(map[string]bool),
	}
}

func (r *StrategyDefaultsCache) Load() error {
	rows, _, err := r.sc.client.From("strategy_registry").
		Select("id,defaults,live_only,requires_futures", "", false).
		Eq("enabled", "true").
		Execute()
	if err != nil {
		return err
	}

	var entries []struct {
		ID              string          `json:"id"`
		Defaults        json.RawMessage `json:"defaults"`
		LiveOnly        bool            `json:"live_only"`
		RequiresFutures bool            `json:"requires_futures"`
	}
	if err := json.Unmarshal(rows, &entries); err != nil {
		return err
	}

	m := make(map[string]map[string]any, len(entries))
	lo := make(map[string]bool)
	rf := make(map[string]bool)
	for _, e := range entries {
		if len(e.Defaults) > 0 && string(e.Defaults) != "{}" {
			var defs map[string]any
			if err := json.Unmarshal(e.Defaults, &defs); err == nil && len(defs) > 0 {
				m[e.ID] = defs
			}
		}
		if e.LiveOnly {
			lo[e.ID] = true
		}
		if e.RequiresFutures {
			rf[e.ID] = true
		}
	}

	r.mu.Lock()
	r.defaults = m
	r.liveOnly = lo
	r.requiresFutures = rf
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

func (r *StrategyDefaultsCache) IsLiveOnly(strategyID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.liveOnly[strategyID]
}

func (r *StrategyDefaultsCache) RequiresFutures(strategyID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.requiresFutures[strategyID]
}
