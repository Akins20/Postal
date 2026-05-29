#!/usr/bin/env bash
# Scheduling endpoint test against a running server. Verifies authz (publish vs
# read capability), validation, calendar, and isolation. The full schedule->fire
# ->publish path needs a connected channel + the worker, and is covered by the Go
# integration test (internal/worker/integration_test.go).
set -uo pipefail

API="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1"
AUTH="$API/auth"
PASS="sched-test-pw"
RUN="$(date +%s)-$RANDOM"
pass=0; fail=0
ok()  { echo "PASS  $*"; pass=$((pass+1)); }
bad() { echo "FAIL  $*"; fail=$((fail+1)); }
field() { sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p"; }
signup_login() {
	curl -sS -o /dev/null -H 'Content-Type: application/json' -X POST "$AUTH/signup" -d "{\"email\":\"$1\",\"password\":\"$PASS\"}"
	local b; b="$(curl -sS -H 'Content-Type: application/json' -X POST "$AUTH/login" -d "{\"email\":\"$1\",\"password\":\"$PASS\"}")"
	TOKEN="$(printf '%s' "$b" | field access_token)"
}
codeauth() { local m="$1" tok="$2" url="$3" data="${4:-}"
	if [ -n "$data" ]; then curl -sS -o /dev/null -w '%{http_code}' -X "$m" "$url" -H "Authorization: Bearer $tok" -H 'Content-Type: application/json' -d "$data"
	else curl -sS -o /dev/null -w '%{http_code}' -X "$m" "$url" -H "Authorization: Bearer $tok"; fi
}

signup_login "schedowner-$RUN@example.com"; AT="$TOKEN"
signup_login "schedviewer-$RUN@example.com"; VT="$TOKEN"
signup_login "schedstranger-$RUN@example.com"; ST="$TOKEN"
[ -n "$AT" ] && ok "actors provisioned" || bad "provisioning failed"
WS="$(curl -sS "$API/workspaces" -H "Authorization: Bearer $AT" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)"
[ -n "$WS" ] && ok "got workspace" || bad "no workspace"
S="$API/workspaces/$WS"

# 1. Calendar (empty) -> 200
c=$(codeauth GET "$AT" "$S/calendar"); [ "$c" = "200" ] && ok "calendar -> 200" || bad "calendar -> $c"

# 2. Schedule with no post_id -> 400
c=$(codeauth POST "$AT" "$S/schedule" '{}'); [ "$c" = "400" ] && ok "schedule no post_id -> 400" || bad "schedule empty -> $c (want 400)"

# 3. Schedule a non-existent (but non-nil) post -> 404
FUTURE="$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u +%Y-%m-%dT%H:%M:%SZ)"
c=$(codeauth POST "$AT" "$S/schedule" "{\"post_id\":\"11111111-1111-1111-1111-111111111111\",\"run_at\":\"$FUTURE\"}")
[ "$c" = "404" ] && ok "schedule missing post -> 404" || bad "schedule missing post -> $c (want 404)"

# 4. Create slot for a non-existent channel -> 404
c=$(codeauth POST "$AT" "$S/slots" '{"channel_id":"00000000-0000-0000-0000-000000000000","day_of_week":1,"time_of_day":"09:00","timezone":"UTC"}')
[ "$c" = "404" ] && ok "slot for missing channel -> 404" || bad "slot missing channel -> $c (want 404)"

# 5. Add viewer (read only): can read calendar, cannot schedule (needs publish)
codeauth POST "$AT" "$S/members" "{\"email\":\"schedviewer-$RUN@example.com\",\"role\":\"viewer\"}" >/dev/null
c=$(codeauth GET "$VT" "$S/calendar"); [ "$c" = "200" ] && ok "viewer reads calendar -> 200" || bad "viewer calendar -> $c"
c=$(codeauth POST "$VT" "$S/schedule" '{}'); [ "$c" = "403" ] && ok "viewer cannot schedule -> 403" || bad "viewer schedule -> $c (want 403)"

# 6. Non-member + unauth
c=$(codeauth GET "$ST" "$S/calendar"); [ "$c" = "403" ] && ok "non-member -> 403" || bad "non-member -> $c"
c=$(curl -sS -o /dev/null -w '%{http_code}' "$S/calendar"); [ "$c" = "401" ] && ok "unauthenticated -> 401" || bad "unauth -> $c"

echo; echo "Summary: $pass passed, $fail failed"; [ "$fail" -eq 0 ]
