#!/usr/bin/env bash
# Channel endpoint test against a running server. The server's OAuth provider
# registry is empty until Phase 4 (X/Twitter), so this verifies HTTP wiring,
# capability gating, workspace isolation, and validation — not a live OAuth round
# trip (that is covered by the Go integration test with a fake provider).
set -uo pipefail

API="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1"
AUTH="$API/auth"
PASS="channels-test-pw"
RUN="$(date +%s)-$RANDOM"
pass=0; fail=0

ok()  { echo "PASS  $*"; pass=$((pass+1)); }
bad() { echo "FAIL  $*"; fail=$((fail+1)); }
field() { sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p"; }

signup_login() {
	curl -sS -o /dev/null -H 'Content-Type: application/json' -X POST "$AUTH/signup" -d "{\"email\":\"$1\",\"password\":\"$PASS\"}"
	local body; body="$(curl -sS -H 'Content-Type: application/json' -X POST "$AUTH/login" -d "{\"email\":\"$1\",\"password\":\"$PASS\"}")"
	TOKEN="$(printf '%s' "$body" | field access_token)"
}
codeauth() { # METHOD TOKEN URL [JSON]
	local m="$1" tok="$2" url="$3" data="${4:-}"
	if [ -n "$data" ]; then
		curl -sS -o /dev/null -w '%{http_code}' -X "$m" "$url" -H "Authorization: Bearer $tok" -H 'Content-Type: application/json' -d "$data"
	else
		curl -sS -o /dev/null -w '%{http_code}' -X "$m" "$url" -H "Authorization: Bearer $tok"
	fi
}

signup_login "chanowner-$RUN@example.com"; AT="$TOKEN"
signup_login "chanviewer-$RUN@example.com"; VT="$TOKEN"
signup_login "chanstranger-$RUN@example.com"; ST="$TOKEN"
[ -n "$AT" ] && ok "actors provisioned" || bad "provisioning failed"

WS="$(curl -sS "$API/workspaces" -H "Authorization: Bearer $AT" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)"
[ -n "$WS" ] && ok "got workspace ($WS)" || bad "no workspace id"
CH="$API/workspaces/$WS/channels"

# 1. Owner lists channels -> 200 (empty)
c=$(codeauth GET "$AT" "$CH/")
[ "$c" = "200" ] && ok "owner lists channels -> 200" || bad "list -> $c (want 200)"

# 2. Connect unsupported platform -> 400 (no provider registered)
c=$(codeauth POST "$AT" "$CH/connect" '{"platform":"twitter"}')
[ "$c" = "400" ] && ok "connect unsupported platform -> 400" || bad "connect -> $c (want 400)"

# 3. Owner adds viewer (read only)
codeauth POST "$AT" "$API/workspaces/$WS/members" "{\"email\":\"chanviewer-$RUN@example.com\",\"role\":\"viewer\"}" >/dev/null

# 4. Viewer can list (read) but cannot connect (needs manage_channels)
c=$(codeauth GET "$VT" "$CH/")
[ "$c" = "200" ] && ok "viewer lists channels -> 200" || bad "viewer list -> $c (want 200)"
c=$(codeauth POST "$VT" "$CH/connect" '{"platform":"twitter"}')
[ "$c" = "403" ] && ok "viewer cannot connect -> 403" || bad "viewer connect -> $c (want 403)"

# 5. Disconnect requires manage_channels: viewer -> 403
c=$(codeauth DELETE "$VT" "$CH/00000000-0000-0000-0000-000000000000")
[ "$c" = "403" ] && ok "viewer cannot disconnect -> 403" || bad "viewer disconnect -> $c (want 403)"

# 6. Workspace isolation: stranger (non-member) -> 403
c=$(codeauth GET "$ST" "$CH/")
[ "$c" = "403" ] && ok "non-member denied (isolation) -> 403" || bad "non-member -> $c (want 403)"

# 7. Unauthenticated -> 401
c=$(curl -sS -o /dev/null -w '%{http_code}' "$CH/")
[ "$c" = "401" ] && ok "unauthenticated -> 401" || bad "unauth -> $c (want 401)"

echo
echo "Summary: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
