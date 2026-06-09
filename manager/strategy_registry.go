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
	fields          map[string][]FieldDef
}

type FieldDef struct {
	Key      string   `json:"key"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

func NewStrategyDefaultsCache(sc *SupabaseClient) *StrategyDefaultsCache {
	return &StrategyDefaultsCache{
		sc:              sc,
		defaults:        make(map[string]map[string]any),
		liveOnly:        make(map[string]bool),
		requiresFutures: make(map[string]bool),
		fields:          make(map[string][]FieldDef),
	}
}

func (r *StrategyDefaultsCache) Load() error {
	rows, _, err := r.sc.client.From("strategy_registry").
		Select("id,defaults,live_only,requires_futures,fields", "", false).
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
		Fields          json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(rows, &entries); err != nil {
		return err
	}

	defs := make(map[string]map[string]any, len(entries))
	lo := make(map[string]bool)
	rf := make(map[string]bool)
	fl := make(map[string][]FieldDef)
	for _, e := range entries {
		if len(e.Defaults) > 0 && string(e.Defaults) != "{}" {
			var d map[string]any
			if err := json.Unmarshal(e.Defaults, &d); err == nil && len(d) > 0 {
				defs[e.ID] = d
			}
		}
		if e.LiveOnly {
			lo[e.ID] = true
		}
		if e.RequiresFutures {
			rf[e.ID] = true
		}
		if len(e.Fields) > 0 && string(e.Fields) != "[]" {
			var fields []FieldDef
			if err := json.Unmarshal(e.Fields, &fields); err == nil && len(fields) > 0 {
				fl[e.ID] = fields
			}
		}
	}

	r.mu.Lock()
	r.defaults = defs
	r.liveOnly = lo
	r.requiresFutures = rf
	r.fields = fl
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

func (r *StrategyDefaultsCache) GetFields(strategyID string) []FieldDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.fields[strategyID]
}
