package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

const testEncryptionKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

const testUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

// newTestStore creates an InstanceStore backed by a temp directory.
func newTestStore(t *testing.T) (*InstanceStore, string) {
	t.Helper()
	dir := t.TempDir()
	return NewInstanceStore(dir, testRegistry), dir
}

// createTestInstance creates a strategy instance in the store for testing.
func createTestInstance(t *testing.T, store *InstanceStore, userID, mode, strategy, symbol string, config map[string]any) *StrategyInstance {
	t.Helper()
	if config == nil && strategy == "grid2" {
		config = map[string]any{"gridNumber": 10, "upperPrice": "65000", "lowerPrice": "58000"}
	}
	rawConfig := rawJSON("{}")
	if config != nil {
		if b, err := json.Marshal(config); err == nil {
			rawConfig = b
		}
	}
	inst := &StrategyInstance{
		UserID:   userID,
		Mode:     mode,
		Strategy: strategy,
		Exchange: "binance",
		Symbol:   symbol,
		Config:   rawConfig,
		Name:     strategy + "-" + symbol,
	}
	inst.InstanceID = computeInstanceID(strategy, symbol, rawConfig)
	if err := store.CreateInstance(inst, func(string) bool { return false }); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	return inst
}

func testRouter(api *API) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)
	return r
}

func serveJSON(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}

func doRequest(r chi.Router, method, path string, body any) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("X-Manager-Token", "test-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func rawJSON(s string) []byte {
	return []byte(s)
}

// botFromStrategy builds a Bot from a bbgo strategy state map (test helper).
func botFromStrategy(s BBGoStrategyState, mode string) Bot {
	id, _ := s["strategyInstanceID"].(string)
	strategy, _ := s["strategy"].(string)

	var exchange string
	if on, ok := s["on"].([]any); ok && len(on) > 0 {
		exchange, _ = on[0].(string)
	}

	var symbol string
	var config json.RawMessage
	if cfg, ok := s[strategy]; ok {
		if raw, err := json.Marshal(cfg); err == nil {
			config = raw
		}
		if m, ok := cfg.(map[string]any); ok {
			symbol, _ = m["symbol"].(string)
		}
	}

	state, _ := json.Marshal(s)

	return Bot{
		ID:              id,
		Strategy:        strategy,
		Symbol:          symbol,
		Exchange:        exchange,
		Config:          config,
		State:           state,
		ContainerStatus: StatusRunning,
		Mode:            mode,
	}
}
