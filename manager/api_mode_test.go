package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupModeTestAPI(t *testing.T, existingMode string) *API {
	t.Helper()
	users := NewUserContainerManager()
	strategies := []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}}
	if existingMode != "" {
		strategies[0].Mode = existingMode
	}
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusStopped,
		Strategies: strategies,
	}
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
		DataDir:      t.TempDir(),
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	return NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
}

func TestAPI_CreateStrategy_LiveOnlyRejectsPaper(t *testing.T) {
	api := setupModeTestAPI(t, "paper")
	r := testRouter(api)

	for _, strategy := range []string{"bollmaker", "supertrend", "dca2", "sentinel_anomaly"} {
		t.Run(strategy, func(t *testing.T) {
			body := map[string]interface{}{
				"name":     "test",
				"exchange": "binance",
				"strategy": strategy,
				"config":   map[string]interface{}{"symbol": "BTCUSDT"},
				"mode":     "paper",
			}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for liveOnly strategy %s in paper mode, got %d: %s", strategy, w.Code, w.Body.String())
			}
		})
	}
}

func TestAPI_CreateStrategy_LiveOnlyAcceptsLive(t *testing.T) {
	api := setupModeTestAPI(t, "live")
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "BollMaker",
		"exchange": "binance",
		"strategy": "bollmaker",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
		"mode":     "live",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for liveOnly strategy in live mode, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_MixedModeAcceptsPaper(t *testing.T) {
	api := setupModeTestAPI(t, "live")
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "Paper Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
		"mode":     "paper",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for paper strategy alongside live (separate containers), got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_MixedModeAcceptsLive(t *testing.T) {
	api := setupModeTestAPI(t, "paper")
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "Live Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
		"mode":     "live",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for live strategy alongside paper (separate containers), got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_SameModeAccepts(t *testing.T) {
	api := setupModeTestAPI(t, "paper")
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "Another Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "ETHUSDT"},
		"mode":     "paper",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for same-mode strategy, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_LiveOnlyLegacyAlias(t *testing.T) {
	api := setupModeTestAPI(t, "paper")
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "Sentinel",
		"exchange": "binance",
		"strategy": "sentinel",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
		"mode":     "paper",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for legacy alias 'sentinel' (→ sentinel_anomaly) in paper mode, got %d: %s", w.Code, w.Body.String())
	}
}

// frontendLiveOnlyStrategies mirrors the liveOnly flags from web/src/lib/bbgo/strategies.ts.
// Keep this list in sync with the frontend. Test below validates consistency.
var frontendLiveOnlyStrategies = map[string]bool{
	"bollmaker": true, "linregmaker": true, "rsmaker": true, "scmaker": true,
	"supertrend": true, "dca2": true, "dca3": true, "wall": true,
	"sentinel_anomaly": true, "audacitymaker": true, "liquiditymaker": true,
	"drift": true, "elliottwave": true, "factorzoo": true, "xvs": true,
	"autoborrow": true, "convert": true, "deposit2transfer": true,
	"autobuy_scheduled": true, "rebalance_portfolio": true, "support": true,
	"xpremium": true, "xnav": true,
}

func TestLiveOnlyLists_FrontendBackendSync(t *testing.T) {
	t.Helper()
	for strategy := range liveOnlyStrategies {
		found := false
		for feID := range frontendLiveOnlyStrategies {
			resolved, _ := normalizeStrategyConfig(feID, nil)
			if resolved == strategy {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("backend liveOnly strategy %q has no matching frontend entry", strategy)
		}
	}
	for feID := range frontendLiveOnlyStrategies {
		resolved, _ := normalizeStrategyConfig(feID, nil)
		if !liveOnlyStrategies[resolved] {
			t.Errorf("frontend liveOnly strategy %q (resolves to %q) missing from backend liveOnlyStrategies", feID, resolved)
		}
	}
}

func TestAPI_CreateStrategy_EmptyNameRejected(t *testing.T) {
	api := setupModeTestAPI(t, "paper")
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
		"mode":     "paper",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_CreateStrategy_InvalidModeRejected(t *testing.T) {
	api := setupModeTestAPI(t, "")
	r := testRouter(api)

	for _, mode := range []string{"test123", "PAPER", "LIVE", "demo", "backtest"} {
		t.Run(mode, func(t *testing.T) {
			body := map[string]interface{}{
				"name":     "test",
				"exchange": "binance",
				"strategy": "grid2",
				"config":   map[string]interface{}{"symbol": "BTCUSDT"},
				"mode":     mode,
			}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for invalid mode %q, got %d: %s", mode, w.Code, w.Body.String())
			}
		})
	}
}

func TestAPI_CreateStrategy_NoModeWithExistingMode(t *testing.T) {
	api := setupModeTestAPI(t, "live")
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "No Mode Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", bytes.NewReader(b))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 when empty mode (defaults to paper) alongside live (separate containers), got %d: %s", w.Code, w.Body.String())
	}
}
