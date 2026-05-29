#!/usr/bin/env bash
# Composer endpoint test against a running server. Verifies authz (capability
# gating), validation, and UTM preview. The full create→validate happy path
# needs a connected channel (FK) and is covered by the Go integration test
# (internal/post/integration_test.go); here we exercise the HTTP/authz/validation
# surface that doesn't require a live channel.
set -uo pipefail

API="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1"
AUTH="$API/auth"
PASS="posts-test-pw"
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

signup_login "postowner-$RUN@example.com"; AT="$TOKEN"
signup_login "postviewer-$RUN@example.com"; VT="$TOKEN"
signup_login "poststranger-$RUN@example.com"; ST="$TOKEN"
[ -n "$AT" ] && ok "actors provisioned" || bad "provisioning failed"

WS="$(curl -sS "$API/workspaces" -H "Authorization: Bearer $AT" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)"
[ -n "$WS" ] && ok "got workspace ($WS)" || bad "no workspace id"
P="$API/workspaces/$WS/posts"

# 1. Owner lists posts -> 200 (empty)
c=$(codeauth GET "$AT" "$P/")
[ "$c" = "200" ] && ok "owner lists posts -> 200" || bad "list -> $c (want 200)"

# 2. Create with no variants -> 400
c=$(codeauth POST "$AT" "$P/" '{"variants":[]}')
[ "$c" = "400" ] && ok "create with no variants -> 400" || bad "no-variants -> $c (want 400)"

# 3. Create targeting a non-existent/foreign channel -> 404
c=$(codeauth POST "$AT" "$P/" '{"variants":[{"channel_id":"00000000-0000-0000-0000-000000000000","body":"hi"}]}')
[ "$c" = "404" ] && ok "create with foreign channel -> 404" || bad "foreign channel -> $c (want 404)"

# 4. UTM preview -> 200 and tags the link
BODY="$(curl -sS -X POST "$P/utm-preview" -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' -d '{"text":"see https://example.com/x","utm":{"utm_source":"postal"}}')"
printf '%s' "$BODY" | grep -q 'utm_source=postal' && ok "utm-preview tags link" || bad "utm-preview missing tag: $BODY"

# 5. Owner adds a viewer (read only)
codeauth POST "$AT" "$API/workspaces/$WS/members" "{\"email\":\"postviewer-$RUN@example.com\",\"role\":\"viewer\"}" >/dev/null

# 6. Viewer can list (read) but cannot create (needs create capability)
c=$(codeauth GET "$VT" "$P/")
[ "$c" = "200" ] && ok "viewer lists posts -> 200" || bad "viewer list -> $c (want 200)"
c=$(codeauth POST "$VT" "$P/" '{"variants":[]}')
[ "$c" = "403" ] && ok "viewer cannot create -> 403" || bad "viewer create -> $c (want 403)"

# 7. Non-member -> 403
c=$(codeauth GET "$ST" "$P/")
[ "$c" = "403" ] && ok "non-member denied (isolation) -> 403" || bad "non-member -> $c (want 403)"

# 8. Unauthenticated -> 401
c=$(curl -sS -o /dev/null -w '%{http_code}' "$P/")
[ "$c" = "401" ] && ok "unauthenticated -> 401" || bad "unauth -> $c (want 401)"

echo
echo "Summary: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
