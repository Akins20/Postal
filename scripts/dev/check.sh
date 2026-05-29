#!/usr/bin/env bash
# Definition-of-Done enforcement for Postal (run by `make check`).
# Fails on the FIRST violation so problems are caught early. See
# docs/CODING_STANDARDS.md for the rules enforced here.
set -euo pipefail

cd "$(dirname "$0")/../.." # repo root

MAX_LINES=800
fail() { echo "✗ $*" >&2; exit 1; }
ok()   { echo "✓ $*"; }

echo "==> 1/6 gofmt"
unformatted="$(gofmt -l . | grep -v '^vendor/' || true)"
[ -z "$unformatted" ] || fail "gofmt: these files are not formatted:"$'\n'"$unformatted"
ok "gofmt clean"

echo "==> 2/6 goimports"
imports="$(go run golang.org/x/tools/cmd/goimports@latest -l -local github.com/Akins20/postal . | grep -v '^vendor/' || true)"
[ -z "$imports" ] || fail "goimports: these files need import fixes:"$'\n'"$imports"
ok "goimports clean"

echo "==> 3/6 go vet"
go vet ./... || fail "go vet reported issues"
ok "go vet clean"

echo "==> 4/6 golangci-lint"
go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run ./... || fail "golangci-lint reported issues"
ok "golangci-lint clean"

echo "==> 5/6 file length (<= ${MAX_LINES} lines per .go file)"
violations=""
while IFS= read -r -d '' f; do
	lines="$(wc -l < "$f")"
	if [ "$lines" -gt "$MAX_LINES" ]; then
		violations+="  $f: $lines lines"$'\n'
	fi
done < <(find . -type f -name '*.go' -not -path './vendor/*' -not -name '*.pb.go' -print0)
[ -z "$violations" ] || fail "files exceed ${MAX_LINES} lines (split by responsibility):"$'\n'"$violations"
ok "all .go files within ${MAX_LINES} lines"

echo "==> 6/6 go test -race"
go test -race ./... || fail "tests failed"
ok "tests pass"

echo
echo "All checks passed ✅"
