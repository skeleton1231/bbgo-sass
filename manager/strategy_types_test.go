package main

import (
	"encoding/json"
	"testing"
)

func TestNewStrategyConfig_KnownStrategies(t *testing.T) {
	for id := range StrategyRegistry {
		cfg := NewStrategyConfig(id)
		if cfg == nil {
			t.Errorf("NewStrategyConfig(%q) returned nil", id)
		}
		if _, ok := cfg.(*map[string]any); ok {
			t.Errorf("NewStrategyConfig(%q) returned generic map, not a typed config", id)
		}
	}
}

func TestNewStrategyConfig_UnknownStrategy(t *testing.T) {
	cfg := NewStrategyConfig("nonexistent_strategy")
	if cfg == nil {
		t.Error("unknown strategy should return a fallback map, not nil")
	}
	m, ok := cfg.(*map[string]any)
	if !ok {
		t.Errorf("unknown strategy should return *map[string]any, got %T", cfg)
	}
	if len(*m) != 0 {
		t.Errorf("fallback map should be empty, got %v", *m)
	}
}

func TestNewStrategyConfig_UnmarshalsJSON(t *testing.T) {
	tests := []struct {
		id   string
		json string
	}{
		{"grid2", `{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":50000,"lowerPrice":40000}`},
		{"dca", `{"symbol":"ETHUSDT","quantity":0.1,"investmentInterval":"1h"}`},
		{"supertrend", `{"symbol":"BTCUSDT"}`},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			cfg := NewStrategyConfig(tt.id)
			if err := json.Unmarshal([]byte(tt.json), cfg); err != nil {
				t.Fatalf("json.Unmarshal into %s config: %v", tt.id, err)
			}
			roundtrip, err := json.Marshal(cfg)
			if err != nil {
				t.Fatalf("json.Marshal %s config: %v", tt.id, err)
			}
			if len(roundtrip) == 0 {
				t.Errorf("roundtrip of %s config produced empty JSON", tt.id)
			}
		})
	}
}

func TestStrategyRegistry_Completeness(t *testing.T) {
	if len(StrategyRegistry) == 0 {
		t.Fatal("StrategyRegistry is empty")
	}

	for id, meta := range StrategyRegistry {
		if meta.ID != id {
			t.Errorf("registry key %q has meta.ID %q", id, meta.ID)
		}
		if meta.Label == "" {
			t.Errorf("strategy %q has empty label", id)
		}
		if meta.Category == "" {
			t.Errorf("strategy %q has empty category", id)
		}
		if len(meta.SupportedExchanges) == 0 {
			t.Errorf("strategy %q has no supported exchanges", id)
		}
	}
}

func TestStrategyRegistry_EveryEntryHasFactory(t *testing.T) {
	for id := range StrategyRegistry {
		cfg := NewStrategyConfig(id)
		if _, ok := cfg.(*map[string]any); ok {
			t.Errorf("strategy %q in registry but returns generic map from factory", id)
		}
	}
}

func TestStrategyRegistry_CrossExchangeStrategies(t *testing.T) {
	crossCount := 0
	for _, meta := range StrategyRegistry {
		if meta.CrossExchange {
			crossCount++
			if len(meta.SessionRoles) == 0 {
				t.Errorf("cross-exchange strategy %q has no session roles", meta.ID)
			}
		}
	}
	if crossCount == 0 {
		t.Error("expected at least one cross-exchange strategy")
	}
}

func TestStrategyRegistry_LiveOnlyStrategies(t *testing.T) {
	liveOnlyCount := 0
	for _, meta := range StrategyRegistry {
		if meta.LiveOnly {
			liveOnlyCount++
			if meta.Category == "" {
				t.Errorf("liveOnly strategy %q has no category", meta.ID)
			}
		}
	}
	if liveOnlyCount == 0 {
		t.Error("expected at least one liveOnly strategy")
	}
}
