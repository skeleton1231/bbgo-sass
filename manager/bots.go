package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Bot represents a single strategy instance (a "bot" in the web UI).
type Bot struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Exchange        string              `json:"exchange"`
	Strategy        string              `json:"strategy"`
	Config          json.RawMessage     `json:"config"`
	Mode            string              `json:"mode"`
	CrossExchange   bool                `json:"crossExchange"`
	Sessions        []SessionRoleConfig `json:"sessions,omitempty"`
	ContainerStatus string              `json:"container_status"`
}

// ListBots returns all bots (strategy instances) for a user, optionally filtered by mode.
func (api *API) ListBots(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	mode := r.URL.Query().Get("mode")
	containers := api.users.GetByUser(userID)

	var bots []Bot
	for _, uc := range containers {
		if mode != "" && uc.Mode != mode {
			continue
		}
		api.refreshContainerStatus(uc)
		for _, s := range uc.Strategies {
			bots = append(bots, Bot{
				ID:              s.ID,
				Name:            s.Name,
				Exchange:        s.Exchange,
				Strategy:        s.Strategy,
				Config:          s.Config,
				Mode:            s.Mode,
				CrossExchange:   s.CrossExchange,
				Sessions:        s.Sessions,
				ContainerStatus: uc.Status,
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

// GetBot returns a single bot (strategy instance) by its ID.
func (api *API) GetBot(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	botID := chi.URLParam(r, "botID")
	mode, found := api.users.FindStrategy(userID, botID)
	if !found {
		writeError(w, http.StatusNotFound, "bot not found")
		return
	}

	uc, exists := api.users.Get(userID, mode)
	if !exists {
		writeError(w, http.StatusNotFound, "bot not found")
		return
	}

	api.refreshContainerStatus(uc)

	var entry *StrategyEntry
	for i := range uc.Strategies {
		if uc.Strategies[i].ID == botID {
			e := uc.Strategies[i]
			entry = &e
			break
		}
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "bot not found")
		return
	}

	writeJSON(w, http.StatusOK, Bot{
		ID:              entry.ID,
		Name:            entry.Name,
		Exchange:        entry.Exchange,
		Strategy:        entry.Strategy,
		Config:          entry.Config,
		Mode:            entry.Mode,
		CrossExchange:   entry.CrossExchange,
		Sessions:        entry.Sessions,
		ContainerStatus: uc.Status,
	})
}
