package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

const credsUID = "cccccccc-dddd-eeee-aaaa-bbbbbbbbbbbb"

func setupCredsAPI(t *testing.T) (*API, func()) {
	t.Helper()
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	store := NewInstanceStore(dir, nil)
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := &ContainerManager{cfg: cfg, store: store}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, nil, nil, nil, nil, nil, nil, nil)
	return api, func() {}
}

func containerRunningFor(running map[string]bool) func(string, string, string) bool {
	return func(userID, mode, instanceID string) bool {
		return running[userID+":"+mode+":"+instanceID]
	}
}

// --- botFromStrategy unit tests ---

func TestBotFromStrategy_ExtractsNestedSymbol(t *testing.T) {
	s := map[string]any{
		"strategy":           "grid2",
		"strategyInstanceID": "grid2-BTCUSDT-size-5-75500-73000",
		"on":                 []any{"binance"},
		"grid2": map[string]any{
			"symbol":     "BTCUSDT",
			"gridNumber": float64(5),
			"upperPrice": float64(75500),
			"lowerPrice": float64(73000),
		},
	}

	bot := botFromStrategy(s, "paper")

	if bot.ID != "grid2-BTCUSDT-size-5-75500-73000" {
		t.Errorf("ID: got %q", bot.ID)
	}
	if bot.Strategy != "grid2" {
		t.Errorf("Strategy: got %q", bot.Strategy)
	}
	if bot.Symbol != "BTCUSDT" {
		t.Errorf("Symbol: got %q", bot.Symbol)
	}
	if bot.Exchange != "binance" {
		t.Errorf("Exchange: got %q", bot.Exchange)
	}
	if bot.Mode != "paper" {
		t.Errorf("Mode: got %q", bot.Mode)
	}
	if bot.ContainerStatus != StatusRunning {
		t.Errorf("ContainerStatus: got %q", bot.ContainerStatus)
	}
}

func TestBotFromStrategy_ConfigContainsNestedStrategyParams(t *testing.T) {
	s := map[string]any{
		"strategy":           "grid2",
		"strategyInstanceID": "grid2-BTCUSDT-size-5-75500-73000",
		"on":                 []any{"binance"},
		"grid2": map[string]any{
			"symbol":     "BTCUSDT",
			"gridNumber": float64(5),
			"upperPrice": float64(75500),
			"lowerPrice": float64(73000),
		},
	}

	bot := botFromStrategy(s, "live")

	var cfg map[string]any
	if err := json.Unmarshal(bot.Config, &cfg); err != nil {
		t.Fatalf("Config: failed to unmarshal: %v", err)
	}
	if cfg["symbol"] != "BTCUSDT" {
		t.Errorf("Config.symbol: got %v", cfg["symbol"])
	}
	if cfg["gridNumber"] != float64(5) {
		t.Errorf("Config.gridNumber: got %v", cfg["gridNumber"])
	}
	if cfg["upperPrice"] != float64(75500) {
		t.Errorf("Config.upperPrice: got %v", cfg["upperPrice"])
	}
}

func TestBotFromStrategy_StateContainsFullMap(t *testing.T) {
	s := map[string]any{
		"strategy":           "supertrend",
		"strategyInstanceID": "supertrend-ETHUSDT",
		"on":                 []any{"binance"},
		"supertrend": map[string]any{
			"symbol": "ETHUSDT",
		},
	}

	bot := botFromStrategy(s, "live")

	var state map[string]any
	if err := json.Unmarshal(bot.State, &state); err != nil {
		t.Fatalf("State: failed to unmarshal: %v", err)
	}
	if state["strategy"] != "supertrend" {
		t.Errorf("State.strategy: got %v", state["strategy"])
	}
	if state["strategyInstanceID"] != "supertrend-ETHUSDT" {
		t.Errorf("State.strategyInstanceID: got %v", state["strategyInstanceID"])
	}
}

func TestBotFromStrategy_MissingStrategyInstanceID(t *testing.T) {
	s := map[string]any{
		"strategy": "grid2",
		"on":       []any{"binance"},
		"grid2": map[string]any{
			"symbol": "BTCUSDT",
		},
	}

	bot := botFromStrategy(s, "live")

	if bot.ID != "" {
		t.Errorf("ID should be empty when missing, got %q", bot.ID)
	}
	if bot.Symbol != "BTCUSDT" {
		t.Errorf("Symbol should still be extracted: got %q", bot.Symbol)
	}
}

func TestBotFromStrategy_MissingOnArray(t *testing.T) {
	s := map[string]any{
		"strategy":           "grid2",
		"strategyInstanceID": "grid2-BTCUSDT",
		"grid2": map[string]any{
			"symbol": "BTCUSDT",
		},
	}

	bot := botFromStrategy(s, "live")

	if bot.Exchange != "" {
		t.Errorf("Exchange should be empty when on is missing, got %q", bot.Exchange)
	}
	if bot.Symbol != "BTCUSDT" {
		t.Errorf("Symbol: got %q", bot.Symbol)
	}
}

func TestBotFromStrategy_EmptyOnArray(t *testing.T) {
	s := map[string]any{
		"strategy":           "grid2",
		"strategyInstanceID": "grid2-BTCUSDT",
		"on":                 []any{},
		"grid2": map[string]any{
			"symbol": "BTCUSDT",
		},
	}

	bot := botFromStrategy(s, "live")

	if bot.Exchange != "" {
		t.Errorf("Exchange should be empty with empty on, got %q", bot.Exchange)
	}
}

func TestBotFromStrategy_NoNestedConfig(t *testing.T) {
	s := map[string]any{
		"strategy":           "grid2",
		"strategyInstanceID": "grid2",
		"on":                 []any{"binance"},
	}

	bot := botFromStrategy(s, "paper")

	if bot.Symbol != "" {
		t.Errorf("Symbol should be empty without nested config, got %q", bot.Symbol)
	}
	if bot.Config != nil {
		t.Errorf("Config should be nil without nested config, got %v", bot.Config)
	}
	if bot.Exchange != "binance" {
		t.Errorf("Exchange should still come from on, got %q", bot.Exchange)
	}
}

// --- Integration tests for bot listing from instance store ---

func setupBotsTestAPI(t *testing.T) (*API, *chi.Mux) {
	t.Helper()
	store, dir := newTestStore(t)

	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := &ContainerManager{cfg: cfg, store: store}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, store, cm, proxy, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	return api, testRouter(api)
}

func TestListBots_FilterByMode(t *testing.T) {
	api, r := setupBotsTestAPI(t)

	grid2Cfg := map[string]any{"gridNumber": 10, "upperPrice": "65000", "lowerPrice": "58000"}
	createTestInstance(t, api.store, testUUID, ModeLive, "grid2", "BTCUSDT", grid2Cfg)
	createTestInstance(t, api.store, testUUID, ModeLive, "dca", "ETHUSDT", nil)
	createTestInstance(t, api.store, testUUID, ModePaper, "grid2", "BTCUSDT", grid2Cfg)

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
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Bots) != 2 {
		t.Fatalf("expected 2 bots, got %d", len(resp.Bots))
	}

	var gridBot *Bot
	for i := range resp.Bots {
		if resp.Bots[i].ID == "grid2-BTCUSDT-size-10-65000-58000" {
			gridBot = &resp.Bots[i]
			break
		}
	}
	if gridBot == nil {
		t.Fatal("grid2-BTCUSDT-size-10-65000-58000 bot not found")
	}
	if gridBot.Strategy != "grid2" {
		t.Errorf("Strategy: got %q", gridBot.Strategy)
	}
	if gridBot.Symbol != "BTCUSDT" {
		t.Errorf("Symbol: got %q", gridBot.Symbol)
	}
	if gridBot.Exchange != "binance" {
		t.Errorf("Exchange: got %q", gridBot.Exchange)
	}
	if gridBot.Mode != "live" {
		t.Errorf("Mode: got %q", gridBot.Mode)
	}
}

func TestGetBot_ReturnsCorrectFields(t *testing.T) {
	api, r := setupBotsTestAPI(t)

	grid2Cfg := map[string]any{"gridNumber": 10, "upperPrice": "65000", "lowerPrice": "58000"}
	createTestInstance(t, api.store, testUUID, ModeLive, "grid2", "BTCUSDT", grid2Cfg)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/grid2-BTCUSDT-size-10-65000-58000", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var bot Bot
	if err := json.NewDecoder(w.Body).Decode(&bot); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if bot.ID != "grid2-BTCUSDT-size-10-65000-58000" {
		t.Errorf("ID: got %q", bot.ID)
	}
	if bot.Symbol != "BTCUSDT" {
		t.Errorf("Symbol: got %q", bot.Symbol)
	}
	if bot.Exchange != "binance" {
		t.Errorf("Exchange: got %q", bot.Exchange)
	}
}

func TestGetBot_SecondStrategy(t *testing.T) {
	api, r := setupBotsTestAPI(t)

	createTestInstance(t, api.store, testUUID, ModeLive, "dca", "ETHUSDT", nil)

	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bots/dca-ETHUSDT", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var bot Bot
	json.NewDecoder(w.Body).Decode(&bot)

	if bot.Strategy != "dca" {
		t.Errorf("Strategy: got %q", bot.Strategy)
	}
	if bot.Symbol != "ETHUSDT" {
		t.Errorf("Symbol: got %q", bot.Symbol)
	}
}
