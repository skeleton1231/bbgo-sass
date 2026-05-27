package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateStrategy_EmptyMode_DefaultsToPaper verifies that omitting mode
// defaults to "paper" so all validation (liveOnly, credentials) still runs.
func TestCreateStrategy_EmptyMode_DefaultsToPaper(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "No Mode Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", bytes.NewReader(b))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for empty-mode strategy (defaults to paper), got %d: %s", w.Code, w.Body.String())
	}

	uc, _ := api.users.Get(userID, ModePaper)
	if uc == nil || len(uc.Strategies) < 1 {
		t.Fatalf("expected 1 strategy in paper container, got %d", func() int {
			if uc == nil {
				return 0
			}
			return len(uc.Strategies)
		}())
	}
	var newStrat *StrategyEntry
	for i := range uc.Strategies {
		if uc.Strategies[i].Name == "No Mode Grid" {
			newStrat = &uc.Strategies[i]
		}
	}
	if newStrat == nil {
		t.Fatal("new strategy not found")
	}
	if newStrat.Mode != "paper" {
		t.Errorf("expected mode to default to 'paper', got %q", newStrat.Mode)
	}
}

// TestCreateStrategy_EmptyMode_LiveOnlyRejected verifies that liveOnly
// strategies are rejected even when mode is omitted (because it defaults to paper).
func TestCreateStrategy_EmptyMode_LiveOnlyRejected(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	r := testRouter(api)

	for _, strategy := range []string{"bollmaker", "supertrend", "dca2"} {
		t.Run(strategy, func(t *testing.T) {
			body := map[string]interface{}{
				"name":     "No Mode LiveOnly",
				"exchange": "binance",
				"strategy": strategy,
				"config":   map[string]interface{}{"symbol": "BTCUSDT"},
			}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", bytes.NewReader(b))
			req.Header.Set("X-User-Id", userID)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for liveOnly strategy without mode (defaults to paper), got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

// TestCreateStrategy_EmptyMode_MixedWithExistingLiveAllowed verifies that
// adding a strategy without mode (defaults to paper) is allowed when existing
// strategies are live.
func TestCreateStrategy_EmptyMode_MixedWithExistingLiveAllowed(t *testing.T) {
	api, cleanup := setupTestAPIWithCreds(t)
	defer cleanup()

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	api.users.users[userID+":"+ModeLive].Strategies = []StrategyEntry{
		{ID: "s1", Exchange: "binance", Strategy: "grid", Mode: "live"},
	}

	r := testRouter(api)

	body := map[string]interface{}{
		"name":     "No Mode Grid",
		"exchange": "binance",
		"strategy": "grid2",
		"config":   map[string]interface{}{"symbol": "BTCUSDT"},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", bytes.NewReader(b))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 when empty-mode (paper) alongside live - separate containers, got %d: %s", w.Code, w.Body.String())
	}
}

// TestEnvArgs_EmptyModeStrategy_ProducesPaperTrade verifies that a strategy
// with empty mode (now stored as "paper") produces PAPER_TRADE=1 in Docker env.
func TestEnvArgs_EmptyModeStrategy_ProducesPaperTrade(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	cm := &ContainerManager{cfg: &Config{DataDir: tmpDir, BBGOPort: 8080, BBGOGRPCPort: 9090}, creds: creds}

	uc := &UserContainer{

		Mode: ModePaper,

		UserID: "test-user",

		Strategies: []StrategyEntry{

			{Exchange: "binance", Strategy: "grid", Mode: "paper"},
		},
	}

	args := cm.envArgs(uc)

	hasPaper := false
	for i := range args {
		if args[i] == "PAPER_TRADE=1" {
			hasPaper = true
			break
		}
	}
	if !hasPaper {
		t.Error("expected PAPER_TRADE=1 in env args for strategy with defaulted paper mode")
	}
}

// TestBuildUserYAML_EmptyModeStrategy_ProducesPaperTrade verifies YAML generation
// treats empty-mode (now stored as "paper") correctly with paperTrade: "1".
func TestBuildUserYAML_EmptyModeStrategy_ProducesPaperTrade(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModePaper,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{ID: "s1", Exchange: "binance", Strategy: "grid", Mode: "paper", Config: json.RawMessage(`{"symbol":"BTCUSDT"}`)},
		},
	}

	yaml, err := buildUserYAML(uc, func(exchange string) bool {
		return false
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(yaml), "PAPER_TRADE") {
		t.Errorf("expected PAPER_TRADE in YAML for paper-mode strategy, got:\n%s", string(yaml))
	}
}
