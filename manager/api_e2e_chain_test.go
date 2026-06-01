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
	ucUserID := "user-1"
	ucMode := ModeLive
	ucStrategies := []StrategyEntry{
			{
					Name:     "BTC Grid",
				Exchange: "binance",
				Strategy: "grid2",
				Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`),
				Mode:     "live",
			},
		}

	// Step 3: Verify YAML generation for live mode
	yamlBytes, err := buildUserYAML(ucUserID, ucMode, ucStrategies, func(exchange string) bool {
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

	ucUserID := "user-1"
	ucMode := ModePaper
	ucStrategies := []StrategyEntry{
			{
					Name:     "BTC Grid Paper",
				Exchange: "binance",
				Strategy: "grid2",
				Config:   rawJSON(`{"symbol":"BTCUSDT","gridNumber":10,"upperPrice":70000,"lowerPrice":50000,"quantity":0.001}`),
				Mode:     "paper",
			},
		}

	yamlBytes, err := buildUserYAML(ucUserID, ucMode, ucStrategies, func(exchange string) bool {
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

	ucUserID := "user-1"
	ucMode := ModeLive
	ucStrategies := []StrategyEntry{
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
		}

	yamlBytes, err := buildUserYAML(ucUserID, ucMode, ucStrategies, func(exchange string) bool {
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
	ucUserID := "user-1"
	ucMode := ModeLive
	ucStrategies := []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "paper",
				Config:        rawJSON(`{"symbol":"BTCUSDT"}`),
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance"},              // EnvVarPrefix intentionally empty
					{Name: "hedge", Exchange: "bybit", Futures: true}, // EnvVarPrefix intentionally empty
				},
			},
		}

	yamlBytes, err := buildUserYAML(ucUserID, ucMode, ucStrategies, func(exchange string) bool { return false })
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

// TestMixedModePrevention verifies that creating strategies with different modes is now allowed
// since they go to separate containers (live → bbgo-{userID}, paper → bbgo-{userID}-paper).
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

	tnKeyEnc, _ := enc.Encrypt("tn-key")
	tnSecretEnc, _ := enc.Encrypt("tn-secret")
	credStore.Upsert(ExchangeCredential{
		ID: "c2", UserID: testUUID, Exchange: "binance",
		APIKeyEncrypted: tnKeyEnc, APISecretEncrypted: tnSecretEnc, IsTestnet: true,
	})

	store := NewStrategyStore("")
	api := NewAPI(cfg, store, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil, nil)
	r := testRouter(api)

	// Create first strategy as live
	createBody := `{"name":"Live Grid","strategy":"grid2","exchange":"binance","config":{"symbol":"BTCUSDT"},"mode":"live"}`
	req := makeStrategyRequest("POST", "/api/users/"+testUUID+"/strategies", createBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first strategy creation failed: %d %s", w.Code, w.Body.String())
	}

	// Create second strategy as paper — should now be ALLOWED (goes to paper container)
	createBody2 := `{"name":"Paper Grid","strategy":"grid","exchange":"binance","config":{"symbol":"ETHUSDT"},"mode":"paper"}`
	req2 := makeStrategyRequest("POST", "/api/users/"+testUUID+"/strategies", createBody2)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("expected 201 for mixed mode (separate containers), got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestLiveModeRequiresCredentials verifies that creating a live strategy without credentials is rejected.
func TestLiveModeRequiresCredentials(t *testing.T) {
	cfg := &Config{SupabaseURL: "http://localhost:1", SupabaseKey: "test", ManagerToken: "test-token"}
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	credStore := NewCredentialStore(dir, enc)

	store := NewStrategyStore("")
	api := NewAPI(cfg, store, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil, nil)
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

	store := NewStrategyStore("")
	api := NewAPI(cfg, store, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil, nil)
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

	store := NewStrategyStore("")
	api := NewAPI(cfg, store, &ContainerManager{cfg: cfg}, nil, credStore, enc, nil, nil, nil, nil, nil, nil)
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

	ucUserID := "user-1"
	ucMode := ModePaper
	ucStrategies := []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "paper"},
		}

	args := cm.envArgs(ucUserID, ucMode, ucStrategies)

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "PAPER_TRADE=1") {
		t.Error("paper mode must inject PAPER_TRADE=1")
	}
	if !strings.Contains(argsStr, "DB_DRIVER=sqlite3") {
		t.Error("paper mode must inject DB_DRIVER=sqlite3")
	}
	if strings.Contains(argsStr, "SUPABASE_URL") {
		t.Error("paper mode must NOT inject Supabase env vars")
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

	ucUserID := "user-1"
	ucMode := ModeLive
	ucStrategies := []StrategyEntry{
			{Exchange: "okex", Strategy: "grid2", Mode: "live"},
		}

	args := cm.envArgs(ucUserID, ucMode, ucStrategies)

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
	if !strings.Contains(argsStr, "OKEX_API_PASSPHRASE=my-passphrase") {
		t.Error("must inject OKEX_API_PASSPHRASE for exchanges that need it")
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

	ucUserID := "user-1"
	ucMode := ModeLive
	ucStrategies := []StrategyEntry{
			{
				Strategy:      "xmaker",
				CrossExchange: true,
				Mode:          "live",
				Sessions: []SessionRoleConfig{
					{Name: "maker", Exchange: "binance"},
					{Name: "hedge", Exchange: "bybit", Futures: true},
				},
			},
		}

	args := cm.envArgs(ucUserID, ucMode, ucStrategies)

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

	ucUserID := "user-1"
	ucMode := ModeLive
	ucStrategies := []StrategyEntry{
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
		}

	yamlBytes, err := buildUserYAML(ucUserID, ucMode, ucStrategies, func(exchange string) bool {
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
	proxy.resolveAddr = func(userID string, _ string) string { return bbgoSrv.URL }

	req := httptest.NewRequest("GET", "/api/bbgo/user-1/sessions", nil)
	req.Header.Set("X-Manager-Token", "secret-token")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("Authorization", "Bearer token")

	w := httptest.NewRecorder()
	proxy.ProxyToBot(w, req, "user-1", ModeLive)

	mu.Lock()
	defer mu.Unlock()
	if receivedHeaders.Get("X-Manager-Token") != "" {
		t.Error("proxy must strip X-Manager-Token before forwarding to bbgo container")
	}
	if receivedHeaders.Get("X-User-Id") != "" {
		t.Error("proxy must strip X-User-Id before forwarding to bbgo container")
	}
}

// TestPaperModeFullLifecycle traces: create strategy → query all bbgo endpoints → verify mode routing
func TestPaperModeFullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	store, _ := newTestStore(t)
	cfg := &Config{DataDir: tmpDir, ManagerToken: "test-token", SupabaseURL: "http://localhost:1", SupabaseKey: "test"}
	cm := &ContainerManager{cfg: cfg, creds: creds}
	proxy := NewBotProxy(cm)
	supaClient, _ := NewSupabaseClient("http://localhost:1", "test")
	syncer := NewSyncer(supaClient)
	api := NewAPI(cfg, store, cm, proxy, creds, enc, syncer, nil, nil, nil, nil, NewBacktestJobStore(tmpDir))
	api.verifyCredFn = func(_, _, _, _ string, _ bool) VerifyResult { return VerifyResult{Verified: true} }

	// Insert testnet credential so paper mode start is allowed
	tnKey, _ := enc.Encrypt("tn-key")
	tnSec, _ := enc.Encrypt("tn-secret")
	creds.Upsert(ExchangeCredential{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Exchange: "binance",
		APIKeyEncrypted: tnKey, APISecretEncrypted: tnSec, IsTestnet: true, IsVerified: true,
	})

	var bbgoRequests []string
	var mu sync.Mutex
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		bbgoRequests = append(bbgoRequests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/sessions":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sessions": []map[string]interface{}{{"name": "binance", "exchange": "binance"}},
			})
		case "/api/strategies/single":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"strategies": []map[string]interface{}{
					{"strategyInstanceID": "strat-1", "strategy": "grid2", "symbol": "BTCUSDT", "session": "binance"},
				},
			})
		case "/api/assets":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"assets": map[string]interface{}{
					"USDT": map[string]string{"currency": "USDT", "total": "10000", "available": "8000"},
				},
			})
		case "/api/trades":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"trades": []map[string]interface{}{
					{"gid": 1, "symbol": "BTCUSDT", "side": "BUY", "price": "50000", "quantity": "0.1", "fee": "2.5", "tradedAt": "2024-01-01"},
				},
			})
		case "/api/orders/closed":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"orders": []map[string]interface{}{
					{"gid": 1, "symbol": "BTCUSDT", "side": "BUY", "orderType": "LIMIT", "price": "50000", "status": "FILLED"},
				},
			})
		case "/api/trading-volume":
			json.NewEncoder(w).Encode(map[string]interface{}{"tradingVolumes": []interface{}{}})
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	}))
	t.Cleanup(bbgoSrv.Close)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
	}
	api.containerRunning = func(uid, mode string) bool {
		return uid == "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" && mode == ModePaper
	}
	api.container.apiURLFn = func(_, _ string) string { return bbgoSrv.URL }

	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	r := testRouter(api)

	// Step 1: Create paper strategy
	body := `{"name":"Paper Grid","exchange":"binance","strategy":"grid2","config":{"symbol":"BTCUSDT"},"mode":"paper"}`
	req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create strategy: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: Query all bbgo data endpoints with mode=paper
	endpoints := []struct {
		path, key string
	}{
		{"/bbgo/ping?mode=paper", "status"},
		{"/bbgo/sessions?mode=paper", "sessions"},
		{"/bbgo/assets?mode=paper", "assets"},
		{"/bbgo/trades?mode=paper", "trades"},
		{"/bbgo/orders/closed?mode=paper", "orders"},
		{"/bbgo/strategies?mode=paper", "strategies"},
		{"/bbgo/trading-volume?mode=paper", "tradingVolumes"},
	}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID+ep.path, nil)
		req.Header.Set("X-User-Id", userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d: %s", ep.path, w.Code, w.Body.String())
			continue
		}
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Errorf("GET %s: decode: %v", ep.path, err)
			continue
		}
		if _, ok := resp[ep.key]; !ok {
			t.Errorf("GET %s: missing key %q", ep.path, ep.key)
		}
	}

	// Step 3: Verify bbgo container received all 7 requests
	mu.Lock()
	reqs := bbgoRequests
	mu.Unlock()
	for _, ep := range []string{"/api/ping", "/api/sessions", "/api/assets", "/api/trades", "/api/orders/closed", "/api/strategies/single", "/api/trading-volume"} {
		found := false
		for _, r := range reqs {
			if r == "GET "+ep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("bbgo never received %s (got: %v)", ep, reqs)
		}
	}

	// Step 4: Live mode returns 503 (no live container running)
	req = httptest.NewRequest(http.MethodGet, "/api/users/"+userID+"/bbgo/ping?mode=live", nil)
	req.Header.Set("X-User-Id", userID)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("live ping: expected 503, got %d", w.Code)
	}
}
