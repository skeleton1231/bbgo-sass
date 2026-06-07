package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// --- CreateStrategy API handler ---

func setupInstanceAPI(t *testing.T) (*API, *chi.Mux) {
	t.Helper()
	store, dir := newTestStore(t)
	cfg := &Config{ManagerToken: "test-token", DataDir: dir}
	cm := NewContainerManager(cfg, nil, nil, store)
	cm.checkRunningFn = func(name string) (bool, error) { return false, nil }
	cm.dockerFn = func(args ...string) (string, error) { return "", nil }
	api := NewAPI(cfg, store, cm, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	r := testRouter(api)
	return api, r
}

func TestCreateStrategy_Success(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{
		"name":     "My Grid",
		"strategy": "grid2",
		"exchange": "binance",
		"config":   map[string]any{"gridNumber": 5},
		"mode":     "paper",
	}
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: %d, body: %s", w.Code, w.Body.String())
	}
	var resp instanceInfo
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Strategy != "grid2" {
		t.Errorf("Strategy: got %q", resp.Strategy)
	}
	if resp.Mode != "paper" {
		t.Errorf("Mode: got %q", resp.Mode)
	}
	if resp.Status != "starting" {
		t.Errorf("Status: got %q", resp.Status)
	}
	if resp.Name != "My Grid" {
		t.Errorf("Name: got %q", resp.Name)
	}
}

func TestCreateStrategy_MissingStrategy(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{"name": "test", "exchange": "binance", "mode": "paper"}
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateStrategy_MissingName(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{"strategy": "grid2", "exchange": "binance", "mode": "paper"}
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateStrategy_MissingExchange(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{"name": "test", "strategy": "grid2", "mode": "paper"}
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateStrategy_InvalidMode(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{"name": "test", "strategy": "grid2", "exchange": "binance", "mode": "invalid"}
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateStrategy_DefaultModeIsPaper(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{"name": "test", "strategy": "grid2", "exchange": "binance"}
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: %d, body: %s", w.Code, w.Body.String())
	}
	var resp instanceInfo
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Mode != "paper" {
		t.Errorf("default mode should be paper, got %q", resp.Mode)
	}
}

func TestCreateStrategy_DuplicateInstance(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{
		"name": "test", "strategy": "grid2", "exchange": "binance", "mode": "paper",
	}
	w1 := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first create: %d", w1.Code)
	}
	w2 := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate should be 409, got %d", w2.Code)
	}
}

// --- ListStrategies API handler ---

func TestListStrategies_Empty(t *testing.T) {
	_, r := setupInstanceAPI(t)
	w := doRequest(r, "GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	instances := resp["instances"].([]any)
	if len(instances) != 0 {
		t.Errorf("expected empty instances, got %d", len(instances))
	}
}

func TestListStrategies_AfterCreate(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{
		"name": "test", "strategy": "grid2", "exchange": "binance", "mode": "paper",
	}
	doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	w := doRequest(r, "GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	instances := resp["instances"].([]any)
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}
	inst := instances[0].(map[string]any)
	if inst["strategy"] != "grid2" {
		t.Errorf("strategy: got %v", inst["strategy"])
	}
}

// --- DeleteStrategy API handler ---

func TestDeleteStrategy_Success(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{
		"name": "test", "strategy": "grid2", "exchange": "binance", "mode": "paper",
	}
	doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)

	w := doRequest(r, "GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	var listResp map[string]any
	json.NewDecoder(w.Body).Decode(&listResp)
	inst := listResp["instances"].([]any)[0].(map[string]any)
	instanceID := inst["instance_id"].(string)

	w = doRequest(r, "DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/"+instanceID+"?mode=paper", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status: %d, body: %s", w.Code, w.Body.String())
	}

	w = doRequest(r, "GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	json.NewDecoder(w.Body).Decode(&listResp)
	instances := listResp["instances"].([]any)
	if len(instances) != 0 {
		t.Errorf("expected empty after delete, got %d", len(instances))
	}
}

func TestDeleteStrategy_NotFound(t *testing.T) {
	_, r := setupInstanceAPI(t)
	w := doRequest(r, "DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies/nonexistent?mode=paper", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- StartInstance / StopInstance API handlers ---

func TestStartInstance_Success(t *testing.T) {
	api, r := setupInstanceAPI(t)
	body := map[string]any{
		"name": "test", "strategy": "grid2", "exchange": "binance", "mode": "paper",
	}
	doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)

	w := doRequest(r, "GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	var listResp map[string]any
	json.NewDecoder(w.Body).Decode(&listResp)
	inst := listResp["instances"].([]any)[0].(map[string]any)
	instanceID := inst["instance_id"].(string)

	api.starting.Delete(instanceID)

	w = doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/instances/"+instanceID+"/start", nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("start status: %d, body: %s", w.Code, w.Body.String())
	}
	var resp instanceInfo
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "starting" {
		t.Errorf("status: got %q", resp.Status)
	}
}

func TestStartInstance_NotFound(t *testing.T) {
	_, r := setupInstanceAPI(t)
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/instances/nonexistent/start", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestStopInstance_Success(t *testing.T) {
	_, r := setupInstanceAPI(t)
	body := map[string]any{
		"name": "test", "strategy": "grid2", "exchange": "binance", "mode": "paper",
	}
	doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)

	w := doRequest(r, "GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	var listResp map[string]any
	json.NewDecoder(w.Body).Decode(&listResp)
	inst := listResp["instances"].([]any)[0].(map[string]any)
	instanceID := inst["instance_id"].(string)

	w = doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/instances/"+instanceID+"/stop", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("stop status: %d, body: %s", w.Code, w.Body.String())
	}
}

func TestStopInstance_NotFound(t *testing.T) {
	_, r := setupInstanceAPI(t)
	w := doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/instances/nonexistent/stop", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- ClearAllStrategies ---

func TestClearAllStrategies(t *testing.T) {
	_, r := setupInstanceAPI(t)
	for i := 0; i < 3; i++ {
		body := map[string]any{
			"name": "test", "strategy": "grid2", "exchange": "binance", "mode": "paper",
			"config": map[string]any{"gridNumber": i + 1},
		}
		doRequest(r, "POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", body)
	}

	w := doRequest(r, "DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies?mode=paper", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("clear status: %d", w.Code)
	}

	w = doRequest(r, "GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/strategies", nil)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	instances := resp["instances"].([]any)
	if len(instances) != 0 {
		t.Errorf("expected empty after clear, got %d", len(instances))
	}
}
