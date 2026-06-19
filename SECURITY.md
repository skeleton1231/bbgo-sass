# Security Policy

## Reporting a vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

Email reports to **security@your-domain.com** (replace with your production domain). Include:

1. Affected component (`bbgo` core, `saas/manager`, `saas/web`, docker compose, Caddy config)
2. Version / commit hash
3. Reproduction steps (proof of concept, logs, request payloads)
4. Impact assessment (what an attacker gains)
5. Suggested fix (optional)

### Response SLAs

| Event | Target |
|-------|--------|
| Acknowledge receipt | 48 hours |
| Initial assessment (severity, scope) | 5 business days |
| Fix or mitigation for HIGH/CRITICAL | 30 days from disclosure |
| Public release of advisory | After fix is shipped, coordinated with reporter |

We follow coordinated disclosure. Credit will be given to reporters unless they prefer anonymity.

## Scope

| In scope | Out of scope |
|----------|--------------|
| bbgo SaaS manager API (`saas/manager/`) | bbgo upstream issues — report at https://github.com/c9s/bbgo |
| Web dashboard (`saas/web/`) | Supabase cloud infra — report to Supabase directly |
| Docker compose / Caddy hardening | Exchange-side bugs (Binance, OKX, etc.) |
| Credential encryption (`crypto.go`) | Self-inflicted credential leaks (don't paste keys in issues) |
| Multi-tenant isolation | DoS via high-frequency legitimate trading |
| Realtime subscription auth | Social engineering / phishing |

## Threat model summary

| Threat | Mitigation |
|--------|------------|
| Credential theft at rest | AES-256-GCM with per-deployment `ENCRYPTION_KEY` (env-injected, never logged) |
| Credential leak to container logs | Env injection only; containers log at INFO level by default |
| Cross-tenant data access | `user_id` + `strategy_instance_id` columns + Supabase RLS policies |
| Cross-tenant container access | Containers on shared `saas_bbgo-net`; only manager talks to them |
| Manager token leak | Rotate `MANAGER_TOKEN`; traffic between web ↔ manager must be on private network |
| XSS via strategy state | React default-escaping; strategy state rendered as data, not HTML |
| CSRF on state-changing routes | Same-site cookies + Origin header check on Supabase-auth routes |
| Brute-force on /api/health or /api/ws | Unauthenticated routes rate-limited at Caddy layer |
| Dangling containers after user deletion | Manager runs `container_recovery.go` periodic reaper |

For the full architecture picture, see [`docs/architecture.md`](docs/architecture.md).

## Production hardening checklist

Before exposing the stack to real users:

- [ ] `.env.prod` exists with strong values for `ENCRYPTION_KEY` and `MANAGER_TOKEN` (≥32 bytes)
- [ ] `.env.prod` is owned by `root` with mode `0600` and never committed
- [ ] `DOMAIN` resolves to the host and TLS is being issued (check Caddy logs for ACME success)
- [ ] Supabase project has RLS enabled on every table that carries `user_id`
- [ ] `SUPABASE_SERVICE_KEY` is the **service-role** key (not anon) — only the manager has it
- [ ] Web frontend only ever has `NEXT_PUBLIC_SUPABASE_ANON_KEY` (never the service key)
- [ ] Caddy returns `404` for `/metrics`, `/livez`, `/readyz` externally (verify via `smoke-test.sh`)
- [ ] Docker socket is mounted read-only on the manager container
- [ ] Host firewall limits inbound to 80/443 only
- [ ] `backup-data.sh` runs on cron with retention policy enforced
- [ ] All containers have resource limits (memory + CPU) set in `docker-compose.prod.yml`
- [ ] Log rotation is configured (`json-file`, `max-size` 10m, `max-file` 3)
- [ ] Manager runs with `LOG_FORMAT=json` for structured ingestion (Datadog/Loki/GCP Logging)
- [ ] `govulncheck` passes in CI (`.github/workflows/saas.yml`)
- [ ] `pnpm audit --prod` is clean in web
- [ ] CSP header does not include `'unsafe-eval'` (allowed: `'self' 'unsafe-inline'`)

## Secret rotation procedures

### Rotate `ENCRYPTION_KEY` (rotates all user API credentials)

```bash
# 1. Decrypt and export all credentials with the OLD key
OLD_KEY="$ENCRYPTION_KEY"
docker compose -f docker/docker-compose.prod.yml exec manager \
  /bin/sh -c 'cat /data/*/credentials.json' > /tmp/creds-plaintext.json
# (You need a small one-off tool to decrypt — speak with the maintainers if it's not already in the manager CLI.)

# 2. Generate a new key
NEW_KEY=$(openssl rand -base64 32)

# 3. Re-encrypt all credentials with NEW_KEY (script TBD — see #rotations epic)

# 4. Update .env.prod, restart manager
$EDITOR .env.prod       # ENCRYPTION_KEY=$NEW_KEY
./scripts/deploy.sh restart manager

# 5. Verify one user can still start a container and trade
```

### Rotate `MANAGER_TOKEN`

```bash
NEW_TOKEN=$(openssl rand -hex 32)
# Update both: saas/.env.prod (MANAGER_TOKEN) and saas/web/.env.production (MANAGER_TOKEN)
# Restart manager and web together
./scripts/deploy.sh restart manager web
```

### Rotate exchange API keys

Per-user operation done through the dashboard. The old key is overwritten on disk and in any running container after the next start. No manager-level action required.

## Disclosure policy

- We credit reporters in release notes unless they request anonymity.
- We do not pursue legal action against good-faith reporters.
- We reserve the right to publish details after the fix is shipped.
