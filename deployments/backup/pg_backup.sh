#!/usr/bin/env bash
# IOP daily PG backup.  Runs from cron (or k8s CronJob).  Writes to BACKUP_DIR with timestamp.
# Retention: rolling 7 days + Mondays kept for 1 year.

set -euo pipefail

PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-5433}"
PGUSER="${PGUSER:-iop}"
PGPASSWORD="${PGPASSWORD:-iop_dev}"
PGDATABASE="${PGDATABASE:-iop}"

BACKUP_DIR="${BACKUP_DIR:-/var/backups/iop}"
mkdir -p "$BACKUP_DIR/daily" "$BACKUP_DIR/weekly"

TS=$(date -u +"%Y-%m-%dT%H%M%SZ")
DOW=$(date -u +%u) # 1=Mon
TARGET="$BACKUP_DIR/daily/iop-$TS.dump"

export PGPASSWORD
echo "[backup] dumping $PGDATABASE → $TARGET"
pg_dump -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -F c -Z 6 -f "$TARGET" "$PGDATABASE"

echo "[backup] verifying readability"
pg_restore -l "$TARGET" > /dev/null

# Monday → also copy to weekly retention
if [ "$DOW" = "1" ]; then
  cp "$TARGET" "$BACKUP_DIR/weekly/iop-$TS.dump"
fi

# Rolling 7-day retention for daily
find "$BACKUP_DIR/daily" -type f -name 'iop-*.dump' -mtime +7 -delete
# Yearly retention for weekly
find "$BACKUP_DIR/weekly" -type f -name 'iop-*.dump' -mtime +365 -delete

# Optional offsite sync (S3/MinIO) — set IOP_BACKUP_S3=s3://bucket/path
if [ -n "${IOP_BACKUP_S3:-}" ]; then
  echo "[backup] syncing to $IOP_BACKUP_S3"
  aws s3 sync "$BACKUP_DIR" "$IOP_BACKUP_S3" --exclude '*' --include 'iop-*.dump'
fi

echo "[backup] done: $TARGET"
