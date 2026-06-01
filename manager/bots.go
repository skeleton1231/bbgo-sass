package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Bot represents a single strategy instance (a "bot" in the web UI).
type Bot struct {
	ID              string          `json:"id"`
	Strategy        string          `json:"strategy"`
	Symbol          string          `json:"symbol"`
	Session         string          `json:"session"`
	State           interface{}     `json:"state"`
	ContainerStatus string          `json:"container_status"`
	Mode            string          `json:"mode"`
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
			id, _ := s["strategyInstanceID"].(string)
			strat, _ := s["strategy"].(string)
			symbol, _ := s["symbol"].(string)
			session, _ := s["session"].(string)
			bots = append(bots, Bot{
				ID:              id,
				Strategy:        strat,
				Symbol:          symbol,
				Session:         session,
				State:           s,
				ContainerStatus: StatusRunning,
				Mode:            m,
			})
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
			strat, _ := s["strategy"].(string)
			symbol, _ := s["symbol"].(string)
			session, _ := s["session"].(string)
			writeJSON(w, http.StatusOK, Bot{
				ID:              id,
				Strategy:        strat,
				Symbol:          symbol,
				Session:         session,
				State:           s,
				ContainerStatus: StatusRunning,
				Mode:            m,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, fmt.Sprintf("bot %s not found (container may be stopped)", botID))
}
