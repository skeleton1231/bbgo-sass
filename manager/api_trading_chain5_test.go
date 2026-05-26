package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/c9s/bbgo/saas/manager/pool"
	"github.com/go-chi/chi/v5"
)

func chiReqWithParam(method, url string, body string, key, val string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- Gap 1: Strategy added to previously-stopped user does not auto-start ---

func TestCreateStrategy_StoppedUser_NoAutoStart(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000001", StrategyEntry{Exchange: "binance", Strategy: "grid2", Mode: "paper"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000001", StatusStopped)

	started := false
	api := &API{
		users:            users,
		wsTickets:        NewWSTicketStore(),
		container:        &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		containerStart:   func(uc *UserContainer) error { started = true; return nil },
		containerStop:    func(string) {},
		containerRunning: func(string) bool { return false },
	}
	defer api.Close()

	body := `{"name":"second","exchange":"binance","strategy":"grid2","mode":"paper","config":{"symbol":"ETHUSDT"}}`
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeee000001/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000001")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	time.Sleep(50 * time.Millisecond)
	if started {
		t.Error("container should NOT auto-start when user was previously stopped (not newly created)")
	}
}

// --- Gap 2: First strategy on new user DOES auto-start ---

func TestCreateStrategy_NewUser_AutoStarts(t *testing.T) {
	users := NewUserContainerManager()
	started := make(chan string, 1)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`ok`))
	}))
	defer bbgoSrv.Close()

	api := &API{
		users:            users,
		wsTickets:        NewWSTicketStore(),
		container:        &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		containerStart:   func(uc *UserContainer) error { started <- uc.UserID; return nil },
		containerStop:    func(string) {},
		containerRunning: func(string) bool { return false },
		newBBGoClient:    func(baseURL string) *BBGoClient { return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()} },
	}
	defer api.Close()

	body := `{"name":"first","exchange":"binance","strategy":"grid2","mode":"paper","config":{"symbol":"BTCUSDT"}}`
	req := httptest.NewRequest("POST", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeee000002/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000002")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	select {
	case uid := <-started:
		if uid != "aaaaaaaa-bbbb-cccc-dddd-eeeeee000002" {
			t.Errorf("started user = %q, want new-u1", uid)
		}
	case <-time.After(time.Second):
		t.Error("container should auto-start for newly created user")
	}
}

// --- Gap 3: DeleteStrategy with remaining strategies triggers restart ---

func TestDeleteStrategy_RemainingStrategies_RestartsContainer(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000003", StrategyEntry{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "paper"})
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000003", StrategyEntry{ID: "s2", Exchange: "binance", Strategy: "grid2", Mode: "paper"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000003", StatusRunning)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`ok`))
	}))
	defer bbgoSrv.Close()

	restarted := make(chan string, 1)
	api := &API{
		users:            users,
		wsTickets:        NewWSTicketStore(),
		container:        &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		containerStart:   func(uc *UserContainer) error { restarted <- uc.UserID; return nil },
		containerStop:    func(string) {},
		containerRunning: func(string) bool { return true },
		newBBGoClient:    func(baseURL string) *BBGoClient { return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()} },
	}
	defer api.Close()

	req := chiReqWithParam("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeee000003/strategies/s1", "", "strategyID", "s1")
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000003")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.DeleteStrategy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	select {
	case uid := <-restarted:
		if uid != "aaaaaaaa-bbbb-cccc-dddd-eeeeee000003" {
			t.Errorf("restarted user = %q, want del-u1", uid)
		}
	case <-time.After(time.Second):
		t.Error("container should restart when strategy deleted but others remain")
	}
}

// --- Gap 4: DeleteStrategy last strategy stops container ---

func TestDeleteStrategy_LastStrategy_StopsContainer(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000004", StrategyEntry{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "paper"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000004", StatusRunning)

	stopped := false
	api := &API{
		users:            users,
		wsTickets:        NewWSTicketStore(),
		container:        &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		containerStop:    func(string) { stopped = true },
		containerRunning: func(string) bool { return true },
	}
	defer api.Close()

	req := chiReqWithParam("DELETE", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeee000004/strategies/s1", "", "strategyID", "s1")
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000004")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.DeleteStrategy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !stopped {
		t.Error("container should stop when last strategy deleted")
	}
	uc, _ := users.Get("aaaaaaaa-bbbb-cccc-dddd-eeeeee000004")
	if uc.Status != StatusStopped {
		t.Errorf("status = %q, want stopped", uc.Status)
	}
}

// --- Gap 5: Mode mixing prevention with unspecified mode ---

func TestCreateStrategy_ModeMixing_FirstUnspecified(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000005", StrategyEntry{Exchange: "binance", Strategy: "grid2", Mode: ""})

	api := &API{users: users, wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"second","exchange":"binance","strategy":"grid2","mode":"paper","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000005")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("unspecified->paper should succeed, got status %d", w.Code)
	}
}

func TestCreateStrategy_ModeMixing_PaperThenLive(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000006", StrategyEntry{Exchange: "binance", Strategy: "grid2", Mode: "paper"})

	api := &API{users: users, wsTickets: NewWSTicketStore()}
	defer api.Close()

	body := `{"name":"second","exchange":"binance","strategy":"grid2","mode":"live","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000006")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("paper->live should be rejected, got status %d", w.Code)
	}
}

// --- Gap 6: Live mode with nil creds store (creds check skipped) ---

func TestCreateStrategy_LiveMode_NilCredsStore(t *testing.T) {
	users := NewUserContainerManager()
	api := &API{
		users:     users,
		creds:     nil,
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	body := `{"name":"test","exchange":"binance","strategy":"grid2","mode":"live","config":{}}`
	req := httptest.NewRequest("POST", "/api/users/"+testUUID+"/strategies", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000007")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateStrategy(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("live mode with nil creds store: status = %d (creds check skipped when store is nil)", w.Code)
	}
}

// --- Gap 7: Full live YAML verification ---

func TestBuildUserYAML_LiveWithCredentials(t *testing.T) {
	uc := &UserContainer{
		UserID: "live-yaml-u1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "live", Config: json.RawMessage(`{"symbol":"BTCUSDT","grid_num":5,"lower_price":40000,"upper_price":50000}`)},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)

	if strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Error("live mode should not have PAPER_TRADE in YAML")
	}
	if strings.Contains(yamlStr, "publicOnly: true") {
		t.Error("live mode with credentials should not set publicOnly")
	}
	if !strings.Contains(yamlStr, "grid2:") {
		t.Error("strategy config should be in YAML")
	}
	if !strings.Contains(yamlStr, "grid_num:") {
		t.Error("strategy params should be in YAML")
	}
}

// --- Gap 8: Cross-exchange EnvVarPrefix auto-fill ---

func TestBuildUserYAML_CrossExchange_AutoEnvVarPrefix(t *testing.T) {
	uc := &UserContainer{
		UserID: "xenv-u1",
		Strategies: []StrategyEntry{
			{
				CrossExchange: true,
				Strategy:      "xmaker",
				Mode:          "paper",
				Config:        json.RawMessage(`{"symbol":"BTCUSDT"}`),
				Sessions: []SessionRoleConfig{
					{Name: "binance_spot", Exchange: "binance", EnvVarPrefix: ""},
					{Name: "okex_spot", Exchange: "okex", EnvVarPrefix: ""},
				},
			},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)

	if !strings.Contains(yamlStr, "BINANCE") {
		t.Error("binance session should have BINANCE envVarPrefix auto-filled")
	}
	if !strings.Contains(yamlStr, "OKEX") {
		t.Error("okex session should have OKEX envVarPrefix auto-filled")
	}
	if !strings.Contains(yamlStr, "publicOnly: true") {
		t.Error("no credentials should set publicOnly")
	}
}

// --- Gap 9: Credential update triggers container restart ---

func TestCreateCredential_RunningContainer_TriggersRestart(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000008", StrategyEntry{Exchange: "binance", Strategy: "grid2", Mode: "live"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000008", StatusRunning)

	enc, _ := NewEncryptor("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	creds := NewCredentialStore("/tmp/test-creds-chain5", enc)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`ok`))
	}))
	defer bbgoSrv.Close()

	restarted := make(chan string, 1)
	api := &API{
		users:            users,
		creds:            creds,
		encryptor:        enc,
		wsTickets:        NewWSTicketStore(),
		container:        &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		containerStart:   func(uc *UserContainer) error { restarted <- uc.UserID; return nil },
		containerStop:    func(string) {},
		containerRunning: func(string) bool { return true },
		newBBGoClient:    func(baseURL string) *BBGoClient { return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()} },
	}
	defer api.Close()

	body := `{"exchange":"binance","api_key":"testkey","api_secret":"testsecret"}`
	req := httptest.NewRequest("POST", "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000008")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	select {
	case uid := <-restarted:
		if uid != "aaaaaaaa-bbbb-cccc-dddd-eeeeee000008" {
			t.Errorf("restarted user = %q, want cred-u1", uid)
		}
	case <-time.After(time.Second):
		t.Error("credential update should trigger container restart when running")
	}
}

// --- Gap 10: Unsupported exchange rejected ---

func TestCreateCredential_UnsupportedExchange_Verify(t *testing.T) {
	enc, _ := NewEncryptor("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	creds := NewCredentialStore("/tmp/test-creds-chain5b", enc)

	api := &API{
		creds:     creds,
		encryptor: enc,
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	body := `{"exchange":"unknown_exchange","api_key":"key","api_secret":"secret"}`
	req := httptest.NewRequest("POST", "/api/credentials", strings.NewReader(body))
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.CreateCredential(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unsupported exchange: status = %d, want 400", w.Code)
	}
}

// --- Gap 11: Supabase PnL fallback to container ---

func TestBBGoPnL_SupabaseFallback(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000009", StrategyEntry{Exchange: "binance", Strategy: "grid2"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000009", StatusRunning)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"trades":[{"id":1,"symbol":"BTCUSDT","side":"BUY","price":"40000","quantity":"1","fee":"0.01","tradedAt":"2024-01-01","gid":1},{"id":2,"symbol":"BTCUSDT","side":"SELL","price":"50000","quantity":"1","fee":"0.01","tradedAt":"2024-01-02","gid":2}]}`))
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	syncer := &Syncer{
		cfg:    &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		client: supabaseSrv.Client(),
	}

	api := &API{
		users:     users,
		syncer:    syncer,
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		newBBGoClient: func(baseURL string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	req := chiReqWithParam("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeee000009/bbgo/pnl", "", "userID", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000009")
	req.Header.Set("X-User-Id", "aaaaaaaa-bbbb-cccc-dddd-eeeeee000009")
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoPnL(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["totalTrades"] != float64(2) {
		t.Errorf("totalTrades = %v, want 2", result["totalTrades"])
	}
}

// --- Gap 12: SyncAll with running container syncs data ---

func TestSyncAll_RunningContainer_SyncsData(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000010", StrategyEntry{Exchange: "binance", Strategy: "grid2"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000010", StatusRunning)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/ping" {
			w.Write([]byte(`ok`))
		} else if path == "/api/orders/closed" {
			w.Write([]byte(`{"orders":[]}`))
		} else if path == "/api/trades" {
			w.Write([]byte(`{"trades":[]}`))
		} else {
			w.WriteHeader(404)
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	p := pool.New(5)
	s := &Syncer{
		users:     users,
		cfg:       &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		pool:      p,
		client:    supabaseSrv.Client(),
		newBBGoClientFn: func(_ string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
	}
	s.SyncAll()
}

// --- Gap 13: UserStatus for unknown user returns stopped ---

func TestUserStatus_UnknownUser(t *testing.T) {
	api := &API{
		users:     NewUserContainerManager(),
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	req := chiReqWithParam("GET", "/api/users/"+testUUID+"/status", "", "userID", testUUID)
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.UserStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["status"] != StatusStopped {
		t.Errorf("unknown user status = %v, want stopped", result["status"])
	}
}

// --- Gap 14: StartUser with no strategies fails ---

func TestStartUser_NoStrategies(t *testing.T) {
	users := NewUserContainerManager()
	api := &API{
		users:         users,
		wsTickets:     NewWSTicketStore(),
		containerStop: func(string) {},
	}
	defer api.Close()

	req := chiReqWithParam("POST", "/api/users/"+testUUID+"/start", "", "userID", testUUID)
	req.Header.Set("X-User-Id", testUUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.StartUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("start with no strategies: status = %d, want 400", w.Code)
	}
}

// --- Gap 15: buildBacktestYAML with explicit params ---

func TestBuildBacktestYAML_ExplicitParams(t *testing.T) {
	yamlBytes, err := buildBacktestYAML(
		"grid2",
		json.RawMessage(`{"symbol":"ETHUSDT","grid_num":10}`),
		"2024-01-01", "2024-06-01",
		"okex", "ETHUSDT",
	)
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)
	if !strings.Contains(yamlStr, "okex") {
		t.Error("should contain exchange override")
	}
	if !strings.Contains(yamlStr, "ETHUSDT") {
		t.Error("should contain symbol override")
	}
	if !strings.Contains(yamlStr, "grid_num:") {
		t.Error("should contain strategy params")
	}
	if !strings.Contains(yamlStr, "startTime:") {
		t.Error("should contain backtest timing")
	}
}

// --- Gap 16: fullSyncOrders pagination ---

func TestFullSyncOrders_MultiplePages(t *testing.T) {
	page := 0
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		if page <= 2 {
			orders := make([]map[string]interface{}, syncPageSize)
			for i := range orders {
				orders[i] = map[string]interface{}{
					"orderID": uint64(page*syncPageSize + i),
					"symbol":  "BTCUSDT", "side": "BUY",
					"price": "50000", "quantity": "1",
					"status": "FILLED", "type": "LIMIT",
					"creationTime": "2024-01-01",
				}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"orders": orders})
		} else {
			w.Write([]byte(`{"orders":[]}`))
		}
	}))
	defer bbgoSrv.Close()

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	client := &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}

	s := &Syncer{
		cfg:    &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		client: supabaseSrv.Client(),
	}

	s.fullSyncOrders("multi-page-u1", client)
}

// --- Gap 17: CredentialStore round-trip ---

func TestCredentialStore_RoundTrip(t *testing.T) {
	enc, _ := NewEncryptor("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	creds := NewCredentialStore("/tmp/test-creds-chain5c", enc)

	cred := ExchangeCredential{
		ID: "cred-rt-1", UserID: "rt-u1", Exchange: "binance",
	}
	key, _ := enc.Encrypt("my-api-key")
	secret, _ := enc.Encrypt("my-secret")
	cred.APIKeyEncrypted = key
	cred.APISecretEncrypted = secret
	creds.Upsert(cred)

	apiKey, apiSecret, passphrase, err := creds.GetDecrypted("rt-u1", "binance")
	if err != nil {
		t.Fatal(err)
	}
	if apiKey != "my-api-key" {
		t.Errorf("apiKey = %q, want my-api-key", apiKey)
	}
	if apiSecret != "my-secret" {
		t.Errorf("apiSecret = %q, want my-secret", apiSecret)
	}
	if passphrase != "" {
		t.Errorf("passphrase = %q, want empty", passphrase)
	}
}

// --- Gap 18: ContainerManager envArgs with paper vs live mode ---

func TestContainerManager_EnvArgs_PaperMode(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{}, creds: nil}
	uc := &UserContainer{
		UserID: "env-u1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "paper"},
		},
	}
	args := cm.envArgs(uc)
	found := false
	for i, a := range args {
		if a == "PAPER_TRADE=1" && i > 0 && args[i-1] == "-e" {
			found = true
		}
	}
	if !found {
		t.Errorf("paper mode env args should contain PAPER_TRADE=1, got %v", args)
	}
}

func TestContainerManager_EnvArgs_LiveMode(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{}, creds: nil}
	uc := &UserContainer{
		UserID: "env-u2",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "live"},
		},
	}
	args := cm.envArgs(uc)
	for _, a := range args {
		if a == "PAPER_TRADE=1" {
			t.Error("live mode should not have PAPER_TRADE=1")
		}
	}
}

// --- Gap 19: Syncer.bbgoClient uses injected function ---

func TestSyncer_BBGoClient_Injection(t *testing.T) {
	called := false
	s := &Syncer{
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		newBBGoClientFn: func(baseURL string) *BBGoClient {
			called = true
			return &BBGoClient{}
		},
	}
	client := s.bbgoClient("test-u1")
	if !called {
		t.Error("should call injected newBBGoClientFn")
	}
	if client == nil {
		t.Error("client should not be nil")
	}
}

// --- Gap 20: MarketDataHub broadcast to empty key is safe ---

func TestMarketDataHub_Broadcast_EmptyKey(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{})}
	hub.broadcast("nonexistent", json.RawMessage(`{}`))
}

// --- Gap 21: cleanupBackups edge cases ---

func TestCleanupBackups_NoMatch(t *testing.T) {
	cleanupBackups("/nonexistent", "prefix", 3)
}

// --- Gap 22: exchangeEnvPrefix unknown exchange ---

func TestExchangeEnvPrefix_Unknown(t *testing.T) {
	if got := exchangeEnvPrefix("unknown"); got != "EXCHANGE" {
		t.Errorf("unknown exchange prefix = %q, want EXCHANGE", got)
	}
}

// --- Gap 23: MarketDataHub SubscribeMarket and cleanup ---

func TestMarketDataHub_SubscribeMarket_Close(t *testing.T) {
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		done:     make(chan struct{}),
	}
	ch, err := hub.SubscribeMarket(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ch == nil {
		t.Fatal("channel should not be nil")
	}
	hub.Close()
	hub.Unsubscribe("market", ch)
}

// --- Gap 24: Full chain - live mode without credentials still sets publicOnly ---

func TestFullChain_LiveYAMLWithoutCredentials(t *testing.T) {
	uc := &UserContainer{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeee000011",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "live", Config: json.RawMessage(`{"symbol":"BTCUSDT"}`)},
		},
	}
	yamlBytes, err := buildUserYAML(uc, func(exchange string) bool {
		return false
	})
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)

	if !strings.Contains(yamlStr, "publicOnly: true") {
		t.Error("live mode without credentials should set publicOnly: true")
	}
	if strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Error("live mode should not have PAPER_TRADE even without credentials")
	}
}
