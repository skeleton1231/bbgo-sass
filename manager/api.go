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
	"github.com/go-chi/chi/v5/middleware"
)

type API struct {
	cfg        *Config
	strategies *StrategyStore
	container  *ContainerManager
	proxy      *BotProxy
	creds      *CredentialStore
	encryptor  *Encryptor
	syncer     *Syncer
	hub        *MarketDataHub
	testnetHub *MarketDataHub
	notifier   *Notifier
	wsTickets  *WSTicketStore
	btExec     *BacktestExecutor
	btJobs     *BacktestJobStore
	btSyncSem  chan struct{}

	stopCtx    context.Context
	stopCancel context.CancelFunc
	starting   sync.Map

	newBBGoClient    func(baseURL string) *BBGoClient
	containerStart   func(userID, mode string) error
	containerStop    func(userID, mode string)
	containerRunning func(userID, mode string) bool
	verifyCredFn     func(exchange, apiKey, apiSecret, passphrase string, isTestnet bool) VerifyResult
}

func NewAPI(cfg *Config, strategies *StrategyStore, cm *ContainerManager, proxy *BotProxy, creds *CredentialStore, enc *Encryptor, syncer *Syncer, hub *MarketDataHub, testnetHub *MarketDataHub, notifier *Notifier, btExec *BacktestExecutor, btJobs *BacktestJobStore) *API {
	ctx, cancel := context.WithCancel(context.Background())
	return &API{
		cfg:           cfg,
		strategies:    strategies,
		container:     cm,
		proxy:         proxy,
		creds:         creds,
		encryptor:     enc,
		syncer:        syncer,
		hub:           hub,
		testnetHub:    testnetHub,
		notifier:      notifier,
		wsTickets:     NewWSTicketStore(),
		btExec:        btExec,
		btJobs:        btJobs,
		btSyncSem:     make(chan struct{}, 2),
		newBBGoClient: NewBBGoClient,
		stopCtx:       ctx,
		stopCancel:    cancel,
	}
}

func (api *API) Close() {
	if api.stopCancel != nil {
		api.stopCancel()
	}
	if api.wsTickets != nil {
		api.wsTickets.Close()
	}
}

func (api *API) hubForMode(mode string) *MarketDataHub {
	if mode == ModePaper && api.testnetHub != nil {
		return api.testnetHub
	}
	return api.hub
}

func (api *API) RegisterRoutes(r chi.Router) {

	// WS routes — no request timeout (long-lived connections)
	r.Get("/api/ws/ticket", api.IssueWSTicket)
	r.Get("/api/ws", api.HandleWebSocket)

	// Long-running operations — no request timeout
	r.Post("/api/backtest/sync", api.SyncBacktestData)

	// Standard routes — 60s request timeout
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * time.Second))

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
		})

		r.Get("/api/users/{userID}/bots", api.ListBots)
		r.Get("/api/users/{userID}/bots/{botID}", api.GetBot)
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

		r.HandleFunc("/api/bbgo/{userID}/*", api.ProxyToBot)

		r.Get("/api/backtest/jobs", api.ListBacktestJobs)
		r.Get("/api/backtest/jobs/{jobID}", api.GetBacktestJob)
		r.Get("/api/backtest/status", api.BacktestSyncStatus)
		r.Get("/api/credentials", api.ListCredentials)
	})
}

func (api *API) Health(w http.ResponseWriter, _ *http.Request) {
	users := api.strategies.ScanUsers()
	running := 0
	for _, um := range users {
		if api.isContainerRunning(um.UserID, um.Mode) {
			running++
		}
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Users:   len(users),
		Running: running,
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

func modeFromQuery(r *http.Request) string {
	m := r.URL.Query().Get("mode")
	if m == ModePaper {
		return ModePaper
	}
	return ModeLive
}

func containerKey(userID, mode string) string {
	return userID + ":" + mode
}

func (api *API) isContainerStarting(userID, mode string) bool {
	_, ok := api.starting.Load(containerKey(userID, mode))
	return ok
}

func (api *API) isContainerRunning(userID, mode string) bool {
	if api.containerRunning != nil {
		return api.containerRunning(userID, mode)
	}
	return api.container.IsRunning(userID, mode)
}

func (api *API) startContainer(userID, mode string) error {
	if api.containerStart != nil {
		return api.containerStart(userID, mode)
	}
	if api.container == nil {
		return fmt.Errorf("container manager not configured")
	}
	return api.container.CreateAndStart(userID, mode)
}

func (api *API) stopContainer(userID, mode string) {
	if api.containerStop != nil {
		api.containerStop(userID, mode)
		return
	}
	api.container.Stop(userID, mode)
}

// containerStatusForMode returns the status info for a single mode.
// When running, queries bbgo API for strategy details.
// When stopped, returns status only (no strategy data).
func (api *API) containerStatusForMode(userID, mode string) *containerInfo {
	if !api.strategies.YAMLExists(userID, mode) {
		return nil
	}

	info := &containerInfo{
		UserID: userID,
		Mode:   mode,
		Status: StatusStopped,
	}

	if api.isContainerStarting(userID, mode) {
		info.Status = StatusStarting
		return info
	}

	if !api.isContainerRunning(userID, mode) {
		return info
	}

	info.Status = StatusRunning
	client := api.newBBGoClient(api.container.APIURL(userID, mode)).WithContext(context.Background())
	strategies, err := client.GetStrategies()
	if err != nil {
		log.Printf("list strategies from bbgo for %s (%s): %v", userID, mode, err)
		info.Strategies = []BBGoStrategyState{}
	} else {
		info.Strategies = strategies
	}
	return info
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
	if req.Mode == ModePaper {
		if !req.CrossExchange && req.Exchange != "binance" {
			writeError(w, http.StatusBadRequest, "paper mode only supports Binance exchange")
			return
		}
		if req.CrossExchange {
			for _, sr := range req.Sessions {
				if sr.Exchange != paperExchange {
					writeError(w, http.StatusBadRequest, "paper mode only supports Binance exchange for all sessions")
					return
				}
			}
		}
	}

	if api.creds != nil {
		wantTestnet := req.Mode == ModePaper
		for _, ex := range collectExchanges([]StrategyEntry{{
			Exchange: req.Exchange, CrossExchange: req.CrossExchange, Sessions: req.Sessions,
		}}) {
			if _, _, _, err := api.creds.GetDecryptedByMode(userID, ex, wantTestnet); err != nil {
				writeError(w, http.StatusBadRequest, credModeError(req.Mode, ex))
				return
			}
		}
	}

	entry := StrategyEntry{
		Name:          req.Name,
		Exchange:      req.Exchange,
		Strategy:      req.Strategy,
		Config:        req.Config,
		Mode:          req.Mode,
		CrossExchange: req.CrossExchange,
		Sessions:      req.Sessions,
	}

	hasCredFn := func(exchange string) bool {
		if api.creds == nil {
			return false
		}
		_, err := api.creds.GetByMode(userID, exchange, req.Mode == ModePaper)
		return err == nil
	}

	if err := api.strategies.AddStrategy(userID, req.Mode, entry, hasCredFn); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if api.isContainerRunning(userID, req.Mode) {
		if _, loaded := api.starting.LoadOrStore(containerKey(userID, req.Mode), true); !loaded {
			go api.startUserContainer(userID, req.Mode)
		}
	}

	writeJSON(w, http.StatusCreated, strategyCreatedResponse{
		Status: "created",
		UserID: userID,
		Mode:   req.Mode,
	})
}

func (api *API) ListStrategies(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	byMode := make(map[string]*containerInfo)
	for _, mode := range []string{ModeLive, ModePaper} {
		if info := api.containerStatusForMode(userID, mode); info != nil {
			byMode[mode] = info
		}
	}
	writeJSON(w, http.StatusOK, containerStatusResponse{
		UserID:     userID,
		Containers: byMode,
	})
}

// DeleteStrategy removes a strategy by bbgo strategyInstanceID.
// Only works when the container is running (we query bbgo API to resolve the ID).
func (api *API) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	strategyID := chi.URLParam(r, "strategyID")
	mode := modeFromQuery(r)

	if !api.isContainerRunning(userID, mode) {
		writeError(w, http.StatusBadRequest, "container is not running — start it first to delete strategies")
		return
	}

	// Query bbgo API to find the strategy by instanceID and get its type+symbol
	client := api.newBBGoClient(api.container.APIURL(userID, mode)).WithContext(r.Context())
	bbgoStrategies, err := client.GetStrategies()
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to query strategies: %v", err))
		return
	}

	var target BBGoStrategyState
	for _, s := range bbgoStrategies {
		if id, _ := s["strategyInstanceID"].(string); id == strategyID {
			target = s
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "strategy not found")
		return
	}

	strategy, _ := target["strategy"].(string)
	symbol, _ := target["symbol"].(string)

	found, err := api.strategies.RemoveStrategy(userID, mode, strategy, symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "strategy not found in bbgo.yaml")
		return
	}

	// If no strategies left, stop the container
	strategies, _ := api.strategies.ListStrategies(userID, mode)
	if len(strategies) == 0 {
		api.stopContainer(userID, mode)
		writeJSON(w, http.StatusOK, statusStopped{Status: "stopped", Reason: "no strategies left"})
		return
	}

	if _, loaded := api.starting.LoadOrStore(containerKey(userID, mode), true); !loaded {
		go api.startUserContainer(userID, mode)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "restarting",
		"user_id": userID,
		"mode":    mode,
	})
}

func (api *API) StartUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	mode := modeFromQuery(r)

	if !api.strategies.YAMLExists(userID, mode) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("no strategies configured for %s mode", mode))
		return
	}

	// Validate credentials
	strategies, _ := api.strategies.ListStrategies(userID, mode)
	if api.creds != nil {
		wantTestnet := mode == ModePaper
		for _, ex := range collectExchanges(strategies) {
			if mode == ModePaper && ex != paperExchange {
				continue
			}
			cred, err := api.creds.GetByMode(userID, ex, wantTestnet)
			if err != nil {
				writeError(w, http.StatusBadRequest, credModeError(mode, ex))
				return
			}
			if !cred.IsVerified && exchangeHasVerifier(ex) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("API key for %s (%s) is not verified — verification failed or has not been attempted", ex, modeLabel(wantTestnet)))
				return
			}
		}
	}

	if api.isContainerRunning(userID, mode) {
		writeJSON(w, http.StatusOK, api.containerStatusForMode(userID, mode))
		return
	}

	if api.isContainerStarting(userID, mode) {
		writeJSON(w, http.StatusAccepted, api.containerStatusForMode(userID, mode))
		return
	}

	// Return starting status immediately, start async
	info := api.containerStatusForMode(userID, mode)
	if info != nil {
		info.Status = StatusStarting
	} else {
		info = &containerInfo{UserID: userID, Mode: mode, Status: StatusStarting}
	}
	writeJSON(w, http.StatusAccepted, info)

	if _, loaded := api.starting.LoadOrStore(containerKey(userID, mode), true); !loaded {
		go api.startUserContainer(userID, mode)
	}
}

func (api *API) startUserContainer(userID, mode string) {
	key := containerKey(userID, mode)
	defer api.starting.Delete(key)

	if err := api.startContainer(userID, mode); err != nil {
		log.Printf("start %s container for user %s failed: %v", mode, userID, err)
		return
	}

	ctx, cancel := context.WithTimeout(api.stopCtx, 30*time.Second)
	defer cancel()

	baseURL := ""
	if api.container != nil {
		baseURL = api.container.APIURL(userID, mode)
	}
	if baseURL == "" {
		log.Printf("user %s %s: cannot determine container URL", userID, mode)
		return
	}
	if api.newBBGoClient == nil {
		log.Printf("user %s %s: bbgo client factory not configured", userID, mode)
		return
	}
	client := api.newBBGoClient(baseURL)
	var reachable bool
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
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
		log.Printf("user %s %s container started but health check failed", userID, mode)
		return
	}

	log.Printf("user %s %s container started and healthy", userID, mode)

	if api.syncer != nil {
		strategies, _ := api.strategies.ListStrategies(userID, mode)
		go api.syncer.MarkCredentialsVerified(userID, mode, strategies)
	}

	go func() {
		grpcAddr := api.container.ContainerGRPCAddr(userID, mode)
		for i := 0; i < 10; i++ {
			select {
			case <-api.stopCtx.Done():
				return
			default:
			}
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "user_id": userID, "mode": mode})
}

func (api *API) UserStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	byMode := make(map[string]*containerInfo)
	for _, mode := range []string{ModeLive, ModePaper} {
		if info := api.containerStatusForMode(userID, mode); info != nil {
			byMode[mode] = info
		}
	}
	writeJSON(w, http.StatusOK, containerStatusResponse{UserID: userID, Containers: byMode})
}

// userFromURL extracts userID and mode from URL parameters, validates access.
func (api *API) userFromURL(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	userID := chi.URLParam(r, "userID")
	if !isValidUUID(userID) {
		writeError(w, http.StatusBadRequest, "invalid user ID format")
		return "", "", false
	}
	if headerID, ok := userIDFromRequest(r); ok && headerID != userID {
		writeError(w, http.StatusForbidden, "user ID mismatch")
		return "", "", false
	}
	mode := modeFromQuery(r)
	return userID, mode, true
}

func (api *API) bbgoClientForUser(w http.ResponseWriter, r *http.Request) (*BBGoClient, string, bool) {
	userID, mode, ok := api.userFromURL(w, r)
	if !ok {
		return nil, "", false
	}

	if !api.isContainerRunning(userID, mode) {
		writeError(w, http.StatusServiceUnavailable, "container is not running")
		return nil, "", false
	}

	return api.newBBGoClient(api.container.APIURL(userID, mode)).WithContext(r.Context()), userID, true
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
	if !api.strategies.YAMLExists(userID, mode) {
		writeError(w, http.StatusNotFound, "user container not found")
		return
	}

	api.proxy.ProxyToBot(w, r, userID, mode)
}

func (api *API) ContainerLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}

	mode := modeFromQuery(r)
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}

	logs, err := api.container.Logs(userID, mode, tail)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logsResponse{Logs: logs})
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
	writeJSON(w, http.StatusOK, sessionsResponse{Sessions: sessions})
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
	writeJSON(w, http.StatusOK, sessionDetailResponse{Session: *detail})
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
	writeJSON(w, http.StatusOK, tradesResponse{Trades: trades})
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
	writeJSON(w, http.StatusOK, ordersResponse{Orders: orders})
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
	writeJSON(w, http.StatusOK, accountResponse{Account: account})
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
	writeJSON(w, http.StatusOK, balancesResponse{Balances: balances})
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
	writeJSON(w, http.StatusOK, symbolsResponse{Symbols: filterTradingPairs(symbols)})
}

func (api *API) MarketSymbols(w http.ResponseWriter, r *http.Request) {
	exchange := chi.URLParam(r, "exchange")
	if exchange == "" {
		writeError(w, http.StatusBadRequest, "exchange is required")
		return
	}
	mode := r.URL.Query().Get("mode")
	var restAddr string
	if mode == ModePaper && api.cfg.MarketDataTestnetRESTAddr != "" {
		restAddr = api.cfg.MarketDataTestnetRESTAddr
	} else {
		restAddr = api.cfg.MarketDataRESTAddr
	}
	base := "http://" + restAddr
	client := api.newBBGoClient(base)
	symbols, err := client.GetSessionSymbols(exchange)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("marketdata symbols: %s", err))
		return
	}
	writeJSON(w, http.StatusOK, symbolsResponse{Symbols: filterTradingPairs(symbols)})
}

func (api *API) MarketTicker(w http.ResponseWriter, r *http.Request) {
	exchange := chi.URLParam(r, "exchange")
	symbol := r.URL.Query().Get("symbol")
	if exchange == "" || symbol == "" {
		writeError(w, http.StatusBadRequest, "exchange and symbol are required")
		return
	}
	mode := r.URL.Query().Get("mode")
	hub := api.hubForMode(mode)
	if hub == nil || hub.conn == nil {
		writeError(w, http.StatusServiceUnavailable, "market data service not connected")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	client := pb.NewMarketDataQueryClient(hub.conn)
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
	writeJSON(w, http.StatusOK, tickerResponse{Ticker: tickerData{
		Symbol: resp.Ticker.Symbol,
		Open:   resp.Ticker.Open,
		High:   resp.Ticker.High,
		Low:    resp.Ticker.Low,
		Close:  resp.Ticker.Close,
		Volume: resp.Ticker.Volume,
	}})
}

func (api *API) MarketKlines(w http.ResponseWriter, r *http.Request) {
	exchange := chi.URLParam(r, "exchange")
	symbol := r.URL.Query().Get("symbol")
	if exchange == "" || symbol == "" {
		writeError(w, http.StatusBadRequest, "exchange and symbol are required")
		return
	}
	// Klines are public market data — always use the live hub regardless of mode.
	// Testnet exchanges have very limited historical data.
	hub := api.hub
	if hub == nil || hub.conn == nil {
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

	client := pb.NewMarketDataQueryClient(hub.conn)
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

	klines := make([]klineEntry, 0, len(resp.Klines))
	for _, k := range resp.Klines {
		klines = append(klines, klineEntry{
			Time:        k.StartTime,
			Open:        k.Open,
			High:        k.High,
			Low:         k.Low,
			Close:       k.Close,
			Volume:      k.Volume,
			QuoteVolume: k.QuoteVolume,
			Closed:      k.Closed,
		})
	}
	writeJSON(w, http.StatusOK, klinesResponse{Klines: klines})
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
	writeJSON(w, http.StatusOK, assetsResponse{Assets: assets})
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
	writeJSON(w, http.StatusOK, bbgoStrategiesResponse{Strategies: strategies})
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
		session := exchange
		if session == "" {
			if sessions, serr := client.GetSessions(); serr == nil && len(sessions) > 0 {
				session = sessions[0].Name
			}
		}
		if session != "" {
			if st, serr := client.GetSessionTrades(session); serr == nil {
				trades = st
				err = nil
			}
		}
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tradesResponse{Trades: trades})
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
	writeJSON(w, http.StatusOK, ordersResponse{Orders: orders})
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
	writeJSON(w, http.StatusOK, tradingVolumeResponse{TradingVolumes: volumes})
}

func (api *API) BBGoPnL(w http.ResponseWriter, r *http.Request) {
	userID, mode, ok := api.userFromURL(w, r)
	if !ok {
		return
	}

	// Paper mode stores data in container SQLite, not Supabase
	if api.syncer != nil && mode != ModePaper {
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

	if !api.isContainerRunning(userID, mode) {
		writeError(w, http.StatusServiceUnavailable, "container is not running")
		return
	}
	if api.container == nil {
		writeError(w, http.StatusInternalServerError, "container manager not available")
		return
	}
	client := api.newBBGoClient(api.container.APIURL(userID, mode)).WithContext(r.Context())

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
		session := exchange
		if session == "" {
			if sessions, serr := client.GetSessions(); serr == nil && len(sessions) > 0 {
				session = sessions[0].Name
			}
		}
		if session != "" {
			if st, serr := client.GetSessionTrades(session); serr == nil {
				trades = st
				err = nil
			}
		}
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	report := calculatePnL(trades)
	writeJSON(w, http.StatusOK, report)
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

	writeJSON(w, http.StatusOK, backtestResultResponse{Output: string(result)})
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

	log.Printf("backtest sync requested by user %s: %s %v %s-%s", userID, req.Exchange, req.Symbols, req.StartTime, req.EndTime)
	results := make([]syncResult, len(req.Symbols))
	var wg sync.WaitGroup
	for i, sym := range req.Symbols {
		wg.Add(1)
		go func(idx int, sym string) {
			defer wg.Done()
			out, err := api.container.SyncBacktest(userID, req.Exchange, sym, req.StartTime, req.EndTime)
			r := syncResult{Symbol: sym, Output: out}
			if err != nil {
				r.Error = err.Error()
			}
			results[idx] = r
		}(i, sym)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, backtestSyncResponse{Exchange: req.Exchange, Synced: results})
}

func (api *API) BacktestSyncStatus(w http.ResponseWriter, _ *http.Request) {
	dbPath := api.container.cfg.BacktestSharedDir
	if dbPath == "" {
		dbPath = api.container.cfg.DataDir + "/backtest-shared"
	}
	dbPath += "/backtest.db"
	info, err := os.Stat(dbPath)
	if err != nil {
		writeJSON(w, http.StatusOK, backtestSyncStatusResponse{Available: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, backtestSyncStatusResponse{Available: true, Size: info.Size(), Modified: info.ModTime().Format(time.RFC3339)})
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

	verifyFn := api.verifyCredFn
	if verifyFn == nil {
		verifyFn = verifyCredential
	}
	result := verifyFn(req.Exchange, req.APIKey, req.APISecret, req.Passphrase, req.IsTestnet)
	cred.IsVerified = result.Verified

	if err := api.creds.Upsert(cred); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if api.syncer != nil {
		go api.syncer.SyncCredential(cred)
	}

	// Restart container if running and credentials verified
	if cred.IsVerified {
		mode := ModeLive
		if req.IsTestnet {
			mode = ModePaper
		}
		if api.isContainerRunning(userID, mode) {
			if _, loaded := api.starting.LoadOrStore(containerKey(userID, mode), true); !loaded {
				go api.startUserContainer(userID, mode)
			}
		}
	}

	resp := credentialResponse{
		ID:         cred.ID,
		UserID:     cred.UserID,
		Exchange:   cred.Exchange,
		IsTestnet:  cred.IsTestnet,
		IsVerified: cred.IsVerified,
	}
	if !result.Verified {
		resp.VerifyError = result.Error
	}
	writeJSON(w, http.StatusCreated, resp)
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
	safe := make([]credentialResponse, len(creds))
	for i, c := range creds {
		safe[i] = credentialResponse{
			ID:         c.ID,
			UserID:     c.UserID,
			Exchange:   c.Exchange,
			IsTestnet:  c.IsTestnet,
			IsVerified: c.IsVerified,
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
			go api.syncer.DeleteCredential(userID, exchange, isTestnet)
		}
	}

	mode := ModeLive
	if isTestnet {
		mode = ModePaper
	}
	if api.isContainerRunning(userID, mode) {
		if _, loaded := api.starting.LoadOrStore(containerKey(userID, mode), true); !loaded {
			go api.startUserContainer(userID, mode)
		}

	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
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

	writeJSON(w, http.StatusCreated, notifConfigResponse{ID: ch.ID, Type: ch.Type, Enabled: ch.Enabled, Rules: cfg.Rules})
}

func (api *API) ListNotificationConfigs(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	configs := api.notifier.List(userID)
	safe := make([]notifConfigResponse, len(configs))
	for i, c := range configs {
		safe[i] = notifConfigResponse{
			ID:      c.Channel.ID,
			Type:    c.Channel.Type,
			Enabled: c.Channel.Enabled,
			Rules:   c.Rules,
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
		var sym struct{ Symbol string `json:"symbol"` }
		if err := json.Unmarshal(req.Config, &sym); err == nil && sym.Symbol != "" {
			req.Symbol = sym.Symbol
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
	writeJSON(w, http.StatusAccepted, backtestSubmitResponse{JobID: job.ID, Status: status, NeedSync: needSync})
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
	writeJSON(w, http.StatusOK, backtestJobsResponse{Jobs: jobs})
}

func (api *API) hasDataForRange(exchange, symbol, startTime, endTime string) bool {
	return false
}

func credModeError(mode, ex string) string {
	if mode == ModeLive {
		return fmt.Sprintf("live mode requires API credentials for %s — add them in Settings first", ex)
	}
	return "paper mode requires Binance testnet API credentials — add them in Settings first"
}

func collectExchanges(strategies []StrategyEntry) []string {
	seen := map[string]bool{}
	var result []string
	for _, s := range strategies {
		if s.CrossExchange {
			for _, sr := range s.Sessions {
				if !seen[sr.Exchange] {
					seen[sr.Exchange] = true
					result = append(result, sr.Exchange)
				}
			}
		} else if s.Exchange != "" && !seen[s.Exchange] {
			seen[s.Exchange] = true
			result = append(result, s.Exchange)
		}
	}
	return result
}
