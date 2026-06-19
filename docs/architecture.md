# Architecture

This document describes the BBGO SaaS system: components, data flows, multi-tenant isolation, and operational characteristics. For hands-on bring-up, see [`deployment-guide.md`](deployment-guide.md). For per-component code reference, see [`../CLAUDE.md`](../CLAUDE.md).

## System overview

```
                      Internet
                         │
                         ▼
                  ┌──────────────┐
                  │    Caddy      │   TLS termination, HSTS, CSP, /metrics block
                  │  (auto-HTTPS) │
                  └──────┬───────┘
            ┌────────────┼─────────────┐
            ▼            ▼             ▼
      ┌─────────┐  ┌──────────┐  ┌──────────────┐
      │  Web    │  │ Manager  │  │  (exposed    │
      │ Next.js │  │ Go HTTP  │  │   only via   │
      │ :3142   │  │ :8090    │  │   Caddy)     │
      └────┬────┘  └─────┬────┘  └──────────────┘
           │             │
           │             │ Docker socket
           │             ▼
           │      ┌────────────────────────────┐
           │      │ BBGO instances (per user)  │
           │      │ ┌──────────┐ ┌──────────┐  │
           │      │ │ user-A-  │ │ user-B-  │  │
           │      │ │ inst-1   │ │ inst-1   │  │
           │      │ └────┬─────┘ └────┬─────┘  │
           │      └──────┼────────────┼───────┘
           │             │            │
           │             ▼            ▼
           │      ┌──────────────────────────┐
           │      │  Market Data Hub         │
           │      │  bbgo-marketdata         │
           │      │  bbgo-marketdata-testnet │
           │      │  (gRPC :9090)            │
           │      └──────────────────────────┘
           │
           ▼
    ┌──────────────────────────────────────────┐
    │  Supabase (PostgreSQL + Realtime)         │
    │  - Auth                                    │
    │  - RLS on every user-scoped table          │
    │  - Realtime broadcast for live/paper       │
    └──────────────────────────────────────────┘
```

## Component responsibilities

### Caddy

Reverse proxy, TLS termination, security headers. Single source of truth for:

- ACME certificate issuance (Let's Encrypt via `ACME_EMAIL`)
- HSTS preload, CSP, X-Frame-Options, Referrer-Policy
- Static asset caching (immutable `Cache-Control` for `/_next/static/`)
- Internal-only route blocking (`/metrics`, `/livez`, `/readyz` return 404 to external clients)

See [`docker/Caddyfile`](../docker/Caddyfile).

### Web (Next.js 16)

Single-page dashboard. Standalone build (`output: 'standalone'`) running as uid 10001 in production.

Responsibilities:

- Supabase auth (sign-in / sign-out, session refresh)
- Strategy config UI (50 strategies across 10 categories)
- Bot lifecycle (start/stop, status, logs)
- Backtest submission and result viewing
- Live market data via Supabase Realtime + manager WebSocket
- Read paths for trades / orders / PnL via Supabase direct queries (fallback when container is down)

All backend calls are proxied through `/api/manager/*` which enforces Supabase auth before forwarding. The manager never sees the user's Supabase session — it only sees `X-User-Id` and `X-Manager-Token`.

See [`../web/src`](../web/src) and [`../CLAUDE.md`](../CLAUDE.md) "Web Frontend" section.

### Manager (Go HTTP)

Orchestration brain. Responsibilities:

- **Instance lifecycle** — start/stop/restart Docker containers per user strategy
- **Credential management** — AES-GCM encrypted on disk under `{DATA_DIR}/{userID}/credentials.json`
- **Config generation** — deep-merge registry defaults with user config → YAML for the bbgo binary
- **Backtest execution** — isolated `docker run --rm` jobs with own config/DB
- **Market data hub** — pooled gRPC connections to the shared marketdata service, ref-counted per user container
- **WebSocket relay** — ticket-based auth, broadcasts market data and container events to the frontend
- **Verification** — exchange API key validation via HMAC-signed test requests for 8 exchanges
- **Container recovery** — periodic reaper for dead containers (5-worker pool)

Auth model: every request must carry `X-Manager-Token` (env-injected shared secret). Public endpoints (`/api/health`, `/api/ws`, `/livez`, `/readyz`, `/metrics`) are exempted via a switch in `auth.go`.

See [`../manager`](../manager) and [`../CLAUDE.md`](../CLAUDE.md) "Manager" section.

### BBGO instances (per user / per strategy)

Each strategy instance runs in its own Docker container named `bbgo-{userID}-{instanceID}`. The container:

- Reads config YAML from `{DATA_DIR}/{userID}/{instanceID}/bbgo.yaml`
- Connects to the exchange (real or paper trade engine based on mode)
- Subscribes to market data via the shared gRPC service
- Writes orders / trades / positions / profits directly to Supabase
- Paper mode writes to `paper_*` tables via `SUPABASE_TABLE_PREFIX=paper_`
- Exposes a REST API on `:8080` (for manager queries) and gRPC on `:9090` (unused by manager today)
- Joins the `saas_bbgo-net` bridge network — only the manager can reach it

### Market data hub

Two long-running bbgo containers provide shared market data:

- `bbgo-marketdata:9090` (gRPC) — mainnet
- `bbgo-marketdata-testnet:9090` (gRPC) — testnet

They use the 3-layer kline cache (memory → SQLite at `KLINE_DB_PATH` → exchange API). All user containers connect via pooled gRPC connections managed by the manager's `MarketDataHub`.

### Supabase

PostgreSQL + Realtime, hosted in the cloud.

Key responsibilities:

- **Auth** — user accounts, sessions, JWTs consumed by the web frontend
- **Storage** — backtest equity curve JSON, CSVs
- **Realtime** — broadcasts inserts on `orders`, `trades`, `positions`, `profits`, and all `paper_*` mirrors to subscribed frontend clients (migration `00023_realtime_tables.sql`)
- **RLS** — every user-scoped table has a policy restricting rows to the owning `user_id`

The manager uses the **service-role key** to bypass RLS. The web frontend only ever has the **anon key** (RLS-protected).

## Data flows

### 1. User starts a bot (live or paper)

```
Browser
  → POST /api/manager/users/{id}/strategies  (Supabase JWT)
     Next.js route handler authenticates with Supabase, adds X-User-Id + X-Manager-Token
  → Manager: CreateStrategy handler
     1. Resolve strategy registry defaults (cached)
     2. Deep-merge user config over defaults
     3. Compute deterministic instance_id (pkg/instanceid)
     4. Generate bbgo.yaml on disk under {DATA_DIR}/{userID}/{instanceID}/
     5. Insert into strategy_instances table
     6. Return 201 with instance info
  → Browser: POST /api/manager/users/{id}/instances/{inst}/start?mode=live
  → Manager: StartInstance handler
     1. Resolve credentials, decrypt via ENCRYPTION_KEY
     2. Build env args (mode-specific: live gets real keys, paper gets PAPER_TRADE=true)
     3. docker run -d --name bbgo-{uid}-{iid} --network saas_bbgo-net ...
     4. Return 202 immediately
     5. Background goroutine: poll container health, mark as "running"
  → Container boots bbgo, opens WebSocket to exchange, starts strategy
  → BBGO writes orders/trades/positions/profits directly to Supabase
  → Supabase Realtime broadcasts to the frontend
```

### 2. Frontend reads bot data

Multiple paths depending on freshness needs:

| Path | When | Source |
|------|------|--------|
| Supabase Realtime subscription | Always-on live updates | Realtime broadcast |
| Supabase direct query (`lib/bbgo/supabase-queries.ts`) | Default reads, fallback when container down | `orders`, `trades`, etc. tables |
| Manager proxy to bbgo container (`lib/bbgo/manager.ts`) | Live state the container knows but Supabase doesn't (e.g. open orders) | bbgo's in-memory state via `:8080` REST |

The frontend picks the right path per data type. Trade markers and PnL curves use Supabase data; open-order depth and balances use the container.

### 3. Backtest

```
User submits config (strategy, exchange, symbol, date range)
  → Manager: POST /api/backtest/submit
     1. Acquire semaphore (max 2 concurrent)
     2. Create isolated config dir: /data/backtest-{userID}/{jobID}/
     3. Download klines from marketdata gRPC service
     4. Generate backtest bbgo.yaml with SQLite DB (NOT Supabase)
     5. docker run --rm --name bt-{jobID} bbgo-backtest --sync --config ...
     6. Poll until container exits
     7. Parse results from output dir
     8. Upload equity curve JSON + CSV to Supabase Storage
     9. Insert row in backtest_reports table
     10. Release semaphore
```

Stale jobs from previous manager restarts are auto-failed on startup.

## Multi-tenant isolation

### Network isolation

All backend containers live on the `saas_bbgo-net` Docker bridge network. Containers cannot reach the host network or each other except via the manager.

| Direction | Allowed? |
|-----------|----------|
| Web → Manager | Yes (via Caddy internal route) |
| Manager → BBGO user container | Yes |
| BBGO user container A → BBGO user container B | No (network policy) |
| BBGO user container → Host LAN | No |
| BBGO user container → Exchange | Yes (outbound internet) |
| BBGO user container → Supabase | Yes (outbound HTTPS) |
| BBGO user container → Other user's Supabase rows | No (RLS by user_id) |

### Data isolation

Every user-scoped row carries a `user_id` column. Supabase RLS policies enforce that the web client (using anon key) can only see their own rows.

For per-instance isolation within a user, trading records carry `strategy_instance_id`. Most read paths filter by it. The exceptions are documented as gaps (e.g. `nav_history_details`, `rewards`, `withdraws`, `deposits` — see [`production-readiness-gaps.md`](production-readiness-gaps.md)).

### Credential isolation

User API keys are stored AES-GCM encrypted on the manager's volume at `{DATA_DIR}/{userID}/credentials.json`. Decryption happens in-memory in the manager at container start. Keys are passed to containers via env vars (`BINANCE_API_KEY`, etc.), never written to disk inside the container.

Containers run as uid 10001 (bbgo) — they cannot read other users' config directories because the manager owns them and the container's filesystem is layered.

## Observability

### Logs

Manager emits structured JSON logs (`slog`) to stdout. Captured by Docker's `json-file` log driver with rotation (`max-size: 10m`, `max-file: 3`). In production, ship to Loki/Datadog/GCP Logging via a fluentd sidecar or host-level collector.

Switch to human-readable for debugging: `LOG_FORMAT=text`.

### Metrics

The manager exposes a JSON snapshot at `/metrics` (internal only — Caddy returns 404 externally):

```json
{
  "build_version": "v1.4.2",
  "http_requests_total": 84321,
  "http_errors_total": 12,
  "container_starts_total": 87,
  "container_stops_total": 85,
  "backtests_run_total": 23,
  "backtests_failed_total": 1,
  "ws_clients_current": 14,
  "uptime_seconds": 86412
}
```

Use a scraper like Prometheus with a JSON exporter, or poll from a monitoring script.

### Health

| Endpoint | Cache | Use |
|----------|-------|-----|
| `/livez` | none | Liveness probe — process is up |
| `/readyz` | none | Readiness probe — returns 503 during graceful shutdown |
| `/api/health` | 15s TTL | Snapshot: container count, user count, uptime, version |
| `/metrics` | none | JSON snapshot of counters (internal only) |

## Operational characteristics

### Startup order

```
1. marketdata     (waits for gRPC healthcheck)
2. manager        (depends on marketdata being ready for backtest data download)
3. web            (depends on manager being ready for API calls)
4. caddy          (depends on web + manager being ready to serve)
```

`deploy.sh up` enforces this order with a 60s readiness probe on `/readyz`.

### Failure modes

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Manager crash | Docker restarts; readyz flips 503 | Caddy returns 503 for ~3s, then traffic resumes |
| Marketdata crash | Docker restarts; gRPC healthcheck fails | User containers reconnect, gap fills from exchange API |
| User container crash | `container_recovery.go` periodic scan | Auto-restart within 30s |
| Supabase outage | Realtime subscriptions drop; writes 5xx | Containers buffer nothing — trades still go to exchange, but Supabase records lost until restored |
| Caddy ACME fail | Caddy logs error; old cert continues | Renew within 30 days; alert via log monitor |
| Disk full | Manager cannot write configs/credentials | Manual cleanup, restore from backup |
| Docker daemon crash | Everything goes down | Host-level monitoring; restart daemon |

### Capacity assumptions

Current design targets (single-host deployment):

- ~50 active users
- ~150 running containers (3 strategies per user avg)
- ~5 concurrent backtests (semaphore capped at 2)
- ~50 WebSocket clients

For larger scale, split the manager and bbgo instances across hosts with a shared Docker socket proxy or migrate to Kubernetes.

## Security boundaries

1. **Internet → Caddy** — TLS termination, header hardening, internal route blocking
2. **Caddy → Web** — Next.js standalone server, no direct exposure
3. **Web → Manager** — Supabase JWT auth at Next.js route handler, manager token to manager
4. **Manager → BBGO container** — Docker socket, name-based resolution on private network
5. **BBGO container → Exchange** — Outbound only, credentials from env
6. **Anything → Supabase** — RLS by user_id; anon key for web, service key for manager only

For the full hardening checklist, see [`../SECURITY.md`](../SECURITY.md).
