package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Bot represents a single strategy instance (a "bot" in the web UI).
type Bot struct {
	ID              string      `json:"id"`
	Strategy        string      `json:"strategy"`
	Symbol          string      `json:"symbol"`
	Exchange        string      `json:"exchange"`
	Session         string      `json:"session"`
	Config          interface{} `json:"config"`
	State           interface{} `json:"state"`
	ContainerStatus string      `json:"container_status"`
	Mode            string      `json:"mode"`
}

// botFromStrategy builds a Bot from a bbgo strategy state map.
// bbgo returns {"on": [...], "grid2": {symbol, ...}, "strategy": "grid2", "strategyInstanceID": "..."}
func botFromStrategy(s map[string]interface{}, mode string) Bot {
	id, _ := s["strategyInstanceID"].(string)
	strategy, _ := s["strategy"].(string)

	var session string
	if on, ok := s["on"].([]interface{}); ok && len(on) > 0 {
		session, _ = on[0].(string)
	}

	var symbol string
	var config interface{}
	if cfg, ok := s[strategy]; ok {
		config = cfg
		if m, ok := cfg.(map[string]interface{}); ok {
			symbol, _ = m["symbol"].(string)
		}
	}

	return Bot{
		ID:              id,
		Strategy:        strategy,
		Symbol:          symbol,
		Exchange:        session,
		Session:         session,
		Config:          config,
		State:           s,
		ContainerStatus: StatusRunning,
		Mode:            mode,
	}
}

// ListBots returns all bots for a user. Only available when container is running
// (data comes from bbgo API). Returns empty list for stopped containers.
func (api *API) ListBots(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	mode := r.URL.Query().Get("mode")
	modes := []string{ModeLive, ModePaper}
	if mode != "" {
		modes = []string{mode}
	}

	var bots []Bot
	for _, m := range modes {
		if !api.isContainerRunning(userID, m) {
			continue
		}
		client := api.newBBGoClient(api.container.APIURL(userID, m))
		strategies, err := client.GetStrategies()
		if err != nil {
			continue
		}
		for _, s := range strategies {
			bots = append(bots, botFromStrategy(s, m))
		}
	}

	if bots == nil {
		bots = []Bot{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"bots": bots,
	})
}

// GetBot returns a single bot by bbgo strategyInstanceID. Only works when container is running.
func (api *API) GetBot(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	botID := chi.URLParam(r, "botID")

	// Search across both modes
	for _, m := range []string{ModeLive, ModePaper} {
		if !api.isContainerRunning(userID, m) {
			continue
		}
		client := api.newBBGoClient(api.container.APIURL(userID, m))
		strategies, err := client.GetStrategies()
		if err != nil {
			continue
		}
		for _, s := range strategies {
			id, _ := s["strategyInstanceID"].(string)
			if id != botID {
				continue
			}
			writeJSON(w, http.StatusOK, botFromStrategy(s, m))
			return
		}
	}

	writeError(w, http.StatusNotFound, fmt.Sprintf("bot %s not found (container may be stopped)", botID))
}
