package main

import (
	"encoding/json"
	"testing"
)

func TestTypedResponseJSON(t *testing.T) {
	t.Run("healthResponse", func(t *testing.T) {
		resp := healthResponse{Status: "ok", Users: 3, Running: 2}
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		json.Unmarshal(b, &m)
		if m["status"] != "ok" {
			t.Errorf("status: got %v", m["status"])
		}
		if m["users"] != float64(3) {
			t.Errorf("users: got %v", m["users"])
		}
		if m["running"] != float64(2) {
			t.Errorf("running: got %v", m["running"])
		}
	})

	t.Run("containerStatusResponse", func(t *testing.T) {
		resp := containerStatusResponse{
			UserID: "user-1",
			Containers: map[string]*containerInfo{
				"live": {UserID: "user-1", Mode: "live", Status: "running"},
			},
		}
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		json.Unmarshal(b, &m)
		if m["user_id"] != "user-1" {
			t.Errorf("user_id: got %v", m["user_id"])
		}
		containers, _ := m["containers"].(map[string]any)
		if containers == nil {
			t.Fatal("missing containers map")
		}
		live, _ := containers["live"].(map[string]any)
		if live["status"] != "running" {
			t.Errorf("live.status: got %v", live["status"])
		}
	})

	t.Run("credentialResponse", func(t *testing.T) {
		resp := credentialResponse{
			ID: "cred-1", UserID: "user-1", Exchange: "binance",
			IsTestnet: false, IsVerified: true, VerifyError: "",
		}
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		json.Unmarshal(b, &m)
		if m["is_verified"] != true {
			t.Errorf("is_verified: got %v", m["is_verified"])
		}
		if _, exists := m["verify_error"]; exists {
			t.Error("verify_error should be omitted when empty")
		}
	})

	t.Run("credentialResponse_WithVerifyError", func(t *testing.T) {
		resp := credentialResponse{
			ID: "cred-1", UserID: "user-1", Exchange: "binance",
			IsVerified: false, VerifyError: "invalid signature",
		}
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		json.Unmarshal(b, &m)
		if m["verify_error"] != "invalid signature" {
			t.Errorf("verify_error: got %v", m["verify_error"])
		}
	})

	t.Run("klineEntry", func(t *testing.T) {
		entry := klineEntry{
			Time: 1700000000, Open: "42000.5", High: "42500.0",
			Low: "41800.0", Close: "42300.0", Volume: "1234.5",
			QuoteVolume: "50000000.0", Closed: true,
		}
		b, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		json.Unmarshal(b, &m)
		if m["time"] != float64(1700000000) {
			t.Errorf("time: got %v", m["time"])
		}
		if m["open"] != "42000.5" {
			t.Errorf("open: got %v", m["open"])
		}
		if m["closed"] != true {
			t.Errorf("closed: got %v", m["closed"])
		}
	})
}

func TestBotJSONRoundtrip(t *testing.T) {
	configJSON := `{"symbol":"BTCUSDT","gridNumber":10}`
	stateJSON := `{"strategy":"grid2","symbol":"BTCUSDT"}`

	bot := Bot{
		ID:              "grid2-BTCUSDT",
		Strategy:        "grid2",
		Symbol:          "BTCUSDT",
		Exchange:        "binance",
		Session:         "binance",
		Config:          json.RawMessage(configJSON),
		State:           json.RawMessage(stateJSON),
		ContainerStatus: "running",
		Mode:            "live",
	}

	b, err := json.Marshal(bot)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	json.Unmarshal(b, &parsed)

	if parsed["id"] != "grid2-BTCUSDT" {
		t.Errorf("id: got %v", parsed["id"])
	}

	cfg, _ := parsed["config"].(map[string]any)
	if cfg == nil {
		t.Error("config should be a JSON object")
	} else if cfg["symbol"] != "BTCUSDT" {
		t.Errorf("config.symbol: got %v", cfg["symbol"])
	}

	state, _ := parsed["state"].(map[string]any)
	if state == nil {
		t.Error("state should be a JSON object")
	} else if state["strategy"] != "grid2" {
		t.Errorf("state.strategy: got %v", state["strategy"])
	}
}

func TestBotFromStrategy_ConfigIsRawMessage(t *testing.T) {
	s := BBGoStrategyState{
		"strategy":           "grid2",
		"strategyInstanceID": "grid2-BTCUSDT",
		"on":                 []any{"binance"},
		"grid2": map[string]any{
			"symbol":     "BTCUSDT",
			"gridNumber": float64(10),
		},
	}

	bot := botFromStrategy(s, "live")

	if bot.Strategy != "grid2" {
		t.Errorf("Strategy: got %q", bot.Strategy)
	}
	if bot.Symbol != "BTCUSDT" {
		t.Errorf("Symbol: got %q", bot.Symbol)
	}

	var cfg map[string]any
	if err := json.Unmarshal(bot.Config, &cfg); err != nil {
		t.Fatalf("Config unmarshal: %v", err)
	}
	if cfg["symbol"] != "BTCUSDT" {
		t.Errorf("Config.symbol: got %v", cfg["symbol"])
	}
	if cfg["gridNumber"] != float64(10) {
		t.Errorf("Config.gridNumber: got %v", cfg["gridNumber"])
	}

	var state map[string]any
	if err := json.Unmarshal(bot.State, &state); err != nil {
		t.Fatalf("State unmarshal: %v", err)
	}
	if state["strategy"] != "grid2" {
		t.Errorf("State.strategy: got %v", state["strategy"])
	}
}
