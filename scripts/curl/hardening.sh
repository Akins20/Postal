#!/usr/bin/env bash
# Phase 9 hardening checks against a running server: global security headers and
# the CORS allowlist behavior. Start the server with
#   POSTAL_CORS_ALLOWED_ORIGINS=https://app.postal.test
# so the CORS checks are meaningful (the script notes if CORS is disabled).
set -uo pipefail

BASE="${POSTAL_BASE_URL:-http://localhost:8080}"
ORIGIN="https://app.postal.test"
EVIL="https://evil.example"
pass=0; fail=0
ok()  { echo "PASS  $*"; pass=$((pass+1)); }
bad() { echo "FAIL  $*"; fail=$((fail+1)); }

# header NAME EXPECTED-SUBSTRING  (from a GET /healthz)
H="$(curl -sS -D - -o /dev/null "$BASE/healthz")"
hdr() { printf '%s' "$H" | grep -i "^$1:" | tr -d '\r'; }

printf '%s' "$H" | grep -qi '^X-Content-Type-Options: *nosniff' && ok "nosniff present" || bad "missing X-Content-Type-Options"
printf '%s' "$H" | grep -qi '^X-Frame-Options: *DENY'           && ok "X-Frame-Options DENY" || bad "missing/!DENY X-Frame-Options: $(hdr X-Frame-Options)"
printf '%s' "$H" | grep -qi '^Referrer-Policy: *no-referrer'    && ok "Referrer-Policy no-referrer" || bad "missing Referrer-Policy"
printf '%s' "$H" | grep -qi "^Content-Security-Policy: *default-src 'none'" && ok "CSP default-src none" || bad "missing/weak CSP"

# HSTS must be ABSENT in dev (plain HTTP). It only appears in production.
if printf '%s' "$H" | grep -qi '^Strict-Transport-Security:'; then
	bad "HSTS present in dev (should be production-only)"
else
	ok "HSTS absent in dev (correct)"
fi

# CORS preflight from an allowed origin -> 204 + reflected origin (only if configured).
PRE="$(curl -sS -D - -o /dev/null -X OPTIONS "$BASE/api/v1/ping" -H "Origin: $ORIGIN" -H 'Access-Control-Request-Method: GET')"
if printf '%s' "$PRE" | grep -qi "^Access-Control-Allow-Origin: *$ORIGIN"; then
	ok "CORS preflight reflects allowed origin"
	printf '%s' "$PRE" | grep -qi '^Access-Control-Allow-Credentials: *true' && ok "CORS allows credentials" || bad "missing ACAC"
	# A disallowed origin must NOT be reflected.
	EV="$(curl -sS -D - -o /dev/null -X OPTIONS "$BASE/api/v1/ping" -H "Origin: $EVIL" -H 'Access-Control-Request-Method: GET')"
	printf '%s' "$EV" | grep -qi "^Access-Control-Allow-Origin:" && bad "disallowed origin reflected!" || ok "disallowed origin not reflected"
else
	echo "NOTE  CORS not configured (POSTAL_CORS_ALLOWED_ORIGINS unset); skipping CORS reflect checks."
fi

echo
echo "Summary: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
