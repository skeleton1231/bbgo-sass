#!/usr/bin/env bash
# Restore bbgo-data Docker volume from a backup tarball.
#
# Usage:
#   saas/scripts/restore-data.sh path/to/bbgo-data-20260617T031700Z.tar.gz
#
# Stops the manager first so we don't fight over file locks. Volume must
# already exist (docker volume create saas_bbgo-data if missing).
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <backup-tarball>" >&2
  exit 2
fi

BACKUP="$1"
VOLUME="${DATA_VOLUME:-saas_bbgo-data}"

if [ ! -f "$BACKUP" ]; then
  echo "ERROR: backup not found: $BACKUP" >&2
  exit 1
fi

# Pause the manager so we don't race with credential writes.
echo "[restore] stopping manager..."
docker compose -f saas/docker/docker-compose.prod.yml --env-file saas/.env.prod \
  stop manager || true

echo "[restore] restoring $BACKUP → volume $VOLUME"
docker run --rm \
  -v "$VOLUME":/data \
  -v "$(cd "$(dirname "$BACKUP")" && pwd)":/backup:ro \
  alpine \
  sh -c "rm -rf /data/* /data/.[!.]* 2>/dev/null; tar -C /data -xzf /backup/$(basename "$BACKUP")"

echo "[restore] restarting manager..."
docker compose -f saas/docker/docker-compose.prod.yml --env-file saas/.env.prod \
  start manager

echo "[restore] done"
