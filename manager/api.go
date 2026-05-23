package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type API struct {
	users     *UserContainerManager
	container *ContainerManager
	proxy     *BotProxy
	creds     *CredentialStore
	encryptor *Encryptor
	syncer    *Syncer

	newBBGoClient func(baseURL string) *BBGoClient
}

func NewAPI(users *UserContainerManager, cm *ContainerManager, proxy *BotProxy, creds *CredentialStore, enc *Encryptor, syncer *Syncer) *API {
	return &API{
		users:         users,
		container:     cm,
		proxy:         proxy,
		creds:         creds,
		encryptor:     enc,
		syncer:        syncer,
		newBBGoClient: NewBBGoClient,
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

	// Aggregated bbgo data endpoints (Manager → bbgo REST API)
	r.Get("/api/users/{userID}/bbgo/ping", api.BBGoPing)
	r.Get("/api/users/{userID}/bbgo/sessions", api.BBGoSessions)
	r.Get("/api/users/{userID}/bbgo/session/{session}", api.BBGoSessionDetail)
	r.Get("/api/users/{userID}/bbgo/session/{session}/trades", api.BBGoSessionTrades)
	r.Get("/api/users/{userID}/bbgo/session/{session}/open-orders", api.BBGoSessionOpenOrders)
	r.Get("/api/users/{userID}/bbgo/session/{session}/account", api.BBGoSessionAccount)
	r.Get("/api/users/{userID}/bbgo/session/{session}/balances", api.BBGoSessionBalances)
	r.Get("/api/users/{userID}/bbgo/session/{session}/symbols", api.BBGoSessionSymbols)
	r.Get("/api/users/{userID}/bbgo/assets", api.BBGoAssets)
	r.Get("/api/users/{userID}/bbgo/strategies", api.BBGoStrategies)
	r.Get("/api/users/{userID}/bbgo/trades", api.BBGoTrades)
	r.Get("/api/users/{userID}/bbgo/orders/closed", api.BBGoClosedOrders)
	r.Get("/api/users/{userID}/bbgo/trading-volume", api.BBGoTradingVolume)

	// Container logs
	r.Get("/api/users/{userID}/logs", api.ContainerLogs)

	// Generic proxy for any other bbgo API calls
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

func (api *API) resolveUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	urlID := chi.URLParam(r, "userID")
	if urlID != "" {
		if !isValidUUID(urlID) {
			writeError(w, http.StatusBadRequest, "invalid user ID format")
			return "", false
		}
		return urlID, true
	}
	id, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return "", false
	}
	return id, true
}

func (api *API) CreateStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	var req struct {
		Name          string              `json:"name"`
		Exchange      string              `json:"exchange"`
		Strategy      string              `json:"strategy"`
		Config        json.RawMessage     `json:"config"`
		Mode          string              `json:"mode"`
		CrossExchange bool                `json:"crossExchange"`
		Sessions      []SessionRoleConfig `json:"sessions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Strategy == "" {
		writeError(w, http.StatusBadRequest, "strategy is required")
		return
	}
	if !req.CrossExchange && req.Exchange == "" {
		writeError(w, http.StatusBadRequest, "exchange is required for single-exchange strategies")
		return
	}
	if req.CrossExchange && len(req.Sessions) == 0 {
		writeError(w, http.StatusBadRequest, "sessions are required for cross-exchange strategies")
		return
	}

	if req.Mode == "live" && api.creds != nil {
		exchanges := []string{}
		if req.CrossExchange {
			for _, sr := range req.Sessions {
				exchanges = append(exchanges, sr.Exchange)
			}
		} else {
			exchanges = append(exchanges, req.Exchange)
		}
		for _, ex := range exchanges {
			if _, _, _, err := api.creds.GetDecrypted(userID, ex); err != nil {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("live mode requires API credentials for %s — add them in Settings first", ex))
				return
			}
		}
	}

	entry := StrategyEntry{
		ID:            fmt.Sprintf("strat-%d", time.Now().UnixNano()),
		Name:          req.Name,
		Exchange:      req.Exchange,
		Strategy:      req.Strategy,
		Config:        req.Config,
		Mode:          req.Mode,
		CrossExchange: req.CrossExchange,
		Sessions:      req.Sessions,
	}

	uc, created := api.users.AddStrategy(userID, entry)

	if uc.Status == StatusRunning {
		if err := api.container.Restart(uc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if created {
		if err := api.container.CreateAndStart(uc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		api.users.UpdateStatus(userID, StatusRunning)
		uc.Status = StatusRunning
	}

	if api.syncer != nil { go api.syncer.SyncUser(userID) }
	writeJSON(w, http.StatusCreated, uc)
}

func (api *API) ListStrategies(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	uc, found := api.users.Get(userID)
	if !found {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user_id":    userID,
			"status":     StatusStopped,
			"strategies": []StrategyEntry{},
		})
		return
	}
	api.refreshContainerStatus(uc)
	writeJSON(w, http.StatusOK, uc)
}

func (api *API) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	strategyID := chi.URLParam(r, "strategyID")

	if !api.users.RemoveStrategy(userID, strategyID) {
		writeError(w, http.StatusNotFound, "strategy not found")
		return
	}

	uc, found := api.users.Get(userID)
	if !found || len(uc.Strategies) == 0 {
		api.container.Stop(userID)
		api.users.UpdateStatus(userID, StatusStopped)
		if api.syncer != nil { go api.syncer.SyncUser(userID) }
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
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	uc, found := api.users.Get(userID)
	if !found || len(uc.Strategies) == 0 {
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
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	api.container.Stop(userID)
	api.users.UpdateStatus(userID, StatusStopped)
	if api.syncer != nil { go api.syncer.SyncUser(userID) }
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "user_id": userID})
}

func (api *API) UserStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	uc, found := api.users.Get(userID)
	if !found {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user_id":    userID,
			"status":     StatusStopped,
			"strategies": []StrategyEntry{},
		})
		return
	}
	api.refreshContainerStatus(uc)
	writeJSON(w, http.StatusOK, uc)
}

func (api *API) ProxyToBot(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if !isValidUUID(userID) {
		writeError(w, http.StatusBadRequest, "invalid user ID format")
		return
	}

	if _, found := api.users.Get(userID); !found {
		writeError(w, http.StatusNotFound, "user container not found")
		return
	}

	api.proxy.ProxyToBot(w, r, userID)
}

func (api *API) RunBacktest(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
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

	yamlContent, err := buildBacktestYAML(req.Strategy, req.Config, req.StartTime, req.EndTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", err))
		return
	}

	result, err := api.container.RunBacktest(userID, yamlContent)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"output": string(result),
	})
}

func (api *API) CreateCredential(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
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

	if api.syncer != nil { go api.syncer.SyncCredential(cred) }

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         cred.ID,
		"user_id":    cred.UserID,
		"exchange":   cred.Exchange,
		"is_testnet": cred.IsTestnet,
	})
}

func (api *API) ListCredentials(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
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
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	id := chi.URLParam(r, "id")

	creds, _ := api.creds.List(userID)
	var exchange string
	for _, c := range creds {
		if c.ID == id {
			exchange = c.Exchange
			break
		}
	}

	if err := api.creds.Delete(userID, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if exchange != "" {
		if api.syncer != nil { go api.syncer.DeleteCredential(userID, exchange) }
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

func (api *API) bbgoClientForUser(w http.ResponseWriter, r *http.Request) (*BBGoClient, string, bool) {
	userID := chi.URLParam(r, "userID")
	if !isValidUUID(userID) {
		writeError(w, http.StatusBadRequest, "invalid user ID format")
		return nil, "", false
	}

	uc, found := api.users.Get(userID)
	if !found {
		writeError(w, http.StatusNotFound, "user container not found")
		return nil, "", false
	}

	if uc.Status != StatusRunning {
		writeError(w, http.StatusServiceUnavailable, "container is not running")
		return nil, "", false
	}

	return api.newBBGoClient(api.container.APIURL(userID)), userID, true
}

func (api *API) BBGoPing(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	if err := client.Ping(); err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("bbgo ping failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (api *API) BBGoSessions(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	sessions, err := client.GetSessions()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": sessions})
}

func (api *API) BBGoSessionDetail(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	session := chi.URLParam(r, "session")
	detail, err := client.GetSession(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": detail})
}

func (api *API) BBGoSessionTrades(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	session := chi.URLParam(r, "session")
	trades, err := client.GetSessionTrades(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"trades": trades})
}

func (api *API) BBGoSessionOpenOrders(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	session := chi.URLParam(r, "session")
	orders, err := client.GetSessionOpenOrders(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": orders})
}

func (api *API) BBGoSessionAccount(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	session := chi.URLParam(r, "session")
	account, err := client.GetSessionAccount(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"account": account})
}

func (api *API) BBGoSessionBalances(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	session := chi.URLParam(r, "session")
	balances, err := client.GetSessionBalances(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"balances": balances})
}

func (api *API) BBGoSessionSymbols(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	session := chi.URLParam(r, "session")
	symbols, err := client.GetSessionSymbols(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"symbols": symbols})
}

func (api *API) BBGoAssets(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	assets, err := client.GetAssets()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"assets": assets})
}

func (api *API) BBGoStrategies(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	strategies, err := client.GetStrategies()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"strategies": strategies})
}

func (api *API) BBGoTrades(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	exchange := r.URL.Query().Get("exchange")
	symbol := r.URL.Query().Get("symbol")
	gidStr := r.URL.Query().Get("gid")
	var lastGID int64
	if gidStr != "" {
		if v, err := strconv.ParseInt(gidStr, 10, 64); err == nil {
			lastGID = v
		}
	}
	trades, err := client.GetTrades(exchange, symbol, lastGID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"trades": trades})
}

func (api *API) BBGoClosedOrders(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	exchange := r.URL.Query().Get("exchange")
	symbol := r.URL.Query().Get("symbol")
	gidStr := r.URL.Query().Get("gid")
	var lastGID int64
	if gidStr != "" {
		if v, err := strconv.ParseInt(gidStr, 10, 64); err == nil {
			lastGID = v
		}
	}
	orders, err := client.GetClosedOrders(exchange, symbol, lastGID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": orders})
}

func (api *API) BBGoTradingVolume(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	period := r.URL.Query().Get("period")
	segment := r.URL.Query().Get("segment")
	volumes, err := client.GetTradingVolume(period, segment)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tradingVolumes": volumes})
}

// refreshContainerStatus checks Docker for the actual container state and updates
// the in-memory UserContainer status. If Docker is unreachable, the in-memory
// status is left unchanged.
func (api *API) refreshContainerStatus(uc *UserContainer) {
	if uc.Status != StatusRunning {
		return
	}
	running, err := api.container.CheckRunning(uc.UserID)
	if err != nil {
		return
	}
	if !running {
		api.users.UpdateStatus(uc.UserID, StatusStopped)
		uc.Status = StatusStopped
	}
}

func (api *API) ContainerLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	uc, found := api.users.Get(userID)
	if !found {
		writeError(w, http.StatusNotFound, "user container not found")
		return
	}

	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}

	logs, err := api.container.Logs(uc.UserID, tail)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"logs": logs})
}
