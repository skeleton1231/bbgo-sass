package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
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

// chiParam returns the URL-decoded value of a chi path parameter. chi.URLParam
// returns the raw percent-encoded segment, so an instance ID like
// "emacross:BTCUSDT:15m:5-13" arrives as "emacross%3A..." and never matches the
// stored value — which silently broke start/stop/delete on every colon-bearing
// instance. Decode here so every handler sees the real value.
func chiParam(r *http.Request, name string) string {
	v := chi.URLParam(r, name)
	if d, err := url.PathUnescape(v); err == nil {
		return d
	}
	return v
}

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
	metrics    *Metrics
	health     *cachedHealth

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
	api := &API{
		cfg: cfg, store: store, container: cm, proxy: proxy,
		creds: creds, encryptor: enc, syncer: syncer,
		hub: hub, testnetHub: testnetHub, notifier: notifier,
		wsTickets: NewWSTicketStore(), btExec: btExec, btJobs: btJobs,
		btSyncSem: make(chan struct{}, 2), storage: storage,
		metrics:       NewMetrics(),
		newBBGoClient: NewBBGoClient,
		stopCtx:       ctx, stopCancel: cancel,
	}
	api.health = newCachedHealth(api.refreshHealth, 15*time.Second)
	return api
}

// WithMetrics overrides the default Metrics instance — used by main.go to share
// a single Metrics across the API, container recovery loop, and WS ticket store.
func (api *API) WithMetrics(m *Metrics) *API {
	if m != nil {
		api.metrics = m
	}
	return api
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
		r.Get("/livez", api.LivezHandler)
		r.Get("/readyz", api.ReadyzHandler)
		r.Get("/metrics", api.MetricsHandler)
		r.Get("/api/markets/{exchange}/symbols", api.MarketSymbols)
		r.Get("/api/markets/{exchange}/ticker", api.MarketTicker)
		r.Get("/api/markets/{exchange}/klines", api.MarketKlines)

		r.Route("/", func(r chi.Router) {
			r.Use(UserRateLimit(3*time.Second, 20))
			r.Post("/api/users/{userID}/strategies", api.CreateStrategy)
			r.Patch("/api/users/{userID}/strategies/{strategyID}", api.UpdateStrategy)
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
		r.Get("/api/users/{userID}/bbgo/session/{session}/symbols", api.BBGoSessionSymbols)
		r.Get("/api/users/{userID}/bbgo/strategies", api.BBGoStrategies)
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
	urlID := chiParam(r, "userID")
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

// resolveTarget returns the session name to use for gRPC routing.
// If session is specified, it overrides the exchange (e.g., "binance_futures").
func resolveTarget(exchange, session string) string {
	if session != "" {
		return session
	}
	return exchange
}

func (api *API) isInstanceStarting(instanceID string) bool {
	_, ok := api.starting.Load(instanceID)
	return ok
}

func (api *API) isInstanceRunning(userID, mode, instanceID string) bool {
	return api.container.IsInstanceRunning(userID, mode, instanceID)
}

// instanceEffectiveStatus classifies an instance's actual liveness, accounting
// for the crashloop window where Docker reports running=true between restart
// cycles of a --restart=unless-stopped container. Returns one of
// StatusRunning/StatusError/StatusStopped plus the captured reason (if any).
//
// isInstanceRunning (Docker .State.Running only) is unsafe for restart
// decisions — a crashlooping bbgo whose process exits non-zero on startup
// still appears running for the brief moment between Docker's restart cycles,
// which makes StartInstance treat the click as a no-op while the container
// silently keeps failing. This helper is what those callers should use.
func (api *API) instanceEffectiveStatus(userID, mode, instanceID string) (status, reason string) {
	if api.isInstanceStarting(instanceID) {
		return StatusStarting, ""
	}
	if !api.container.IsInstanceRunning(userID, mode, instanceID) {
		return StatusStopped, ""
	}
	health, err := api.container.CheckInstanceHealth(userID, mode, instanceID)
	if err != nil || health.Status == HealthStatusStopped {
		return StatusStopped, health.Reason
	}
	if health.Status == HealthStatusError {
		return StatusError, health.Reason
	}
	return StatusRunning, ""
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

func (api *API) Health(w http.ResponseWriter, r *http.Request) {
	if api.health != nil {
		writeJSON(w, http.StatusOK, api.health.Get(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, HealthSnapshot{Status: "ok"})
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
		RiskConfig    *RiskConfig         `json:"riskConfig,omitempty"`
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

	// Inject FuturesConfig.leverage into config so it propagates everywhere:
	// validator, instance YAML (strategy struct field), backtest YAML, and persistence.
	// FuturesConfig is the UI's source of truth; config.leverage must mirror it.
	if req.FuturesConfig != nil && req.FuturesConfig.Leverage > 0 {
		var raw map[string]any
		if len(req.Config) == 0 || string(req.Config) == "null" {
			raw = map[string]any{}
		} else if err := json.Unmarshal(req.Config, &raw); err != nil {
			raw = map[string]any{}
		}
		raw["leverage"] = req.FuturesConfig.Leverage
		if req.FuturesConfig.MarginType != "" {
			raw["marginType"] = req.FuturesConfig.MarginType
		}
		if b, err := json.Marshal(raw); err == nil {
			req.Config = b
		}
	}

	if req.RiskConfig != nil {
		if err := req.RiskConfig.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	inst := &StrategyInstance{
		UserID: userID, Mode: req.Mode, Strategy: normalizedStrategy,
		Exchange: req.Exchange, Config: req.Config, Name: req.Name,
		CrossExchange: req.CrossExchange, Sessions: req.Sessions,
		FuturesConfig: req.FuturesConfig,
		RiskConfig:    req.RiskConfig,
	}

	symbol := extractSymbolFromConfig(req.Config)
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	inst.Symbol = symbol

	var mergedConfig json.RawMessage = req.Config
	if defaults := api.store.Defaults(); defaults != nil {
		if d := defaults.GetDefaults(normalizedStrategy); d != nil {
			var raw map[string]any
			if len(req.Config) == 0 || string(req.Config) == "null" {
				raw = map[string]any{}
			} else if err := json.Unmarshal(req.Config, &raw); err != nil {
				raw = map[string]any{}
			}
			merged := deepMerge(d, raw)
			if b, err := json.Marshal(merged); err == nil {
				mergedConfig = b
			}
		}
	}
	inst.InstanceID = computeInstanceID(inst.Strategy, inst.Symbol, mergedConfig)

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

	var warnings []StrategyWarning
	if w := ValidateStrategyConfig(normalizedStrategy, mergedConfig); len(w) > 0 {
		warnings = w
	}

	writeJSON(w, http.StatusCreated, struct {
		instanceInfo
		Warnings []StrategyWarning `json:"warnings,omitempty"`
	}{
		instanceInfo: instanceInfo{
			InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
			Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
			Name: inst.Name, Status: StatusStarting,
		},
		Warnings: warnings,
	})
}

func (api *API) ListStrategies(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	runningSet := api.container.ListRunningInstanceContainers(userID)
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
			} else if runningSet[api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)] {
				info.Status = StatusRunning
			}
			instances = append(instances, info)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "instances": instances})
}

// UpdateStrategy handles PATCH requests to update an existing instance's strategy
// params (config), FuturesConfig, and/or RiskConfig. All three are optional but at
// least one must be present. Config uses deep-merge semantics (nested maps recurse,
// scalars in the patch overwrite); FuturesConfig and RiskConfig use field-level
// merge semantics (zero-valued fields in the patch do not clear existing values,
// with the exception that an all-zero RiskConfig clears risk controls entirely).
// Triggers a container restart if the instance is currently running so the new
// YAML/env vars take effect immediately.
//
// Symbol and strategy cannot be changed via the config patch: the deterministic
// instance ID is computed from those fields (see pkg/instanceid), so mutating
// them would orphan historical trades/orders under the old ID. Callers wanting
// a different symbol or strategy must delete and recreate the instance.
//
// Restarts are serialized via api.starting: we explicitly clear any stale flag from a
// prior in-flight start BEFORE stopping the container, then unconditionally spawn a
// fresh start goroutine. This avoids the race where a previous start's health-check
// loop holds the flag, blocks our restart, and then times out leaving the container
// dead while the API reports "starting".
func (api *API) UpdateStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	instanceID := chiParam(r, "strategyID")
	mode := modeFromQuery(r)

	var req struct {
		Config        map[string]any `json:"config"`
		FuturesConfig *FuturesConfig `json:"futuresConfig"`
		RiskConfig    *RiskConfig    `json:"riskConfig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Config) == 0 && req.FuturesConfig == nil && req.RiskConfig == nil {
		writeError(w, http.StatusBadRequest, "config, futuresConfig, or riskConfig is required")
		return
	}
	const maxLeverage = 125
	if req.FuturesConfig != nil {
		if req.FuturesConfig.Leverage < 0 || req.FuturesConfig.Leverage > maxLeverage {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("leverage must be between 1 and %d", maxLeverage))
			return
		}
		if mt := req.FuturesConfig.MarginType; mt != "" && mt != "cross" && mt != "isolated" {
			writeError(w, http.StatusBadRequest, "marginType must be 'cross' or 'isolated'")
			return
		}
	}
	if req.RiskConfig != nil {
		if err := req.RiskConfig.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	existing, err := api.store.GetInstance(userID, mode, instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	// L1: leverage/marginType only affect futures strategies. Reject early so users
	// don't see a misleading "leverage updated" toast on a spot strategy that
	// silently ignores the change. RiskConfig applies to any strategy and is not
	// gated by this check.
	if req.FuturesConfig != nil && api.store.Defaults() != nil && !api.store.Defaults().RequiresFutures(existing.Strategy) {
		writeError(w, http.StatusBadRequest, "strategy does not support futures config")
		return
	}

	wasRunning := api.isInstanceRunning(existing.UserID, existing.Mode, existing.InstanceID)
	if wasRunning {
		// H1: clear any stale starting flag from an in-flight start BEFORE stop.
		// The previous start's health-check goroutine holds this flag while polling;
		// if we don't clear it, LoadOrStore below returns loaded=true and we skip
		// the restart, leaving the container dead after stop.
		api.starting.Delete(existing.InstanceID)
		// H3: surface stop failure instead of reporting success on a container
		// that may still be running with the old YAML.
		if err := api.container.StopInstance(existing.UserID, existing.Mode, existing.InstanceID); err != nil {
			log.Printf("update-strategy: stop instance %s for user %s failed: %v", existing.InstanceID, userID, err)
			writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("failed to stop running container: %v", err))
			return
		}
	}

	hasCredFn := func(exchange string) bool {
		if api.creds == nil || existing.Mode == ModePaper {
			return false
		}
		_, err := api.creds.GetByMode(userID, exchange, false)
		return err == nil
	}

	inst := existing
	if len(req.Config) > 0 {
		inst, err = api.store.UpdateInstanceConfig(userID, existing.Mode, instanceID, req.Config, hasCredFn)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.FuturesConfig != nil {
		inst, err = api.store.UpdateInstanceFuturesConfig(userID, existing.Mode, instanceID, req.FuturesConfig, hasCredFn)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if req.RiskConfig != nil {
		inst, err = api.store.UpdateInstanceRiskConfig(userID, existing.Mode, instanceID, req.RiskConfig, hasCredFn)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// M2: store applies merge semantics — inst.FuturesConfig may differ from req.FuturesConfig.
	merged := inst.FuturesConfig
	mergedRisk := inst.RiskConfig
	mergedConfig := inst.Config

	if wasRunning {
		// We already cleared the flag above and stopped the container. Spawn a fresh
		// start unconditionally (LoadOrStore just guards against concurrent calls
		// racing in between here and the goroutine startup).
		if _, loaded := api.starting.LoadOrStore(inst.InstanceID, true); !loaded {
			go api.startInstanceContainer(inst)
		}
	}

	writeJSON(w, http.StatusOK, struct {
		InstanceID    string          `json:"instance_id"`
		UserID        string          `json:"user_id"`
		Mode          string          `json:"mode"`
		Status        string          `json:"status"`
		Config        json.RawMessage `json:"config"`
		FuturesConfig *FuturesConfig  `json:"futuresConfig"`
		RiskConfig    *RiskConfig     `json:"riskConfig"`
	}{
		InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
		Status:        map[bool]string{true: StatusStarting, false: StatusStopped}[wasRunning],
		Config:        mergedConfig,
		FuturesConfig: merged,
		RiskConfig:    mergedRisk,
	})
}

func (api *API) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	instanceID := chiParam(r, "strategyID")
	mode := modeFromQuery(r)

	inst, err := api.store.GetInstance(userID, mode, instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	if api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
		_ = api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID)
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
	names := make([]string, 0, len(instances))
	for _, inst := range instances {
		names = append(names, api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID))
	}
	_ = api.container.StopInstanceNames(names)
	for _, inst := range instances {
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
	instanceID := chiParam(r, "instanceID")
	inst, err := api.store.GetInstance(userID, "", instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	status, reason := api.instanceEffectiveStatus(inst.UserID, inst.Mode, inst.InstanceID)
	if status == StatusRunning {
		writeJSON(w, http.StatusOK, instanceInfo{
			InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
			Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
			Name: inst.Name, Status: StatusRunning,
		})
		return
	}
	if status == StatusStarting {
		writeJSON(w, http.StatusAccepted, instanceInfo{
			InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
			Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
			Name: inst.Name, Status: StatusStarting,
		})
		return
	}
	// StatusError or StatusStopped: force-stop the container so the start
	// goroutine gets a clean slate. For StatusError (crashloop), Docker is
	// still restarting the failed container in a tight loop — without this
	// stop, CreateAndStartInstance's own StopInstance races against Docker's
	// restart and the new container can fail to bind the name. For
	// StatusStopped this is a cheap no-op.
	if status == StatusError {
		log.Printf("instance %s in error state, forcing restart (reason: %s)", inst.InstanceID, reason)
		if err := api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID); err != nil {
			log.Printf("force-stop errored instance %s: %v", inst.InstanceID, err)
		}
		// Clear stale error so the UI shows "starting" instead of lingering red
		// while the new container comes up. captureAndMarkInstanceError will
		// re-populate if the new container also fails.
		if api.store != nil {
			api.store.ClearInstanceError(inst.UserID, inst.Mode, inst.InstanceID)
		}
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
	instanceID := chiParam(r, "instanceID")
	inst, err := api.store.GetInstance(userID, "", instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	if err := api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID); err != nil {
		log.Printf("stop instance %s for user %s: %v", inst.InstanceID, userID, err)
	}
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
		status, reason := api.instanceEffectiveStatus(inst.UserID, inst.Mode, inst.InstanceID)
		switch status {
		case StatusRunning:
			info.Status = StatusRunning
		case StatusStarting:
			info.Status = StatusStarting
		case StatusError:
			// Force-stop the crashlooping container so CreateAndStartInstance
			// doesn't race with Docker's --restart=unless-stopped retry loop.
			log.Printf("instance %s in error state during StartAll, forcing restart (reason: %s)", inst.InstanceID, reason)
			if err := api.container.StopInstance(inst.UserID, inst.Mode, inst.InstanceID); err != nil {
				log.Printf("force-stop errored instance %s: %v", inst.InstanceID, err)
			}
			if api.store != nil {
				api.store.ClearInstanceError(inst.UserID, inst.Mode, inst.InstanceID)
			}
			info.Status = StatusStarting
			if _, loaded := api.starting.LoadOrStore(inst.InstanceID, true); !loaded {
				go api.startInstanceContainer(inst)
			}
		default:
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
	names := make([]string, 0, len(instances))
	for _, inst := range instances {
		names = append(names, api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID))
		api.starting.Delete(inst.InstanceID)
	}
	_ = api.container.StopInstanceNames(names)
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "user_id": userID, "mode": mode})
}

func (api *API) UserStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := api.resolveUserID(w, r)
	if !ok {
		return
	}
	runningSet := api.container.ListRunningInstanceContainers(userID)
	var instances []instanceInfo
	for _, mode := range []string{ModeLive, ModePaper} {
		insts, _ := api.store.ListInstances(userID, mode)
		for _, inst := range insts {
			info := instanceInfo{
				InstanceID: inst.InstanceID, UserID: inst.UserID, Mode: inst.Mode,
				Strategy: inst.Strategy, Symbol: inst.Symbol, Exchange: inst.Exchange,
				Name: inst.Name, Status: StatusStopped,
				LastError: inst.LastError, LastErrorAt: inst.LastErrorAt,
			}
			if api.isInstanceStarting(inst.InstanceID) {
				info.Status = StatusStarting
			} else if runningSet[api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)] {
				// Cheap crashloop check — see ListBots for the rationale.
				if health, err := api.container.CheckInstanceHealth(inst.UserID, inst.Mode, inst.InstanceID); err == nil && health.Status == HealthStatusError {
					info.Status = StatusError
					if info.LastError == "" {
						info.LastError = health.Reason
					}
				} else {
					info.Status = StatusRunning
				}
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
		api.captureAndMarkInstanceError(inst, fmt.Sprintf("docker run failed: %v", err))
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
		api.captureAndMarkInstanceError(inst, "container started but bbgo /ping unreachable after 30s")
		return
	}
	log.Printf("instance %s container started and healthy", inst.InstanceID)

	// Container is healthy — clear any stale error from a previous crashloop
	// so the frontend stops showing the old failure message.
	if api.store != nil {
		api.store.ClearInstanceError(inst.UserID, inst.Mode, inst.InstanceID)
	}

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

// captureAndMarkInstanceError grabs the container's last N log lines, extracts
// the logrus level=fatal / level=error line, and persists it to
// strategy_instances.last_error so the frontend can show *why* the container
// is failing. `fallback` is used when docker logs are unavailable (e.g.
// `docker run` itself errored before any container existed).
//
// Strategy-agnostic: whatever made bbgo crash (grid2 spread-too-small,
// bollmaker missing indicator, panic in any strategy, OOM, credential
// failure) shows up as a logrus level=fatal line — we don't hardcode
// strategy-specific keywords.
func (api *API) captureAndMarkInstanceError(inst *StrategyInstance, fallback string) {
	if api.store == nil {
		return
	}
	captured := api.container.CaptureContainerError(inst.UserID, inst.Mode, inst.InstanceID)
	if captured == "" {
		captured = fallback
	}
	api.store.MarkInstanceError(inst.UserID, inst.Mode, inst.InstanceID, captured)
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
	session := chiParam(r, "session")
	detail, err := client.GetSession(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessionDetailResponse{Session: *detail})
}

func (api *API) BBGoSessionSymbols(w http.ResponseWriter, r *http.Request) {
	client, _, ok := api.bbgoClientForUser(w, r)
	if !ok {
		return
	}
	session := chiParam(r, "session")
	symbols, err := client.GetSessionSymbols(session)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, symbolsResponse{Symbols: filterTradingPairs(symbols)})
}

func (api *API) MarketSymbols(w http.ResponseWriter, r *http.Request) {
	exchange := chiParam(r, "exchange")
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
	exchange := chiParam(r, "exchange")
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
	target := resolveTarget(exchange, r.URL.Query().Get("session"))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	req := &pb.QueryTickerRequest{Exchange: target, Symbol: symbol}
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
	exchange := chiParam(r, "exchange")
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
	target := resolveTarget(exchange, r.URL.Query().Get("session"))
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var err error
	req := &pb.QueryKLinesRequest{
		Exchange: target, Symbol: symbol, Interval: interval,
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

func (api *API) RunBacktest(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	var req struct {
		Strategy      string          `json:"strategy"`
		Config        json.RawMessage `json:"config"`
		Exchange      string          `json:"exchange"`
		StartTime     string          `json:"start_time"`
		EndTime       string          `json:"end_time"`
		FuturesConfig *FuturesConfig  `json:"futuresConfig,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	yamlContent, err := buildBacktestYAML(req.Strategy, req.Config, req.StartTime, req.EndTime, req.Exchange, "", api.store.Defaults(), req.FuturesConfig)
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
		Strategy      string          `json:"strategy"`
		Config        json.RawMessage `json:"config"`
		Exchange      string          `json:"exchange"`
		Symbol        string          `json:"symbol"`
		StartTime     string          `json:"start_time"`
		EndTime       string          `json:"end_time"`
		FuturesConfig *FuturesConfig  `json:"futuresConfig,omitempty"`
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
		var sym struct {
			Symbol string `json:"symbol"`
		}
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
		EndTime: req.EndTime, NeedSync: needSync, FuturesConfig: req.FuturesConfig,
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
	jobID := chiParam(r, "jobID")
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
	jobID := chiParam(r, "jobID")
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
	id := chiParam(r, "id")
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
	if err := api.notifier.Delete(userID, chiParam(r, "id")); err != nil {
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
	runningSet := api.container.ListRunningInstanceContainers(userID)

	// Gather instances across the requested mode(s). Mode order is preserved so
	// the response stays grouped (live before paper) as before.
	var allInsts []StrategyInstance
	for _, m := range modes {
		instances, _ := api.store.ListInstances(userID, m)
		allInsts = append(allInsts, instances...)
	}
	// Pre-compute container health CONCURRENTLY. Calling CheckInstanceHealth
	// serially inside the loop below (one `docker inspect` per running
	// container) made ListBots take 20s+ with ~40 paper containers, which
	// tripped the Next.js proxy's fetch timeout (ECONNRESET → HTTP 500) and
	// made the bots page unusable. Batched it is ~2s.
	healthMap := api.container.CheckInstanceHealthBatch(allInsts, runningSet)

	var bots []Bot
	for _, inst := range allInsts {
		bot := Bot{
			ID: inst.InstanceID, Strategy: inst.Strategy, Symbol: inst.Symbol,
			Exchange: inst.Exchange, Mode: inst.Mode, Name: inst.Name,
			LastError: inst.LastError, LastErrorAt: inst.LastErrorAt,
		}
		if api.isInstanceStarting(inst.InstanceID) {
			bot.ContainerStatus = StatusStarting
		} else {
			name := api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
			if !runningSet[name] {
				bot.ContainerStatus = StatusStopped
			} else {
				// Docker reports the container as running, but with
				// --restart=unless-stopped a crashlooping bbgo process
				// still shows up in `docker ps`. Cheap second check via
				// docker inspect detects crashloop (high RestartCount +
				// non-zero ExitCode) so we can mark it as error instead
				// of reporting a phantom-active container.
				bot.ContainerName = name
				if health, ok := healthMap[name]; ok && health.Status == HealthStatusError {
					bot.ContainerStatus = StatusError
					if bot.LastError == "" {
						bot.LastError = health.Reason
					}
				} else {
					bot.ContainerStatus = StatusRunning
				}
			}
		}
		bots = append(bots, bot)
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
	botID := chiParam(r, "botID")
	for _, mode := range []string{ModeLive, ModePaper} {
		inst, err := api.store.GetInstance(userID, mode, botID)
		if err == nil {
			bot := Bot{
				ID: inst.InstanceID, Strategy: inst.Strategy, Symbol: inst.Symbol,
				Exchange: inst.Exchange, Mode: inst.Mode, Name: inst.Name,
				LastError: inst.LastError, LastErrorAt: inst.LastErrorAt,
			}
			status, reason := api.instanceEffectiveStatus(inst.UserID, inst.Mode, inst.InstanceID)
			bot.ContainerStatus = status
			bot.ContainerName = api.container.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
			if status == StatusError && bot.LastError == "" && reason != "" {
				bot.LastError = reason
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
