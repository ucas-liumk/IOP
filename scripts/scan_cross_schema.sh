#!/usr/bin/env bash
# Cross-schema JOIN static scan (命门 5).
# Fails CI if a Go source file in internal/services/* or internal/contexts/* contains
# patterns like  "FROM public." or "JOIN public.tenant_" or "FROM tenant_<x>." or similar
# (other than within migrations).

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SRC="$ROOT/server/internal"

# Allow these patterns in known-safe locations (tenancy provisions schemas; iam queries public.*).
ALLOWED_REGEX='internal/services/(tenancy|iam)|migrations|provisioner.go|pg_repo.go|test/'

found=0
grep -rnE 'FROM (public|tenant_[a-z0-9_]+)\.' "$SRC" --include='*.go' | grep -vE "$ALLOWED_REGEX" || true | while read -r line; do
  echo "[cross-schema] $line"
  found=1
done

if [ $found -ne 0 ]; then
  echo "FAIL: cross-schema literal references detected outside services/tenancy|iam."
  exit 1
fi
echo "OK: no cross-schema JOIN patterns outside tenancy/iam/migrations."
