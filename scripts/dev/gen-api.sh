#!/usr/bin/env bash
# Generate the web client's typed API schema from the frozen OpenAPI contract.
# The spec (docs/openapi.yaml) is the single source of truth; never hand-edit the
# generated file. See docs/FRONTEND_PLAN.md §2/§7.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SPEC="$ROOT/docs/openapi.yaml"
OUT="$ROOT/web/src/api/schema.d.ts"

mkdir -p "$(dirname "$OUT")"
cd "$ROOT/web"
npx --no-install openapi-typescript "$SPEC" -o "$OUT" \
  || npx --yes openapi-typescript "$SPEC" -o "$OUT"
echo "Generated $OUT from $SPEC"
