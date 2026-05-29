#!/usr/bin/env bash
# Smoke test for Postal's health endpoints. Requires the server running
# (make run) with dependencies up (make up). Prints PASS/FAIL per check and
# exits non-zero if any check fails.
set -uo pipefail

BASE_URL="${POSTAL_BASE_URL:-http://localhost:8080}"
pass=0
fail=0

check_status() {
	local name="$1" path="$2" want="$3"
	local got
	got="$(curl -sS -o /dev/null -w '%{http_code}' "${BASE_URL}${path}" 2>/dev/null || echo "000")"
	if [ "$got" = "$want" ]; then
		echo "PASS  ${name}: ${path} -> ${got}"
		pass=$((pass + 1))
	else
		echo "FAIL  ${name}: ${path} -> ${got} (want ${want})"
		fail=$((fail + 1))
	fi
}

echo "Target: ${BASE_URL}"
echo

# Liveness: always 200 when the process is up.
check_status "healthz" "/healthz" "200"

# Readiness: 200 only when Postgres AND Redis are reachable.
check_status "readyz" "/readyz" "200"

# Show the readiness body for visibility.
echo
echo "readyz body:"
curl -sS "${BASE_URL}/readyz" || true
echo

echo
echo "Summary: ${pass} passed, ${fail} failed"
[ "$fail" -eq 0 ]
