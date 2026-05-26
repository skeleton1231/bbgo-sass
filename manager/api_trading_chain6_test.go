package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func chiReq(method, url, body, paramKey, paramVal string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	if paramKey != "" {
		rctx.URLParams.Add(paramKey, paramVal)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func chiReq2(method, url, body string, params map[string]string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func setupProxyAPI(t *testing.T, bbgoHandler http.HandlerFunc) (*API, func()) {
	t.Helper()
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000001", StrategyEntry{Exchange: "binance", Strategy: "grid2"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000001", StatusRunning)

	bbgoSrv := httptest.NewServer(http.HandlerFunc(bbgoHandler))
	api := &API{
		users:     users,
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		newBBGoClient: func(baseURL string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
		wsTickets: NewWSTicketStore(),
	}
	return api, func() { api.Close(); bbgoSrv.Close() }
}

const proxyUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeee000001"

// --- Session detail proxy ---

func TestBBGoSessionDetail_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance" {
			w.Write([]byte(`{"session":{"name":"binance","exchangeName":"binance"}}`))
		}
	})
	defer cleanup()

	req := chiReq2("GET", "/api/users/"+proxyUID+"/bbgo/sessions/binance", "", map[string]string{"userID": proxyUID, "session": "binance"})
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoSessionDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

func TestBBGoSessionDetail_NotRunning(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy(proxyUID, StrategyEntry{Exchange: "binance", Strategy: "grid2"})

	api := &API{
		users:     users,
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	req := chiReq2("GET", "/api/users/"+proxyUID+"/bbgo/sessions/binance", "", map[string]string{"userID": proxyUID, "session": "binance"})
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoSessionDetail(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("not running: status = %d, want 503", w.Code)
	}
}

// --- Session trades proxy ---

func TestBBGoSessionTrades_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance/trades" {
			w.Write([]byte(`{"trades":[{"id":1,"symbol":"BTCUSDT"}]}`))
		}
	})
	defer cleanup()

	req := chiReq2("GET", "/api/users/"+proxyUID+"/bbgo/sessions/binance/trades", "", map[string]string{"userID": proxyUID, "session": "binance"})
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoSessionTrades(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- Session open orders proxy ---

func TestBBGoSessionOpenOrders_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance/open-orders" {
			w.Write([]byte(`{"orders":[{"orderID":1,"symbol":"BTCUSDT","side":"BUY"}]}`))
		}
	})
	defer cleanup()

	req := chiReq2("GET", "/api/users/"+proxyUID+"/bbgo/sessions/binance/orders/open", "", map[string]string{"userID": proxyUID, "session": "binance"})
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoSessionOpenOrders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- Session balances proxy ---

func TestBBGoSessionBalances_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance/account/balances" {
			w.Write([]byte(`{"balances":{"USDT":{"currency":"USDT","available":"1000","locked":"0"}}}`))
		}
	})
	defer cleanup()

	req := chiReq2("GET", "/api/users/"+proxyUID+"/bbgo/sessions/binance/balances", "", map[string]string{"userID": proxyUID, "session": "binance"})
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoSessionBalances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- BBGoPing proxy ---

func TestBBGoPing_Success(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message":"ok"}`))
	})
	defer cleanup()

	req := chiReq("GET", "/api/users/"+proxyUID+"/bbgo/ping", "", "userID", proxyUID)
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoPing(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- BBGoTradingVolume proxy ---

func TestBBGoTradingVolume_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"volume":[{"symbol":"BTCUSDT","buyVolume":"1.5","sellVolume":"0.5"}]}`))
	})
	defer cleanup()

	req := chiReq("GET", "/api/users/"+proxyUID+"/bbgo/trading-volume", "", "userID", proxyUID)
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoTradingVolume(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- SyncCredential ---

func TestSyncCredential_UpsertsUser(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("aaaaaaaa-bbbb-cccc-dddd-eeeeee000020", StrategyEntry{Exchange: "binance", Strategy: "grid2", Mode: "live"})
	users.UpdateStatus("aaaaaaaa-bbbb-cccc-dddd-eeeeee000020", StatusRunning)

	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "user_containers") {
			w.WriteHeader(200)
			w.Write([]byte(`[{"user_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeee000020"}]`))
		} else {
			w.WriteHeader(200)
			w.Write([]byte(`[]`))
		}
	}))
	defer supabaseSrv.Close()

	s := &Syncer{
		users:  users,
		cfg:    &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		client: supabaseSrv.Client(),
	}
	s.SyncCredential(ExchangeCredential{UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeee000020", Exchange: "binance"})
}

// --- SyncBacktestData validation ---

func TestSyncBacktestData_InvalidBody(t *testing.T) {
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: &Config{BBGOPort: 8080}}
	api := &API{
		users:     users,
		container: cm,
		btSyncSem: make(chan struct{}, 2),
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	req := httptest.NewRequest("POST", "/api/backtest/sync", strings.NewReader(`not json`))
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.SyncBacktestData(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid body: status = %d, want 400", w.Code)
	}
}

func TestSyncBacktestData_TooManySymbols(t *testing.T) {
	users := NewUserContainerManager()
	cm := &ContainerManager{cfg: &Config{BBGOPort: 8080}}
	api := &API{
		users:     users,
		container: cm,
		btSyncSem: make(chan struct{}, 2),
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	symbols := make([]string, 11)
	for i := range symbols {
		symbols[i] = "SYM" + strings.Repeat("A", 10)
	}
	body := `{"exchange":"binance","symbols":["A","B","C","D","E","F","G","H","I","J","K"]}`
	req := httptest.NewRequest("POST", "/api/backtest/sync", strings.NewReader(body))
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.SyncBacktestData(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("too many symbols: status = %d, want 400", w.Code)
	}
}

// --- UpsertUser edge cases ---

func TestUpsertUser_NewUser(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	s := &Syncer{
		cfg:    &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		client: supabaseSrv.Client(),
	}

	uc := &UserContainer{
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeee000021",
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "live"},
		},
	}
	s.UpsertUser(uc)
}

func TestUpsertUser_SupabaseError(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer supabaseSrv.Close()

	s := &Syncer{
		cfg:    &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		client: supabaseSrv.Client(),
	}

	uc := &UserContainer{UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeee000022", Status: StatusRunning}
	// UpsertUser logs errors but does not return them; just ensure it doesn't panic
	s.UpsertUser(uc)
}

// --- bbgoClientForUser stopped container ---

func TestBBGoClientForUser_StoppedContainer(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy(proxyUID, StrategyEntry{Exchange: "binance", Strategy: "grid2"})

	api := &API{
		users:     users,
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	req := chiReq("GET", "/api/users/"+proxyUID+"/bbgo/ping", "", "userID", proxyUID)
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.BBGoPing(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("stopped container: status = %d, want 503", w.Code)
	}
}
