package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// HTTP-level tests for PATCH /api/users/{userID}/strategies/{strategyID}.
//
// These guard the behaviors added during the 2026-06-12 review fix pass:
//   - Leverage range validation (1..125) — H2
//   - Non-futures strategy rejection — L1
//   - Merge semantics (PATCH must not clear unrelated FuturesConfig fields) — M2
//   - Stop error surfacing as 503 — H3
//   - Restart orchestration when instance is running — H1
//
// The leverage-propagation unit tests in leverage_chain_test.go cover the
// buildInstanceYAML internals; these tests cover the HTTP handler contract.

const (
	pivotshortTestUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	grid2TestUUID      = "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"
)

// setupUpdateStrategyAPI spins up an API with a known futures instance
// (pivotshort, requiresFutures=true) and a known spot instance (grid2).
// Both use ModePaper so we don't need credentials. checkRunningFn is wired
// to `running` so individual tests can toggle container state.
func setupUpdateStrategyAPI(t *testing.T, running bool) (*API, *chi.Mux, *InstanceStore) {
	t.Helper()
	store, dir := newTestStore(t)
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(name string) (bool, error) { return running, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }
	api := NewAPI(cfg, store, cm, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	r := testRouter(api)

	// Futures strategy instance — pivotshort is in testRegistry.requiresFutures.
	psCfg := rawJSON(`{"symbol":"BTCUSDT","interval":"1h","quantity":0.001}`)
	psInst := &StrategyInstance{
		UserID:   pivotshortTestUUID,
		Mode:     ModePaper,
		Strategy: "pivotshort",
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Config:   psCfg,
		Name:     "ps-test",
		FuturesConfig: &FuturesConfig{
			Leverage:   5,
			MarginType: "cross",
		},
	}
	psInst.InstanceID = computeInstanceID(psInst.Strategy, psInst.Symbol, psCfg)
	if err := store.CreateInstance(psInst, func(string) bool { return false }); err != nil {
		t.Fatalf("create pivotshort instance: %v", err)
	}

	// Spot strategy instance — grid2 is NOT in requiresFutures.
	createTestInstance(t, store, grid2TestUUID, ModePaper, "grid2", "BTCUSDT", map[string]any{"gridNumber": 5})

	return api, r, store
}

func firstInstanceID(t *testing.T, store *InstanceStore, userID, mode, strategy string) string {
	t.Helper()
	instances, _ := store.ListInstances(userID, mode)
	for _, in := range instances {
		if in.Strategy == strategy {
			return in.InstanceID
		}
	}
	t.Fatalf("no %s instance for user %s", strategy, userID)
	return ""
}

func TestUpdateStrategy_Success_PaperNotRunning(t *testing.T) {
	_, r, store := setupUpdateStrategyAPI(t, false)
	instanceID := firstInstanceID(t, store, pivotshortTestUUID, ModePaper, "pivotshort")

	body := map[string]any{
		"futuresConfig": map[string]any{"leverage": 10, "marginType": "isolated"},
	}
	path := "/api/users/" + pivotshortTestUUID + "/strategies/" + instanceID + "?mode=paper"
	w := doRequest(r, "PATCH", path, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		InstanceID    string         `json:"instance_id"`
		Status        string         `json:"status"`
		FuturesConfig *FuturesConfig `json:"futuresConfig"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != StatusStopped {
		t.Errorf("status: got %q, want %q (wasRunning=false)", resp.Status, StatusStopped)
	}
	if resp.FuturesConfig == nil || resp.FuturesConfig.Leverage != 10 {
		t.Errorf("FuturesConfig.Leverage: got %+v, want 10", resp.FuturesConfig)
	}
	if resp.FuturesConfig.MarginType != "isolated" {
		t.Errorf("FuturesConfig.MarginType: got %q, want %q", resp.FuturesConfig.MarginType, "isolated")
	}
}

func TestUpdateStrategy_RejectsInvalidLeverage(t *testing.T) {
	_, r, store := setupUpdateStrategyAPI(t, false)
	instanceID := firstInstanceID(t, store, pivotshortTestUUID, ModePaper, "pivotshort")
	path := "/api/users/" + pivotshortTestUUID + "/strategies/" + instanceID + "?mode=paper"

	cases := []struct {
		name   string
		body   map[string]any
		errMsg string
	}{
		{
			name:   "negative",
			body:   map[string]any{"futuresConfig": map[string]any{"leverage": -1}},
			errMsg: "leverage must be between 1 and 125",
		},
		{
			name:   "too large",
			body:   map[string]any{"futuresConfig": map[string]any{"leverage": 1000}},
			errMsg: "leverage must be between 1 and 125",
		},
		{
			name:   "nil futuresConfig",
			body:   map[string]any{},
			errMsg: "config, futuresConfig, or riskConfig is required",
		},
		{
			name:   "invalid marginType",
			body:   map[string]any{"futuresConfig": map[string]any{"leverage": 5, "marginType": "weird"}},
			errMsg: "marginType must be",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(r, "PATCH", path, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status: got %d, want 400. body: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.errMsg) {
				t.Errorf("body: got %q, want substring %q", w.Body.String(), tc.errMsg)
			}
		})
	}
}

func TestUpdateStrategy_RejectsNonFuturesStrategy(t *testing.T) {
	_, r, store := setupUpdateStrategyAPI(t, false)
	instanceID := firstInstanceID(t, store, grid2TestUUID, ModePaper, "grid2")
	path := "/api/users/" + grid2TestUUID + "/strategies/" + instanceID + "?mode=paper"

	body := map[string]any{"futuresConfig": map[string]any{"leverage": 5}}
	w := doRequest(r, "PATCH", path, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400 (non-futures). body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "does not support futures") {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestUpdateStrategy_MergeSemanticsPreservesExistingFields(t *testing.T) {
	_, r, store := setupUpdateStrategyAPI(t, false)
	instanceID := firstInstanceID(t, store, pivotshortTestUUID, ModePaper, "pivotshort")
	path := "/api/users/" + pivotshortTestUUID + "/strategies/" + instanceID + "?mode=paper"

	// Initial FuturesConfig on the instance is {Leverage:5, MarginType:"cross"}.
	// PATCH only leverage — marginType must be preserved by merge semantics.
	body := map[string]any{"futuresConfig": map[string]any{"leverage": 8}}
	w := doRequest(r, "PATCH", path, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		FuturesConfig *FuturesConfig `json:"futuresConfig"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.FuturesConfig.Leverage != 8 {
		t.Errorf("leverage: got %d, want 8", resp.FuturesConfig.Leverage)
	}
	if resp.FuturesConfig.MarginType != "cross" {
		t.Errorf("marginType: got %q, want %q (must be preserved by merge)", resp.FuturesConfig.MarginType, "cross")
	}
}

func TestUpdateStrategy_StopFailureSurfacesAs503(t *testing.T) {
	api, r, store := setupUpdateStrategyAPI(t, true /* running */)
	api.container.dockerFn = func(args ...string) (string, error) {
		if len(args) > 0 && args[0] == "stop" {
			return "docker daemon down", errFake
		}
		return "", nil
	}

	instanceID := firstInstanceID(t, store, pivotshortTestUUID, ModePaper, "pivotshort")
	path := "/api/users/" + pivotshortTestUUID + "/strategies/" + instanceID + "?mode=paper"

	body := map[string]any{"futuresConfig": map[string]any{"leverage": 7}}
	w := doRequest(r, "PATCH", path, body)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503 (stop failed). body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "failed to stop running container") {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestUpdateStrategy_RestartsRunningInstance(t *testing.T) {
	api, r, store := setupUpdateStrategyAPI(t, true /* running */)

	var stopCalled, rmCalled bool
	api.container.dockerFn = func(args ...string) (string, error) {
		if len(args) > 0 {
			switch args[0] {
			case "stop":
				stopCalled = true
			case "rm":
				rmCalled = true
			}
		}
		return "", nil
	}

	instanceID := firstInstanceID(t, store, pivotshortTestUUID, ModePaper, "pivotshort")

	// Pre-seed api.starting with a stale flag to simulate the H1 race scenario.
	// The handler must clear it before stop, otherwise the restart goroutine
	// never spawns (LoadOrStore returns loaded=true and skips the start).
	api.starting.Store(instanceID, true)

	path := "/api/users/" + pivotshortTestUUID + "/strategies/" + instanceID + "?mode=paper"
	body := map[string]any{"futuresConfig": map[string]any{"leverage": 7}}
	w := doRequest(r, "PATCH", path, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d, body: %s", w.Code, w.Body.String())
	}

	if !stopCalled || !rmCalled {
		t.Errorf("expected docker stop+rm to be invoked; stop=%v rm=%v", stopCalled, rmCalled)
	}

	var resp struct {
		Status string `json:"status"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != StatusStarting {
		t.Errorf("status: got %q, want %q (wasRunning=true)", resp.Status, StatusStarting)
	}
}

// TestUpdateStrategy_ConfigPatch covers the general config-patch path added
// when abstracting UpdateStrategy to accept arbitrary strategy params (e.g.
// fixing a "spread too small" grid2 config without delete+recreate).
func TestUpdateStrategy_ConfigPatch(t *testing.T) {
	_, r, store := setupUpdateStrategyAPI(t, false)
	instanceID := firstInstanceID(t, store, grid2TestUUID, ModePaper, "grid2")
	path := "/api/users/" + grid2TestUUID + "/strategies/" + instanceID + "?mode=paper"

	body := map[string]any{
		"config": map[string]any{
			"gridNumber":  3,
			"upperPrice":  80000,
			"lowerPrice":  60000,
			"quantity":    0.01,
			"nested":      map[string]any{"deep": map[string]any{"value": 7}},
		},
	}
	w := doRequest(r, "PATCH", path, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v, ok := resp.Config["gridNumber"].(float64); !ok || v != 3 {
		t.Errorf("config.gridNumber: got %v, want 3", resp.Config["gridNumber"])
	}
	if v, ok := resp.Config["symbol"].(string); !ok || v != "BTCUSDT" {
		t.Errorf("config.symbol (preserved): got %v, want BTCUSDT", resp.Config["symbol"])
	}
	nested, ok := resp.Config["nested"].(map[string]any)
	if !ok {
		t.Errorf("config.nested should be an object, got %T", resp.Config["nested"])
	} else if deep, ok := nested["deep"].(map[string]any); !ok || deep["value"].(float64) != 7 {
		t.Errorf("config.nested.deep.value missing or wrong: %v", nested["deep"])
	}
}

// TestUpdateStrategy_ConfigPatch_RejectsSymbolChange verifies the
// instance-ID-stability invariant: changing symbol via patch must be rejected
// because the deterministic instance ID is derived from (strategy, symbol,
// config) and rewriting it would orphan historical trades under the old ID.
func TestUpdateStrategy_ConfigPatch_RejectsSymbolChange(t *testing.T) {
	_, r, store := setupUpdateStrategyAPI(t, false)
	instanceID := firstInstanceID(t, store, grid2TestUUID, ModePaper, "grid2")
	path := "/api/users/" + grid2TestUUID + "/strategies/" + instanceID + "?mode=paper"

	body := map[string]any{"config": map[string]any{"symbol": "ETHUSDT"}}
	w := doRequest(r, "PATCH", path, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "cannot change symbol") {
		t.Errorf("body: got %q, want substring 'cannot change symbol'", w.Body.String())
	}
}
