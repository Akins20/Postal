#!/usr/bin/env bash
# Regenerate the mobile API types from the frozen OpenAPI contract.
set -euo pipefail
cd "$(dirname "$0")/../.."
(cd web && npx openapi-typescript ../docs/openapi.yaml -o ../mobile/src/api/schema.d.ts)
