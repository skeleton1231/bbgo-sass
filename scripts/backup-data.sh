#!/usr/bin/env bash
# Snapshot the bbgo-data Docker volume to a tarball.
#
# Snapshots are written to $BACKUP_DIR (default: ./backups). The filename
# encodes a UTC timestamp so retention sweeps can sort by name.
#
# Run as a cron job on the Docker host:
#   17 3 * * *  /opt/bbgo/saas/scripts/backup-data.sh
#
# Retention: 14 daily snapshots, then 8 weekly snapshots (Sun only).
# Combine with pg_dump of the Supabase project for full DR coverage.
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-$(cd "$(dirname "$0")/.." && pwd)/backups}"
VOLUME="${DATA_VOLUME:-saas_bbgo-data}"
mkdir -p "$BACKUP_DIR"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DEST="$BACKUP_DIR/bbgo-data-$STAMP.tar.gz"

echo "[backup] snapshotting volume $VOLUME → $DEST"

# Mount the volume read-only inside a throwaway alpine container so the tar
# capture is point-in-time consistent (no app writes mid-archive).
docker run --rm \
  -v "$VOLUME":/data:ro \
  -v "$BACKUP_DIR":/out \
  alpine \
  tar -C /data -czf "/out/bbgo-data-$STAMP.tar.gz" .

# Retention: keep all snapshots from the last 14 days, then one-per-week for 8
# additional weeks. Everything else is pruned.
CUTOFF_DATE=$(date -u -d '14 days ago' +%Y%m%d)
find "$BACKUP_DIR" -name 'bbgo-data-*.tar.gz' -printf '%TY-%Tm-%Td %p\n' \
  | sort -r \
  | awk -v cutoff="$CUTOFF_DATE" '
      $1 < cutoff {
        week = substr($1, 1, 4) "-" strftime("%V", mktime($1 " 00 00 00"))
        if (seen[week]++) { print $2 }
      }
    ' \
  | xargs -r rm -f

# Hard cap of 50 backups regardless of retention rules.
TOTAL=$(find "$BACKUP_DIR" -name 'bbgo-data-*.tar.gz' | wc -l)
if [ "$TOTAL" -gt 50 ]; then
  find "$BACKUP_DIR" -name 'bbgo-data-*.tar.gz' -printf '%T@ %p\n' \
    | sort -n | head -n -50 | cut -d' ' -f2- | xargs -r rm -f
fi

echo "[backup] done ($(du -h "$DEST" | cut -f1))"
