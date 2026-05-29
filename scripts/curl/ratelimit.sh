#!/usr/bin/env bash
# Proves the rate-limit middleware returns 429 past the threshold, and that
# /metrics is exposed. Requires the server running (make run). The ping bucket
# is capacity 5, refill 1/sec, so a quick burst of >5 requests trips the limit.
set -uo pipefail

BASE_URL="${POSTAL_BASE_URL:-http://localhost:8080}"
BURST="${BURST:-8}"
pass=0
fail=0

note() { printf '%s\n' "$*"; }

note "Target: ${BASE_URL}"
note "Bursting ${BURST} requests at /api/v1/ping (bucket: capacity 5, 1/sec)"
note

saw_200=0
saw_429=0
for i in $(seq 1 "$BURST"); do
	code="$(curl -sS -o /dev/null -w '%{http_code}' "${BASE_URL}/api/v1/ping" 2>/dev/null || echo 000)"
	printf '  req %-2s -> %s\n' "$i" "$code"
	[ "$code" = "200" ] && saw_200=1
	[ "$code" = "429" ] && saw_429=1
done

note
if [ "$saw_200" = "1" ]; then echo "PASS  some requests succeeded (200)"; pass=$((pass+1)); else echo "FAIL  no 200s seen"; fail=$((fail+1)); fi
if [ "$saw_429" = "1" ]; then echo "PASS  threshold tripped (429)"; pass=$((pass+1)); else echo "FAIL  never hit 429"; fail=$((fail+1)); fi

note
note "429 response body (standard error envelope):"
curl -sS "${BASE_URL}/api/v1/ping" >/dev/null 2>&1 # consume one more to ensure empty
for i in $(seq 1 "$BURST"); do curl -sS -o /dev/null "${BASE_URL}/api/v1/ping"; done
curl -sS -D - "${BASE_URL}/api/v1/ping" 2>/dev/null | grep -iE '^(HTTP/|retry-after|x-ratelimit)' || true
curl -sS "${BASE_URL}/api/v1/ping" 2>/dev/null
note

# /metrics smoke
mcode="$(curl -sS -o /dev/null -w '%{http_code}' "${BASE_URL}/metrics" 2>/dev/null || echo 000)"
if [ "$mcode" = "200" ]; then echo "PASS  /metrics exposed (200)"; pass=$((pass+1)); else echo "FAIL  /metrics -> ${mcode}"; fail=$((fail+1)); fi

note
note "Summary: ${pass} passed, ${fail} failed"
[ "$fail" -eq 0 ]
