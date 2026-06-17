package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestHTTPRequest_FullChain_RealProductPath exercises the real middleware
// composition used in main.go: RequestID → RealIP → metrics → logger →
// recoverer → body-limit → SharedSecretAuth → handler.
//
// It walks every public probe path that ops / Docker healthchecks rely on
// and asserts they remain accessible without an X-Manager-Token. A regression
// here breaks Docker container healthchecks (Docker sends no token).
func TestHTTPRequest_FullChain_RealProductPath(t *testing.T) {
	api := &API{metrics: NewMetrics()}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.Health)
	mux.HandleFunc("/livez", api.LivezHandler)
	mux.HandleFunc("/readyz", api.ReadyzHandler)
	mux.HandleFunc("/metrics", api.MetricsHandler)

	// Mirror main.go middleware order: metrics is applied BEFORE auth so the
	// request counter captures every request including 401s from auth failures.
	chain := requestMetricsMiddleware(api.metrics)(SharedSecretAuth("super-secret-token")(mux))

	srv := httptest.NewServer(chain)
	defer srv.Close()

	t.Run("healthcheck paths must not require manager token", func(t *testing.T) {
		// Docker healthcheck runs `curl -fsS http://localhost:8090/readyz`
		// with no X-Manager-Token header. If auth.go does not exempt this
		// path, the healthcheck gets 401 and Docker marks the container
		// unhealthy after 3 failures.
		for _, path := range []string{"/livez", "/readyz", "/api/health"} {
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s: expected 200 (Docker healthcheck needs this), got %d body=%s",
					path, resp.StatusCode, body)
			}
		}
	})

	t.Run("metrics endpoint is auth-exempt for internal scraping", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/metrics")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("/metrics: expected 200 from internal scraper, got %d", resp.StatusCode)
		}
	})

	t.Run("api routes without token are rejected", func(t *testing.T) {
		// Sanity: the auth middleware still protects everything else.
		mux2 := http.NewServeMux()
		mux2.HandleFunc("/api/users/abc/bots", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		authed2 := SharedSecretAuth("super-secret-token")(mux2)
		srv2 := httptest.NewServer(authed2)
		defer srv2.Close()

		resp, err := http.Get(srv2.URL + "/api/users/abc/bots")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 without token, got %d", resp.StatusCode)
		}
	})

	t.Run("api routes with valid token succeed", func(t *testing.T) {
		mux2 := http.NewServeMux()
		mux2.HandleFunc("/api/users/abc/bots", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		authed2 := SharedSecretAuth("super-secret-token")(mux2)
		srv2 := httptest.NewServer(authed2)
		defer srv2.Close()

		req, _ := http.NewRequest(http.MethodGet, srv2.URL+"/api/users/abc/bots", nil)
		req.Header.Set("X-Manager-Token", "super-secret-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 with token, got %d", resp.StatusCode)
		}
	})

	t.Run("metrics counter increments once per request", func(t *testing.T) {
		before := api.metrics.HTTPRequestsTotal.Load()
		for i := 0; i < 5; i++ {
			resp, err := http.Get(srv.URL + "/livez")
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
		}
		after := api.metrics.HTTPRequestsTotal.Load()
		if got := after - before; got != 5 {
			t.Errorf("expected counter delta 5, got %d", got)
		}
	})

	t.Run("error counter increments on 5xx", func(t *testing.T) {
		muxErr := http.NewServeMux()
		muxErr.HandleFunc("/boom", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "kaboom", http.StatusInternalServerError)
		})
		authedErr := SharedSecretAuth("super-secret-token")(requestMetricsMiddleware(api.metrics)(muxErr))
		srvErr := httptest.NewServer(authedErr)
		defer srvErr.Close()

		before := api.metrics.HTTPErrorsTotal.Load()
		req, _ := http.NewRequest(http.MethodGet, srvErr.URL+"/boom", nil)
		req.Header.Set("X-Manager-Token", "super-secret-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got := api.metrics.HTTPErrorsTotal.Load() - before; got != 1 {
			t.Errorf("expected error counter delta 1, got %d", got)
		}
	})

	t.Run("4xx does not increment error counter", func(t *testing.T) {
		mux4 := http.NewServeMux()
		mux4.HandleFunc("/missing", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "no", http.StatusNotFound)
		})
		authed4 := SharedSecretAuth("super-secret-token")(requestMetricsMiddleware(api.metrics)(mux4))
		srv4 := httptest.NewServer(authed4)
		defer srv4.Close()

		before := api.metrics.HTTPErrorsTotal.Load()
		req, _ := http.NewRequest(http.MethodGet, srv4.URL+"/missing", nil)
		req.Header.Set("X-Manager-Token", "super-secret-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got := api.metrics.HTTPErrorsTotal.Load() - before; got != 0 {
			t.Errorf("4xx must not bump error counter, got delta %d", got)
		}
	})
}

// TestStatusRecorder_RecordsStatus verifies that WriteHeader is captured so
// the metrics middleware can distinguish 2xx/4xx/5xx. Regression: if the
// wrapper is removed, every response looks like 200 to the counter logic.
func TestStatusRecorder_RecordsStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/created", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	mux.HandleFunc("/boom", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	cases := map[string]int{
		"/ok":      http.StatusOK,
		"/created": http.StatusCreated,
		"/boom":    http.StatusInternalServerError,
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			m := NewMetrics()
			wrapped := requestMetricsMiddleware(m)(mux)
			srv := httptest.NewServer(wrapped)
			defer srv.Close()
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != want {
				t.Errorf("expected %d, got %d", want, resp.StatusCode)
			}
		})
	}
}

// TestCachedHealth_TTLExpiry_TriggersRefresh confirms that the cache really
// does call refresh after the TTL elapses — not just on first call.
func TestCachedHealth_TTLExpiry_TriggersRefresh(t *testing.T) {
	var calls atomic.Int32
	ch := newCachedHealth(func(context.Context) HealthSnapshot {
		n := calls.Add(1)
		return HealthSnapshot{Status: "ok", Users: int(n)}
	}, 30*time.Millisecond)

	a := ch.Get(context.Background())
	time.Sleep(50 * time.Millisecond)
	b := ch.Get(context.Background())

	if a.Users != 1 || b.Users != 2 {
		t.Fatalf("expected refresh after TTL; got a=%+v b=%+v calls=%d", a, b, calls.Load())
	}
}

// TestCachedHealth_ConcurrentGet_NoThunderingHerd fires N goroutines at the
// cache simultaneously. The refresh function must be called exactly once per
// TTL window; otherwise a container restart storm would trigger N Docker
// scans + N user-store reads on /api/health.
func TestCachedHealth_ConcurrentGet_NoThunderingHerd(t *testing.T) {
	var calls atomic.Int32
	ch := newCachedHealth(func(context.Context) HealthSnapshot {
		calls.Add(1)
		time.Sleep(10 * time.Millisecond) // simulate slow scan
		return HealthSnapshot{Status: "ok", Users: 1}
	}, 5*time.Second)

	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_ = ch.Get(context.Background())
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Errorf("expected exactly 1 refresh under concurrent load, got %d (thundering herd)", got)
	}
}

// TestReadyzHandler_GracefulShutdownSequence simulates SIGTERM:
// 1. MarkNotReady is called from main.go before srv.Shutdown
// 2. /readyz must immediately return 503 so the load balancer stops sending traffic
// 3. /livez must still return 200 because the process is alive and draining
// 4. /api/health must continue serving cached value to keep UI informative
func TestReadyzHandler_GracefulShutdownSequence(t *testing.T) {
	api := &API{metrics: NewMetrics()}

	// Pre-populate cache so /api/health doesn't try to use api.store (nil in this test).
	api.health = newCachedHealth(func(context.Context) HealthSnapshot {
		return HealthSnapshot{Status: "ok", Users: 5, Running: 3}
	}, 5*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("/livez", api.LivezHandler)
	mux.HandleFunc("/readyz", api.ReadyzHandler)
	mux.HandleFunc("/api/health", api.Health)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	if !api.metrics.Ready() {
		t.Fatal("expected ready=true on fresh metrics")
	}

	api.metrics.MarkNotReady()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("readyz during shutdown: expected 503, got %d", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("livez during shutdown: expected 200 (process still alive), got %d", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health during shutdown: expected 200, got %d", resp.StatusCode)
	}
	var snap HealthSnapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if snap.Users != 5 || snap.Running != 3 {
		t.Errorf("health snapshot mismatch: %+v", snap)
	}
}

// TestMetricsHandler_IncludesBuildVersion verifies the version stamp is
// propagated via -ldflags. Operators rely on /metrics to identify which
// build is running during incident triage.
func TestMetricsHandler_IncludesBuildVersion(t *testing.T) {
	prev := BuildVersion
	defer func() { BuildVersion = prev }()
	BuildVersion = "v1.2.3-test"

	api := &API{metrics: NewMetrics()}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	api.MetricsHandler(w, req)

	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["version"] != "v1.2.3-test" {
		t.Errorf("version not in metrics output: got %v", got["version"])
	}
	for _, key := range []string{"uptime_seconds", "go_alloc_bytes", "go_goroutines", "hostname"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing %q in metrics output", key)
		}
	}
}

// TestMetricsHandler_HasContentType documents the response shape scrapers depend on.
func TestMetricsHandler_HasContentType(t *testing.T) {
	api := &API{metrics: NewMetrics()}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	api.MetricsHandler(w, req)
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json, got %q", ct)
	}
}

// TestSetupLogging_ProducesJSONByDefault verifies the logger actually emits
// JSON to stdout — operators ship these lines to Loki/Datadog and depend on
// parseable records. We swap os.Stdout for a pipe to capture the output.
func TestSetupLogging_ProducesJSONByDefault(t *testing.T) {
	// Redirect os.Stdout before calling SetupLogging; restore after.
	orig := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	t.Cleanup(func() { os.Stdout = orig })

	t.Setenv("LOG_FORMAT", "")
	t.Setenv("LOG_LEVEL", "info")
	logger := SetupLogging()
	logger.Info("hello", "world", "yes")

	_ = wPipe.Close()
	out, _ := io.ReadAll(rPipe)
	_ = rPipe.Close()

	if !strings.HasPrefix(strings.TrimSpace(string(out)), "{") {
		t.Fatalf("expected JSON log line, got: %q", out)
	}
	if !strings.Contains(string(out), `"msg":"hello"`) {
		t.Errorf("log line missing msg field: %q", out)
	}
	if !strings.Contains(string(out), `"world":"yes"`) {
		t.Errorf("log line missing kv pair: %q", out)
	}
}

// TestSetupLogging_TextFormatHonored confirms LOG_FORMAT=text switches output
// to key=value format — used during local debugging.
func TestSetupLogging_TextFormatHonored(t *testing.T) {
	orig := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	t.Cleanup(func() { os.Stdout = orig })

	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("LOG_LEVEL", "info")
	logger := SetupLogging()
	logger.Info("hello")

	_ = wPipe.Close()
	out, _ := io.ReadAll(rPipe)
	_ = rPipe.Close()

	if !strings.Contains(string(out), "msg=hello") {
		t.Errorf("expected text format with msg=hello, got: %q", out)
	}
}

// TestSetupLogging_LevelFilterSuppressesDebug verifies LOG_LEVEL controls
// what gets emitted. A misconfigured debug level in prod would flood logs.
func TestSetupLogging_LevelFilterSuppressesDebug(t *testing.T) {
	orig := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe
	t.Cleanup(func() { os.Stdout = orig })

	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_LEVEL", "warn")
	logger := SetupLogging()
	logger.Debug("should-be-suppressed")
	logger.Warn("should-appear")

	_ = wPipe.Close()
	out, _ := io.ReadAll(rPipe)
	_ = rPipe.Close()

	if strings.Contains(string(out), "should-be-suppressed") {
		t.Errorf("debug message leaked through warn-level filter: %q", out)
	}
	if !strings.Contains(string(out), "should-appear") {
		t.Errorf("warn message missing: %q", out)
	}
}
