package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TDD verification for the universal risk controller feature. These tests
// exercise the full SaaS path from API request through to the env vars
// injected into the bbgo container. They complement the struct-level tests
// in risk_config_test.go by proving the wire-up.

func TestCreateStrategy_PersistsRiskConfig(t *testing.T) {
	api, r := setupInstanceAPI(t)

	// Hook the Supabase upsert path so we can capture what would be persisted
	// (file-only test store doesn't round-trip RiskConfig — that lives in
	// Supabase, mirroring FuturesConfig).
	var captured *StrategyInstance
	api.store.supabaseUpsertFn = func(inst *StrategyInstance) {
		captured = inst
	}

	body := map[string]any{
		"name":     "risk-test",
		"strategy": "grid2",
		"exchange": "binance",
		"config":   map[string]any{"gridNumber": 5},
		"mode":     "paper",
		"riskConfig": map[string]any{
			"stopLossPrice":   19000,
			"takeProfitPrice": 22000,
			"maxPositionQty":  3,
		},
	}
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	w := doRequest(r, "POST", "/api/users/"+userID+"/strategies", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: %d, body: %s", w.Code, w.Body.String())
	}

	if captured == nil {
		t.Fatal("upsertToSupabase was not called")
	}
	if captured.RiskConfig == nil {
		t.Fatal("upserted instance has nil RiskConfig")
	}
	if captured.RiskConfig.StopLossPrice != 19000 {
		t.Errorf("StopLossPrice: got %v, want 19000", captured.RiskConfig.StopLossPrice)
	}
	if captured.RiskConfig.TakeProfitPrice != 22000 {
		t.Errorf("TakeProfitPrice: got %v, want 22000", captured.RiskConfig.TakeProfitPrice)
	}
	if captured.RiskConfig.MaxPositionQty != 3 {
		t.Errorf("MaxPositionQty: got %v, want 3", captured.RiskConfig.MaxPositionQty)
	}
}

func TestCreateStrategy_RejectsInvalidRiskConfig(t *testing.T) {
	_, r := setupInstanceAPI(t)

	body := map[string]any{
		"name":     "bad-risk",
		"strategy": "grid2",
		"exchange": "binance",
		"config":   map[string]any{"gridNumber": 5},
		"mode":     "paper",
		"riskConfig": map[string]any{
			"stopLossPrice": -100,
		},
	}
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	w := doRequest(r, "POST", "/api/users/"+userID+"/strategies", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative stopLossPrice, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestStartInstance_InjectsRiskEnvVars proves the highest-risk wiring:
// creating a strategy with risk_config causes the resulting docker
// invocation to contain the BBGO_UNIVERSAL_RISK_* env vars.
func TestStartInstance_InjectsRiskEnvVars(t *testing.T) {
	api, r := setupInstanceAPI(t)

	body := map[string]any{
		"name":     "env-test",
		"strategy": "grid2",
		"exchange": "binance",
		"config":   map[string]any{"gridNumber": 5},
		"mode":     "paper",
		"riskConfig": map[string]any{
			"stopLossPrice":      19000,
			"takeProfitPrice":    22000,
			"roiStopLoss":        0.05,
			"roiTakeProfit":      0.10,
			"trailingActivation": 0.03,
			"trailingCallback":   0.02,
			"maxPositionQty":     3,
		},
	}
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	w := doRequest(r, "POST", "/api/users/"+userID+"/strategies", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status: %d, body: %s", w.Code, w.Body.String())
	}
	var create struct {
		InstanceID string `json:"instance_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &create)

	var capturedArgs []string
	api.container.dockerFn = func(args ...string) (string, error) {
		capturedArgs = append(capturedArgs, args...)
		return "", nil
	}

	startPath := "/api/users/" + userID + "/instances/" + create.InstanceID + "/start?mode=paper"
	w = doRequest(r, "POST", startPath, nil)
	if w.Code != http.StatusAccepted && w.Code != http.StatusOK {
		t.Fatalf("start status: %d, body: %s", w.Code, w.Body.String())
	}

	if len(capturedArgs) == 0 {
		t.Fatal("dockerFn was not invoked on start")
	}

	wantEnvs := []string{
		"BBGO_UNIVERSAL_RISK_STOP_LOSS_PRICE=19000",
		"BBGO_UNIVERSAL_RISK_TAKE_PROFIT_PRICE=22000",
		"BBGO_UNIVERSAL_RISK_ROI_STOP_LOSS=0.05",
		"BBGO_UNIVERSAL_RISK_ROI_TAKE_PROFIT=0.1",
		"BBGO_UNIVERSAL_RISK_TRAILING_ACTIVATION=0.03",
		"BBGO_UNIVERSAL_RISK_TRAILING_CALLBACK=0.02",
		"BBGO_UNIVERSAL_RISK_MAX_POSITION_QTY=3",
	}
	for _, want := range wantEnvs {
		found := false
		for i := 0; i < len(capturedArgs)-1; i++ {
			if capturedArgs[i] == "-e" && capturedArgs[i+1] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing env entry %q in docker args:\n%s", want, strings.Join(capturedArgs, " "))
		}
	}
}

func TestStartInstance_NoRiskConfig_NoEnvVars(t *testing.T) {
	api, r := setupInstanceAPI(t)

	body := map[string]any{
		"name":     "no-risk",
		"strategy": "grid2",
		"exchange": "binance",
		"config":   map[string]any{"gridNumber": 5},
		"mode":     "paper",
	}
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	w := doRequest(r, "POST", "/api/users/"+userID+"/strategies", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status: %d, body: %s", w.Code, w.Body.String())
	}
	var create struct {
		InstanceID string `json:"instance_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &create)

	var capturedArgs []string
	api.container.dockerFn = func(args ...string) (string, error) {
		capturedArgs = append(capturedArgs, args...)
		return "", nil
	}

	startPath := "/api/users/" + userID + "/instances/" + create.InstanceID + "/start?mode=paper"
	w = doRequest(r, "POST", startPath, nil)
	if w.Code != http.StatusAccepted && w.Code != http.StatusOK {
		t.Fatalf("start status: %d, body: %s", w.Code, w.Body.String())
	}

	for i := 0; i < len(capturedArgs)-1; i++ {
		if capturedArgs[i] == "-e" && strings.HasPrefix(capturedArgs[i+1], "BBGO_UNIVERSAL_RISK_") {
			t.Errorf("expected no BBGO_UNIVERSAL_RISK_* env vars without riskConfig, got %s", capturedArgs[i+1])
		}
	}
}
