#!/usr/bin/env bash
# Analytics endpoint test against a running server. Verifies authz (capability
# gating), validation, and response shapes. Populating real metrics needs a
# connected channel + the poller (covered by the Go integration test
# internal/worker/analytics_integration_test.go); here we exercise the HTTP/authz/
# validation surface, which works without live platform data.
set -uo pipefail

API="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1"
AUTH="$API/auth"
PASS="analytics-test-pw"
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
code() { # METHOD TOKEN URL
	curl -sS -o /dev/null -w '%{http_code}' -X "$1" "$3" -H "Authorization: Bearer $2"
}

signup_login "anaowner-$RUN@example.com";   AT="$TOKEN"
signup_login "anaviewer-$RUN@example.com";  VT="$TOKEN"
signup_login "anastranger-$RUN@example.com"; ST="$TOKEN"
[ -n "$AT" ] && ok "actors provisioned" || bad "provisioning failed"

WS="$(curl -sS "$API/workspaces" -H "Authorization: Bearer $AT" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)"
[ -n "$WS" ] && ok "got workspace ($WS)" || bad "no workspace id"
A="$API/workspaces/$WS/analytics"
POST_ID="11111111-1111-1111-1111-111111111111"
CHAN_ID="22222222-2222-2222-2222-222222222222"

# 1. Owner reads overview -> 200 (empty).
c=$(code GET "$AT" "$A/")
[ "$c" = "200" ] && ok "owner overview -> 200" || bad "overview -> $c (want 200)"

# 2. Per-post metrics for an unknown post -> 200 (empty metrics, not an error).
c=$(code GET "$AT" "$A/posts/$POST_ID")
[ "$c" = "200" ] && ok "per-post metrics -> 200" || bad "per-post -> $c (want 200)"

# 3. Series requires a channel_id -> 400 when omitted.
c=$(code GET "$AT" "$A/posts/$POST_ID/series?metric=likes")
[ "$c" = "400" ] && ok "series without channel_id -> 400" || bad "series no-channel -> $c (want 400)"

# 4. Series requires a metric -> 400 when omitted (channel present).
c=$(code GET "$AT" "$A/posts/$POST_ID/series?channel_id=$CHAN_ID")
[ "$c" = "400" ] && ok "series without metric -> 400" || bad "series no-metric -> $c (want 400)"

# 5. Series with channel_id + metric -> 200.
c=$(code GET "$AT" "$A/posts/$POST_ID/series?channel_id=$CHAN_ID&metric=likes")
[ "$c" = "200" ] && ok "series with channel+metric -> 200" || bad "series -> $c (want 200)"

# 6. Invalid time range -> 400.
c=$(code GET "$AT" "$A/posts/$POST_ID/series?channel_id=$CHAN_ID&metric=likes&from=not-a-date")
[ "$c" = "400" ] && ok "series bad from -> 400" || bad "series bad from -> $c (want 400)"

# 6. CSV export -> 200 with header row.
EXP="$(curl -sS "$A/export.csv" -H "Authorization: Bearer $AT")"
printf '%s' "$EXP" | grep -q 'post_id,channel_id,platform_post_id,metric,value,captured_at' \
	&& ok "csv export has header" || bad "csv export missing header: $EXP"

# 7. Owner adds a read-only viewer; viewer can read analytics.
curl -sS -o /dev/null -X POST "$API/workspaces/$WS/members" -H "Authorization: Bearer $AT" \
	-H 'Content-Type: application/json' -d "{\"email\":\"anaviewer-$RUN@example.com\",\"role\":\"viewer\"}"
c=$(code GET "$VT" "$A/")
[ "$c" = "200" ] && ok "viewer reads analytics -> 200" || bad "viewer read -> $c (want 200)"

# 8. Non-member -> 403.
c=$(code GET "$ST" "$A/")
[ "$c" = "403" ] && ok "non-member denied (isolation) -> 403" || bad "non-member -> $c (want 403)"

# 9. Unauthenticated -> 401.
c=$(curl -sS -o /dev/null -w '%{http_code}' "$A/")
[ "$c" = "401" ] && ok "unauthenticated -> 401" || bad "unauth -> $c (want 401)"

echo
echo "Summary: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
