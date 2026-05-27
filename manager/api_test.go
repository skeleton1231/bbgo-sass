package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestAPI(bbgoHandler http.HandlerFunc) (*API, *httptest.Server) {
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		Mode:       ModeLive,
		UserID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status:     StatusRunning,
		Strategies: []StrategyEntry{{ID: "s1", Exchange: "binance", Strategy: "grid"}},
	}

	bbgoSrv := httptest.NewServer(bbgoHandler)

	cfg := &Config{
		SupabaseURL:  "http://localhost:1",
		SupabaseKey:  "test",
		ManagerToken: "test-token",
	}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)
	api.newBBGoClient = func(_ string) *BBGoClient {
		return NewBBGoClient(bbgoSrv.URL)
	}
	return api, bbgoSrv
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

func TestAPI_BBGoPing(t *testing.T) {
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/ping", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoSessions(t *testing.T) {
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sessions": []BBGoSession{{Name: "binance", ExchangeName: "binance"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/sessions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	sessions := resp["sessions"].([]interface{})
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestAPI_BBGoBalances(t *testing.T) {
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/binance/account/balances" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"balances": map[string]interface{}{
					"BTC": map[string]string{"currency": "BTC", "available": "1.5", "locked": "0.5"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/session/binance/balances", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoTrades_WithQueryParams(t *testing.T) {
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trades" {
			if r.URL.Query().Get("exchange") != "binance" {
				t.Errorf("expected exchange=binance, got %s", r.URL.Query().Get("exchange"))
			}
			if r.URL.Query().Get("gid") != "100" {
				t.Errorf("expected gid=100, got %s", r.URL.Query().Get("gid"))
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"trades": []interface{}{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/trades?exchange=binance&gid=100", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGo_UserNotFound(t *testing.T) {
	users := NewUserContainerManager()
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/ping", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown user, got %d", w.Code)
	}
}

func TestAPI_BBGo_InvalidUserID(t *testing.T) {
	users := NewUserContainerManager()
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/not-a-uuid/bbgo/ping", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestAPI_BBGo_ContainerStopped(t *testing.T) {
	users := NewUserContainerManager()
	users.users["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:"+ModeLive] = &UserContainer{
		Mode:   ModeLive,
		UserID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Status: StatusStopped,
	}
	cfg := &Config{ManagerToken: "test-token"}
	cm := &ContainerManager{cfg: cfg, pool: nil}
	proxy := NewBotProxy(cm)
	api := NewAPI(cfg, users, cm, proxy, nil, nil, nil, nil, nil, nil, nil)

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/ping", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for stopped container, got %d", w.Code)
	}
}

func TestAPI_BBGoAssets(t *testing.T) {
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/assets" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"assets": map[string]interface{}{"BTC": map[string]string{"currency": "BTC", "total": "1.0", "available": "1.0", "lock": "0", "netAsset": "1.0", "netAssetInUSD": "43000"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/assets", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoClosedOrders(t *testing.T) {
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/orders/closed" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{"orderID": 100, "symbol": "BTCUSDT", "status": "FILLED"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/orders/closed", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoTradingVolume(t *testing.T) {
	api, bbgoSrv := setupTestAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/trading-volume" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tradingVolumes": []interface{}{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bbgoSrv.Close()

	r := testRouter(api)
	req := httptest.NewRequest("GET", "/api/users/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/bbgo/trading-volume?period=day", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
