package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestListCredentials_DoesNotLeakEncryptedData verifies the credential list
// endpoint never returns encrypted API keys or secrets — only safe metadata.
func TestListCredentials_DoesNotLeakEncryptedData(t *testing.T) {
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	insertTestCredential(t, creds, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "binance", "my-api-key", "my-api-secret")

	r := testRouterWithUser(api, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	req := httptest.NewRequest("GET", "/api/credentials", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, forbidden := range []string{"api_key", "api_secret", "encrypted", "passphrase", "my-api-key", "my-api-secret"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("credential list response should not contain %q, got: %s", forbidden, body)
		}
	}

	var result []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(result))
	}
	c := result[0]
	for _, key := range []string{"id", "user_id", "exchange", "is_testnet", "is_verified"} {
		if _, ok := c[key]; !ok {
			t.Errorf("expected field %q in safe credential response", key)
		}
	}
}

// TestBBGoData_NonRunningContainer_Returns503 verifies that all bbgo data
// endpoints return 503 when the container is not running.
func TestBBGoData_NonRunningContainer_Returns503(t *testing.T) {
	cfg := &Config{ManagerToken: "test-token"}
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Manager-Token", "test-token")
			r.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
			next.ServeHTTP(w, r)
		})
	})
	api.RegisterRoutes(r)

	endpoints := []struct {
		name string
		path string
	}{
		{"ping", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/ping"},
		{"sessions", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/sessions"},
		{"trades", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/trades"},
		{"assets", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/assets"},
		{"strategies", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/strategies"},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", ep.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("expected 503 for %s on stopped container, got %d: %s", ep.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestPaperTrading_HappyPath exercises the complete paper trading flow:
// create strategy (paper) → start → status=running → query bbgo data.
func TestPaperTrading_HappyPath(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)
	defer api.Close()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		case "/api/sessions":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sessions": []map[string]interface{}{
					{"name": "binance", "exchangeName": "binance"},
				},
			})
		case "/api/trades":
			json.NewEncoder(w).Encode(map[string]interface{}{"trades": []interface{}{}})
		default:
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer bbgoSrv.Close()

	api.containerStart = func(_ *UserContainer) error { return nil }
	api.containerRunning = func(_ string) bool { return false }
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}

	r := testRouter(api)

	// Step 1: Create paper strategy — should auto-start (first strategy)
	body := `{"name":"Grid Paper","exchange":"binance","strategy":"grid2","config":{"symbol":"BTCUSDT"},"mode":"paper"}`
	req := httptest.NewRequest("POST", "/api/users/"+userID+"/strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("step1 create strategy: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: Wait for async start to complete (simulate health check success)
	time.Sleep(200 * time.Millisecond)
	users.UpdateStatus(userID, StatusRunning)

	// Step 3: Check status
	req = httptest.NewRequest("GET", "/api/users/"+userID+"/status", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step3 status: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var statusResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&statusResp)
	if statusResp["status"] != StatusRunning {
		t.Errorf("step3: expected running, got %v", statusResp["status"])
	}

	// Step 4: Query bbgo data (ping)
	req = httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/ping", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step4 ping: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Step 5: Query bbgo sessions
	req = httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/sessions", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step5 sessions: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Step 6: Verify YAML was generated (paper mode)
	uc, _ := users.Get(userID)
	yaml, err := buildUserYAML(uc, func(_ string) bool { return false })
	if err != nil {
		t.Fatalf("step6 yaml: %v", err)
	}
	s := string(yaml)
	if !strings.Contains(s, "PAPER_TRADE: \"1\"") {
		t.Errorf("step6: expected PAPER_TRADE in YAML, got:\n%s", s)
	}

	// Step 7: Stop container
	req = httptest.NewRequest("POST", "/api/users/"+userID+"/stop", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step7 stop: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Step 8: Verify bbgo data now returns 503
	req = httptest.NewRequest("GET", "/api/users/"+userID+"/bbgo/ping", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("step8: expected 503 after stop, got %d", w.Code)
	}
}

// TestLiveTrading_HappyPath exercises the complete live trading flow:
// create credential → create strategy (live) → verify YAML + env args.
func TestLiveTrading_HappyPath(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	api.containerStart = func(_ *UserContainer) error { return nil }

	r := testRouter(api)

	// Step 1: Create credential for binance
	credBody := `{"exchange":"binance","api_key":"real-key","api_secret":"real-secret"}`
	req := httptest.NewRequest("POST", "/api/credentials", strings.NewReader(credBody))
	req.Header.Set("Content-Type", "application/json")
	r2 := testRouterWithUser(api, userID)
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("step1 credential: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: Create live strategy
	stratBody := `{"name":"Grid Live","exchange":"binance","strategy":"grid2","config":{"symbol":"BTCUSDT"},"mode":"live"}`
	req = httptest.NewRequest("POST", "/api/users/"+userID+"/strategies", strings.NewReader(stratBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("step2 strategy: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Step 3: Verify YAML does NOT have PAPER_TRADE
	uc, _ := users.Get(userID)
	yaml, err := buildUserYAML(uc, func(_ string) bool { return false })
	if err != nil {
		t.Fatalf("step3 yaml: %v", err)
	}
	s := string(yaml)
	if strings.Contains(s, "PAPER_TRADE") {
		t.Errorf("step3: live mode should NOT have PAPER_TRADE in YAML, got:\n%s", s)
	}

	// Step 4: Verify env args include real credentials
	args := cm.envArgs(uc)
	foundKey := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-e" && strings.HasPrefix(args[i+1], "BINANCE_API_KEY=") {
			foundKey = true
			if !strings.Contains(args[i+1], "real-key") {
				t.Errorf("step4: expected real API key in env args, got %s", args[i+1])
			}
		}
	}
	if !foundKey {
		t.Error("step4: BINANCE_API_KEY not found in env args")
	}
}

// TestLiveOnlyStrategy_PaperModeBlocked verifies that strategies marked liveOnly
// cannot be created in paper mode even with valid credentials.
func TestLiveOnlyStrategy_PaperModeBlocked(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	insertTestCredential(t, creds, userID, "binance", "key", "secret")

	r := testRouter(api)
	body := `{"name":"BollMaker","exchange":"binance","strategy":"bollmaker","config":{"symbol":"BTCUSDT"},"mode":"paper"}`
	req := httptest.NewRequest("POST", "/api/users/"+userID+"/strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for liveOnly paper, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "only supports live mode") {
		t.Errorf("expected liveOnly error message, got: %s", w.Body.String())
	}
}

// TestLiveOnlyStrategy_LiveModeAccepted verifies that strategies marked liveOnly
// can be created in live mode when credentials exist.
func TestLiveOnlyStrategy_LiveModeAccepted(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	insertTestCredential(t, creds, userID, "binance", "key", "secret")
	api.containerStart = func(_ *UserContainer) error { return nil }

	r := testRouter(api)
	body := `{"name":"BollMaker","exchange":"binance","strategy":"bollmaker","config":{"symbol":"BTCUSDT"},"mode":"live"}`
	req := httptest.NewRequest("POST", "/api/users/"+userID+"/strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for liveOnly live, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteCredential_StoppedContainer_NoRestart verifies that deleting
// credentials from a stopped container does NOT trigger restart.
func TestDeleteCredential_StoppedContainer_NoRestart(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	users := NewUserContainerManager()
	users.users[userID] = &UserContainer{
		UserID:     userID,
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live"}},
	}
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	insertTestCredential(t, creds, userID, "binance", "k", "s")
	stored, _ := creds.List(userID)
	credID := stored[0].ID

	var restartCalled bool
	api.containerStart = func(_ *UserContainer) error {
		restartCalled = true
		return nil
	}

	r := testRouterWithUser(api, userID)
	req := httptest.NewRequest("DELETE", "/api/credentials/"+credID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	time.Sleep(200 * time.Millisecond)
	if restartCalled {
		t.Error("deleting credential on stopped container should NOT trigger restart")
	}
}

// TestCreateCredential_PassphraseEncrypted verifies that optional passphrase
// is also encrypted and stored.
func TestCreateCredential_PassphraseEncrypted(t *testing.T) {
	userID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(dir, enc)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
		DataDir:      dir,
	}
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: cfg, creds: creds, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, creds, enc, nil, nil, nil, nil, nil)

	r := testRouterWithUser(api, userID)
	body := `{"exchange":"okex","api_key":"okx-key","api_secret":"okx-secret","passphrase":"my-pass"}`
	req := httptest.NewRequest("POST", "/api/credentials", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	_, _, passPlain, err := creds.GetDecrypted(userID, "okex")
	if err != nil {
		t.Fatalf("GetDecrypted error: %v", err)
	}
	if passPlain != "my-pass" {
		t.Errorf("expected passphrase 'my-pass', got %q", passPlain)
	}
}

// TestRefreshContainerStatus_ErrorLeavesUnchanged verifies that
// refreshContainerStatus leaves the status unchanged when Docker
// is unreachable (error from CheckRunning).
func TestRefreshContainerStatus_ErrorLeavesUnchanged(t *testing.T) {
	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey: "test",
		ManagerToken: "test-token",
	}
	users := NewUserContainerManager()
	uc := &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = uc

	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	// pool is nil → CheckRunning calls docker CLI which fails → error returned
	// refreshContainerStatus should leave status unchanged on error
	api.refreshContainerStatus(uc)
	if uc.Status != StatusRunning {
		t.Errorf("expected status unchanged (running) on docker error, got %s", uc.Status)
	}
}

// TestRefreshContainerStatus_SkipsStoppedContainer verifies refreshContainerStatus
// is a no-op for already-stopped containers.
func TestRefreshContainerStatus_SkipsStoppedContainer(t *testing.T) {
	cfg := &Config{ManagerToken: "test-token"}
	users := NewUserContainerManager()
	uc := &UserContainer{
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusStopped,
		Strategies: []StrategyEntry{{ID: "s1"}},
	}
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] = uc

	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	api.refreshContainerStatus(uc)
	if uc.Status != StatusStopped {
		t.Errorf("expected status unchanged, got %s", uc.Status)
	}
}
