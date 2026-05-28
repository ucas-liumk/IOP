#!/usr/bin/env bash
# scripts/dev.sh — one-command local dev: starts infra, migrates, runs server + web.

set -euo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

cleanup() {
  echo
  echo "[dev.sh] shutting down ..."
  jobs -p | xargs -I {} kill {} 2>/dev/null || true
  # Leave docker compose running — re-runs are faster. Stop with: cd deployments && docker compose down
}
trap cleanup EXIT

echo "[1/4] Starting infra (db/redis/minio)..."
(cd "$ROOT/deployments" && docker compose up -d db redis minio)

echo "[2/4] Waiting for PG healthy..."
for i in $(seq 1 30); do
  if (cd "$ROOT/deployments" && docker compose exec -T db pg_isready -U iop -d iop > /dev/null 2>&1); then
    break
  fi
  sleep 1
done

echo "[3/4] Running migrations..."
(cd "$ROOT/server" && go run ./cmd/migrate up)

echo "[4/4] Starting server (port 8080) + web (port 5174) in background..."
echo "  Server:  http://localhost:8080"
echo "  Web:     http://localhost:5174"
echo "  Stop:    Ctrl-C"
(cd "$ROOT/server" && go run ./cmd/server) &
(cd "$ROOT/web" && npm run dev) &
wait
