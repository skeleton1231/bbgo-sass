package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAPI_DownloadBacktestReport_JobNotCompleted(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-pending", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-pending/download", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-completed job, got %d", w.Code)
	}
}

func TestAPI_DownloadBacktestReport_SignedURL(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-signed", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-signed", JobCompleted, "done")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"signedUrl": "/signed/path"})
	}))
	defer srv.Close()

	api.storage = NewStorageClient(srv.URL, "test-key")

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-signed/download?signed=1&file=summary.json", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DownloadBacktestReport_SignedUnsupportedFile(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-signed2", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-signed2", JobCompleted, "done")

	api.storage = NewStorageClient("http://127.0.0.1:1", "test-key")

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-signed2/download?signed=1&file=bad.exe", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported file, got %d", w.Code)
	}
}

func TestAPI_DownloadCSV_OrdersFile(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-orders", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-orders", JobCompleted, "done")

	reportDir := api.container.BacktestReportDir(testUUID, "bt-orders")
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "orders.tsv"), []byte("order_id\tprice\n1\t50000\n"), 0o644)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-orders/download?file=orders", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/csv") {
		t.Errorf("expected csv, got %s", w.Header().Get("Content-Type"))
	}
}

func TestAPI_DownloadCSV_EquityFile(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-eq", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config:    json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-eq", JobCompleted, "done")
	api.btJobs.SetReport("bt-eq", json.RawMessage(`{}`), "timestamp,equity\n1,100\n")

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-eq/download?file=equity", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DownloadCSV_EquityNotAvailable(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-noeq", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-noeq", JobCompleted, "done")

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-noeq/download?file=equity", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing equity, got %d", w.Code)
	}
}

func TestAPI_DownloadCSV_KlineFile(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-kline", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-kline", JobCompleted, "done")

	reportDir := api.container.BacktestReportDir(testUUID, "bt-kline")
	os.MkdirAll(reportDir, 0o755)
	os.WriteFile(filepath.Join(reportDir, "BTCUSDT-1h.tsv"), []byte("time\topen\thigh\tlow\tclose\n1\t100\t110\t90\t105\n"), 0o644)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-kline/download?file=kline", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_DownloadCSV_KlineNotFound(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-nokline", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-nokline", JobCompleted, "done")

	reportDir := api.container.BacktestReportDir(testUUID, "bt-nokline")
	os.MkdirAll(reportDir, 0o755)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-nokline/download?file=kline", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing kline, got %d", w.Code)
	}
}

func TestAPI_DownloadCSV_UnsupportedFileType(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-unsupported", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-unsupported", JobCompleted, "done")

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-unsupported/download?file=badfile", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported file type, got %d", w.Code)
	}
}

func TestAPI_Health_WithRunningInstance(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)
	api.container.checkRunningFn = func(string) (bool, error) { return true, nil }

	w := doRequest(r, "GET", "/api/health", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("expected ok, got %s", resp.Status)
	}
	if resp.Running != 1 {
		t.Errorf("expected 1 running, got %d", resp.Running)
	}
}

func TestAPI_HasDataForRange_NoDB(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	api.container.cfg.BacktestSharedDir = t.TempDir()

	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("expected false when no DB exists")
	}
}

func TestAPI_HasDataForRange_SmallFile(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	dir := t.TempDir()
	api.container.cfg.BacktestSharedDir = dir
	os.WriteFile(filepath.Join(dir, "backtest.db"), []byte("tiny"), 0o644)

	if api.hasDataForRange("binance", "BTCUSDT", "2024-01-01", "2024-06-01") {
		t.Error("expected false for small DB file")
	}
}

func TestAPI_HasDataForRange_BadDates(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	dir := t.TempDir()
	api.container.cfg.BacktestSharedDir = dir
	os.WriteFile(filepath.Join(dir, "backtest.db"), make([]byte, 2048), 0o644)

	if api.hasDataForRange("binance", "BTCUSDT", "bad-date", "2024-06-01") {
		t.Error("expected false for bad dates")
	}
}

func TestAPI_DownloadCSV_TradesNotFound(t *testing.T) {
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)

	job := &BacktestJob{
		ID: "bt-notrades", UserID: testUUID, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		StartTime: "2024-01-01", EndTime: "2024-03-01",
		Config: json.RawMessage(`{}`),
	}
	api.btJobs.Create(job)
	api.btJobs.UpdateStatus("bt-notrades", JobCompleted, "done")

	reportDir := api.container.BacktestReportDir(testUUID, "bt-notrades")
	os.MkdirAll(reportDir, 0o755)

	w := doRequest(r, "GET", "/api/backtest/jobs/bt-notrades/download?file=trades", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing trades, got %d", w.Code)
	}
}


func TestAPI_UploadLocalToStorage_DisallowedFile(t *testing.T) {
	api, _ := setupHandlerAPI(t)
	api.storage = NewStorageClient("http://127.0.0.1:1", "key")

	job := &BacktestJob{ID: "bt-ul", UserID: testUUID}
	if api.uploadLocalToStorage(job, "bad.exe") {
		t.Error("expected false for disallowed file")
	}
}

// --- BBGo handler tests with mock server ---

func setupMockBBGo(t *testing.T, handler http.HandlerFunc) (*API, *chi.Mux, *httptest.Server) {
	t.Helper()
	api, r := setupHandlerAPI(t)
	createTestInstance(t, api.store, testUUID, "live", "grid2", "BTCUSDT", nil)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	api.newBBGoClient = func(baseURL string) *BBGoClient {
		return &BBGoClient{baseURL: srv.URL, client: srv.Client()}
	}
	return api, r, srv
}

func TestAPI_BBGoPing_MockServer(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/ping?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoPing_Error(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/ping?mode=live", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestAPI_BBGoSessions_MockServer(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BBGoSessionsResponse{Sessions: []BBGoSession{{Name: "binance"}}})
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/sessions?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoSessions_Error(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/sessions?mode=live", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestAPI_BBGoSessionDetail_Error(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/binance?mode=live", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestAPI_BBGoSessionSymbols_MockServer(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BBGoSymbolsResponse{Symbols: []string{"BTCUSDT", "ETHUSDT"}})
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/binance/symbols?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_BBGoSessionSymbols_Error(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/session/binance/symbols?mode=live", nil)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestAPI_BBGoStrategies_MockServer(t *testing.T) {
	_, r, _ := setupMockBBGo(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BBGoStrategiesResponse{Strategies: []BBGoStrategyState{
			{"strategy": "grid"},
		}})
	})
	w := doRequest(r, "GET", "/api/users/"+testUUID+"/bbgo/strategies?mode=live", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAPI_MarketTicker_MissingParams(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/markets/binance/ticker", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_MarketKlines_MissingParams(t *testing.T) {
	_, r := setupHandlerAPI(t)
	w := doRequest(r, "GET", "/api/markets/binance/klines", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
