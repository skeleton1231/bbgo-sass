package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type API struct {
	users       *UserContainerManager
	container   *ContainerManager
	proxy       *BotProxy
	creds       *CredentialStore
	encryptor   *Encryptor
	syncer      *Syncer
}

func NewAPI(users *UserContainerManager, cm *ContainerManager, proxy *BotProxy, creds *CredentialStore, enc *Encryptor, syncer *Syncer) *API {
	return &API{
		users:     users,
		container: cm,
		proxy:     proxy,
		creds:     creds,
		encryptor: enc,
		syncer:    syncer,
	}
}

func (api *API) RegisterRoutes(r chi.Router) {
	r.Get("/api/health", api.Health)

	r.Post("/api/users/{userID}/strategies", api.CreateStrategy)
	r.Get("/api/users/{userID}/strategies", api.ListStrategies)
	r.Delete("/api/users/{userID}/strategies/{strategyID}", api.DeleteStrategy)

	r.Post("/api/users/{userID}/start", api.StartUser)
	r.Post("/api/users/{userID}/stop", api.StopUser)
	r.Get("/api/users/{userID}/status", api.UserStatus)

	r.HandleFunc("/api/bbgo/{userID}/*", api.ProxyToBot)

	r.Post("/api/backtest", api.RunBacktest)

	r.Post("/api/credentials", api.CreateCredential)
	r.Get("/api/credentials", api.ListCredentials)
	r.Delete("/api/credentials/{id}", api.DeleteCredential)
}

func (api *API) Health(w http.ResponseWriter, _ *http.Request) {
	users := api.users.ListUsers()
	running := 0
	for _, u := range users {
		if u.Status == StatusRunning {
			running++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"users":   len(users),
		"running": running,
	})
}

func (api *API) CreateStrategy(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}

	var req struct {
		Name     string          `json:"name"`
		Exchange string          `json:"exchange"`
		Strategy string          `json:"strategy"`
		Config   json.RawMessage `json:"config"`
		Mode     string          `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Exchange == "" || req.Strategy == "" {
		writeError(w, http.StatusBadRequest, "exchange and strategy are required")
		return
	}

	entry := StrategyEntry{
		ID:       fmt.Sprintf("strat-%d", time.Now().UnixNano()),
		Name:     req.Name,
		Exchange: req.Exchange,
		Strategy: req.Strategy,
		Config:   req.Config,
		Mode:     req.Mode,
	}

	uc := api.users.AddStrategy(userID, entry)

	if uc.Status == StatusRunning {
		if err := api.container.Restart(uc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		if err := api.container.CreateAndStart(uc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		api.users.UpdateStatus(userID, StatusRunning)
		uc.Status = StatusRunning
	}

	go api.syncer.SyncUser(userID)
	writeJSON(w, http.StatusCreated, uc)
}

func (api *API) ListStrategies(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}
	uc, ok := api.users.Get(userID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user_id":     userID,
			"status":      StatusStopped,
			"strategies":  []StrategyEntry{},
		})
		return
	}
	writeJSON(w, http.StatusOK, uc)
}

func (api *API) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	strategyID := chi.URLParam(r, "strategyID")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}

	if !api.users.RemoveStrategy(userID, strategyID) {
		writeError(w, http.StatusNotFound, "strategy not found")
		return
	}

	uc, ok := api.users.Get(userID)
	if !ok || len(uc.Strategies) == 0 {
		api.container.Stop(userID)
		api.users.UpdateStatus(userID, StatusStopped)
		go api.syncer.SyncUser(userID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "reason": "no strategies left"})
		return
	}

	if uc.Status == StatusRunning {
		if err := api.container.Restart(uc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, uc)
}

func (api *API) StartUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}

	uc, ok := api.users.Get(userID)
	if !ok || len(uc.Strategies) == 0 {
		writeError(w, http.StatusBadRequest, "no strategies configured")
		return
	}

	if err := api.container.CreateAndStart(uc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.users.UpdateStatus(userID, StatusRunning)
	uc.Status = StatusRunning
	writeJSON(w, http.StatusOK, uc)
}

func (api *API) StopUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}

	api.container.Stop(userID)
	api.users.UpdateStatus(userID, StatusStopped)
	go api.syncer.SyncUser(userID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "user_id": userID})
}

func (api *API) UserStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}

	uc, ok := api.users.Get(userID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user_id":     userID,
			"status":      StatusStopped,
			"strategies":  []StrategyEntry{},
		})
		return
	}
	writeJSON(w, http.StatusOK, uc)
}

func (api *API) ProxyToBot(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}

	_, ok := api.users.Get(userID)
	if !ok {
		writeError(w, http.StatusNotFound, "user container not found")
		return
	}

	api.proxy.ProxyToBot(w, r, userID)
}

func (api *API) RunBacktest(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}

	var req struct {
		Strategy  string          `json:"strategy"`
		Config    json.RawMessage `json:"config"`
		StartTime string          `json:"start_time"`
		EndTime   string          `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	yamlContent := buildBacktestYAML(req.Strategy, req.Config, req.StartTime, req.EndTime)

	result, err := api.container.RunBacktest(userID, []byte(yamlContent))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"output": string(result),
	})
}

func buildBacktestYAML(strategy string, rawConfig json.RawMessage, startTime, endTime string) string {
	var cfg struct {
		Symbol   string `json:"symbol"`
		Exchange string `json:"exchange"`
		Interval string `json:"interval"`
	}
	json.Unmarshal(rawConfig, &cfg)

	if cfg.Interval == "" {
		cfg.Interval = "1h"
	}
	if cfg.Exchange == "" {
		cfg.Exchange = "binance"
	}
	if startTime == "" {
		startTime = "2024-01-01"
	}
	if endTime == "" {
		endTime = "2024-06-01"
	}

	return fmt.Sprintf("exchange:\n  %s:\n    symbol: %s\n    interval: %s\nstrategy:\n  %s: {}\nbacktest:\n  sessions: [%s]\n  startTime: \"%s\"\n  endTime: \"%s\"\n",
		cfg.Exchange, cfg.Symbol, cfg.Interval, strategy, cfg.Exchange, startTime, endTime)
}

func (api *API) CreateCredential(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}

	var req struct {
		Exchange   string `json:"exchange"`
		APIKey     string `json:"api_key"`
		APISecret  string `json:"api_secret"`
		Passphrase string `json:"passphrase"`
		IsTestnet  bool   `json:"is_testnet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Exchange == "" || req.APIKey == "" || req.APISecret == "" {
		writeError(w, http.StatusBadRequest, "exchange, api_key, api_secret are required")
		return
	}

	keyEnc, err := api.encryptor.Encrypt(req.APIKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypt api key failed")
		return
	}
	secretEnc, err := api.encryptor.Encrypt(req.APISecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypt api secret failed")
		return
	}
	var passEnc string
	if req.Passphrase != "" {
		passEnc, err = api.encryptor.Encrypt(req.Passphrase)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "encrypt passphrase failed")
			return
		}
	}

	cred := ExchangeCredential{
		ID:                  fmt.Sprintf("cred-%d", time.Now().UnixNano()),
		UserID:              userID,
		Exchange:            req.Exchange,
		APIKeyEncrypted:     keyEnc,
		APISecretEncrypted:  secretEnc,
		PassphraseEncrypted: passEnc,
		IsTestnet:           req.IsTestnet,
	}

	if err := api.creds.Create(cred); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         cred.ID,
		"user_id":    cred.UserID,
		"exchange":   cred.Exchange,
		"is_testnet": cred.IsTestnet,
	})
}

func (api *API) ListCredentials(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	creds, err := api.creds.List(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	safe := make([]map[string]interface{}, len(creds))
	for i, c := range creds {
		safe[i] = map[string]interface{}{
			"id":          c.ID,
			"user_id":     c.UserID,
			"exchange":    c.Exchange,
			"is_testnet":  c.IsTestnet,
			"is_verified": c.IsVerified,
		}
	}
	writeJSON(w, http.StatusOK, safe)
}

func (api *API) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if err := api.creds.Delete(userID, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
