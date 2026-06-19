# Deployment Guide

End-to-end production bring-up for the BBGO SaaS stack. Assumes a single Linux host. For architecture, see [`architecture.md`](architecture.md). For troubleshooting and DR, see [`operations-runbook.md`](operations-runbook.md).

## Prerequisites

### Host requirements

- **OS**: Ubuntu 22.04+ or Debian 12+ (any modern Linux works if Docker supports it)
- **CPU**: 4 cores minimum, 8 recommended
- **RAM**: 8 GB minimum, 16 GB recommended (marketdata caching + per-user bbgo containers add up)
- **Disk**: 100 GB SSD minimum (more if backtesting many symbols/date ranges)
- **Network**: Public IP with ports 80 + 443 open inbound; outbound to exchange APIs and Supabase

### Software

```bash
# Verify versions
docker --version            # 28+
docker compose version      # v2+
openssl version             # any recent
curl --version              # any recent
```

### External services

1. **Supabase project** — create a project. Note:
   - Project URL (`https://<project-ref>.supabase.co`)
   - Service-role key (sensitive — only the manager gets this)
   - Anon key (public — the web frontend uses this)
   - Database URL (PostgreSQL connection string)
2. **Domain** — DNS A record pointing to your host's public IP
3. **Email for ACME** — used by Caddy for Let's Encrypt notifications

## Phase 1: Supabase setup

```bash
# Clone and enter the repo
git clone https://github.com/skeleton1231/bbgo.git
cd bbgo/saas

# Install Supabase CLI (one-time)
cd web && pnpm install && cd ..

# Push the schema (23+ migrations: tables, RLS policies, realtime publication, strategy registry)
cd web
pnpm sb push
pnpm sb types        # regenerate src/lib/supabase/types.ts
pnpm sb go-types     # regenerate manager/supabase_types.go + ../../pkg/supabasetypes/database_types.go
cd ..
```

### Seed the strategy registry

The `strategy_registry` table is the source of truth for strategy defaults. After the initial migration, verify 50 strategies are present:

```sql
-- via Supabase Studio SQL editor
SELECT COUNT(*) FROM strategy_registry;  -- expect ~50
SELECT id, live_only, requires_futures FROM strategy_registry ORDER BY id;
```

If empty, restore from the latest migration that ships seed data.

## Phase 2: Local environment

```bash
cp .env.production.example .env.prod
$EDITOR .env.prod
```

Required values:

| Variable | How to set |
|----------|------------|
| `DOMAIN` | Your domain (e.g. `bbgo.example.com`) |
| `ACME_EMAIL` | Ops email for Let's Encrypt |
| `SUPABASE_URL` | `https://<project-ref>.supabase.co` |
| `SUPABASE_SERVICE_KEY` | Service-role key from Supabase dashboard |
| `SUPABASE_DB_URL` | PostgreSQL URL from Supabase dashboard |
| `NEXT_PUBLIC_SUPABASE_URL` | Same as `SUPABASE_URL` |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Anon key from Supabase dashboard |
| `SUPABASE_SERVICE_ROLE_KEY` | Same as `SUPABASE_SERVICE_KEY` |
| `ENCRYPTION_KEY` | `openssl rand -base64 32` |
| `MANAGER_TOKEN` | `openssl rand -hex 32` |
| `BINANCE_API_KEY` / `BINANCE_API_SECRET` | For the shared marketdata container; can be empty if only using public market data |
| `BUILD_VERSION` | Git tag or `prod-YYYYMMDD` |
| `LOG_LEVEL` | `info` (use `debug` only when investigating) |
| `LOG_FORMAT` | `json` (production); `text` for human reading |

Lock down the file:

```bash
chmod 600 .env.prod
chown root:root .env.prod
```

**Never commit `.env.prod`.** `.gitignore` excludes it.

## Phase 3: Build the bbgo base image

The bbgo base image is built from the repo root (not `saas/`):

```bash
cd ../../  # repo root
docker build --network host -f saas/docker/Dockerfile.bbgo-base -t bbgo-base:latest .
docker tag bbgo-base:latest bbgo:latest           # manager uses this tag
docker tag bbgo-base:latest bbgo-backtest:latest  # backtest containers
```

CI also builds this image — see `.github/workflows/saas.yml`.

## Phase 4: First bring-up

```bash
cd saas
./scripts/deploy.sh up
```

What happens:

1. `docker compose -f docker/docker-compose.prod.yml --env-file .env.prod build` — builds manager, web
2. `docker compose ... up -d` — starts all services
3. Polls `http://localhost/readyz` for up to 60s — must return 200
4. Prints status summary

If `up` fails, check logs:

```bash
./scripts/deploy.sh logs manager
./scripts/deploy.sh logs caddy
```

## Phase 5: Verify

### Smoke tests

```bash
./scripts/smoke-test.sh
# All probes should be green:
#   ok  livez → 200
#   ok  readyz → 200
#   ok  api-health → 200
#   ok  homepage → 200
#   ok  robots.txt → 200
#   ok  static-404 → 404
#   ok  metrics-blocked → 404   ← confirms Caddy blocks internal routes
```

### Manual checks

1. Visit `https://<DOMAIN>/` — should see the landing page
2. Sign up / sign in — Supabase auth should redirect you to the dashboard
3. Create a paper-trading strategy — should start within ~10s
4. Check container: `docker ps | grep bbgo-`
5. Force a paper trade — see it appear in the trades table
6. Test WebSocket — open browser devtools, confirm WS connection to `/api/ws` stays open

### Security verification

```bash
# Confirm /metrics is blocked externally but reachable internally
curl -I https://<DOMAIN>/metrics      # expect 404
docker compose -f docker/docker-compose.prod.yml exec manager wget -qO- http://localhost:8090/metrics  # 200

# Confirm HSTS header
curl -sI https://<DOMAIN>/ | grep -i strict-transport-security
# expect: max-age=31536000; includeSubDomains; preload

# Confirm CSP
curl -sI https://<DOMAIN>/ | grep -i content-security-policy
# should NOT contain 'unsafe-eval'
```

## Phase 6: Set up backups

Schedule daily volume snapshots via cron or systemd timer:

```bash
# crontab -e
0 3 * * * /opt/bbgo/saas/scripts/backup-data.sh >> /var/log/bbgo-backup.log 2>&1
```

Retention: 14 days daily + 8 weeks weekly, capped at 50 archives total. Old archives auto-deleted.

Verify a backup works by doing a test restore on a staging host:

```bash
./scripts/restore-data.sh /opt/backups/bbgo-data-YYYYMMDDTHHMMSSZ.tar.gz
```

## Phase 7: Monitoring

Pick one of:

### Option A: Uptime monitoring

Use UptimeRobot / Pingdom / Better Uptime to ping `https://<DOMAIN>/readyz` every 60s. Alert if non-200.

### Option B: Log aggregation

Run a sidecar collector (fluentd, Promtail, Vector) that tails Docker logs and ships to Loki / Datadog / GCP Logging. Manager logs are JSON-structured for easy ingestion.

### Option C: Metric scraping

Poll `http://<internal-ip>:8090/metrics` from inside the network (or via `docker exec` script) and feed to Prometheus with a JSON exporter.

Alert thresholds (suggested):

- `http_errors_total` increases by > 10 in 5 min
- `container_starts_total - container_stops_total` (running count) drops unexpectedly
- `ws_clients_current` is 0 for > 5 min during business hours
- `/readyz` returns non-200 for > 30s

## Phase 8: Hardening checklist

Before exposing to real users, walk through [`../SECURITY.md`](../SECURITY.md) "Production hardening checklist". Specifically:

- [ ] `.env.prod` is mode 0600, owned by root, never committed
- [ ] `ENCRYPTION_KEY` and `MANAGER_TOKEN` are ≥32 bytes
- [ ] Host firewall: inbound only 80/443, outbound only to exchanges + Supabase
- [ ] Docker socket mounted read-only on manager
- [ ] Caddy returns 404 for internal routes externally (smoke test confirms)
- [ ] Backups running and tested
- [ ] CI green on `main` (`.github/workflows/saas.yml`)
- [ ] `govulncheck` clean
- [ ] `pnpm audit --prod` clean
- [ ] Resource limits set on all containers
- [ ] Log rotation configured
- [ ] At least one team member trained on the runbook (`operations-runbook.md`)

## Phase 9: User onboarding

The first user is usually an admin. After signup:

1. Verify their row appears in `user_profiles` table
2. Have them add an exchange API key via `/settings/api-keys`
3. Manager verifies the key (HMAC test request to the exchange)
4. Have them create a paper strategy first to validate end-to-end
5. If paper works, allow live trading

## Ongoing operations

- **Updates**: `./scripts/deploy.sh update` does a rolling restart
- **Backups**: `./scripts/backup-data.sh` (cron-scheduled)
- **Restores**: see [`operations-runbook.md`](operations-runbook.md)
- **Incident response**: see [`operations-runbook.md`](operations-runbook.md)

## Common pitfalls

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| Caddy can't get cert | DNS not pointing to host | Wait for propagation; verify with `dig <DOMAIN>` |
| Manager logs "decryption failed" | Wrong `ENCRYPTION_KEY` | Restore `.env.prod` from backup or re-encrypt all credentials |
| Containers start but no trades | API key invalid or IP-restricted | Re-verify key; check exchange IP whitelist |
| Frontend shows old data | Supabase Realtime subscription dropped | Refresh page; check browser console for WS errors |
| Backtest hangs forever | Stale job from crash | `docker ps | grep bt-`, kill it; manager auto-fails job on next scan |
| `/api/health` slow | First call after restart (cache cold) | Should be fast after first call (15s TTL) |

For more, see [`operations-runbook.md`](operations-runbook.md).
