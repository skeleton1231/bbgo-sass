#!/usr/bin/env bash
# Deploy BBGO SaaS to a Docker host.
#
# Usage:
#   saas/scripts/deploy.sh up        Build + start the prod stack
#   saas/scripts/deploy.sh down      Stop + remove containers (keeps volumes)
#   saas/scripts/deploy.sh restart   Restart all services
#   saas/scripts/deploy.sh logs      Tail logs
#   saas/scripts/deploy.sh status    Show service status
#   saas/scripts/deploy.sh update    Pull + rebuild + rolling restart
#
# Required env: copy saas/.env.production.example to saas/.env.prod and fill in.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_ROOT="$(cd "$ROOT_DIR/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker/docker-compose.prod.yml"
ENV_FILE="$ROOT_DIR/.env.prod"

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: $ENV_FILE not found. Copy .env.production.example to .env.prod and fill in real values." >&2
  exit 1
fi

DC=(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE")
cd "$REPO_ROOT"

cmd="${1:-up}"
shift || true
case "$cmd" in
  up)
    echo "[deploy] building images..."
    "${DC[@]}" build
    echo "[deploy] starting stack..."
    "${DC[@]}" up -d --remove-orphans
    echo "[deploy] waiting for manager readiness..."
    for _ in $(seq 1 30); do
      if "${DC[@]}" exec -T manager curl -fsS http://localhost:8090/readyz >/dev/null 2>&1; then
        echo "[deploy] manager ready"
        exit 0
      fi
      sleep 2
    done
    echo "[deploy] manager failed to become ready within 60s — check logs with: $0 logs" >&2
    exit 1
    ;;
  down)
    "${DC[@]}" down --remove-orphans
    ;;
  restart)
    "${DC[@]}" restart
    ;;
  logs)
    "${DC[@]}" logs -f --tail=200 "$@"
    ;;
  status)
    "${DC[@]}" ps
    ;;
  update)
    echo "[deploy] pulling base images..."
    "${DC[@]}" pull --ignore-pull-failures
    echo "[deploy] rebuilding..."
    "${DC[@]}" build
    echo "[deploy] rolling restart..."
    for svc in marketdata manager web caddy; do
      "${DC[@]}" up -d --no-deps "$svc"
      sleep 5
    done
    ;;
  *)
    echo "Usage: $0 {up|down|restart|logs|status|update}" >&2
    exit 2
    ;;
esac
