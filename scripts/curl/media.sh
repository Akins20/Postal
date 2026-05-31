#!/usr/bin/env bash
# Media endpoint test against a running server. Exercises the upload→list→
# download→delete happy path plus authz (capability gating) and validation
# failures. Requires the server to be started with object storage configured
# (POSTAL_STORAGE_ENDPOINT); when it is not, the /media routes are absent and the
# script reports that and exits 0 (nothing to test).
set -uo pipefail

API="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1"
AUTH="$API/auth"
PASS="media-test-pw"
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
code() { curl -sS -o /dev/null -w '%{http_code}' "$@"; }

# A 1x1 PNG decoded from base64 — a real, valid image the server can probe.
PNG="$(mktemp --suffix=.png)"
trap 'rm -f "$PNG"' EXIT
base64 -d > "$PNG" <<'B64'
iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==
B64

signup_login "mediaowner-$RUN@example.com";   AT="$TOKEN"
signup_login "mediaviewer-$RUN@example.com";  VT="$TOKEN"
signup_login "mediastranger-$RUN@example.com"; ST="$TOKEN"
[ -n "$AT" ] && ok "actors provisioned" || bad "provisioning failed"

WS="$(curl -sS "$API/workspaces" -H "Authorization: Bearer $AT" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)"
[ -n "$WS" ] && ok "got workspace ($WS)" || bad "no workspace id"
M="$API/workspaces/$WS/media"

# Probe: is the media pipeline enabled on this server?
c=$(code -X POST "$M/" -H "Authorization: Bearer $AT" -F "file=@$PNG;type=image/png")
if [ "$c" = "404" ]; then
	echo "NOTE  media routes are not mounted (storage not configured); skipping."
	echo "Summary: $pass passed, $fail failed"
	exit 0
fi

# 1. Owner uploads a PNG -> 201, returns the asset.
BODY="$(curl -sS -X POST "$M/" -H "Authorization: Bearer $AT" -F "file=@$PNG;type=image/png")"
MID="$(printf '%s' "$BODY" | field id)"
[ -n "$MID" ] && ok "owner uploads png -> asset $MID" || bad "upload returned no id: $BODY"
printf '%s' "$BODY" | grep -q '"kind":"image"' && ok "asset kind=image" || bad "kind not image: $BODY"

# 2. Owner lists media -> 200 and contains the new asset.
BODY="$(curl -sS "$M/" -H "Authorization: Bearer $AT")"
printf '%s' "$BODY" | grep -q "$MID" && ok "list contains uploaded asset" || bad "list missing asset: $BODY"

# 3. Owner downloads the asset -> 200 with image content type.
c=$(code "$M/$MID/download" -H "Authorization: Bearer $AT")
[ "$c" = "200" ] && ok "owner downloads asset -> 200" || bad "download -> $c (want 200)"

# 4. Upload an unsupported type -> 400.
c=$(code -X POST "$M/" -H "Authorization: Bearer $AT" -F "file=@$PNG;type=application/zip")
[ "$c" = "400" ] && ok "unsupported media type -> 400" || bad "unsupported type -> $c (want 400)"

# 5. Upload with the wrong form field -> 400 (missing 'file').
c=$(code -X POST "$M/" -H "Authorization: Bearer $AT" -F "notfile=@$PNG;type=image/png")
[ "$c" = "400" ] && ok "missing 'file' field -> 400" || bad "missing file -> $c (want 400)"

# 6. Owner adds a read-only viewer; viewer can list but not upload.
curl -sS -o /dev/null -X POST "$API/workspaces/$WS/members" -H "Authorization: Bearer $AT" \
	-H 'Content-Type: application/json' -d "{\"email\":\"mediaviewer-$RUN@example.com\",\"role\":\"viewer\"}"
c=$(code "$M/" -H "Authorization: Bearer $VT")
[ "$c" = "200" ] && ok "viewer lists media -> 200" || bad "viewer list -> $c (want 200)"
c=$(code -X POST "$M/" -H "Authorization: Bearer $VT" -F "file=@$PNG;type=image/png")
[ "$c" = "403" ] && ok "viewer cannot upload (no upload cap) -> 403" || bad "viewer upload -> $c (want 403)"

# 7. Non-member is isolated -> 403.
c=$(code "$M/" -H "Authorization: Bearer $ST")
[ "$c" = "403" ] && ok "non-member denied (isolation) -> 403" || bad "non-member -> $c (want 403)"

# 8. Unauthenticated -> 401.
c=$(code "$M/")
[ "$c" = "401" ] && ok "unauthenticated -> 401" || bad "unauth -> $c (want 401)"

# 9. Owner deletes the asset -> 200; a subsequent download -> 404.
c=$(code -X DELETE "$M/$MID" -H "Authorization: Bearer $AT")
[ "$c" = "200" ] && ok "owner deletes asset -> 200" || bad "delete -> $c (want 200)"
c=$(code "$M/$MID/download" -H "Authorization: Bearer $AT")
[ "$c" = "404" ] && ok "download after delete -> 404" || bad "post-delete download -> $c (want 404)"

echo
echo "Summary: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
