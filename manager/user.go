package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

const (
	StatusRunning = "running"
	StatusStopped = "stopped"
	StatusError   = "error"
)

type StrategyEntry struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Exchange string          `json:"exchange"`
	Strategy string          `json:"strategy"`
	Config   json.RawMessage `json:"config"`
	Mode     string          `json:"mode"`
}

type UserContainer struct {
	UserID     string          `json:"user_id"`
	Status     string          `json:"status"`
	Strategies []StrategyEntry `json:"strategies"`
}

type UserContainerManager struct {
	mu    sync.RWMutex
	users map[string]*UserContainer
}

func NewUserContainerManager() *UserContainerManager {
	return &UserContainerManager{users: make(map[string]*UserContainer)}
}

func (m *UserContainerManager) getOrCreate(userID string) *UserContainer {
	uc := &UserContainer{
		UserID:     userID,
		Status:     StatusStopped,
		Strategies: []StrategyEntry{},
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.users[userID]; ok {
		return existing
	}
	m.users[userID] = uc
	return uc
}

func (m *UserContainerManager) Get(userID string) (*UserContainer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	uc, ok := m.users[userID]
	return uc, ok
}

func (m *UserContainerManager) UpdateStatus(userID, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if uc, ok := m.users[userID]; ok {
		uc.Status = status
	}
}

func (m *UserContainerManager) AddStrategy(userID string, entry StrategyEntry) *UserContainer {
	uc := m.getOrCreate(userID)
	m.mu.Lock()
	defer m.mu.Unlock()
	uc.Strategies = append(uc.Strategies, entry)
	return uc
}

func (m *UserContainerManager) RemoveStrategy(userID, strategyID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	uc, ok := m.users[userID]
	if !ok {
		return false
	}
	for i, s := range uc.Strategies {
		if s.ID == strategyID {
			uc.Strategies = append(uc.Strategies[:i], uc.Strategies[i+1:]...)
			return true
		}
	}
	return false
}

func (m *UserContainerManager) ListUsers() []*UserContainer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*UserContainer, 0, len(m.users))
	for _, uc := range m.users {
		list = append(list, uc)
	}
	return list
}

func (m *UserContainerManager) Restore(users []*UserContainer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, uc := range users {
		m.users[uc.UserID] = uc
	}
}

func buildUserYAML(uc *UserContainer) string {
	type exchangeConf struct {
		Symbol string
	}
	exchanges := map[string]*exchangeConf{}
	var strategyLines string
	seenKeys := map[string]int{}

	for _, s := range uc.Strategies {
		var params map[string]interface{}
		if err := json.Unmarshal(s.Config, &params); err != nil {
			var rawStr string
			if err2 := json.Unmarshal(s.Config, &rawStr); err2 == nil {
				strategyLines += rawStr + "\n"
				continue
			}
			continue
		}

		symbol := "BTCUSDT"
		if v, ok := params["symbol"].(string); ok && v != "" {
			symbol = v
		}
		if _, exists := exchanges[s.Exchange]; !exists {
			exchanges[s.Exchange] = &exchangeConf{Symbol: symbol}
		}

		key := s.Strategy
		if count, exists := seenKeys[key]; exists {
			key = fmt.Sprintf("%s_%d", key, count)
		}
		seenKeys[s.Strategy]++

		var lines string
		for k, v := range params {
			if k == "symbol" || k == "interval" {
				continue
			}
			switch val := v.(type) {
			case float64:
				if val == float64(int(val)) {
					lines += fmt.Sprintf("      %s: %d\n", k, int(val))
				} else {
					lines += fmt.Sprintf("      %s: %g\n", k, val)
				}
			case bool:
				lines += fmt.Sprintf("      %s: %v\n", k, val)
			default:
				lines += fmt.Sprintf("      %s: %v\n", k, val)
			}
		}
		strategyLines += fmt.Sprintf("    %s:\n%s", key, lines)
	}

	exchangeYAML := ""
	for name, conf := range exchanges {
		exchangeYAML += fmt.Sprintf("  %s:\n    symbol: %s\n", name, conf.Symbol)
	}

	paperTrade := ""
	for _, s := range uc.Strategies {
		if s.Mode == "paper" {
			paperTrade = "\nenvironment:\n  PAPER_TRADE: \"1\"\n"
			break
		}
	}

	return fmt.Sprintf("exchange:\n%s%sstrategy:\n%s", exchangeYAML, paperTrade, strategyLines)
}
