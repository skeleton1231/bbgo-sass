package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/c9s/bbgo/saas/manager/pb"

	"github.com/go-chi/chi/v5"
)

type API struct {
	cfg       *Config
	users     *UserContainerManager
	container *ContainerManager
	proxy     *BotProxy
	creds     *CredentialStore
	encryptor *Encryptor
	syncer    *Syncer
	hub       *MarketDataHub
	notifier  *Notifier
	wsTickets *WSTicketStore
	btExec    *BacktestExecutor
	btJobs    *BacktestJobStore
	btSyncSem chan struct{}

	newBBGoClient    func(baseURL string) *BBGoClient
	containerStart   func(uc *UserContainer) error
	containerStop    func(userID, mode string)
	containerRunning func(userID, mode string) bool
}

func NewAPI(cfg *Config, users *UserContainerManager, cm *ContainerManager, proxy *BotProxy, creds *CredentialStore, enc *Encryptor, syncer *Syncer, hub *MarketDataHub, notifier *Notifier, btExec *BacktestExecutor, btJobs *BacktestJobStore) *API {
	return &API{
		cfg:           cfg,
		users:         users,
		container:     cm,
		proxy:         proxy,
		creds:         creds,
		encryptor:     enc,
		syncer:        syncer,
		hub:           hub,
		notifier:      notifier,
		wsTickets:     NewWSTicketStore(),
		btExec:        btExec,
		btJobs:        btJobs,
		btSyncSem:     make(chan struct{}, 2),
		newBBGoClient: NewBBGoClient,
	}
}

func (api *API) Close() {
	api.wsTickets.Close()
}

func (api *API) RegisterRoutes(r chi.Router) {
	r.Get("/api/health", api.Health)

	r.Get("/api/markets/{exchange}/symbols", api.MarketSymbols)
	r.Get("/api/markets/{exchange}/ticker", api.MarketTicker)
	r.Get("/api/markets/{exchange}/klines", api.MarketKlines)

	r.Route("/", func(r chi.Router) {
		r.Use(UserRateLimit(3*time.Second, 20))
		r.Post("/api/users/{userID}/strategies", api.CreateStrategy)
		r.Delete("/api/users/{userID}/strategies/{strategyID}", api.DeleteStrategy)
		r.Post("/api/users/{userID}/start", api.StartUser)
		r.Post("/api/users/{userID}/stop", api.StopUser)
		r.Post("/api/credentials", api.CreateCredential)
		r.Delete("/api/credentials/{id}", api.DeleteCredential)
		r.Post("/api/notifications/config", api.CreateNotificationConfig)
		r.Delete("/api/notifications/config/{id}", api.DeleteNotificationConfig)
		r.Post("/api/notifications/test", api.TestNotification)
		r.Post("/api/backtest", api.RunBacktest)
		r.Post("/api/backtest/submit", api.SubmitBacktest)
		r.Post("/api/backtest/sync", api.SyncBacktestData)
	})

	r.Get("/api/users/{userID}/strategies", api.ListStrategies)
	r.Get("/api/users/{userID}/status", api.UserStatus)

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
	r.Get("/api/users/{userID}/bbgo/pnl", api.BBGoPnL)

	r.Get("/api/users/{userID}/logs", api.ContainerLogs)

	r.Get("/api/notifications/config", api.ListNotificationConfigs)

	r.Get("/api/ws/ticket", api.IssueWSTicket)
	r.Get("/api/ws", api.HandleWebSocket)

	r.HandleFunc("/api/bbgo/{userID}/*", api.ProxyToBot)

	r.Get("/api/backtest/jobs", api.ListBacktestJobs)
	r.Get("/api/backtest/jobs/{jobID}", api.GetBacktestJob)
	r.Get("/api/backtest/status", api.BacktestSyncStatus)
	r.Get("/api/credentials", api.ListCredentials)
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
		if headerID, ok := userIDFromRequest(r); ok && headerID != urlID {
			writeError(w, http.StatusForbidden, "user ID mismatch")
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

// modeFromQuery reads the ?mode=paper query parameter, defaulting to ModeLive.
func modeFromQuery(r *http.Request) string {
	m := r.URL.Query().Get("mode")
	if m == ModePaper {
		return ModePaper
	}
	return ModeLive
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
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
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

	normalizedStrategy := req.Strategy
	if alias, ok := legacyStrategyAliases[req.Strategy]; ok {
		normalizedStrategy = alias
	}
	if req.Mode == "" {
		req.Mode = ModePaper
	}
	if req.Mode != ModePaper && req.Mode != ModeLive {
		writeError(w, http.StatusBadRequest, "mode must be 'paper' or 'live'")
		return
	}
	if req.Mode == ModePaper && liveOnlyStrategies[normalizedStrategy] {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("strategy %s only supports live mode", req.Strategy))
		return
	}

	if req.Mode == ModeLive && api.creds != nil {
		exchanges := []string{}
		if req.CrossExchange {
			for _, sr := range req.Sessions {
				exchanges = append(exchanges, sr.Exchange)
			}
		} else {
			exchanges = append(exchanges, req.Exchange)
		}
		for _, ex := range exchanges {
			if _, _, _, err := api.creds.GetDecryptedByMode(userID, ex, false); err != nil {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("live mode requires API credentials for %s — add them in Settings first", ex))
				return
			}
		}
	}

	entry := StrategyEntry{
		ID:            generateID("strat"),
		Name:          req.Name,
		Exchange:      req.Exchange,
		Strategy:      req.Strategy,
		Config:        req.Config,
		Mode:          req.Mode,
		CrossExchange: req.CrossExchange,
		Sessions:      req.Sessions,
	}

	uc, created := api.users.AddStrategy(userID, req.Mode, entry)

	if uc.Status == StatusRunning {
		api.users.UpdateStatus(userID, req.Mode, StatusStarting)
		go api.startUserContainer(userID, req.Mode)
	} else if created {
		api.users.UpdateStatus(userID, req.Mode, StatusStarting)
		go api.startUserContainer(userID, req.Mode)
	}

	if api.syncer != nil {
		go api.syncer.SyncUser(userID, req.Mode)
	}
	writeJSON(w, http.StatusCreated, uc)
}

func (api *API) ListStrategies(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	containers := api.users.GetByUser(userID)
	if len(containers) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user_id":    userID,
			"containers": map[string]interface{}{},
		})
		return
	}
	for _, uc := range containers {
		api.refreshContainerStatus(uc)
	}
	byMode := make(map[string]interface{})
	for _, uc := range containers {
		byMode[uc.Mode] = uc
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":    userID,
		"containers": byMode,
	})
}

func (api *API) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	strategyID := chi.URLParam(r, "strategyID")

	found, mode := api.users.RemoveStrategy(userID, strategyID)
	if !found {
		writeError(w, http.StatusNotFound, "strategy not found")
		return
	}

	uc, exists := api.users.Get(userID, mode)
	if !exists || len(uc.Strategies) == 0 {
		api.stopContainer(userID, mode)
		api.users.UpdateStatus(userID, mode, StatusStopped)
		if api.syncer != nil {
			go api.syncer.SyncUser(userID, mode)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "reason": "no strategies left"})
		return
	}

	if uc.Status == StatusRunning {
		api.users.UpdateStatus(userID, mode, StatusStarting)
		go api.startUserContainer(userID, mode)
	}

	writeJSON(w, http.StatusOK, uc)
}

func (api *API) isContainerRunning(userID, mode string) bool {
	if api.containerRunning != nil {
		return api.containerRunning(userID, mode)
	}
	return api.container.IsRunning(userID, mode)
}

func (api *API) startContainer(uc *UserContainer) error {
	if api.containerStart != nil {
		return api.containerStart(uc)
	}
	if api.container == nil {
		return fmt.Errorf("container manager not configured")
	}
	return api.container.CreateAndStart(uc)
}

func (api *API) stopContainer(userID, mode string) {
	if api.containerStop != nil {
		api.containerStop(userID, mode)
		return
	}
	api.container.Stop(userID, mode)
}

func (api *API) StartUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	mode := modeFromQuery(r)

	uc, found := api.users.Get(userID, mode)
	if !found || len(uc.Strategies) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("no strategies configured for %s mode", mode))
		return
	}

	if api.isContainerRunning(userID, mode) {
		api.users.UpdateStatus(userID, mode, StatusRunning)
		uc.Status = StatusRunning
		writeJSON(w, http.StatusOK, uc)
		return
	}

	if uc.Status == StatusStarting {
		writeJSON(w, http.StatusAccepted, uc)
		return
	}

	if !api.users.CompareAndSetStatus(userID, mode, uc.Status, StatusStarting) {
		writeJSON(w, http.StatusAccepted, uc)
		return
	}
	uc.Status = StatusStarting
	writeJSON(w, http.StatusAccepted, uc)

	go api.startUserContainer(userID, mode)
}

func (api *API) startUserContainer(userID, mode string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uc, found := api.users.Get(userID, mode)
	if !found || uc.Status != StatusStarting {
		return
	}

	if err := api.startContainer(uc); err != nil {
		log.Printf("start %s container for user %s failed: %v", mode, userID, err)
		api.users.UpdateStatus(userID, mode, StatusError)
		return
	}

	var reachable bool
	baseURL := ""
	if api.container != nil {
		baseURL = api.container.APIURL(userID, mode)
	}
	if baseURL == "" {
		api.users.UpdateStatus(userID, mode, StatusError)
		log.Printf("user %s %s: cannot determine container URL", userID, mode)
		return
	}
	if api.newBBGoClient == nil {
		api.users.UpdateStatus(userID, mode, StatusError)
		log.Printf("user %s %s: bbgo client factory not configured", userID, mode)
		return
	}
	client := api.newBBGoClient(baseURL)
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			api.users.UpdateStatus(userID, mode, StatusError)
			log.Printf("user %s %s health check cancelled: %v", userID, mode, ctx.Err())
			return
		default:
		}
		if err := client.WithContext(ctx).Ping(); err == nil {
			reachable = true
			break
		}
		time.Sleep(time.Second)
	}

	if !reachable {
		api.users.UpdateStatus(userID, mode, StatusError)
		log.Printf("user %s %s container started but health check failed", userID, mode)
		return
	}

	api.users.UpdateStatus(userID, mode, StatusRunning)
	log.Printf("user %s %s container started and healthy", userID, mode)

	go func() {
		grpcAddr := api.container.ContainerGRPCAddr(userID, mode)
		for i := 0; i < 10; i++ {
			conn, err := net.DialTimeout("tcp", grpcAddr, time.Second)
			if err == nil {
				conn.Close()
				return
			}
			time.Sleep(time.Second)
		}
		log.Printf("grpc port %s not ready after 10s for user %s %s", grpcAddr, userID, mode)
	}()
}

func (api *API) StopUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	mode := modeFromQuery(r)

	api.stopContainer(userID, mode)
	api.users.UpdateStatus(userID, mode, StatusStopped)
	if api.syncer != nil {
		go api.syncer.SyncUser(userID, mode)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "user_id": userID, "mode": mode})
}

func (api *API) UserStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	containers := api.users.GetByUser(userID)
	if len(containers) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user_id":    userID,
			"containers": map[string]interface{}{},
		})
		return
	}
	for _, uc := range containers {
		api.refreshContainerStatus(uc)
	}
	byMode := make(map[string]interface{})
	for _, uc := range containers {
		byMode[uc.Mode] = uc
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":    userID,
		"containers": byMode,
	})
}

func (api *API) ProxyToBot(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if !isValidUUID(userID) {
		writeError(w, http.StatusBadRequest, "invalid user ID format")
		return
	}
	if headerID, ok := userIDFromRequest(r); ok && headerID != userID {
		writeError(w, http.StatusForbidden, "user ID mismatch")
		return
	}

	mode := modeFromQuery(r)
	if _, found := api.users.Get(userID, mode); !found {
		writeError(w, http.StatusNotFound, "user container not found")
		return
	}

	api.proxy.ProxyToBot(w, r, userID, mode)
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
		Exchange  string          `json:"exchange"`
		StartTime string          `json:"start_time"`
		EndTime   string          `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	yamlContent, err := buildBacktestYAML(req.Strategy, req.Config, req.StartTime, req.EndTime, req.Exchange, "")
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

func (api *API) SyncBacktestData(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}

	var req struct {
		Exchange  string   `json:"exchange"`
		Symbols   []string `json:"symbols"`
		StartTime string   `json:"start_time"`
		EndTime   string   `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	select {
	case api.btSyncSem <- struct{}{}:
	default:
		writeError(w, http.StatusTooManyRequests, "backtest sync already in progress, try again later")
		return
	}
	defer func() { <-api.btSyncSem }()
	if req.Exchange == "" {
		req.Exchange = "binance"
	}
	if len(req.Symbols) == 0 {
		req.Symbols = []string{"BTCUSDT", "ETHUSDT"}
	}
	if len(req.Symbols) > 10 {
		writeError(w, http.StatusBadRequest, "too many symbols (max 10)")
		return
	}
	if req.StartTime == "" {
		req.StartTime = "2024-01-01"
	}
	if req.EndTime == "" {
		req.EndTime = "2025-12-31"
	}

	type syncResult struct {
		Symbol string `json:"symbol"`
		Output string `json:"output"`
		Error  string `json:"error,omitempty"`
	}
	log.Printf("backtest sync requested by user %s: %s %v %s-%s", userID, req.Exchange, req.Symbols, req.StartTime, req.EndTime)
	results := make([]syncResult, len(req.Symbols))
	var wg sync.WaitGroup
	for i, sym := range req.Symbols {
		wg.Add(1)
		go func(idx int, sym string) {
			defer wg.Done()
			out, err := api.container.SyncBacktest(req.Exchange, sym, req.StartTime, req.EndTime)
			r := syncResult{Symbol: sym, Output: out}
			if err != nil {
				r.Error = err.Error()
			}
			results[idx] = r
		}(i, sym)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"exchange": req.Exchange,
		"synced":   results,
	})
}

func (api *API) BacktestSyncStatus(w http.ResponseWriter, _ *http.Request) {
	dbPath := api.container.cfg.BacktestSharedDir
	if dbPath == "" {
		dbPath = api.container.cfg.DataDir + "/backtest-shared"
	}
	dbPath += "/backtest.db"
	info, err := os.Stat(dbPath)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"available": false,
			"error":     err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"available": true,
		"size":      info.Size(),
		"modified":  info.ModTime().Format(time.RFC3339),
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
	if _, ok := exchangePrefixes[req.Exchange]; !ok {
		writeError(w, http.StatusBadRequest, "unsupported exchange: "+req.Exchange)
		return
	}
	if api.encryptor == nil || api.creds == nil {
		writeError(w, http.StatusServiceUnavailable, "credential storage not configured")
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
		ID:                  generateID("cred"),
		UserID:              userID,
		Exchange:            req.Exchange,
		APIKeyEncrypted:     keyEnc,
		APISecretEncrypted:  secretEnc,
		PassphraseEncrypted: passEnc,
		IsTestnet:           req.IsTestnet,
	}

	if err := api.creds.Upsert(cred); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if api.syncer != nil {
		go api.syncer.SyncCredential(cred)
	}

	mode := ModeLive
	if req.IsTestnet {
		mode = ModePaper
	}
	if uc, ok := api.users.Get(userID, mode); ok && uc.Status == StatusRunning {
		api.users.UpdateStatus(userID, mode, StatusStarting)
		go api.startUserContainer(userID, mode)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          cred.ID,
		"user_id":     cred.UserID,
		"exchange":    cred.Exchange,
		"is_testnet":  cred.IsTestnet,
		"is_verified": cred.IsVerified,
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
	var isTestnet bool
	for _, c := range creds {
		if c.ID == id {
			exchange = c.Exchange
			isTestnet = c.IsTestnet
			break
		}
	}

	if err := api.creds.Delete(userID, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if exchange != "" {
		if api.syncer != nil {
			go api.syncer.DeleteCredential(userID, exchange)
		}
	}
	mode := ModeLive
	if isTestnet {
		mode = ModePaper
	}
	if uc, ok := api.users.Get(userID, mode); ok && uc.Status == StatusRunning {
		api.users.UpdateStatus(userID, mode, StatusStarting)
		go api.startUserContainer(userID, mode)
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

func (api *API) userFromURL(w http.ResponseWriter, r *http.Request) (*UserContainer, string, bool) {
	userID := chi.URLParam(r, "userID")
	if !isValidUUID(userID) {
		writeError(w, http.StatusBadRequest, "invalid user ID format")
		return nil, "", false
	}
	if headerID, ok := userIDFromRequest(r); ok && headerID != userID {
		writeError(w, http.StatusForbidden, "user ID mismatch")
		return nil, "", false
	}

	mode := modeFromQuery(r)
	uc, found := api.users.Get(userID, mode)
	if !found {
		writeError(w, http.StatusNotFound, "user container not found")
		return nil, "", false
	}
	return uc, userID, true
}

func (api *API) bbgoClientForUser(w http.ResponseWriter, r *http.Request) (*BBGoClient, string, bool) {
	uc, userID, ok := api.userFromURL(w, r)
	if !ok {
		return nil, "", false
	}

	if uc.Status != StatusRunning {
		writeError(w, http.StatusServiceUnavailable, "container is not running")
		return nil, "", false
	}

	return api.newBBGoClient(api.container.APIURL(userID, uc.Mode)).WithContext(r.Context()), userID, true
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"symbols": filterTradingPairs(symbols)})
}

func (api *API) MarketSymbols(w http.ResponseWriter, r *http.Request) {
	exchange := chi.URLParam(r, "exchange")
	if exchange == "" {
		writeError(w, http.StatusBadRequest, "exchange is required")
		return
	}
	base := "http://" + api.cfg.MarketDataRESTAddr
	client := api.newBBGoClient(base)
	symbols, err := client.GetSessionSymbols(exchange)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("marketdata symbols: %s", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"symbols": filterTradingPairs(symbols)})
}

func (api *API) MarketTicker(w http.ResponseWriter, r *http.Request) {
	exchange := chi.URLParam(r, "exchange")
	symbol := r.URL.Query().Get("symbol")
	if exchange == "" || symbol == "" {
		writeError(w, http.StatusBadRequest, "exchange and symbol are required")
		return
	}
	if api.hub == nil || api.hub.conn == nil {
		writeError(w, http.StatusServiceUnavailable, "market data service not connected")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	client := pb.NewMarketDataQueryClient(api.hub.conn)
	resp, err := client.QueryTicker(ctx, &pb.QueryTickerRequest{Exchange: exchange, Symbol: symbol})
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("query ticker: %s", err))
		return
	}
	if resp.Error != nil {
		writeError(w, http.StatusBadGateway, resp.Error.ErrorMessage)
		return
	}
	if resp.Ticker == nil {
		writeError(w, http.StatusNotFound, "ticker not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ticker": map[string]interface{}{
			"symbol": resp.Ticker.Symbol,
			"open":   resp.Ticker.Open,
			"high":   resp.Ticker.High,
			"low":    resp.Ticker.Low,
			"close":  resp.Ticker.Close,
			"volume": resp.Ticker.Volume,
		},
	})
}

func (api *API) MarketKlines(w http.ResponseWriter, r *http.Request) {
	exchange := chi.URLParam(r, "exchange")
	symbol := r.URL.Query().Get("symbol")
	if exchange == "" || symbol == "" {
		writeError(w, http.StatusBadRequest, "exchange and symbol are required")
		return
	}
	if api.hub == nil || api.hub.conn == nil {
		writeError(w, http.StatusServiceUnavailable, "market data service not connected")
		return
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}
	limitStr := r.URL.Query().Get("limit")
	limit := int64(500)
	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil && l > 0 && l <= 1500 {
			limit = l
		}
	}
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	var startTime, endTime int64
	if startTimeStr != "" {
		if t, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			startTime = t
		}
	}
	if endTimeStr != "" {
		if t, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			endTime = t
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	client := pb.NewMarketDataQueryClient(api.hub.conn)
	resp, err := client.QueryKLines(ctx, &pb.QueryKLinesRequest{
		Exchange:  exchange,
		Symbol:    symbol,
		Interval:  interval,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("query klines: %s", err))
		return
	}
	if resp.Error != nil {
		writeError(w, http.StatusBadGateway, resp.Error.ErrorMessage)
		return
	}

	klines := make([]map[string]interface{}, 0, len(resp.Klines))
	for _, k := range resp.Klines {
		klines = append(klines, map[string]interface{}{
			"time":        k.StartTime,
			"open":        k.Open,
			"high":        k.High,
			"low":         k.Low,
			"close":       k.Close,
			"volume":      k.Volume,
			"quoteVolume": k.QuoteVolume,
			"closed":      k.Closed,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"klines": klines})
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

func (api *API) BBGoPnL(w http.ResponseWriter, r *http.Request) {
	uc, userID, ok := api.userFromURL(w, r)
	if !ok {
		return
	}

	if api.syncer != nil {
		trades, err := api.syncer.GetTradesForPnL(userID)
		if err != nil {
			log.Printf("pnl supabase fallback for user %s: %v", userID, err)
		}
		if err == nil && len(trades) > 0 {
			report := calculatePnL(trades)
			writeJSON(w, http.StatusOK, report)
			return
		}
	}

	if uc.Status != StatusRunning {
		writeError(w, http.StatusServiceUnavailable, "container is not running")
		return
	}
	if api.container == nil {
		writeError(w, http.StatusInternalServerError, "container manager not available")
		return
	}
	client := api.newBBGoClient(api.container.APIURL(userID, uc.Mode)).WithContext(r.Context())

	exchange := r.URL.Query().Get("exchange")
	symbol := r.URL.Query().Get("symbol")

	var lastGID int64
	if gidStr := r.URL.Query().Get("gid"); gidStr != "" {
		if v, err := strconv.ParseInt(gidStr, 10, 64); err == nil {
			lastGID = v
		}
	}

	trades, err := client.GetAllTradesFrom(exchange, symbol, lastGID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	report := calculatePnL(trades)
	writeJSON(w, http.StatusOK, report)
}

func (api *API) refreshContainerStatus(uc *UserContainer) {
	if uc.Status != StatusRunning && uc.Status != StatusStarting {
		return
	}
	running, err := api.container.CheckRunning(uc.UserID, uc.Mode)
	if err != nil {
		return
	}
	if !running {
		api.users.UpdateStatus(uc.UserID, uc.Mode, StatusStopped)
		uc.Status = StatusStopped
	}
}

func (api *API) ContainerLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	mode := modeFromQuery(r)
	uc, found := api.users.Get(userID, mode)
	if !found {
		writeError(w, http.StatusNotFound, "user container not found")
		return
	}

	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}

	logs, err := api.container.Logs(uc.UserID, uc.Mode, tail)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"logs": logs})
}

func (api *API) CreateNotificationConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}

	var req struct {
		Type       string `json:"type"`
		Token      string `json:"token"`
		ChatID     string `json:"chat_id"`
		WebhookURL string `json:"webhook_url"`
		Rules      struct {
			TradeEvents     bool `json:"trade_events"`
			OrderEvents     bool `json:"order_events"`
			ContainerHealth bool `json:"container_health"`
		} `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Type != "telegram" && req.Type != "slack" {
		writeError(w, http.StatusBadRequest, "type must be telegram or slack")
		return
	}

	ch := NotificationChannel{
		ID:      generateID("notif"),
		UserID:  userID,
		Type:    req.Type,
		ChatID:  req.ChatID,
		Enabled: true,
	}

	switch req.Type {
	case "telegram":
		if req.Token == "" || req.ChatID == "" {
			writeError(w, http.StatusBadRequest, "token and chat_id are required for telegram")
			return
		}
		enc, err := api.notifier.EncryptToken(req.Token)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "encryption failed")
			return
		}
		ch.TokenEnc = enc
	case "slack":
		if req.WebhookURL == "" {
			writeError(w, http.StatusBadRequest, "webhook_url is required for slack")
			return
		}
		enc, err := api.notifier.EncryptToken(req.WebhookURL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "encryption failed")
			return
		}
		ch.WebhookURL = enc
	}

	cfg := NotificationConfig{
		Channel: ch,
		Rules: NotificationRule{
			TradeEvents:     req.Rules.TradeEvents,
			OrderEvents:     req.Rules.OrderEvents,
			ContainerHealth: req.Rules.ContainerHealth,
		},
	}

	if err := api.notifier.Create(userID, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      ch.ID,
		"type":    ch.Type,
		"enabled": ch.Enabled,
		"rules":   cfg.Rules,
	})
}

func (api *API) ListNotificationConfigs(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	configs := api.notifier.List(userID)
	safe := make([]map[string]interface{}, len(configs))
	for i, c := range configs {
		safe[i] = map[string]interface{}{
			"id":      c.Channel.ID,
			"type":    c.Channel.Type,
			"enabled": c.Channel.Enabled,
			"rules":   c.Rules,
		}
	}
	writeJSON(w, http.StatusOK, safe)
}

func (api *API) DeleteNotificationConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	id := chi.URLParam(r, "id")
	if err := api.notifier.Delete(userID, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (api *API) TestNotification(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	sent := api.notifier.Dispatch(userID, NotificationEvent{
		Type:    "test",
		Title:   "BBGO Test Notification",
		Message: "If you see this, notifications are working!",
	})
	if !sent {
		configs := api.notifier.List(userID)
		if len(configs) == 0 {
			writeError(w, http.StatusBadRequest, "no notification channels configured")
		} else {
			writeError(w, http.StatusTooManyRequests, "rate limited — try again in a minute")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (api *API) SubmitBacktest(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}

	var req struct {
		Strategy  string          `json:"strategy"`
		Config    json.RawMessage `json:"config"`
		Exchange  string          `json:"exchange"`
		Symbol    string          `json:"symbol"`
		StartTime string          `json:"start_time"`
		EndTime   string          `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Strategy == "" {
		writeError(w, http.StatusBadRequest, "strategy is required")
		return
	}
	if req.Exchange == "" {
		req.Exchange = "binance"
	}
	if req.Symbol == "" {
		var cfg map[string]interface{}
		if err := json.Unmarshal(req.Config, &cfg); err == nil {
			if s, ok := cfg["symbol"].(string); ok && s != "" {
				req.Symbol = s
			}
		}
		if req.Symbol == "" {
			req.Symbol = "BTCUSDT"
		}
	}
	if req.StartTime == "" {
		req.StartTime = "2024-01-01"
	}
	if req.EndTime == "" {
		req.EndTime = "2024-06-01"
	}

	needSync := !api.hasDataForRange(req.Exchange, req.Symbol, req.StartTime, req.EndTime)

	job := &BacktestJob{
		ID:        generateID("bt"),
		UserID:    userID,
		Strategy:  req.Strategy,
		Config:    req.Config,
		Exchange:  req.Exchange,
		Symbol:    req.Symbol,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		NeedSync:  needSync,
	}

	if err := api.btExec.Submit(job); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	refreshed, _ := api.btJobs.Get(job.ID)
	status := JobPending
	if refreshed != nil {
		status = refreshed.Status
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"job_id":    job.ID,
		"status":    status,
		"need_sync": needSync,
	})
}

func (api *API) GetBacktestJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}

	jobID := chi.URLParam(r, "jobID")
	job, found := api.btJobs.Get(jobID)
	if !found {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if job.UserID != userID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (api *API) ListBacktestJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	jobs := api.btJobs.ListByUser(userID)
	if jobs == nil {
		jobs = []*BacktestJob{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs": jobs,
	})
}

func (api *API) hasDataForRange(exchange, symbol, startTime, endTime string) bool {
	return false
}
