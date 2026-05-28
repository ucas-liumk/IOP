#!/usr/bin/env bash
# scripts/openapi-gen.sh
# Generates frontend TypeScript SDK from server/api/openapi.yaml.
# M1: stub. M3+ wires openapi-typescript-codegen.

set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT="$SCRIPT_DIR/.."

echo "[openapi-gen] Source: $ROOT/server/api/openapi.yaml"
echo "[openapi-gen] M1 stub — TODO M3: integrate openapi-typescript-codegen"
