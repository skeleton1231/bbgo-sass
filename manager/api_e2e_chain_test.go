package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ============================================================================
// E2E Chain Tests: Complete Live & Paper Trading Flow
// ============================================================================

// TestLiveTradingChain traces: create credential → create live strategy → verify YAML has no PAPER_TRADE and session has PublicOnly=false
func TestLiveTradingChain(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(dir, enc)

	// Step 1: User stores binance API credentials
	apiKeyEnc, _ := enc.Encrypt("real-api-key")
	apiSecretEnc, _ := enc.Encrypt("real-api-secret")
	creds.Upsert(ExchangeCredential{
		ID:                 "cred-1",
		UserID:             "user-1",
		Exchange:           "binance",
		APIKeyEncrypted:    apiKeyEnc,
		APISecretEncrypted: apiSecretEnc,
	})

	// Step 2: User creates a live grid2 strategy
	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				ID:       "s1",
				Name:     "BTC Grid",
				Exchange: "binance",
				Strategy: "grid2",
				Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`),
				Mode:     "live",
			},
		},
	}

	// Step 3: Verify YAML generation for live mode
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool {
		_, _, _, err := creds.GetDecrypted("user-1", exchange)
		return err == nil
	})
	if err != nil {
		t.Fatalf("buildUserYAML: %v", err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("live mode YAML must NOT contain PAPER_TRADE")
	}
	if !strings.Contains(yaml, "binance:") {
		t.Error("expected binance session in YAML")
	}
	if !strings.Contains(yaml, "BINANCE") {
		t.Error("expected BINANCE env var prefix in YAML")
	}
	if strings.Contains(yaml, "publicOnly: true") {
		t.Error("live mode with credentials must NOT set publicOnly=true")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy in YAML")
	}
	if !strings.Contains(yaml, `"on": binance`) {
		t.Error("expected session binding")
	}

	// Step 4: Verify credential decryption works (container would inject env vars)
	apiKey, apiSecret, _, err := creds.GetDecrypted("user-1", "binance")
	if err != nil {
		t.Fatalf("GetDecrypted: %v", err)
	}
	if apiKey != "real-api-key" {
		t.Errorf("apiKey mismatch: got %q", apiKey)
	}
	if apiSecret != "real-api-secret" {
		t.Errorf("apiSecret mismatch: got %q", apiSecret)
	}
}

// TestPaperTradingChain traces: create paper strategy → verify YAML has PAPER_TRADE=1 and publicOnly=true
func TestPaperTradingChain(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				ID:       "s1",
				Name:     "BTC Grid Paper",
				Exchange: "binance",
				Strategy: "grid2",
				Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`),
				Mode:     "paper",
			},
		},
	}

	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool {
		_, _, _, err := creds.GetDecrypted("user-1", exchange)
		return err == nil
	})
	if err != nil {
		t.Fatalf("buildUserYAML: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("paper mode YAML MUST contain PAPER_TRADE")
	}
	if !strings.Contains(yaml, "publicOnly: true") {
		t.Error("paper mode without credentials MUST set publicOnly=true")
	}
	if strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("single-exchange strategy should not have crossExchangeStrategies")
	}
}

// TestCrossExchangeLiveChain verifies cross-exchange strategy with real credentials on both exchanges
func TestCrossExchangeLiveChain(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	// Store credentials for both exchanges
	for _, ex := range []string{"binance", "bybit"} {
		keyEnc, _ := enc.Encrypt(ex + "-key")
		secretEnc, _ := enc.Encrypt(ex + "-secret")
		creds.Upsert(ExchangeCredential{
			ID:                 "cred-" + ex,
			UserID:             "user-1",
			Exchange:           ex,
			APIKeyEncrypted:    keyEnc,
			APISecretEncrypted: secretEnc,
		})
	}

	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "live",
				Config:        rawJSON(`{"symbol":"BTCUSDT","spread":0.001,"quantity":0.001}`),
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "hedge", Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
				},
			},
		},
	}

	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool {
		_, _, _, err := creds.GetDecrypted("user-1", exchange)
		return err == nil
	})
	if err != nil {
		t.Fatalf("buildUserYAML: %v", err)
	}
	yaml := string(yamlBytes)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("live cross-exchange must NOT have PAPER_TRADE")
	}
	if !strings.Contains(yaml, "crossExchangeStrategies:") {
		t.Error("expected crossExchangeStrategies section")
	}
	if !strings.Contains(yaml, "xmaker:") {
		t.Error("expected xmaker strategy")
	}
	if !strings.Contains(yaml, "maker:") {
		t.Error("expected maker session")
	}
	if !strings.Contains(yaml, "hedge:") {
		t.Error("expected hedge session")
	}
	if !strings.Contains(yaml, "futures: true") {
		t.Error("expected futures:true for hedge session")
	}
	if strings.Contains(yaml, "publicOnly: true") {
		t.Error("live mode with credentials should not set publicOnly=true")
	}
	if !strings.Contains(yaml, "BINANCE") {
		t.Error("expected BINANCE env prefix")
	}
	if !strings.Contains(yaml, "BYBIT") {
		t.Error("expected BYBIT env prefix")
	}
}

// TestCrossExchangeEnvPrefixAutoFill verifies that empty EnvVarPrefix is auto-computed from exchange name
func TestCrossExchangeEnvPrefixAutoFill(t *testing.T) {
	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "paper",
				Config:        rawJSON(`{"symbol":"BTCUSDT"}`),
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance"}, // EnvVarPrefix intentionally empty
					{Name: "hedge", Exchange: "bybit", Futures: true}, // EnvVarPrefix intentionally empty
				},
			},
		},
	}

	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return false })
	if err != nil {
		t.Fatalf("buildUserYAML: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "BINANCE") {
		t.Error("expected auto-filled BINANCE env prefix for maker session")
	}
	if !strings.Contains(yaml, "BYBIT") {
		t.Error("expected auto-filled BYBIT env prefix for hedge session")
	}
}

// TestLiveOnlyStrategiesBackendFrontendConsistency verifies that the backend's liveOnly set
// matches the frontend's liveOnly definitions after legacy alias normalization.
func TestLiveOnlyStrategiesBackendFrontendConsistency(t *testing.T) {
	// Frontend liveOnly strategies (from strategies.ts).
	// These are the IDs the frontend sends — some are legacy aliases that normalize to bbgo IDs.
	frontendLiveOnly := map[string]bool{
		"bollmaker": true, "linregmaker": true, "rsmaker": true, "scmaker": true,
		"supertrend": true, "dca2": true, "dca3": true, "wall": true,
		"sentinel": true, "sentinel_anomaly": true,
		"autobuy": true, "autobuy_scheduled": true,
		"rebalance": true, "rebalance_portfolio": true,
		"audacitymaker": true, "liquiditymaker": true,
		"drift": true, "elliottwave": true, "factorzoo": true, "xvs": true,
		"autoborrow": true, "convert": true, "deposit2transfer": true,
		"support": true, "xpremium": true, "xnav": true,
	}

	// Normalize frontend IDs to bbgo IDs and check against backend
	for frontendID := range frontendLiveOnly {
		normalized := frontendID
		if alias, ok := legacyStrategyAliases[frontendID]; ok {
			normalized = alias
		}
		if !liveOnlyStrategies[normalized] {
			t.Errorf("frontend says %q (-> %q) is liveOnly, but backend liveOnlyStrategies does not include %q", frontendID, normalized, normalized)
		}
	}

	// Check reverse: backend liveOnly that frontend doesn't know about
	for strategy := range liveOnlyStrategies {
		found := frontendLiveOnly[strategy]
		if !found {
			for alias, target := range legacyStrategyAliases {
				if target == strategy && frontendLiveOnly[alias] {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("backend says %q is liveOnly, but frontend does not mark it (may cause confusing UX: frontend allows paper, backend rejects)", strategy)
		}
	}
}

const testUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

// makeStrategyRequest creates a request for the test user with proper auth headers.
func makeStrategyRequest(method, path, body string) *http.Request {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", testUUID)
	return req
}

// TestMixedModePrevention verifies that creating a paper strategy alongside a live one is rejected.
func TestMixedModePrevention(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer srv.Close()

	cfg := &Config{SupabaseURL: srv.URL, SupabaseKey: "test", ManagerToken: "test-token"}
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	credStore := NewCredentialStore(dir, enc)

	keyEnc, _ := enc.Encrypt("key")
	secretEnc, _ := enc.Encrypt("secret")
	credStore.Upsert(ExchangeCredential{
		ID: "c1", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: keyEnc, APISecretEncrypted: secretEnc,
	})

	users := NewUserContainerManager()
	api := NewAPI(cfg, users, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil)
	r := testRouter(api)

	// Create first strategy as live
	createBody := `{"name":"Live Grid","strategy":"grid2","exchange":"binance","config":{"symbol":"BTCUSDT"},"mode":"live"}`
	req := makeStrategyRequest("POST", "/api/users/"+testUUID+"/strategies", createBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first strategy creation failed: %d %s", w.Code, w.Body.String())
	}

	// Try creating a second strategy as paper — should be rejected
	createBody2 := `{"name":"Paper DCA","strategy":"dca","exchange":"binance","config":{"symbol":"ETHUSDT"},"mode":"paper"}`
	req2 := makeStrategyRequest("POST", "/api/users/"+testUUID+"/strategies", createBody2)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for mixed mode, got %d: %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "cannot mix paper and live") {
		t.Errorf("expected mixed mode error message, got: %s", w2.Body.String())
	}
}

// TestLiveModeRequiresCredentials verifies that creating a live strategy without credentials is rejected.
func TestLiveModeRequiresCredentials(t *testing.T) {
	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test", ManagerToken: "test-token"}
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	credStore := NewCredentialStore(dir, enc)

	users := NewUserContainerManager()
	api := NewAPI(cfg, users, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil)
	r := testRouter(api)

	createBody := `{"name":"Live Grid","strategy":"grid2","exchange":"binance","config":{"symbol":"BTCUSDT"},"mode":"live"}`
	req := makeStrategyRequest("POST", "/api/users/"+testUUID+"/strategies", createBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for live without credentials, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "live mode requires API credentials") {
		t.Errorf("expected credential error message, got: %s", w.Body.String())
	}
}

// TestPaperModeWithLiveOnlyStrategy verifies that paper mode is rejected for liveOnly strategies.
func TestPaperModeWithLiveOnlyStrategy(t *testing.T) {
	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test", ManagerToken: "test-token"}
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	credStore := NewCredentialStore(dir, enc)

	users := NewUserContainerManager()
	api := NewAPI(cfg, users, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil)
	r := testRouter(api)

	liveOnlyTests := []string{"bollmaker", "supertrend", "dca2", "wall", "drift"}
	for _, strategy := range liveOnlyTests {
		t.Run(strategy, func(t *testing.T) {
			body := `{"name":"Paper ` + strategy + `","strategy":"` + strategy + `","exchange":"binance","config":{"symbol":"BTCUSDT"},"mode":"paper"}`
			req := makeStrategyRequest("POST", "/api/users/"+testUUID+"/strategies", body)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for paper mode with liveOnly strategy %s, got %d", strategy, w.Code)
			}
		})
	}
}

// TestLegacyAliasPaperRejection verifies that legacy aliases for liveOnly strategies are also rejected in paper mode.
func TestLegacyAliasPaperRejection(t *testing.T) {
	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test", ManagerToken: "test-token"}
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	credStore := NewCredentialStore(dir, enc)

	users := NewUserContainerManager()
	api := NewAPI(cfg, users, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil)
	r := testRouter(api)

	tests := []struct {
		alias  string
		target string
	}{
		{"sentinel_anomaly", "sentinel"},
		{"autobuy_scheduled", "autobuy"},
		{"rebalance_portfolio", "rebalance"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			body := `{"name":"Paper ` + tt.alias + `","strategy":"` + tt.alias + `","exchange":"binance","config":{"symbol":"BTCUSDT"},"mode":"paper"}`
			req := makeStrategyRequest("POST", "/api/users/"+testUUID+"/strategies", body)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("alias %q -> %q should be rejected in paper mode, got %d: %s", tt.alias, tt.target, w.Code, w.Body.String())
			}
		})
	}
}

// TestEnvArgs_PaperMode verifies container env args for paper mode
func TestEnvArgs_PaperMode(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{BBGOPort: 8080, BBGOGRPCPort: 9090, MarketDataAddr: "marketdata:9090"}
	cm := NewContainerManager(cfg, creds, nil)

	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "paper"},
		},
	}

	args := cm.envArgs(uc)

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "PAPER_TRADE=1") {
		t.Error("paper mode must inject PAPER_TRADE=1")
	}
	if !strings.Contains(argsStr, "DB_DRIVER=sqlite3") {
		t.Error("must inject DB_DRIVER")
	}
	if !strings.Contains(argsStr, "MARKET_DATA_SERVICE_URL") {
		t.Error("must inject MARKET_DATA_SERVICE_URL")
	}
	if strings.Contains(argsStr, "_API_KEY=") {
		t.Error("paper mode must NOT inject API keys")
	}
}

// TestEnvArgs_LiveMode verifies container env args for live mode with credentials
func TestEnvArgs_LiveMode(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{BBGOPort: 8080, BBGOGRPCPort: 9090, MarketDataAddr: "marketdata:9090"}
	cm := NewContainerManager(cfg, creds, nil)

	// Store credentials
	keyEnc, _ := enc.Encrypt("my-api-key")
	secretEnc, _ := enc.Encrypt("my-api-secret")
	passEnc, _ := enc.Encrypt("my-passphrase")
	creds.Upsert(ExchangeCredential{
		ID: "c1", UserID: "user-1", Exchange: "okex",
		APIKeyEncrypted: keyEnc, APISecretEncrypted: secretEnc, PassphraseEncrypted: passEnc,
	})

	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{Exchange: "okex", Strategy: "grid2", Mode: "live"},
		},
	}

	args := cm.envArgs(uc)

	argsStr := strings.Join(args, " ")
	if strings.Contains(argsStr, "PAPER_TRADE") {
		t.Error("live mode must NOT inject PAPER_TRADE")
	}
	if !strings.Contains(argsStr, "OKEX_API_KEY=my-api-key") {
		t.Error("must inject OKEX_API_KEY from decrypted credential")
	}
	if !strings.Contains(argsStr, "OKEX_API_SECRET=my-api-secret") {
		t.Error("must inject OKEX_API_SECRET from decrypted credential")
	}
	if !strings.Contains(argsStr, "OKEX_PASSPHRASE=my-passphrase") {
		t.Error("must inject OKEX_PASSPHRASE for exchanges that need it")
	}
}

// TestEnvArgs_CrossExchange_MultipleExchanges verifies that cross-exchange injects keys for all exchanges
func TestEnvArgs_CrossExchange_MultipleExchanges(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)
	cfg := &Config{BBGOPort: 8080, BBGOGRPCPort: 9090}
	cm := NewContainerManager(cfg, creds, nil)

	for _, ex := range []string{"binance", "bybit"} {
		keyEnc, _ := enc.Encrypt(ex + "-key")
		secretEnc, _ := enc.Encrypt(ex + "-secret")
		creds.Upsert(ExchangeCredential{
			ID: "c-" + ex, UserID: "user-1", Exchange: ex,
			APIKeyEncrypted: keyEnc, APISecretEncrypted: secretEnc,
		})
	}

	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "live",
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance"},
					{Name: "hedge", Exchange: "bybit", Futures: true},
				},
			},
		},
	}

	args := cm.envArgs(uc)

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "BINANCE_API_KEY=binance-key") {
		t.Error("must inject BINANCE_API_KEY for maker session")
	}
	if !strings.Contains(argsStr, "BINANCE_API_SECRET=binance-secret") {
		t.Error("must inject BINANCE_API_SECRET for maker session")
	}
	if !strings.Contains(argsStr, "BYBIT_API_KEY=bybit-key") {
		t.Error("must inject BYBIT_API_KEY for hedge session")
	}
	if !strings.Contains(argsStr, "BYBIT_API_SECRET=bybit-secret") {
		t.Error("must inject BYBIT_API_SECRET for hedge session")
	}
}

// TestCrossExchangePublicOnlyPartialCredentials verifies that sessions get individual PublicOnly settings
func TestCrossExchangePublicOnlyPartialCredentials(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	keyEnc, _ := enc.Encrypt("binance-key")
	secretEnc, _ := enc.Encrypt("binance-secret")
	creds.Upsert(ExchangeCredential{
		ID: "c1", UserID: "user-1", Exchange: "binance",
		APIKeyEncrypted: keyEnc, APISecretEncrypted: secretEnc,
	})

	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "paper",
				Config:        rawJSON(`{"symbol":"BTCUSDT"}`),
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance"},
					{Name: "hedge", Exchange: "bybit", Futures: true},
				},
			},
		},
	}

	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool {
		_, _, _, err := creds.GetDecrypted("user-1", exchange)
		return err == nil
	})
	if err != nil {
		t.Fatalf("buildUserYAML: %v", err)
	}
	yaml := string(yamlBytes)

	if !strings.Contains(yaml, "BINANCE") {
		t.Error("maker session should have BINANCE env prefix")
	}
	if !strings.Contains(yaml, "publicOnly: true") {
		t.Error("bybit session without credentials should have publicOnly=true")
	}
}

// TestProxyStripAuthHeaders verifies the proxy strips sensitive headers before forwarding
func TestProxyStripAuthHeaders(t *testing.T) {
	var receivedHeaders http.Header
	var mu sync.Mutex

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedHeaders = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer bbgoSrv.Close()

	cfg := &Config{BBGOPort: 8080, BBGOGRPCPort: 9090}
	cm := &ContainerManager{cfg: cfg}
	proxy := NewBotProxy(cm)
	proxy.resolveAddr = func(userID string) string { return bbgoSrv.URL }

	req := httptest.NewRequest("GET", "/api/bbgo/user-1/sessions", nil)
	req.Header.Set("X-Manager-Token", "secret-token")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("Authorization", "Bearer token")

	w := httptest.NewRecorder()
	proxy.ProxyToBot(w, req, "user-1")

	mu.Lock()
	defer mu.Unlock()
	if receivedHeaders.Get("X-Manager-Token") != "" {
		t.Error("proxy must strip X-Manager-Token before forwarding to bbgo container")
	}
	if receivedHeaders.Get("X-User-Id") != "" {
		t.Error("proxy must strip X-User-Id before forwarding to bbgo container")
	}
}
