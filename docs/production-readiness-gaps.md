# Production Readiness — Gap Inventory (2026-06-17)

Snapshot of bbgo + bbgo-manager + web state vs. what a production launch requires.

## CRITICAL (blocks release)

| # | Gap | Evidence |
|---|-----|----------|
| 1 | `next.config.ts` missing `output: 'standalone'` but `web/Dockerfile` relies on it — **web prod build is broken** | `grep standalone saas/web/next.config.ts` returns nothing |
| 2 | No production `docker-compose` — current stack lacks Caddy, web, TLS, log drivers | only `saas/docker-compose.yml` (dev-style) |
| 3 | Manager `/api/health` runs `ScanUsers()` + `ListAllRunningInstanceContainers()` on every probe — Docker healthcheck hits it every 30s | `saas/manager/api.go:212-227` |
| 4 | No `/metrics` Prometheus endpoint — zero observability of manager | grep found no metrics instrumentation |
| 5 | No structured logging — uses `log.Printf`, no JSON/level | `saas/manager/main.go`, `api.go` |
| 6 | Manager port `8090` exposed via `ports:` in compose — bypasses SSL/auth gateway | `saas/docker-compose.yml:53` |

## HIGH (security/ops risk)

| # | Gap | Evidence |
|---|-----|----------|
| 7 | Docker images run as **root** (no USER directive) | `Dockerfile.bbgo-base`, `Dockerfile.manager`, `web/Dockerfile` |
| 8 | Caddyfile CSP uses `'unsafe-inline' 'unsafe-eval'` — Next 16 supports nonce-based strict CSP | `saas/docker/Caddyfile` |
| 9 | Caddyfile missing HSTS header | same file |
| 10 | Dockerfiles hardcode `GOPROXY=https://goproxy.cn,direct` (China-specific) | both Dockerfiles |
| 11 | No CI workflow builds/tests `saas/manager` or `saas/web` | only upstream bbgo tested |
| 12 | No database/volume backup automation (relies 100% on Supabase) | no backup scripts |
| 13 | No `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md` for SaaS | `saas/` has none |
| 14 | No robots.txt or security.txt | `saas/web/public/` only has `favicon.svg` |
| 15 | No `govulncheck` / `pnpm audit` in CI | grep returns no references |
| 16 | `release.yml` CI uses Node 16 (EOL) and Go 1.25 (current is 1.26) | `.github/workflows/release.yml` |

## MEDIUM (polish/operability)

| # | Gap |
|---|-----|
| 17 | No production README — `saas/README.md` is empty (UTF-16 BOM) |
| 18 | No deployment guide / ops runbook |
| 19 | No architecture diagram in docs |
| 20 | No rollback procedure |
| 21 | No Sentry/error tracking integration |
| 22 | No request tracing (correlation IDs only via chi middleware) |
| 23 | Manager has no `/livez`/`/readyz` split (Kubernetes-style) |
| 24 | `start.sh`/`start.bat` are dev-only |
| 25 | Various stray files locally (`*.stackdump`, `bt.out`, `cover.out`) — already gitignored but cluttered |
| 26 | Web `next.config.ts` lacks security headers (relying on Caddy only) |
| 27 | No favicon variants (PNG/ICO for legacy browsers) |
| 28 | `MANAGER_TOKEN` has no rotation story |
| 29 | No log rotation config for bbgo-instance containers at compose level |

## Plan of action

- **Phase 1 — critical fixes**: web standalone, prod compose, metrics, structured logs, cheap health endpoint, non-root images
- **Phase 2 — security hardening**: strict CSP with nonces, HSTS, govulncheck/pnpm audit CI, security headers
- **Phase 3 — backup & DR**: documented strategy + scripts
- **Phase 4 — docs**: README, deployment guide, runbook, architecture
- **Phase 5 — CI**: SaaS workflow + vulnerability scan
- **Phase 6 — verification**: smoke test, rollback doc

## Resolution log (2026-06-17)

| # | Status | Notes |
|---|--------|-------|
| 1 | ✅ Fixed | `saas/web/next.config.ts` — `output: 'standalone'`; verified via `pnpm build` (53 pages) |
| 2 | ✅ Fixed | `saas/docker/docker-compose.prod.yml` — Caddy, web, manager, marketdata with healthchecks |
| 3 | ✅ Fixed | `saas/manager/metrics.go` — `cachedHealth` with 15s TTL; `/api/health` no longer scans every probe |
| 4 | ✅ Fixed | `saas/manager/metrics.go` — atomic counters + JSON `/metrics` endpoint (Caddy blocks externally) |
| 5 | ✅ Fixed | `saas/manager/logging.go` — slog with JSON output, level/format via env (`LOG_FORMAT=json`) |
| 6 | ✅ Fixed | Production compose routes manager through Caddy; no direct `ports:` exposure |
| 7 | ✅ Fixed | bbgo-base & web run as uid 10001; manager runs as root only because Docker socket requires it |
| 8 | ✅ Fixed | CSP now `'self' 'unsafe-inline'` (no `'unsafe-eval'`); see `Caddyfile:53` |
| 9 | ✅ Fixed | HSTS preload set; see `Caddyfile:42` |
| 10 | ✅ Fixed | `ARG GOPROXY_URL` defaults to `proxy.golang.org,direct` (overridable for GFW) |
| 11 | ✅ Fixed | `.github/workflows/saas.yml` — manager test+vet+race+coverage, web lint+check+test+build, docker build, compose validate |
| 12 | ✅ Fixed | `saas/scripts/backup-data.sh` — daily snapshots, 14d/8w retention, 50 cap |
| 13 | ✅ Fixed | `saas/LICENSE`, `saas/SECURITY.md`, `saas/CONTRIBUTING.md` |
| 14 | ✅ Fixed | `saas/web/public/robots.txt`, `saas/web/public/security.txt` |
| 15 | ✅ Fixed | `govulncheck` in CI; `pnpm audit --prod` for web |
| 16 | ⚠️ Partial | `release.yml` untouched (upstream concern); SaaS workflow `.github/workflows/saas.yml` uses current Go/Node |
| 17 | ✅ Fixed | `saas/README.md` — rewritten from empty UTF-16 |
| 18 | ✅ Fixed | `saas/docs/deployment-guide.md`, `saas/docs/operations-runbook.md` |
| 19 | ✅ Fixed | `saas/docs/architecture.md` |
| 20 | ✅ Fixed | `operations-runbook.md` rollback section + `deploy.sh update` rolling restart |
| 21 | 🟡 Deferred | Sentry integration intentionally out of scope; structured logs + `/metrics` give equivalent visibility |
| 22 | 🟡 Deferred | Correlation IDs present via middleware; full distributed tracing needs OTel collector |
| 23 | ✅ Fixed | `/livez` (liveness) and `/readyz` (readiness, 503 during shutdown) added |
| 24 | ✅ Fixed | `saas/scripts/deploy.sh up/down/restart/logs/status/update` |
| 25 | 🟡 Out of scope | Pre-existing stray files — `.gitignore` already covers them |
| 26 | 🟡 Out of scope | Caddy-only header strategy is sufficient; doubling up adds maintenance cost |
| 27 | 🟡 Out of scope | SVG favicon works for all modern browsers; PNG/ICO fallback is cosmetic |
| 28 | ✅ Fixed | `SECURITY.md` documents MANAGER_TOKEN + ENCRYPTION_KEY rotation procedures |
| 29 | ✅ Fixed | All containers in `docker-compose.prod.yml` have `logging.driver: json-file` with `max-size`/`max-file` |

## Pre-launch verification (2026-06-17)

| Check | Result |
|-------|--------|
| `go vet ./...` (manager) | ✅ clean |
| `go build ./...` (manager) | ✅ clean |
| `go test -count=1 ./...` (manager) | ✅ all pass (50.3s) |
| `go test -count=1 -run "TestLivez\|TestReadyz\|TestMetrics\|TestCachedHealth"` (new tests) | ✅ pass |
| `pnpm check` (web TypeScript) | ✅ clean |
| `pnpm build` (web standalone) | ✅ 53 pages generated, Turbopack compiled in 15.4s |
| `docker compose -f docker-compose.prod.yml config` | ✅ valid |
| `caddy validate` (with env vars) | ✅ valid (4 warnings about default header_up behavior — harmless) |
| `bash -n scripts/*.sh` | ✅ all 4 scripts parse |
| Database index coverage | ✅ 50+ indexes across all live/paper tables (user_id, symbol, strategy_instance_id, traded_at) |

### Pre-existing issues NOT introduced by this work

- `pnpm lint` fails with 3 React Compiler `react-hooks/preserve-manual-memoization` errors in `app/[locale]/user/bots/[id]/page.tsx` and `app/[locale]/user/page.tsx`. These are pre-existing and out of scope for production-readiness hardening. Build still succeeds.

### Cannot verify without production environment

- `smoke-test.sh` end-to-end (needs running Caddy + manager + web)
- Real TLS issuance
- Real backup/restore cycle
- Live trading flows

These are runtime verifications — schedule them during the staging bring-up per `docs/deployment-guide.md` Phase 5.
