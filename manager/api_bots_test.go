package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// setupBotsTestAPI creates an API with pre-populated containers for bots tests.
func setupBotsTestAPI() (*API, *chi.Mux) {
	users := NewUserContainerManager()

	// User 1: live container with 2 strategies, paper with 1
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Mode:   ModeLive,
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{ID: "strat_grid_btc", Name: "BTC Grid", Exchange: "binance", Strategy: "grid2", Config: rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`), Mode: ModeLive},
			{ID: "strat_eth_dca", Name: "ETH DCA", Exchange: "binance", Strategy: "dca", Config: rawJSON(`{"symbol":"ETHUSDT"}`), Mode: ModeLive},
		},
	}
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModePaper] = &UserContainer{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Mode:   ModePaper,
		Status: StatusStopped,
		Strategies: []StrategyEntry{
			{ID: "strat_paper_grid", Name: "Paper Grid", Exchange: "binance", Strategy: "grid2", Config: rawJSON(`{"symbol":"BTCUSDT"}`), Mode: ModePaper},
		},
	}

	// User 2: single live strategy
	users.users["11111111-2222-3333-4444-555555555555:"+ModeLive] = &UserContainer{
		UserID: "11111111-2222-3333-4444-555555555555",
		Mode:   ModeLive,
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{ID: "strat_sol_trend", Name: "SOL Trend", Exchange: "bybit", Strategy: "supertrend", Config: rawJSON(`{"symbol":"SOLUSDT"}`), Mode: ModeLive},
		},
	}

	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.containerRunning = func(_, _ string) bool { return true }

	r := testRouter(api)
	return api, r
}

// --- ListBots tests ---

func TestListBots_ReturnsAllStrategiesForUser(t *testing.T) {
	_, r := setupBotsTestAPI()

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

	if len(resp.Bots) != 3 {
		t.Fatalf("expected 3 bots (2 live + 1 paper), got %d", len(resp.Bots))
	}
}

func TestListBots_FilterByLiveMode(t *testing.T) {
	_, r := setupBotsTestAPI()

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

	if len(resp.Bots) != 2 {
		t.Fatalf("expected 2 live bots, got %d", len(resp.Bots))
	}
	for _, b := range resp.Bots {
		if b.Mode != ModeLive {
			t.Errorf("expected live mode, got %q", b.Mode)
		}
	}
}

func TestListBots_FilterByPaperMode(t *testing.T) {
	_, r := setupBotsTestAPI()

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

	if len(resp.Bots) != 1 {
		t.Fatalf("expected 1 paper bot, got %d", len(resp.Bots))
	}
	if resp.Bots[0].ID != "strat_paper_grid" {
		t.Errorf("expected strat_paper_grid, got %s", resp.Bots[0].ID)
	}
	if resp.Bots[0].ContainerStatus != StatusStopped {
		t.Errorf("expected stopped status for paper bot, got %s", resp.Bots[0].ContainerStatus)
	}
}

func TestListBots_BotFieldsCorrect(t *testing.T) {
	_, r := setupBotsTestAPI()

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
	if bot.Name != "BTC Grid" {
		t.Errorf("expected name 'BTC Grid', got %s", bot.Name)
	}
	if bot.Exchange != "binance" {
		t.Errorf("expected exchange binance, got %s", bot.Exchange)
	}
	if bot.Strategy != "grid2" {
		t.Errorf("expected strategy grid2, got %s", bot.Strategy)
	}
	if bot.ContainerStatus != StatusRunning {
		t.Errorf("expected running status, got %s", bot.ContainerStatus)
	}
}

func TestListBots_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Mode:       ModeLive,
		Status:     StatusStopped,
		Strategies: []StrategyEntry{},
	}
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
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
	users := NewUserContainerManager()
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
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
	_, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestListBots_ConfigPreserved(t *testing.T) {
	_, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots?mode=live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	var cfg map[string]interface{}
	if err := json.Unmarshal(resp.Bots[0].Config, &cfg); err != nil {
		t.Fatalf("config parse error: %v", err)
	}
	if cfg["symbol"] != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %v", cfg["symbol"])
	}
	if cfg["gridNumber"] != float64(10) {
		t.Errorf("expected gridNumber 10, got %v", cfg["gridNumber"])
	}
}

// --- GetBot tests ---

func TestGetBot_Found(t *testing.T) {
	_, r := setupBotsTestAPI()

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
	if bot.Name != "BTC Grid" {
		t.Errorf("expected name 'BTC Grid', got %s", bot.Name)
	}
	if bot.Exchange != "binance" {
		t.Errorf("expected exchange binance, got %s", bot.Exchange)
	}
	if bot.Strategy != "grid2" {
		t.Errorf("expected strategy grid2, got %s", bot.Strategy)
	}
	if bot.Mode != ModeLive {
		t.Errorf("expected live mode, got %s", bot.Mode)
	}
	if bot.ContainerStatus != StatusRunning {
		t.Errorf("expected running status, got %s", bot.ContainerStatus)
	}
}

func TestGetBot_PaperBot(t *testing.T) {
	_, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_paper_grid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	if bot.Mode != ModePaper {
		t.Errorf("expected paper mode, got %s", bot.Mode)
	}
	if bot.ContainerStatus != StatusStopped {
		t.Errorf("expected stopped status for paper bot, got %s", bot.ContainerStatus)
	}
}

func TestGetBot_NotFound(t *testing.T) {
	_, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/nonexistent_id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetBot_InvalidUserID(t *testing.T) {
	_, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/bots/strat_grid_btc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetBot_DifferentUser_CannotSeeOthersBot(t *testing.T) {
	_, r := setupBotsTestAPI()

	// User 1 tries to access User 2's bot
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_sol_trend", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (bot belongs to different user), got %d", w.Code)
	}
}

func TestGetBot_ConfigPreserved(t *testing.T) {
	_, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_eth_dca", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	var cfg map[string]interface{}
	if err := json.Unmarshal(bot.Config, &cfg); err != nil {
		t.Fatalf("config parse error: %v", err)
	}
	if cfg["symbol"] != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %v", cfg["symbol"])
	}
}

func TestGetBot_ContainerStatusReflected(t *testing.T) {
	api, r := setupBotsTestAPI()

	api.users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, StatusError)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_grid_btc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	if bot.ContainerStatus != StatusError {
		t.Errorf("expected error status, got %s", bot.ContainerStatus)
	}
}

func TestGetBot_CrossExchangeStrategy(t *testing.T) {
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Mode:   ModeLive,
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{
				ID:            "strat_xmaker",
				Name:          "Cross XMaker",
				Strategy:      "xmaker",
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
				},
				Config: rawJSON(`{"symbol":"BTCUSDT"}`),
				Mode:   ModeLive,
			},
		},
	}
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	r := testRouter(api)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/strat_xmaker", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	if !bot.CrossExchange {
		t.Error("expected crossExchange=true")
	}
	if len(bot.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(bot.Sessions))
	}
	if bot.Sessions[0].Exchange != "binance" {
		t.Errorf("expected maker exchange binance, got %s", bot.Sessions[0].Exchange)
	}
	if bot.Sessions[1].Futures != true {
		t.Error("expected hedge session futures=true")
	}
}

// --- Bot data isolation tests ---

func TestListBots_UserIsolation(t *testing.T) {
	_, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/11111111-2222-3333-4444-555555555555/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Bots) != 1 {
		t.Fatalf("expected 1 bot for user2, got %d", len(resp.Bots))
	}
	if resp.Bots[0].ID != "strat_sol_trend" {
		t.Errorf("expected strat_sol_trend, got %s", resp.Bots[0].ID)
	}
	if resp.Bots[0].Exchange != "bybit" {
		t.Errorf("expected exchange bybit, got %s", resp.Bots[0].Exchange)
	}
}

func TestListBots_StatusPerContainer(t *testing.T) {
	api, r := setupBotsTestAPI()

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	for _, b := range resp.Bots {
		switch b.Mode {
		case ModeLive:
			if b.ContainerStatus != StatusRunning {
				t.Errorf("live bot %s: expected running, got %s", b.ID, b.ContainerStatus)
			}
		case ModePaper:
			if b.ContainerStatus != StatusStopped {
				t.Errorf("paper bot %s: expected stopped, got %s", b.ID, b.ContainerStatus)
			}
		}
	}

	api.users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ModeLive, StatusError)

	req2 := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots?mode=live", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 struct {
		Bots []Bot `json:"bots"`
	}
	json.NewDecoder(w2.Body).Decode(&resp2)

	for _, b := range resp2.Bots {
		if b.ContainerStatus != StatusError {
			t.Errorf("after status change: expected error, got %s for bot %s", b.ContainerStatus, b.ID)
		}
	}
}
