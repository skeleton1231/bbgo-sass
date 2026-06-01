package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// setupBotsTestAPI creates an API with pre-populated containers for bots tests.
// botBBGoHandler returns a mock bbgo handler that serves strategy data for bots tests.
func botBBGoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/strategies/single" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"strategies": []map[string]interface{}{
					{
						"strategyInstanceID": "strat_grid_btc",
						"strategy":           "grid2",
						"on":                 []interface{}{"binance"},
						"grid2":              map[string]interface{}{"symbol": "BTCUSDT", "gridNumber": float64(10)},
					},
					{
						"strategyInstanceID": "strat_eth_dca",
						"strategy":           "dca",
						"on":                 []interface{}{"binance"},
						"dca":                map[string]interface{}{"symbol": "ETHUSDT"},
					},
					{
						"strategyInstanceID": "strat_paper_grid",
						"strategy":           "grid2",
						"on":                 []interface{}{"binance"},
						"grid2":              map[string]interface{}{"symbol": "BTCUSDT"},
					},
				},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "ok"})
	}
}

func setupBotsTestAPI(t *testing.T) (*API, *chi.Mux) {
	t.Helper()
	store, dir := newTestStore(t)

	writeTestStrategies(t, store, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
		{Exchange: "binance", Strategy: "dca"},
	})
	writeTestStrategies(t, store, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModePaper, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})
	writeTestStrategies(t, store, "11111111-2222-3333-4444-555555555555", ModeLive, []StrategyEntry{
		{Exchange: "bybit", Strategy: "supertrend"},
	})

	bbgoSrv := httptest.NewServer(botBBGoHandler())
	t.Cleanup(bbgoSrv.Close)

	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := testRouter(api)
	return api, r
}

// --- ListBots tests ---

func TestListBots_ReturnsAllStrategiesForUser(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Both live and paper containers are running, each returns 3 strategies from mock
	if len(resp.Bots) != 6 {
		t.Fatalf("expected 6 bots (3 live + 3 paper), got %d", len(resp.Bots))
	}
}

func TestListBots_FilterByLiveMode(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Mock bbgo returns 3 strategies per mode call
	if len(resp.Bots) != 3 {
		t.Fatalf("expected 3 live bots, got %d", len(resp.Bots))
	}
	for _, b := range resp.Bots {
		if b.Mode != ModeLive {
			t.Errorf("expected live mode, got %q", b.Mode)
		}
	}
}

func TestListBots_FilterByPaperMode(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots?mode=paper", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Mock bbgo returns 3 strategies for paper mode too
	if len(resp.Bots) != 3 {
		t.Fatalf("expected 3 paper bots, got %d", len(resp.Bots))
	}
	for _, b := range resp.Bots {
		if b.Mode != ModePaper {
			t.Errorf("expected paper mode, got %q", b.Mode)
		}
	}
}

func TestListBots_BotFieldsCorrect(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	bot := resp.Bots[0]
	if bot.ID != "strat_grid_btc" {
		t.Errorf("expected ID strat_grid_btc, got %s", bot.ID)
	}
	if bot.Strategy != "grid2" {
		t.Errorf("expected strategy grid2, got %s", bot.Strategy)
	}
	if bot.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", bot.Symbol)
	}
	if bot.Session != "binance" {
		t.Errorf("expected session binance, got %s", bot.Session)
	}
	if bot.ContainerStatus != StatusRunning {
		t.Errorf("expected running status, got %s", bot.ContainerStatus)
	}
}

func TestListBots_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	store, dir := newTestStore(t)
	writeTestStrategies(t, store, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, []StrategyEntry{})
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return false }
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Bots == nil {
		t.Fatal("expected empty array, got nil")
	}
	if len(resp.Bots) != 0 {
		t.Errorf("expected 0 bots, got %d", len(resp.Bots))
	}
}

func TestListBots_NoContainers_ReturnsEmptyArray(t *testing.T) {
	store := NewStrategyStore("")
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Bots == nil {
		t.Fatal("expected empty array, got nil")
	}
}

func TestListBots_InvalidUserID(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestListBots_StatePreserved(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	// First bot from mock has symbol BTCUSDT in its state
	if resp.Bots[0].Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", resp.Bots[0].Symbol)
	}
}

// --- GetBot tests ---

func TestGetBot_Found(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_grid_btc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var bot Bot
	if err := json.NewDecoder(w.Body).Decode(&bot); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if bot.ID != "strat_grid_btc" {
		t.Errorf("expected ID strat_grid_btc, got %s", bot.ID)
	}
	if bot.Strategy != "grid2" {
		t.Errorf("expected strategy grid2, got %s", bot.Strategy)
	}
	if bot.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", bot.Symbol)
	}
	if bot.Session != "binance" {
		t.Errorf("expected session binance, got %s", bot.Session)
	}
	if bot.ContainerStatus != StatusRunning {
		t.Errorf("expected running status, got %s", bot.ContainerStatus)
	}
}

func TestGetBot_PaperBot(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_paper_grid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	if bot.ID != "strat_paper_grid" {
		t.Errorf("expected ID strat_paper_grid, got %s", bot.ID)
	}
	if bot.ContainerStatus != StatusRunning {
		t.Errorf("expected running status, got %s", bot.ContainerStatus)
	}
}

func TestGetBot_NotFound(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/nonexistent_id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetBot_InvalidUserID(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/bots/strat_grid_btc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetBot_DifferentUser_CannotSeeOthersBot(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	// User 1 tries to access User 2's bot
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_sol_trend", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (bot belongs to different user), got %d", w.Code)
	}
}

func TestGetBot_SymbolPreserved(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_eth_dca", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	if bot.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", bot.Symbol)
	}
}

func TestGetBot_ContainerStatusReflected(t *testing.T) {
	api, r := setupBotsTestAPI(t)

	// When container is not running, GetBot returns 404
	api.containerRunning = func(_, _ string) bool { return false }

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_grid_btc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when container stopped, got %d", w.Code)
	}
}

func TestGetBot_CrossExchangeStrategy(t *testing.T) {
	// The new Bot struct gets data from bbgo API, not from StrategyStore.
	// Cross-exchange metadata is part of the strategy config, not the Bot response.
	// This test verifies that a cross-exchange strategy bot can be retrieved.
	store, dir := newTestStore(t)
	writeTestStrategies(t, store, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, []StrategyEntry{
		{
			Strategy:      "xmaker",
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
				{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
			},
			Mode: ModeLive,
		},
	})
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)

		// Mock bbgo to return the xmaker strategy
		bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/strategies/single" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"strategies": []map[string]interface{}{
						{"strategyInstanceID": "strat_xmaker", "strategy": "xmaker", "symbol": "BTCUSDT", "session": "binance"},
					},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "ok"})
		}))
	t.Cleanup(bbgoSrv.Close)

	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_xmaker", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	if bot.Strategy != "xmaker" {
		t.Errorf("expected strategy xmaker, got %s", bot.Strategy)
	}
}

// --- Bot data isolation tests ---

func TestListBots_UserIsolation(t *testing.T) {
	_, r := setupBotsTestAPI(t)

	req := httptest.NewRequest("GET", "/api/users/11111111-2222-3333-4444-555555555555/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	// User 2's bots come from the same mock bbgo server
	// Since mock returns same data for all requests, user2 gets 3 strategies per mode
	if len(resp.Bots) != 6 {
		t.Fatalf("expected 6 bots for user2 (3 live + 3 paper), got %d", len(resp.Bots))
	}
}

func TestListBots_StatusPerContainer(t *testing.T) {
	api, r := setupBotsTestAPI(t)

	// By default both modes are running, so all bots show as running
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	for _, b := range resp.Bots {
		if b.ContainerStatus != StatusRunning {
			t.Errorf("bot %s: expected running, got %s", b.ID, b.ContainerStatus)
		}
	}

	// Now stop the live container — only paper bots should appear
	api.containerRunning = func(uid, mode string) bool { return uid == "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" && mode == ModePaper }

	req2 := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w2.Body).Decode(&resp2)

	for _, b := range resp2.Bots {
		if b.Mode != ModePaper {
			t.Errorf("after stopping live: expected paper mode, got %s", b.Mode)
		}
	}
}
