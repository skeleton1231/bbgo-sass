# Operations Runbook

On-call playbook for the BBGO SaaS stack. Covers incidents, DR, routine operations, and troubleshooting.

> If you are new to the system, read [`architecture.md`](architecture.md) first.

## Service overview

| Service | Container | Health | Logs |
|---------|-----------|--------|------|
| Caddy (TLS proxy) | `caddy` | `curl -sk https://localhost/` returns 200 | `docker logs caddy` |
| Web (Next.js) | `web` | `curl http://localhost:3142/` returns 200 | `docker logs saas-web-1` |
| Manager (Go API) | `manager` | `curl http://localhost:8090/readyz` returns 200 | `docker logs saas-manager-1` |
| Marketdata (gRPC) | `bbgo-marketdata`, `bbgo-marketdata-testnet` | `grpc_health_probe` or `curl http://localhost:8080/livez` | `docker logs saas-bbgo-marketdata-1` |
| Per-user BBGO | `bbgo-{uid}-{iid}` | Manager tracks via `container_recovery.go` | `docker logs bbgo-{uid}-{iid}` |

All commands assume you're in the `saas/` directory and `.env.prod` is populated.

## Routine operations

### Deploy an update

```bash
# 1. CI is green on main
# 2. Bump BUILD_VERSION in .env.prod if needed
# 3. Rolling restart
./scripts/deploy.sh update
# Order: marketdata → manager → web → caddy
```

Each step waits for readyz before proceeding. If a step fails, the script stops and prints logs.

### Rollback

```bash
# Revert to previous known-good image tag
$EDITOR .env.prod   # BUILD_VERSION=<previous>

./scripts/deploy.sh down
./scripts/deploy.sh up
```

If the bad change shipped a forward-only DB migration (column drop, type change), rollback won't fully restore state — write a forward fix instead.

### Backup

```bash
./scripts/backup-data.sh
# Creates /opt/backups/bbgo-data-YYYYMMDDTHHMMSSZ.tar.gz
# Prunes to 14 daily + 8 weekly, capped at 50 archives
```

Verify periodically:

```bash
ls -lh /opt/backups/ | head -5
```

### Restore

```bash
./scripts/restore-data.sh /opt/backups/bbgo-data-YYYYMMDDTHHMMSSZ.tar.gz
# Stops manager → restores volume → restarts manager
```

**Warning**: restore overwrites the current volume. Use only when the current state is unrecoverable. Snapshot first if unsure.

### Rotate MANAGER_TOKEN

```bash
NEW_TOKEN=$(openssl rand -hex 32)
$EDITOR .env.prod            # MANAGER_TOKEN=$NEW_TOKEN
$EDITOR web/.env.production  # MANAGER_TOKEN=$NEW_TOKEN
./scripts/deploy.sh restart manager web
```

### Rotate ENCRYPTION_KEY

This re-encrypts every user's API credentials. High-risk operation.

```bash
# 1. Snapshot current state
./scripts/backup-data.sh

# 2. Generate new key
NEW_KEY=$(openssl rand -base64 32)

# 3. (Requires the manager CLI tool — TBD) re-encrypt all credentials
#    For now, contact the maintainer to run the rotation script.

# 4. Update env, restart manager
$EDITOR .env.prod   # ENCRYPTION_KEY=$NEW_KEY
./scripts/deploy.sh restart manager

# 5. Verify a user can still start a bot and trade
```

## Incident response

### P0: Site is down

Symptoms: `https://<DOMAIN>/` returns 5xx or doesn't load.

1. **Triage**:
   ```bash
   ./scripts/smoke-test.sh
   docker compose -f docker/docker-compose.prod.yml ps
   ```
   Identify which service is unhealthy.

2. **Common fixes by service**:

   | Failing service | Try this |
   |-----------------|----------|
   | caddy | `docker logs caddy --tail 50` — usually ACME or upstream timeout |
   | web | `docker restart saas-web-1`, check `docker logs saas-web-1 --tail 50` |
   | manager | `docker restart saas-manager-1`, check `docker logs saas-manager-1 --tail 50` |
   | marketdata | `docker restart saas-bbgo-marketdata-1` |

3. **Last resort**: full restart
   ```bash
   ./scripts/deploy.sh down
   ./scripts/deploy.sh up
   ```

4. **Comms**: post in dashboard banner; notify users if down > 5 min.

### P1: Users can't start bots

Symptoms: `POST /api/users/{id}/instances/.../start` returns 5xx; container never starts.

1. Check manager logs for `docker run` errors:
   ```bash
   docker logs saas-manager-1 --tail 100 | grep -E "docker run|container start"
   ```
2. Try starting manually:
   ```bash
   docker run --rm bbgo:latest --help
   ```
   If this fails, the image is broken — rebuild:
   ```bash
   cd /opt/bbgo
   docker build --network host -f saas/docker/Dockerfile.bbgo-base -t bbgo-base:latest .
   docker tag bbgo-base:latest bbgo:latest
   ```
3. Check disk space:
   ```bash
   df -h /var/lib/docker
   ```
   If > 90% full, clear old containers/images:
   ```bash
   docker system prune -a --volumes
   # WARNING: this removes ALL unused images/volumes. Be careful.
   ```

### P1: Trades not appearing in dashboard

Symptoms: container is running, but trades table is empty.

1. Check if the container is writing to Supabase:
   ```bash
   docker logs bbgo-{uid}-{iid} --tail 50 | grep -i supabase
   ```
   If you see `401 Unauthorized` or `403 Forbidden`, the service-role key may have rotated.
2. Check Supabase project status — is the project paused? (Free tier pauses after inactivity.)
3. Verify the container has the right env vars:
   ```bash
   docker exec bbgo-{uid}-{iid} env | grep SUPABASE
   ```
4. If env is right but writes still fail, check Supabase RLS policies — did a migration accidentally lock down the table?
5. Force a sync: stop and restart the container via the dashboard.

### P2: Backtest jobs stuck

Symptoms: jobs in `running` state for > 30 min.

1. List running backtest containers:
   ```bash
   docker ps | grep ^bt-
   ```
2. If a container is OOM-killed or exited, the manager should auto-fail the job. If it didn't:
   ```bash
   docker kill bt-{jobID}
   # Manager's startup sweep will mark it failed on next restart, or:
   # Manually update Supabase: UPDATE backtest_reports SET status='failed' WHERE id='{jobID}'
   ```
3. Check semaphore: max 2 concurrent. If both are stuck, new submissions will queue.

### P2: Disk full

Symptoms: `df -h` shows `/var/lib/docker` at 100%; new containers fail to start.

1. Stop the manager briefly:
   ```bash
   ./scripts/deploy.sh stop manager
   ```
2. Clean up:
   ```bash
   docker system prune -af --volumes
   ```
   **Caution**: this removes unused volumes. Verify the `saas_bbgo-data` volume is in use (it should be — it's mounted to running containers).
3. Also prune old manager logs:
   ```bash
   find /var/lib/docker/containers -name '*-json.log*' -mtime +7 -delete
   ```
4. Resume manager:
   ```bash
   ./scripts/deploy.sh start manager
   ```

### P2: WebSocket clients disconnecting

Symptoms: dashboard not updating; `ws_clients_current` drops to 0 in metrics.

1. Check manager logs:
   ```bash
   docker logs saas-manager-1 --tail 100 | grep -i websocket
   ```
2. Check Caddy — WS upgrade headers must be passed through (already configured in `Caddyfile`).
3. Restart manager if connection pool is corrupted:
   ```bash
   ./scripts/deploy.sh restart manager
   ```
4. Users will auto-reconnect — no user action needed.

### P3: Single user's bot died

Symptoms: one user's strategy stopped; others fine.

1. Check container:
   ```bash
   docker ps -a | grep bbgo-{uid}-
   ```
2. If `Exited`, check logs:
   ```bash
   docker logs bbgo-{uid}-{iid} --tail 100
   ```
3. `container_recovery.go` should auto-restart within 30s. If it didn't:
   ```bash
   # Via manager API (admin token required)
   curl -X POST -H "X-Manager-Token: $MANAGER_TOKEN" \
     http://localhost:8090/api/users/{uid}/instances/{iid}/start
   ```
4. Common causes: bad config, exchange rate-limit, OOM. Fix root cause before restart.

## Disaster recovery

### Total host failure

Assumes the host is unrecoverable; you have a new host with the same Docker setup.

1. Provision new host per [`deployment-guide.md`](deployment-guide.md).
2. Pull latest code: `git clone https://github.com/skeleton1231/bbgo.git`.
3. Copy `.env.prod` from your secrets store.
4. Push schema to Supabase: `cd saas/web && pnpm sb push`.
5. Build bbgo base image.
6. Restore volume:
   ```bash
   # Copy the latest backup tarball to the new host
   ./scripts/restore-data.sh bbgo-data-YYYYMMDDTHHMMSSZ.tar.gz
   ```
7. Bring up: `./scripts/deploy.sh up`.
8. Smoke test: `./scripts/smoke-test.sh`.

**Data loss**: anything written between the last backup and the failure is lost. This includes new trades (they're in Supabase, which is separate) but not config changes (those are in the bbgo-data volume).

### Supabase outage

Supabase is hosted — outages are rare but possible. During outage:

- Frontend login breaks (auth depends on Supabase)
- New trades fail to persist (container writes will 5xx and retry a few times before giving up)
- Existing containers keep trading — exchange connectivity is unaffected

Recovery:

- Wait for Supabase to come back (status.supabase.com)
- Restart user containers so they pick up where they left off:
  ```bash
  for c in $(docker ps --format '{{.Names}}' | grep '^bbgo-'); do
    docker restart "$c"
  done
  ```

### Credential leak

If `ENCRYPTION_KEY` or a user's API key is suspected compromised:

1. **Rotate `MANAGER_TOKEN`** (per routine ops above).
2. **Rotate `ENCRYPTION_KEY`** (per routine ops above).
3. **Force password reset** for all users (Supabase admin).
4. **Notify users** to rotate their exchange API keys.
5. **Audit logs** for suspicious activity in the 30 days before the leak.

## Troubleshooting reference

### Logs

```bash
# All services, last 100 lines
docker compose -f docker/docker-compose.prod.yml logs --tail 100

# Single service, follow
docker compose -f docker/docker-compose.prod.yml logs -f manager

# Specific user's bot
docker logs bbgo-{uid}-{iid} -f --tail 100
```

### Database queries

Connect via Supabase Studio SQL editor or `psql`:

```bash
psql "$SUPABASE_DB_URL"
```

Useful queries:

```sql
-- Active users
SELECT COUNT(DISTINCT user_id) FROM strategy_instances WHERE status='running';

-- Bots by status
SELECT status, COUNT(*) FROM strategy_instances GROUP BY status;

-- Recent trades
SELECT created_at, user_id, symbol, side, quantity, price
FROM trades
ORDER BY created_at DESC
LIMIT 20;

-- Failed backtests
SELECT id, user_id, strategy, error, created_at
FROM backtest_reports
WHERE status='failed'
ORDER BY created_at DESC
LIMIT 20;

-- Largest tables (for capacity planning)
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC
LIMIT 20;
```

### Performance tuning

If `/api/health` is slow even with caching:

- Check container count — > 200 containers will strain the manager.
- Check Docker daemon CPU — `docker stats`.
- Bump manager memory limit in `docker-compose.prod.yml`.

If frontend feels slow:

- Check Supabase response times in the browser network tab.
- Verify Next.js standalone build (not dev mode).
- Add indexes for common query patterns (e.g. `CREATE INDEX ON trades (user_id, created_at DESC)`).

If WebSocket is laggy:

- Check `ws_clients_current` — if it's growing unbounded, the cleanup logic may have a bug.
- Restart manager to reset the connection pool.

## On-call expectations

- **Response time**: 15 min for P0, 1 hour for P1, 4 hours for P2/P3.
- **Comms**: post in the dashboard banner for user-visible issues; internal channel for everything else.
- **Postmortem**: write one for every P0/P1, within 48 hours. Template: timeline, root cause, action items, what went well, what didn't.

## Contacts

- **Maintainer**: see `git log` for active contributors
- **Supabase status**: https://status.supabase.com
- **Exchange status**: Binance https://status.binance.com, OKX https://status.okx.com, etc.
- **Let's Encrypt**: https://letsencrypt.status.io
