package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	store      *InstanceStore
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
	storage    *StorageClient

	stopCtx    context.Context
	stopCancel context.CancelFunc
	starting   sync.Map // key: instanceID

	newBBGoClient func(baseURL string) *BBGoClient
	verifyCredFn  func(exchange, apiKey, apiSecret, passphrase string, isTestnet bool) VerifyResult
	queryKlinesFn func(ctx context.Context, req *pb.QueryKLinesRequest) (*pb.QueryKLinesResponse, error)
	queryTickerFn func(ctx context.Context, req *pb.QueryTickerRequest) (*pb.QueryTickerResponse, error)
}

func NewAPI(cfg *Config, store *InstanceStore, cm *ContainerManager, proxy *BotProxy, creds *CredentialStore, enc *Encryptor, syncer *Syncer, hub *MarketDataHub, testnetHub *MarketDataHub, notifier *Notifier, btExec *BacktestExecutor, btJobs *BacktestJobStore, storage *StorageClient) *API {
	ctx, cancel := context.WithCancel(context.Background())
	return &API{
		cfg: cfg, store: store, container: cm, proxy: proxy,
		creds: creds, encryptor: enc, syncer: syncer,
		hub: hub, testnetHub: testnetHub, notifier: notifier,
		wsTickets: NewWSTicketStore(), btExec: btExec, btJobs: btJobs,
		btSyncSem: make(chan struct{}, 2), storage: storage,
		newBBGoClient: NewBBGoClient,
		stopCtx: ctx, stopCancel: cancel,
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
	return api.hub
}

func (api *API) RegisterRoutes(r chi.Router) {
	r.Get("/api/ws/ticket", api.IssueWSTicket)
	r.Get("/api/ws", api.HandleWebSocket)
	r.Post("/api/backtest/sync", api.SyncBacktestData)

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
			r.Delete("/api/users/{userID}/strategies", api.ClearAllStrategies)
			r.Post("/api/users/{userID}/start", api.StartUser)
			r.Post("/api/users/{userID}/stop", api.StopUser)
			r.Post("/api/credentials", api.CreateCredential)
			r.Delete("/api/credentials/{id}", api.DeleteCredential)
			r.Post("/api/notifications/config", api.CreateNotificationConfig)
			r.Delete("/api/notifications/config/{id}", api.DeleteNotificationConfig)
			r.Post("/api/notifications/test", api.TestNotification)
			r.Post("/api/backtest", api.RunBacktest)
			r.Post("/api/backtest/submit", api.SubmitBacktest)
			r.Post("/api/users/{userID}/instances/{instanceID}/start", api.StartInstance)
			r.Post("/api/users/{userID}/instances/{instanceID}/stop", api.StopInstance)
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
		r.Get("/api/users/{userID}/bbgo/trades/markers", api.BBGoTradeMarkers)
		r.Get("/api/users/{userID}/bbgo/orders/closed", api.BBGoClosedOrders)
		r.Get("/api/users/{userID}/bbgo/trading-volume", api.BBGoTradingVolume)
		r.Get("/api/users/{userID}/bbgo/pnl", api.BBGoPnL)
		r.Get("/api/users/{userID}/logs", api.ContainerLogs)
		r.Get("/api/notifications/config", api.ListNotificationConfigs)
		r.HandleFunc("/api/bbgo/{userID}/*", api.ProxyToBot)
		r.Get("/api/backtest/jobs", api.ListBacktestJobs)
		r.Get("/api/backtest/jobs/{jobID}", api.GetBacktestJob)
		r.Get("/api/backtest/jobs/{jobID}/download", api.DownloadBacktestReport)
		r.Get("/api/backtest/status", api.BacktestSyncStatus)
		r.Get("/api/credentials", api.ListCredentials)
	})
}

// --- Helpers ---

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

func (api *API) isInstanceStarting(instanceID string) bool {
	_, ok := api.starting.Load(instanceID)
	return ok
}

func (api *API) isInstanceRunning(userID, mode, instanceID string) bool {
	return api.container.IsInstanceRunning(userID, mode, instanceID)
}

func (api *API) resolveInstanceForRequest(w http.ResponseWriter, r *http.Request) (*StrategyInstance, bool) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return nil, false
	}
	mode := modeFromQuery(r)

	if instanceID := r.URL.Query().Get("instanceID"); instanceID != "" {
		inst, err := api.store.GetInstance(userID, mode, instanceID)
		if err != nil {
			writeError(w, http.StatusNotFound, "instance not found")
			return nil, false
		}
		return inst, true
	}

	instances, err := api.store.ListInstances(userID, mode)
	if err != nil || len(instances) == 0 {
		writeError(w, http.StatusNotFound, "no instances found")
		return nil, false
	}

	if len(instances) == 1 {
		return &instances[0], true
	}

	writeError(w, http.StatusBadRequest, "multiple instances found — specify instanceID parameter")
	return nil, false
}

func (api *API) bbgoClientForInstance(inst *StrategyInstance, ctx context.Context) *BBGoClient {
	return api.newBBGoClient(api.container.InstanceAPIURL(inst.UserID, inst.Mode, inst.InstanceID)).WithContext(ctx)
}

// --- Health ---

func (api *API) Health(w http.ResponseWriter, _ *http.Request) {
	users := api.store.ScanUsers()
	running := 0
	for _, um := range users {
		instances, _ := api.store.ListInstances(um.UserID, um.Mode)
		for i := range instances {
			if api.isInstanceRunning(instances[i].UserID, instances[i].Mode, instances[i].InstanceID) {
				running++
			}
		}
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Users: len(users), Running: running})
}

// --- Strategy CRUD ---

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
		FuturesConfig *FuturesConfig      `json:"futuresConfig,omitempty"`
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
	if req.Mode == ModePaper && api.store.IsLiveOnly(normalizedStrategy) {
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

	if api.creds != nil && req.Mode != ModePaper {
		exchanges := []string{req.Exchange}
		if req.CrossExchange {
			exchanges = nil
			for _, sr := range req.Sessions {
				exchanges = append(exchanges, sr.Exchange)
			}
		}
		for _, ex := range exchanges {
			if _, _, _, err := api.creds.GetDecryptedByMode(userID, ex, false); err != nil {
				writeError(w, http.StatusBadRequest, credModeError(req.Mode, ex))
				return
			}
		}
	}

	inst := &StrategyInstance{
		UserID: userID, Mode: req.Mode, Strategy: normalizedStrategy,
		Exchange: req.Exchange, Config: req.Config, Name: req.Name,
		CrossExchange: req.CrossExchange, Sessions: req.Sessions,
		FuturesConfig: req.FuturesConfig,
	}

	symbol := extractSymbolFromConfig(req.Config)
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	inst.Symbol = symbol
	inst.InstanceID = computeInstanceID(inst.Strategy, inst.Symbol, inst.Config)

	hasCredFn := func(exchange string) bool {
		if api.creds == nil || req.Mode == ModePaper {
			return false
		}
		_, err := api.creds.GetByMode(userID, exchange, false)
		return err == nil
	}

	if err := api.store.CreateInstance(inst, hasCredFn); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if _, loaded := api.starting.LoadOrStore(inst.InstanceID, true); !loaded {
		go api.startInstanceContainer(inst)
	}

	writeJSON(w, http.StatusCreated, instanceInfo{
		InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
		Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
		Name: inst.Name, Status: StatusStarting,
	})
}

func (api *API) ListStrategies(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	instances := make([]instanceInfo, 0)
	for _, mode := range []string{ModeLive, ModePaper} {
		insts, _ := api.store.ListInstances(userID, mode)
		for _, inst := range insts {
			info := instanceInfo{
				InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
				Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
				Name: inst.Name, Status: StatusStopped,
			}
			if api.isInstanceStarting(inst.InstanceID) {
				info.Status = StatusStarting
			} else if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
				info.Status = StatusRunning
			}
			instances = append(instances, info)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "instances": instances})
}

func (api *API) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	instanceID := chi.URLParam(r, "strategyID")
	mode := modeFromQuery(r)

	inst, err := api.store.GetInstance(userID, mode, instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
		api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID)
	}

	if err := api.store.RemoveInstance(inst.UserID, inst.Mode, inst.InstanceID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "deleted", "instance_id": inst.InstanceID,
		"user_id": userID, "mode": mode,
	})
}

func (api *API) ClearAllStrategies(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	mode := modeFromQuery(r)
	instances, _ := api.store.ListInstances(userID, mode)
	for _, inst := range instances {
		api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID)
		api.store.RemoveInstance(inst.UserID, inst.Mode, inst.InstanceID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// --- Instance start/stop ---

func (api *API) StartInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	instanceID := chi.URLParam(r, "instanceID")
	inst, err := api.store.GetInstance(userID, "", instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
		writeJSON(w, http.StatusOK, instanceInfo{
			InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
			Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
			Name: inst.Name, Status: StatusRunning,
		})
		return
	}
	if api.isInstanceStarting(inst.InstanceID) {
		writeJSON(w, http.StatusAccepted, instanceInfo{
			InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
			Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
			Name: inst.Name, Status: StatusStarting,
		})
		return
	}
	writeJSON(w, http.StatusAccepted, instanceInfo{
		InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
		Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
		Name: inst.Name, Status: StatusStarting,
	})
	if _, loaded := api.starting.LoadOrStore(inst.InstanceID, true); !loaded {
		go api.startInstanceContainer(inst)
	}
}

func (api *API) StopInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	instanceID := chi.URLParam(r, "instanceID")
	inst, err := api.store.GetInstance(userID, "", instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID)
	api.starting.Delete(inst.InstanceID)
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "stopped", "instance_id": inst.InstanceID,
		"user_id": userID, "mode": inst.Mode,
	})
}

func (api *API) StartUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	mode := modeFromQuery(r)
	instances, err := api.store.ListInstances(userID, mode)
	if err != nil || len(instances) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("no strategies configured for %s mode", mode))
		return
	}

	if api.creds != nil && mode != ModePaper {
		for _, inst := range instances {
			exchanges := []string{inst.Exchange}
			if inst.CrossExchange {
				exchanges = nil
				for _, sr := range inst.Sessions {
					exchanges = append(exchanges, sr.Exchange)
				}
			}
			for _, ex := range exchanges {
				cred, err := api.creds.GetByMode(userID, ex, false)
				if err != nil {
					writeError(w, http.StatusBadRequest, credModeError(mode, ex))
					return
				}
				if !cred.IsVerified && exchangeHasVerifier(ex) {
					writeError(w, http.StatusBadRequest, fmt.Sprintf("API key for %s is not verified", ex))
					return
				}
			}
		}
	}

	var infos []instanceInfo
	for i := range instances {
		inst := &instances[i]
		info := instanceInfo{
			InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
			Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange, Name: inst.Name,
		}
		if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
			info.Status = StatusRunning
		} else if api.isInstanceStarting(inst.InstanceID) {
			info.Status = StatusStarting
		} else {
			info.Status = StatusStarting
			if _, loaded := api.starting.LoadOrStore(inst.InstanceID, true); !loaded {
				go api.startInstanceContainer(inst)
			}
		}
		infos = append(infos, info)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"user_id": userID, "mode": mode, "instances": infos})
}

func (api *API) StopUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	mode := modeFromQuery(r)
	instances, _ := api.store.ListInstances(userID, mode)
	for _, inst := range instances {
		api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID)
		api.starting.Delete(inst.InstanceID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "user_id": userID, "mode": mode})
}

func (api *API) UserStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	var instances []instanceInfo
	for _, mode := range []string{ModeLive, ModePaper} {
		insts, _ := api.store.ListInstances(userID, mode)
		for _, inst := range insts {
			info := instanceInfo{
				InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
				Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
				Name: inst.Name, Status: StatusStopped,
			}
			if api.isInstanceStarting(inst.InstanceID) {
				info.Status = StatusStarting
			} else if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
				info.Status = StatusRunning
			}
			instances = append(instances, info)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "instances": instances})
}

func (api *API) startInstanceContainer(inst *StrategyInstance) {
	defer api.starting.Delete(inst.InstanceID)

	if err := api.container.CreateAndStartInstance(inst); err != nil {
		log.Printf("start instance %s for user %s failed: %v", inst.InstanceID, inst.UserID, err)
		return
	}

	ctx, cancel := context.WithTimeout(api.stopCtx, 30*time.Second)
	defer cancel()

	baseURL := api.container.InstanceAPIURL(inst.UserID, inst.Mode, inst.InstanceID)
	client := api.newBBGoClient(baseURL)
	var reachable bool
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
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
		log.Printf("instance %s container started but health check failed", inst.InstanceID)
		return
	}
	log.Printf("instance %s container started and healthy", inst.InstanceID)

	if api.syncer != nil && inst.Mode != ModePaper {
		go api.syncer.MarkCredentialsVerified(inst.UserID, inst.Mode, []StrategyEntry{{
			Strategy: inst.Strategy, Exchange: inst.Exchange,
			CrossExchange: inst.CrossExchange, Sessions: inst.Sessions,
			FuturesConfig: inst.FuturesConfig,
		}})
	}

	go func() {
		grpcAddr := api.container.InstanceGRPCAddr(inst.UserID, inst.Mode, inst.InstanceID)
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
	}()
}

// --- Bot data endpoints ---

func (api *API) ProxyToBot(w http.ResponseWriter, r *http.Request) {
	inst, ok := api.resolveInstanceForRequest(w, r)
	if !ok {
		return
	}
	if !api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
		writeError(w, http.StatusNotFound, "instance container not running")
		return
	}
	api.proxy.ProxyToInstance(w, r, inst)
}

func (api *API) ContainerLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	mode := modeFromQuery(r)
	instanceID := r.URL.Query().Get("instanceID")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}

	if instanceID != "" {
		logs, err := api.container.InstanceLogs(userID, mode, instanceID, tail)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, logsResponse{Logs: logs})
		return
	}

	instances, _ := api.store.ListInstances(userID, mode)
	var allLogs []string
	for _, inst := range instances {
		logs, err := api.container.InstanceLogs(inst.UserID, inst.Mode, inst.InstanceID, tail)
		if err == nil && logs != "" {
			allLogs = append(allLogs, fmt.Sprintf("--- %s ---\n%s", inst.InstanceID, logs))
		}
	}
	writeJSON(w, http.StatusOK, logsResponse{Logs: strings.Join(allLogs, "\n")})
}

func (api *API) bbgoClientForUser(w http.ResponseWriter, r *http.Request) (*BBGoClient, string, bool) {
	inst, ok := api.resolveInstanceForRequest(w, r)
	if !ok {
		return nil, "", false
	}
	if !api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
		writeError(w, http.StatusServiceUnavailable, "instance container is not running")
		return nil, "", false
	}
	return api.bbgoClientForInstance(inst, r.Context()), inst.Mode, true
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
	client := api.newBBGoClient("http://" + api.cfg.MarketDataRESTAddr)
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
	hub := api.hubForMode(r.URL.Query().Get("mode"))
	if api.queryTickerFn == nil && (hub == nil || hub.conn == nil) {
		writeError(w, http.StatusServiceUnavailable, "market data service not connected")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	req := &pb.QueryTickerRequest{Exchange: exchange, Symbol: symbol}
	var err error
	var resp *pb.QueryTickerResponse
	if api.queryTickerFn != nil {
		resp, err = api.queryTickerFn(ctx, req)
	} else {
		resp, err = pb.NewMarketDataQueryClient(hub.conn).QueryTicker(ctx, req)
	}
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
		Symbol: resp.Ticker.Symbol, Open: resp.Ticker.Open, High: resp.Ticker.High,
		Low: resp.Ticker.Low, Close: resp.Ticker.Close, Volume: resp.Ticker.Volume,
	}})
}

func (api *API) MarketKlines(w http.ResponseWriter, r *http.Request) {
	exchange := chi.URLParam(r, "exchange")
	symbol := r.URL.Query().Get("symbol")
	if exchange == "" || symbol == "" {
		writeError(w, http.StatusBadRequest, "exchange and symbol are required")
		return
	}
	hub := api.hub
	if api.queryKlinesFn == nil && (hub == nil || hub.conn == nil) {
		writeError(w, http.StatusServiceUnavailable, "market data service not connected")
		return
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}
	limit := int64(500)
	if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64); err == nil && l > 0 && l <= 1500 {
		limit = l
	}
	var startTime, endTime int64
	if t, err := strconv.ParseInt(r.URL.Query().Get("start_time"), 10, 64); err == nil {
		startTime = t
	}
	if t, err := strconv.ParseInt(r.URL.Query().Get("end_time"), 10, 64); err == nil {
		endTime = t
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var err error
	req := &pb.QueryKLinesRequest{
		Exchange: exchange, Symbol: symbol, Interval: interval,
		StartTime: startTime, EndTime: endTime, Limit: limit,
	}
	var resp *pb.QueryKLinesResponse
	if api.queryKlinesFn != nil {
		resp, err = api.queryKlinesFn(ctx, req)
	} else {
		resp, err = pb.NewMarketDataQueryClient(hub.conn).QueryKLines(ctx, req)
	}
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
			Time: k.StartTime, Open: k.Open, High: k.High, Low: k.Low,
			Close: k.Close, Volume: k.Volume, QuoteVolume: k.QuoteVolume, Closed: k.Closed,
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
	client, mode, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	exchange := r.URL.Query().Get("exchange")
	symbol := r.URL.Query().Get("symbol")
	ordering := r.URL.Query().Get("ordering")
	if ordering == "" {
		ordering = "DESC"
	}
	var since, until *time.Time
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = &t
		}
	}
	if u := r.URL.Query().Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			until = &t
		}
	}
	limit := 500
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}
	if api.syncer != nil && mode != ModePaper {
		if trades, err := api.fetchSupabaseTrades(r, exchange, symbol, r.URL.Query().Get("strategy"), since, until, ordering, limit); err == nil && len(trades) > 0 {
			writeJSON(w, http.StatusOK, tradesResponse{Trades: trades})
			return
		}
	}
	trades, err := api.fetchContainerTrades(client, exchange, symbol, r.URL.Query().Get("strategy"), since, until, ordering, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tradesResponse{Trades: trades})
}

func (api *API) fetchContainerTrades(client *BBGoClient, exchange, symbol, strategy string, since, until *time.Time, ordering string, limit int) ([]BBGoTrade, error) {
	trades, err := client.GetTradesRange(exchange, symbol, strategy, since, until, limit, ordering)
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
		return nil, err
	}
	return computeTradesWithPositionTags(trades, client, exchange, symbol), nil
}

func (api *API) fetchSupabaseTrades(r *http.Request, exchange, symbol, strategy string, since, until *time.Time, ordering string, limit int) ([]BBGoTrade, error) {
	userID, _ := userIDFromRequest(r)
	allTrades, err := api.syncer.GetTradesForPnL(userID)
	if err != nil {
		return nil, err
	}
	var filtered []BBGoTrade
	for _, t := range allTrades {
		if exchange != "" && t.Exchange != exchange {
			continue
		}
		if symbol != "" && t.Symbol != symbol {
			continue
		}
		if strategy != "" && !strategyMatch(t.StrategyID, strategy) {
			continue
		}
		if since != nil && t.TradedAt < since.Format(time.RFC3339) {
			continue
		}
		if until != nil && t.TradedAt >= until.Format(time.RFC3339) {
			continue
		}
		filtered = append(filtered, t)
	}
	sortTradesASC(filtered)
	computePositionTags(filtered, 0)
	if ordering == "DESC" {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func computeTradesWithPositionTags(trades []BBGoTrade, client *BBGoClient, exchange, symbol string) []BBGoTrade {
	if len(trades) == 0 {
		return trades
	}
	sortTradesASC(trades)
	var initialNet float64
	if earliest := trades[0].TradedAt; earliest != "" {
		if t, err := time.Parse(time.RFC3339, earliest); err == nil {
			if summary, serr := client.GetTradePositionSummary(exchange, symbol, &t); serr == nil {
				initialNet = summary.NetPosition
			}
		}
	}
	computePositionTags(trades, initialNet)
	return trades
}

func (api *API) BBGoTradeMarkers(w http.ResponseWriter, r *http.Request) {
	client, mode, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}
	var since, until *time.Time
	if s := r.URL.Query().Get("since"); s != "" {
		t, _ := time.Parse(time.RFC3339, s)
		since = &t
	}
	if u := r.URL.Query().Get("until"); u != "" {
		t, _ := time.Parse(time.RFC3339, u)
		until = &t
	}
	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	var trades []BBGoTrade
	var err error
	exchange := r.URL.Query().Get("exchange")
	if api.syncer != nil && mode != ModePaper {
		trades, err = api.fetchSupabaseTrades(r, exchange, symbol, r.URL.Query().Get("strategy"), since, until, "ASC", limit)
	} else {
		trades, err = api.fetchContainerTrades(client, exchange, symbol, r.URL.Query().Get("strategy"), since, until, "ASC", limit)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	type tradeMarker struct {
		Time           int64   `json:"time"`
		Side           string  `json:"side"`
		Price          float64 `json:"price"`
		Quantity       float64 `json:"quantity"`
		PositionAction string  `json:"positionAction"`
	}
	markers := make([]tradeMarker, 0, len(trades))
	for _, t := range trades {
		if t.TradedAt == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, t.TradedAt)
		if err != nil {
			continue
		}
		markers = append(markers, tradeMarker{
			Time: parsed.Unix(), Side: t.Side, Price: parseFloat(t.Price),
			Quantity: parseFloat(t.Quantity), PositionAction: t.PositionAction,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"markers": markers})
}

func (api *API) BBGoClosedOrders(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	exchange := r.URL.Query().Get("exchange")
	symbol := r.URL.Query().Get("symbol")
	var lastGID int64
	if v, err := strconv.ParseInt(r.URL.Query().Get("gid"), 10, 64); err == nil {
		lastGID = v
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
	volumes, err := client.GetTradingVolume(r.URL.Query().Get("period"), r.URL.Query().Get("segment"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tradingVolumeResponse{TradingVolumes: volumes})
}

func (api *API) BBGoPnL(w http.ResponseWriter, r *http.Request) {
	inst, ok := api.resolveInstanceForRequest(w, r)
	if !ok {
		return
	}
	var trades []BBGoTrade
	if api.syncer != nil && inst.Mode != ModePaper {
		if supabaseTrades, err := api.syncer.GetTradesForPnL(inst.UserID); err == nil && len(supabaseTrades) > 0 {
			trades = supabaseTrades
		}
	}
	if len(trades) == 0 {
		if !api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
			writeError(w, http.StatusServiceUnavailable, "instance container is not running")
			return
		}
		client := api.bbgoClientForInstance(inst, r.Context())
		exchange := r.URL.Query().Get("exchange")
		symbol := r.URL.Query().Get("symbol")
		strategy := r.URL.Query().Get("strategy")
		var lastGID int64
		if v, err := strconv.ParseInt(r.URL.Query().Get("gid"), 10, 64); err == nil {
			lastGID = v
		}
		var err error
		trades, err = client.GetAllTradesFromWithStrategy(exchange, symbol, lastGID, strategy)
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
	}
	report := calculatePnL(trades)
	enrichUnrealizedPnl(&report, api.symbolPriceLookup(r.Context()))
	writeJSON(w, http.StatusOK, report)
}

func (api *API) symbolPriceLookup(ctx context.Context) func(symbol string) (float64, error) {
	return func(symbol string) (float64, error) {
		hub := api.hub
		if hub == nil || hub.conn == nil {
			return 0, fmt.Errorf("market data not available")
		}
		queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		resp, err := pb.NewMarketDataQueryClient(hub.conn).QueryTicker(queryCtx, &pb.QueryTickerRequest{Exchange: "binance", Symbol: symbol})
		if err != nil {
			return 0, err
		}
		if resp.Ticker == nil || resp.Ticker.Close <= 0 {
			return 0, fmt.Errorf("invalid price for %s", symbol)
		}
		return resp.Ticker.Close, nil
	}
}

// --- Backtest ---

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
	yamlContent, err := buildBacktestYAML(req.Strategy, req.Config, req.StartTime, req.EndTime, req.Exchange, "", api.store.Defaults())
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", err))
		return
	}
	jobID := generateID("bt")
	result, err := api.container.RunBacktest(userID, jobID, yamlContent)
	if err != nil {
		api.container.CleanupBacktest(userID, jobID)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.container.CleanupBacktest(userID, jobID)
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
		writeError(w, http.StatusTooManyRequests, "backtest sync already in progress")
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
		req.StartTime = time.Now().AddDate(0, -3, 0).Format("2006-01-02")
	}
	if req.EndTime == "" {
		req.EndTime = time.Now().Format("2006-01-02")
	}
	needSync := !api.hasDataForRange(req.Exchange, req.Symbol, req.StartTime, req.EndTime)
	job := &BacktestJob{
		ID: generateID("bt"), UserID: userID, Strategy: req.Strategy, Config: req.Config,
		Exchange: req.Exchange, Symbol: req.Symbol, StartTime: req.StartTime,
		EndTime: req.EndTime, NeedSync: needSync,
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
	summaries := make([]backtestJobSummary, len(jobs))
	for i, j := range jobs {
		summaries[i] = backtestJobSummary{
			ID: j.ID, UserID: j.UserID, Strategy: j.Strategy, Exchange: j.Exchange,
			Symbol: j.Symbol, StartTime: j.StartTime, EndTime: j.EndTime,
			Status: j.Status, Progress: j.Progress, Error: j.Error,
			CreatedAt: j.CreatedAt, StartedAt: j.StartedAt, CompletedAt: j.CompletedAt,
			NeedSync: j.NeedSync, HasReport: len(j.Report) > 0,
		}
	}
	writeJSON(w, http.StatusOK, backtestJobsResponse{Jobs: summaries})
}

func (api *API) DownloadBacktestReport(w http.ResponseWriter, r *http.Request) {
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
	if job.Status != JobCompleted {
		writeError(w, http.StatusBadRequest, "job not completed")
		return
	}
	if api.storage != nil && r.URL.Query().Get("signed") == "1" {
		file := r.URL.Query().Get("file")
		if file == "" {
			file = "summary.json"
		}
		if !allowedStorageFiles[file] {
			writeError(w, http.StatusBadRequest, "unsupported file type")
			return
		}
		signedURL, err := api.storage.CreateSignedURL(userID, jobID, file, 3600)
		if err != nil {
			if uploaded := api.uploadLocalToStorage(job, file); uploaded {
				signedURL, err = api.storage.CreateSignedURL(userID, jobID, file, 3600)
			}
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create download link")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"url": signedURL})
		return
	}
	api.downloadCSV(w, r, job)
}

var allowedStorageFiles = map[string]bool{
	"summary.json": true, "trades.tsv": true, "orders.tsv": true, "equity_curve.tsv": true,
}

func tsvToCsv(data []byte) []byte {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	out := make([]byte, 0, len(data))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		for i, f := range fields {
			if i > 0 {
				out = append(out, ',')
			}
			if strings.ContainsAny(f, ",\"\r\n") {
				out = append(out, '"')
				out = append(out, strings.ReplaceAll(f, `"`, `""`)...)
				out = append(out, '"')
			} else {
				out = append(out, f...)
			}
		}
		out = append(out, '\n')
	}
	return out
}

func writeCSV(w http.ResponseWriter, jobID, name string, data []byte) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="backtest-%s-%s.csv"`, jobID, name))
	w.Write(tsvToCsv(data))
}

func (api *API) downloadCSV(w http.ResponseWriter, r *http.Request, job *BacktestJob) {
	reportDir := api.container.BacktestReportDir(job.UserID, job.ID)
	if reportDir == "" {
		writeError(w, http.StatusInternalServerError, "cannot locate report files")
		return
	}
	file := r.URL.Query().Get("file")
	switch file {
	case "trades", "":
		data, err := os.ReadFile(filepath.Join(reportDir, "trades.tsv"))
		if err != nil {
			writeError(w, http.StatusNotFound, "trades file not found")
			return
		}
		writeCSV(w, job.ID, "trades", data)
	case "orders":
		data, err := os.ReadFile(filepath.Join(reportDir, "orders.tsv"))
		if err != nil {
			writeError(w, http.StatusNotFound, "orders file not found")
			return
		}
		writeCSV(w, job.ID, "orders", data)
	case "equity":
		if job.EquityCurve == "" {
			writeError(w, http.StatusNotFound, "equity curve not available")
			return
		}
		writeCSV(w, job.ID, "equity", []byte(job.EquityCurve))
	case "kline":
		entries, err := os.ReadDir(reportDir)
		if err != nil {
			writeError(w, http.StatusNotFound, "report directory not found")
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.Contains(name, job.Symbol) && strings.HasSuffix(name, ".tsv") &&
				name != "trades.tsv" && name != "orders.tsv" && name != "equity_curve.tsv" {
				data, err := os.ReadFile(filepath.Join(reportDir, name))
				if err != nil {
					continue
				}
				w.Header().Set("Content-Type", "text/tab-separated-values; charset=utf-8")
				w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="backtest-%s-%s"`, job.ID, name))
				w.Write(data)
				return
			}
		}
		writeError(w, http.StatusNotFound, "kline file not found")
	default:
		writeError(w, http.StatusBadRequest, "unsupported file type, use trades, orders, equity, or kline")
	}
}

func (api *API) uploadLocalToStorage(job *BacktestJob, filename string) bool {
	if !allowedStorageFiles[filename] {
		return false
	}
	reportDir := api.container.BacktestReportDir(job.UserID, job.ID)
	if reportDir == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(reportDir, filename))
	if err != nil {
		return false
	}
	if err := api.storage.Upload(job.UserID, job.ID, filename, data); err != nil {
		log.Printf("on-demand upload %s for job %s: %v", filename, job.ID, err)
		return false
	}
	return true
}

func (api *API) hasDataForRange(exchange, symbol, startTime, endTime string) bool {
	dbPath := api.container.cfg.BacktestSharedDir
	if dbPath == "" {
		dbPath = api.container.cfg.DataDir + "/backtest-shared"
	}
	dbPath += "/backtest.db"
	info, err := os.Stat(dbPath)
	if err != nil || info.Size() < 1024 {
		return false
	}
	start, err1 := time.Parse("2006-01-02", startTime)
	end, err2 := time.Parse("2006-01-02", endTime)
	if err1 != nil || err2 != nil {
		return false
	}
	modTime := info.ModTime()
	return !end.After(modTime) && modTime.Sub(start) > 0
}

// --- Credentials ---

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
		ID: generateID("cred"), UserID: userID, Exchange: req.Exchange,
		APIKeyEncrypted: keyEnc, APISecretEncrypted: secretEnc, PassphraseEncrypted: passEnc,
		IsTestnet: req.IsTestnet,
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
	if cred.IsVerified {
		mode := ModeLive
		if req.IsTestnet {
			mode = ModePaper
		}
		instances, _ := api.store.ListInstances(userID, mode)
		for i := range instances {
			inst := &instances[i]
			if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
				if _, loaded := api.starting.LoadOrStore(inst.InstanceID, true); !loaded {
					go api.startInstanceContainer(inst)
				}
			}
		}
	}
	resp := credentialResponse{ID: cred.ID, UserID: cred.UserID, Exchange: cred.Exchange, IsTestnet: cred.IsTestnet, IsVerified: cred.IsVerified}
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
		safe[i] = credentialResponse{ID: c.ID, UserID: c.UserID, Exchange: c.Exchange, IsTestnet: c.IsTestnet, IsVerified: c.IsVerified}
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
	if exchange != "" && api.syncer != nil {
		go api.syncer.DeleteCredential(userID, exchange, isTestnet)
	}
	mode := ModeLive
	if isTestnet {
		mode = ModePaper
	}
	instances, _ := api.store.ListInstances(userID, mode)
	for i := range instances {
		inst := &instances[i]
		if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
			if _, loaded := api.starting.LoadOrStore(inst.InstanceID, true); !loaded {
				go api.startInstanceContainer(inst)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Notifications ---

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
	ch := NotificationChannel{ID: generateID("notif"), UserID: userID, Type: req.Type, ChatID: req.ChatID, Enabled: true}
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
	cfg := NotificationConfig{Channel: ch, Rules: NotificationRule{TradeEvents: req.Rules.TradeEvents, OrderEvents: req.Rules.OrderEvents, ContainerHealth: req.Rules.ContainerHealth}}
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
		safe[i] = notifConfigResponse{ID: c.Channel.ID, Type: c.Channel.Type, Enabled: c.Channel.Enabled, Rules: c.Rules}
	}
	writeJSON(w, http.StatusOK, safe)
}

func (api *API) DeleteNotificationConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	if err := api.notifier.Delete(userID, chi.URLParam(r, "id")); err != nil {
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
	sent := api.notifier.Dispatch(userID, NotificationEvent{Type: "test", Title: "BBGO Test Notification", Message: "If you see this, notifications are working!"})
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

// --- Bots ---

func (api *API) ListBots(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	mode := r.URL.Query().Get("mode")
	var modes []string
	if mode == ModeLive || mode == ModePaper {
		modes = []string{mode}
	} else {
		modes = []string{ModeLive, ModePaper}
	}
	var bots []Bot
	for _, m := range modes {
		instances, _ := api.store.ListInstances(userID, m)
		for _, inst := range instances {
			bot := Bot{ID: inst.InstanceID, Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange, Mode: inst.Mode, Name: inst.Name}
			if api.isInstanceStarting(inst.InstanceID) {
				bot.ContainerStatus = StatusStarting
			} else if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
				bot.ContainerStatus = StatusRunning
				bot.ContainerName = api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
			} else {
				bot.ContainerStatus = StatusStopped
			}
			bots = append(bots, bot)
		}
	}
	if bots == nil {
		bots = []Bot{}
	}
	writeJSON(w, http.StatusOK, botsResponse{Bots: bots})
}

func (api *API) GetBot(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	botID := chi.URLParam(r, "botID")
	for _, mode := range []string{ModeLive, ModePaper} {
		inst, err := api.store.GetInstance(userID, mode, botID)
		if err == nil {
			bot := Bot{ID: inst.InstanceID, Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange, Mode: inst.Mode, Name: inst.Name}
			if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
				bot.ContainerStatus = StatusRunning
				bot.ContainerName = api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
			} else {
				bot.ContainerStatus = StatusStopped
			}
			writeJSON(w, http.StatusOK, bot)
			return
		}
	}
	writeError(w, http.StatusNotFound, "bot not found")
}

// --- Utility ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func credModeError(mode, ex string) string {
	if mode == ModeLive {
		return fmt.Sprintf("live mode requires API credentials for %s — add them in Settings first", ex)
	}
	return "paper mode requires Binance API credentials (live keys) — add them in Settings first"
}

func strategyMatch(tradeStrategy, filterStrategy string) bool {
	if tradeStrategy == filterStrategy {
		return true
	}
	return strings.HasPrefix(tradeStrategy, filterStrategy+"-") || strings.HasPrefix(tradeStrategy, filterStrategy+":")
}
