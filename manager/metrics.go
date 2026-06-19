package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// requestMetricsMiddleware counts requests/errors and records latency.
func requestMetricsMiddleware(m *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)
			m.HTTPRequestsTotal.Add(1)
			if ww.status >= 500 {
				m.HTTPErrorsTotal.Add(1)
			}
			slog.Debug("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.status,
				"latency_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// structuredLoggerMiddleware emits one JSON log line per request when the
// chi RequestID middleware is present.
func structuredLoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Metrics holds lightweight counters reported on /metrics and /api/health.
// All fields are atomic — safe for concurrent increments.
type Metrics struct {
	StartedAt            time.Time
	HTTPRequestsTotal    atomic.Int64
	HTTPErrorsTotal      atomic.Int64
	ContainerStartsTotal atomic.Int64
	ContainerStopsTotal  atomic.Int64
	BacktestsRunTotal    atomic.Int64
	BacktestsFailedTotal atomic.Int64
	WSClientsCurrent     atomic.Int64
	gateReady            atomic.Bool
}

func NewMetrics() *Metrics {
	m := &Metrics{StartedAt: time.Now()}
	m.gateReady.Store(true)
	return m
}

// Ready returns false when an explicit readiness gate is unset (during shutdown).
func (m *Metrics) Ready() bool { return m.gateReady.Load() }

// MarkNotReady flips readiness off — used during graceful shutdown.
func (m *Metrics) MarkNotReady() { m.gateReady.Store(false) }

// UptimeSeconds reports process uptime in seconds.
func (m *Metrics) UptimeSeconds() float64 {
	return time.Since(m.StartedAt).Seconds()
}

// LivezHandler responds 200 as long as the process is running and ready.
func (api *API) LivezHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// ReadyzHandler responds 200 only when readiness gate is true.
func (api *API) ReadyzHandler(w http.ResponseWriter, _ *http.Request) {
	if !api.metrics.Ready() {
		http.Error(w, "shutting down", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

// MetricsHandler exposes a JSON snapshot suitable for scraping.
// We emit JSON to avoid pulling in a Prometheus client dependency. The shape
// follows Prometheus exporter naming conventions so an external exporter
// (e.g. a tiny Go sidecar or caddy-metrics shim) can reformat if needed.
func (api *API) MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	hostname, _ := os.Hostname()
	snapshot := map[string]any{
		"uptime_seconds":         api.metrics.UptimeSeconds(),
		"http_requests_total":    api.metrics.HTTPRequestsTotal.Load(),
		"http_errors_total":      api.metrics.HTTPErrorsTotal.Load(),
		"container_starts_total": api.metrics.ContainerStartsTotal.Load(),
		"container_stops_total":  api.metrics.ContainerStopsTotal.Load(),
		"backtests_run_total":    api.metrics.BacktestsRunTotal.Load(),
		"backtests_failed_total": api.metrics.BacktestsFailedTotal.Load(),
		"ws_clients_current":     api.metrics.WSClientsCurrent.Load(),
		"go_alloc_bytes":         mem.Alloc,
		"go_sys_bytes":           mem.Sys,
		"go_heap_objects":        mem.HeapObjects,
		"go_goroutines":          runtime.NumGoroutine(),
		"hostname":               hostname,
		"version":                BuildVersion,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

// HealthSnapshot is the cached view returned by /api/health.
type HealthSnapshot struct {
	Status    string `json:"status"`
	Users     int    `json:"users"`
	Running   int    `json:"running"`
	Version   string `json:"version,omitempty"`
	UptimeSec int64  `json:"uptime_seconds"`
}

// cachedHealth refreshes the health view periodically to keep the Docker
// healthcheck cheap (was scanning all users + listing Docker containers
// every 30s before this change).
type cachedHealth struct {
	mu       sync.Mutex
	view     HealthSnapshot
	loadedAt time.Time
	ttl      time.Duration
	refresh  func(context.Context) HealthSnapshot
}

func newCachedHealth(refresh func(context.Context) HealthSnapshot, ttl time.Duration) *cachedHealth {
	return &cachedHealth{ttl: ttl, refresh: refresh}
}

func (c *cachedHealth) Get(ctx context.Context) HealthSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.loadedAt) < c.ttl {
		return c.view
	}
	c.view = c.refresh(ctx)
	c.loadedAt = time.Now()
	return c.view
}

func (api *API) refreshHealth(context.Context) HealthSnapshot {
	users := api.store.ScanUsers()
	runningSet := api.container.ListAllRunningInstanceContainers()
	running := 0
	for _, um := range users {
		instances, _ := api.store.ListInstances(um.UserID, um.Mode)
		for i := range instances {
			name := api.container.InstanceContainerName(instances[i].UserID, instances[i].Mode, instances[i].InstanceID)
			if runningSet[name] {
				running++
			}
		}
	}
	return HealthSnapshot{
		Status:    "ok",
		Users:     len(users),
		Running:   running,
		Version:   BuildVersion,
		UptimeSec: int64(api.metrics.UptimeSeconds()),
	}
}

// BuildVersion is set via -ldflags at build time. Default "dev" for local builds.
var BuildVersion = "dev"

func init() {
	if v := os.Getenv("BUILD_VERSION"); v != "" {
		BuildVersion = v
	}
}
