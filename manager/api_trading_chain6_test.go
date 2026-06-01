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
	store, _ := newTestStore(t)
	writeTestStrategies(t, store, "aaaaaaaa-bbbb-cccc-dddd-eeeeee000001", ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	bbgoSrv := httptest.NewServer(http.HandlerFunc(bbgoHandler))
	api := &API{
		strategies: store,
		container:  &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		newBBGoClient: func(baseURL string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
		wsTickets:        NewWSTicketStore(),
		containerRunning: func(_, _ string) bool { return true },
	}
	return api, func() { api.Close(); bbgoSrv.Close() }
}

const proxyUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeee000001"

// --- Session detail proxy ---

func TestBBGoSessionDetail_Proxy(t *testing.T) {
	api, cleanup := setupProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance" {
			w.Write([]byte(`{"session":{"name":"binance","exchange":"binance"}}`))
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
	store, _ := newTestStore(t)
	writeTestStrategies(t, store, proxyUID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	api := &API{
		strategies:       store,
		container:        &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		wsTickets:        NewWSTicketStore(),
		containerRunning: func(_, _ string) bool { return false },
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
			w.Write([]byte(`{"trades":{"BTCUSDT":{"Trades":[{"id":1,"symbol":"BTCUSDT"}]}}}`))
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

func TestSyncCredential_Basic(t *testing.T) {
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	supaClient, err := NewSupabaseClient(supabaseSrv.URL, "k")
	if err != nil {
		t.Fatal(err)
	}
	s := NewSyncer(supaClient)
	s.SyncCredential(ExchangeCredential{Exchange: "binance"})
}

// --- SyncBacktestData validation ---

func TestSyncBacktestData_InvalidBody(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{BBGOPort: 8080}}
	api := &API{
		container: cm,
		btSyncSem: make(chan struct{}, 2),
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

	req := httptest.NewRequest("POST", "/api/backtest/sync", strings.NewReader("not json"))
	req.Header.Set("X-User-Id", proxyUID)
	req.Header.Set("X-Manager-Token", "test")
	w := httptest.NewRecorder()
	api.SyncBacktestData(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid body: status = %d, want 400", w.Code)
	}
}

func TestSyncBacktestData_TooManySymbols(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{BBGOPort: 8080}}
	api := &API{
		container: cm,
		btSyncSem: make(chan struct{}, 2),
		wsTickets: NewWSTicketStore(),
	}
	defer api.Close()

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

// --- bbgoClientForUser stopped container ---

func TestBBGoClientForUser_StoppedContainer(t *testing.T) {
	store, _ := newTestStore(t)
	writeTestStrategies(t, store, proxyUID, ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
	})

	api := &API{
		strategies:       store,
		container:        &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		wsTickets:        NewWSTicketStore(),
		containerRunning: func(_, _ string) bool { return false },
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
