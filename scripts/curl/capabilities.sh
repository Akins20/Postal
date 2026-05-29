#!/usr/bin/env bash
# Capability-based authorization test against a running server. Exercises the
# §5.1 invariants: capability gating, no-privilege-escalation, owner-immutable,
# and workspace isolation. Logins do not require email verification (that only
# gates publishing later), so the flow uses fresh accounts directly.
set -uo pipefail

API="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1"
AUTH="$API/auth"
PASS="capabilities-test-pw"
RUN="$(date +%s)-$RANDOM"
pass=0; fail=0

ok()  { echo "PASS  $*"; pass=$((pass+1)); }
bad() { echo "FAIL  $*"; fail=$((fail+1)); }
field() { sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p"; }

# signup_login EMAIL -> sets globals TOKEN and NUID
signup_login() {
	local email="$1"
	curl -sS -o /dev/null -H 'Content-Type: application/json' -X POST "$AUTH/signup" \
		-d "{\"email\":\"$email\",\"password\":\"$PASS\"}"
	local body
	body="$(curl -sS -H 'Content-Type: application/json' -X POST "$AUTH/login" \
		-d "{\"email\":\"$email\",\"password\":\"$PASS\"}")"
	TOKEN="$(printf '%s' "$body" | field access_token)"
	NUID="$(printf '%s' "$body" | field id)"
}

codeauth() { # METHOD TOKEN URL [JSON]
	local m="$1" tok="$2" url="$3" data="${4:-}"
	if [ -n "$data" ]; then
		curl -sS -o /dev/null -w '%{http_code}' -X "$m" "$url" \
			-H "Authorization: Bearer $tok" -H 'Content-Type: application/json' -d "$data"
	else
		curl -sS -o /dev/null -w '%{http_code}' -X "$m" "$url" -H "Authorization: Bearer $tok"
	fi
}

echo "Run id: $RUN"

# --- actors ---
signup_login "owner-$RUN@example.com";  AT="$TOKEN"; AID="$NUID"
signup_login "bob-$RUN@example.com";    BT="$TOKEN"; BID="$NUID"
signup_login "carol-$RUN@example.com";  CT="$TOKEN"; CID="$NUID"
signup_login "erin-$RUN@example.com";   ET="$TOKEN"; EID="$NUID"
signup_login "dave-$RUN@example.com";   DT="$TOKEN"; DID="$NUID"
[ -n "$AT" ] && [ -n "$BID" ] && ok "actors provisioned" || bad "actor provisioning failed"

# A's personal workspace id
WS="$(curl -sS "$API/workspaces" -H "Authorization: Bearer $AT" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)"
[ -n "$WS" ] && ok "owner lists workspace ($WS)" || bad "could not get workspace id"

M="$API/workspaces/$WS/members"

# 1. Owner adds Bob as viewer
c=$(codeauth POST "$AT" "$M" "{\"email\":\"bob-$RUN@example.com\",\"role\":\"viewer\"}")
[ "$c" = "201" ] && ok "owner adds viewer -> 201" || bad "add viewer -> $c (want 201)"

# 2. Members list shows Bob with read only
BODY="$(curl -sS "$M" -H "Authorization: Bearer $AT")"
printf '%s' "$BODY" | grep -q "\"$BID\"" && ok "members list includes Bob" || bad "Bob missing from members"

# 3. Owner updates Bob -> read+upload
c=$(codeauth PATCH "$AT" "$M/$BID/capabilities" '{"capabilities":["read","upload"]}')
[ "$c" = "200" ] && ok "owner updates Bob caps -> 200" || bad "update caps -> $c (want 200)"

# 4. Owner promotes Bob to admin (gains manage_members)
c=$(codeauth PATCH "$AT" "$M/$BID/capabilities" '{"role":"admin"}')
[ "$c" = "200" ] && ok "owner promotes Bob to admin -> 200" || bad "promote -> $c (want 200)"

# 5. Bob (admin) can add Carol as viewer
c=$(codeauth POST "$BT" "$M" "{\"email\":\"carol-$RUN@example.com\",\"role\":\"viewer\"}")
[ "$c" = "201" ] && ok "admin adds member -> 201" || bad "admin add -> $c (want 201)"

# 6. Bob cannot grant a capability he lacks (manage_workspace) -> escalation 403
c=$(codeauth POST "$BT" "$M" "{\"email\":\"erin-$RUN@example.com\",\"capabilities\":[\"read\",\"manage_workspace\"]}")
[ "$c" = "403" ] && ok "privilege escalation blocked -> 403" || bad "escalation -> $c (want 403)"

# 7. Carol (viewer) can read members but cannot manage them
c=$(codeauth GET "$CT" "$M")
[ "$c" = "200" ] && ok "viewer reads members -> 200" || bad "viewer read -> $c (want 200)"
c=$(codeauth POST "$CT" "$M" "{\"email\":\"dave-$RUN@example.com\",\"role\":\"viewer\"}")
[ "$c" = "403" ] && ok "viewer cannot add members -> 403" || bad "viewer add -> $c (want 403)"

# 8. Owner is immutable
c=$(codeauth PATCH "$AT" "$M/$AID/capabilities" '{"role":"viewer"}')
[ "$c" = "403" ] && ok "owner capabilities immutable -> 403" || bad "owner mutate -> $c (want 403)"

# 9. Workspace isolation: Dave (non-member) cannot read members
c=$(codeauth GET "$DT" "$M")
[ "$c" = "403" ] && ok "non-member denied (isolation) -> 403" || bad "non-member -> $c (want 403)"

echo
echo "Summary: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
