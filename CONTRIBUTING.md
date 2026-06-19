# Contributing to BBGO SaaS

Thanks for contributing. This guide covers dev setup, review norms, and release procedures. For system internals, see [`CLAUDE.md`](CLAUDE.md).

## Dev environment

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.24+ | For `saas/manager` and the `bbgo` core |
| Node | 20 LTS+ | For `saas/web` |
| pnpm | 10+ | Web package manager (do not use npm/yarn) |
| Docker | 28+ | For backend service bring-up |
| docker-compose | v2+ | For `saas/docker/docker-compose*.yml` |
| Supabase CLI | latest | For migrations (`pnpm sb`) |
| make | any | For codegen targets in repo root |

Optional: `protoc` (only if touching `pkg/pb/*.proto`), `rockhopper` (only if touching migrations).

## Repo layout

```
saas/
├── manager/              # Go — orchestration API server
├── web/                  # Next.js 16 dashboard
├── docker/               # Dockerfiles, compose, Caddyfile
├── scripts/              # deploy / backup / restore / smoke
├── docs/                 # architecture / deployment / ops
└── CLAUDE.md             # Deep-dive on the system
```

## Local bring-up

```bash
# Backend (marketdata + manager)
cp .env.example .env       # fill in SUPABASE_*, ENCRYPTION_KEY, MANAGER_TOKEN
docker compose -f docker/docker-compose.yml --env-file .env up --build

# Web (separate shell)
cd web
cp .env.example .env.local # fill in NEXT_PUBLIC_SUPABASE_*, MANAGER_API_URL, MANAGER_TOKEN
pnpm install
pnpm dev                   # http://localhost:3142
```

## Code review checklist

Before requesting review:

### Go (manager)
- [ ] `go vet ./...` clean
- [ ] `gofmt -s -w .` clean
- [ ] `go test -race ./...` passes
- [ ] New HTTP handlers are wrapped by `requestMetricsMiddleware` (auto via `RegisterRoutes`)
- [ ] No `fmt.Println` in committed code (use `slog`)
- [ ] No hardcoded secrets — all env-sourced via `config.go`
- [ ] New endpoints add cases to `auth.go` switch for unauthenticated routes (only if truly public)
- [ ] Touching credential code → `crypto.go` flow unchanged, secrets still AES-GCM encrypted

### TypeScript (web)
- [ ] `pnpm lint` clean
- [ ] `pnpm check` (tsc --noEmit) clean
- [ ] `pnpm test` passes
- [ ] `pnpm build` succeeds (Standalone build must work — Dockerfile depends on it)
- [ ] No `console.log` (use the structured logger where needed)
- [ ] No `dangerouslySetInnerHTML` without sanitization
- [ ] New API routes via `app/api/manager/[...path]/route.ts` (don't bypass Supabase auth)
- [ ] Touching strategy fields → regenerate types: `pnpm gen-strategy-types`

### Docker / compose
- [ ] Containers run as non-root where possible (uid 10001)
- [ ] Resource limits set (`mem_limit`, `cpus`)
- [ ] Log rotation configured (`json-file`, `max-size`)
- [ ] Healthchecks present
- [ ] Secrets via env or `secrets:`, never baked into image

### Schema migrations (`web/supabase/migrations/`)
- [ ] Numbered sequentially (`000NN_description.sql`)
- [ ] Idempotent or guarded (`IF NOT EXISTS`)
- [ ] RLS policies included for new tables with `user_id`
- [ ] Indexes added for any new column used in WHERE/JOIN
- [ ] After merge: `pnpm sb push`, then `pnpm sb types` and `pnpm sb go-types` to regenerate

## Commit conventions

Conventional commits. Examples:

```
feat(web): add trailing-stop config to grid strategy form
fix(manager): leak in WebSocket ticket cleanup on disconnect
chore(docker): bump bbgo-base to v1.4.2
docs: add operations-runbook DR section
test(manager): cover MarkCredentialsVerified race
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`, `ci`.

PR titles should follow the same convention — they get squash-merged.

## Strategy changes (special case)

Strategies have **three sources of truth** that can drift:

1. `pkg/strategy/<name>/strategy.go` `Defaults()` — compiled into the bbgo binary
2. `strategy_registry.defaults` (Supabase JSONB) — what the manager deep-merges
3. `strategy_registry.fields[*].default` — what the frontend form displays

When you change `Defaults()`:

1. Write a new migration that `UPDATE`s `strategy_registry.defaults` to match (see `00043_bollmaker_bandwidth_fix.sql` for the template).
2. Update the corresponding `fields[*].default` entries if user-visible.
3. Run `pnpm sb push` to apply.
4. Restart manager (or wait 5 min for `StrategyDefaultsCache.RefreshLoop`).
5. Existing instance containers won't auto-pick up the change — bbgo-layer defensive `Defaults()` is what catches them.

There is a registry sweep test (`pkg/cmd/strategy/registry_lifecycle_test.go`) that catches silent-accept bugs. Run it whenever you add or change a strategy.

## Release procedure

1. **Verify CI green on `main`** — `.github/workflows/saas.yml` must pass.
2. **Bump version** — tag the bbgo repo (`vMAJOR.MINOR.PATCH`), update `BUILD_VERSION` in `.env.prod`.
3. **Rebuild images**:
   ```bash
   ./scripts/deploy.sh update    # rolling restart: marketdata → manager → web → caddy
   ```
4. **Run smoke tests** — `./scripts/smoke-test.sh` should print all green.
5. **Verify health endpoints** — `/api/health` JSON should show `status: ok`.
6. **Notify users** — for breaking changes, post in the dashboard banner slot.

### Rollback

```bash
# Revert the deploy tag, then:
./scripts/deploy.sh down
# Fix .env.prod: BUILD_VERSION=<previous-good>
./scripts/deploy.sh up

# If a migration went wrong, restore data volume from backup:
./scripts/restore-data.sh /path/to/bbgo-data-YYYYMMDDTHHMMSSZ.tar.gz
```

Rollback assumes backward-compatible migrations. Forward-only migrations (column drops, type changes) need a forward fix, not a rollback.

## Getting help

- Architecture questions → [`docs/architecture.md`](docs/architecture.md) and [`CLAUDE.md`](CLAUDE.md)
- Operational issues → [`docs/operations-runbook.md`](docs/operations-runbook.md)
- Security issues → [`SECURITY.md`](SECURITY.md) (private channel, not GitHub issues)
