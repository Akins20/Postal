#!/usr/bin/env bash
# End-to-end auth flow test against a running server (make run) with deps up.
# Covers signup -> verify -> login -> /me -> refresh -> logout, plus negative
# paths (duplicate email, wrong password, weak password, disposable email,
# unauthorized /me). Reads the verification token from the server log because the
# console mailer logs it.
#
# Usage: POSTAL_SERVE_LOG=/tmp/postal_serve.log ./scripts/curl/auth.sh
set -uo pipefail

BASE="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1/auth"
LOG="${POSTAL_SERVE_LOG:-/tmp/postal_serve.log}"
JAR="$(mktemp)"
pass=0; fail=0
EMAIL="user-$(date +%s)-$RANDOM@example.com"
PASS="sup3r-secret-pw"

ok()   { echo "PASS  $*"; pass=$((pass+1)); }
bad()  { echo "FAIL  $*"; fail=$((fail+1)); }
code() { curl -sS -o /dev/null -w '%{http_code}' "$@"; }

j() { curl -sS -H 'Content-Type: application/json' "$@"; }

echo "Base: $BASE"
echo "Email: $EMAIL"
echo

# 1. signup
c=$(code -X POST "$BASE/signup" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")
[ "$c" = "201" ] && ok "signup -> 201" || bad "signup -> $c (want 201)"

# 2. duplicate signup -> 409
c=$(code -X POST "$BASE/signup" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")
[ "$c" = "409" ] && ok "duplicate signup -> 409" || bad "duplicate signup -> $c (want 409)"

# 3. weak password -> 400
c=$(code -X POST "$BASE/signup" -H 'Content-Type: application/json' -d '{"email":"weak@example.com","password":"short"}')
[ "$c" = "400" ] && ok "weak password -> 400" || bad "weak password -> $c (want 400)"

# 4. disposable email -> 400
c=$(code -X POST "$BASE/signup" -H 'Content-Type: application/json' -d "{\"email\":\"x@mailinator.com\",\"password\":\"$PASS\"}")
[ "$c" = "400" ] && ok "disposable email -> 400" || bad "disposable email -> $c (want 400)"

# 5. wrong password -> 401
c=$(code -X POST "$BASE/login" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"wrong-password\"}")
[ "$c" = "401" ] && ok "wrong password -> 401" || bad "wrong password -> $c (want 401)"

# 6. verify email (pull token from server log)
TOKEN="$(grep -a 'verification_token' "$LOG" 2>/dev/null | tail -1 | sed -n 's/.*verification_token=\([A-Za-z0-9_-]*\).*/\1/p')"
if [ -n "$TOKEN" ]; then
	c=$(code -X POST "$BASE/verify-email" -H 'Content-Type: application/json' -d "{\"token\":\"$TOKEN\"}")
	[ "$c" = "200" ] && ok "verify email -> 200" || bad "verify email -> $c (want 200)"
else
	bad "could not read verification token from $LOG"
fi

# 7. login (capture cookies + body)
BODY="$(j -c "$JAR" -X POST "$BASE/login" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")"
ACCESS="$(printf '%s' "$BODY" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')"
CSRF="$(printf '%s' "$BODY" | sed -n 's/.*"csrf_token":"\([^"]*\)".*/\1/p')"
[ -n "$ACCESS" ] && ok "login -> access token issued" || bad "login did not return access token"

# 8. /me with bearer
c=$(code "$BASE/me" -H "Authorization: Bearer $ACCESS")
[ "$c" = "200" ] && ok "/me with bearer -> 200" || bad "/me with bearer -> $c (want 200)"

# 9. /me without token -> 401
c=$(code "$BASE/me")
[ "$c" = "401" ] && ok "/me without token -> 401" || bad "/me unauth -> $c (want 401)"

# 11. refresh without CSRF header -> 403 (cookie present, header missing)
c=$(code -b "$JAR" -X POST "$BASE/refresh")
[ "$c" = "403" ] && ok "refresh without CSRF -> 403" || bad "refresh w/o csrf -> $c (want 403)"

# 10. refresh via cookie + CSRF header (rotates cookies AND csrf token)
RBODY="$(j -b "$JAR" -c "$JAR" -X POST "$BASE/refresh" -H "X-CSRF-Token: $CSRF")"
NEWCSRF="$(printf '%s' "$RBODY" | sed -n 's/.*"csrf_token":"\([^"]*\)".*/\1/p')"
[ -n "$NEWCSRF" ] && ok "refresh (cookie+csrf) -> rotated tokens" || bad "refresh did not return new csrf token"

# 12. logout — must use the rotated csrf token (double-submit enforced)
c=$(code -b "$JAR" -c "$JAR" -X POST "$BASE/logout" -H "X-CSRF-Token: $NEWCSRF")
[ "$c" = "200" ] && ok "logout (rotated csrf) -> 200" || bad "logout -> $c (want 200)"

# 13. refresh after logout -> 401 (session revoked)
c=$(code -b "$JAR" -X POST "$BASE/refresh" -H "X-CSRF-Token: $NEWCSRF")
[ "$c" = "401" ] && ok "refresh after logout -> 401" || bad "refresh after logout -> $c (want 401)"

rm -f "$JAR"
echo
echo "Summary: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
