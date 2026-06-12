#!/usr/bin/env bash
# Live web-stack e2e (Phase 12.7): drives the SAME requests the web app makes,
# through the Next dev proxy (cookie session + CSRF double-submit), against the
# running backend with the X simulator:
#
#   signup -> login -> me -> workspace -> connect X (real authorize redirect)
#   -> oauth callback -> compose -> validate -> utm -> media upload (real
#   multipart) -> attach -> schedule (exact time + slots) -> calendar ->
#   worker publishes -> analytics -> failure paths -> logout.
#
# Prereqs: docker deps up, `postal sim`, `postal serve`, `postal worker`,
# `npm run dev` in web/ (or POSTAL_WEB_URL pointing at the web origin).
set -uo pipefail

WEB="${POSTAL_WEB_URL:-http://localhost:3000}"
API="$WEB/api/v1"
JAR="$(mktemp)"
RUN="$(date +%s)-$RANDOM"
EMAIL="webe2e-$RUN@example.com"
PW="web-e2e-password"
pass=0; fail=0

ok()  { echo "PASS  $*"; pass=$((pass+1)); }
bad() { echo "FAIL  $*"; fail=$((fail+1)); }
jget() { python3 -c "import json,sys;d=json.load(sys.stdin)
for k in '$1'.split('.'):
    d=d[int(k)] if isinstance(d,list) else d[k]
print(d)" 2>/dev/null; }
csrf() { awk '$6=="postal_csrf"{v=$7} END{print v}' "$JAR"; }

# --- auth (cookie flow, exactly what the browser does) ---
code=$(curl -sS -o /dev/null -w '%{http_code}' -c "$JAR" -H 'Content-Type: application/json' \
	-X POST "$API/auth/signup" -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\"}")
[ "$code" = "201" ] && ok "signup -> 201" || bad "signup -> $code (want 201)"

code=$(curl -sS -o /dev/null -w '%{http_code}' -c "$JAR" -b "$JAR" -H 'Content-Type: application/json' \
	-X POST "$API/auth/login" -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\"}")
[ "$code" = "200" ] && ok "login -> 200 (cookies set)" || bad "login -> $code (want 200)"
CSRF="$(csrf)"
[ -n "$CSRF" ] && ok "csrf cookie readable" || bad "no postal_csrf cookie in jar"

me=$(curl -sS -b "$JAR" "$API/auth/me" | jget data.email)
[ "$me" = "$EMAIL" ] && ok "GET /auth/me -> $me" || bad "me -> '$me' (want $EMAIL)"

WS=$(curl -sS -b "$JAR" "$API/workspaces/" | jget data.0.id)
[ -n "$WS" ] && ok "workspace: $WS" || bad "no workspace"

# --- connect X via the simulator (the browser redirect round trip) ---
AURL=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/channels/connect" -d '{"platform":"twitter"}' | jget data.authorize_url)
[ -n "$AURL" ] && ok "connect -> authorize_url ($(echo "$AURL" | cut -c1-40)…)" || bad "no authorize_url"

LOC=$(curl -sS -o /dev/null -w '%{redirect_url}' "$AURL")
case "$LOC" in
"$WEB/oauth/callback"*) ok "IdP redirects to frontend callback" ;;
*) bad "authorize redirect -> '$LOC'" ;;
esac
STATE=$(printf '%s' "$LOC" | sed -n 's/.*[?&]state=\([^&]*\).*/\1/p')
CODE=$(printf '%s' "$LOC" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')

CHAN_BODY=$(curl -sS -b "$JAR" "$API/channels/oauth/callback?state=$STATE&code=$CODE")
CH=$(printf '%s' "$CHAN_BODY" | jget data.id)
HANDLE=$(printf '%s' "$CHAN_BODY" | jget data.handle)
[ -n "$CH" ] && ok "oauth callback -> channel @$HANDLE ($CH)" || bad "callback failed: $CHAN_BODY"

n=$(curl -sS -b "$JAR" "$API/workspaces/$WS/channels/" | python3 -c 'import json,sys;print(len(json.load(sys.stdin)["data"]))' 2>/dev/null)
[ "$n" = "1" ] && ok "channels list shows 1 connected" || bad "channels list count=$n"

# --- compose -> validate -> utm ---
POST_ID=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/posts/" \
	-d "{\"variants\":[{\"channel_id\":\"$CH\",\"body\":\"Hello from the web e2e $RUN\"}]}" | jget data.id)
[ -n "$POST_ID" ] && ok "draft created ($POST_ID)" || bad "draft create failed"

valid=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -X POST \
	"$API/workspaces/$WS/posts/$POST_ID/validate" | jget data.variants.0.valid)
[ "$valid" = "True" ] && ok "validate -> valid" || bad "validate -> '$valid'"

tagged=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/posts/utm-preview" \
	-d '{"text":"see https://example.com/launch","utm":{"utm_source":"postal"}}' | jget data.text)
case "$tagged" in
*utm_source=postal*) ok "utm preview tags links" ;;
*) bad "utm preview -> '$tagged'" ;;
esac

# --- media: REAL multipart upload through the proxy (jsdom can't test this) ---
PNG="$(mktemp --suffix=.png)"
python3 2>/dev/null -c "
import struct, zlib
def chunk(t, d):
    c = t + d
    return struct.pack('>I', len(d)) + c + struct.pack('>I', zlib.crc32(c))
ihdr = struct.pack('>IIBBBBB', 8, 8, 8, 2, 0, 0, 0)
raw = b''.join(b'\x00' + b'\x7f\x3f\xbf' * 8 for _ in range(8))
png = b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', ihdr) + chunk(b'IDAT', zlib.compress(raw)) + chunk(b'IEND', b'')
open('$PNG','wb').write(png)
"
MEDIA_BODY=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -X POST \
	"$API/workspaces/$WS/media/" -F "file=@$PNG;type=image/png")
MEDIA_ID=$(printf '%s' "$MEDIA_BODY" | jget data.id)
MIME=$(printf '%s' "$MEDIA_BODY" | jget data.mime)
[ -n "$MEDIA_ID" ] && ok "multipart upload -> asset $MEDIA_ID ($MIME)" || bad "upload failed: $MEDIA_BODY"

code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" "$API/workspaces/$WS/media/$MEDIA_ID/download")
[ "$code" = "200" ] && ok "asset bytes downloadable -> 200" || bad "download -> $code"

BYTES=$(printf '%s' "$MEDIA_BODY" | jget data.bytes)
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X PUT "$API/workspaces/$WS/posts/$POST_ID" \
	-d "{\"variants\":[{\"channel_id\":\"$CH\",\"body\":\"Hello from the web e2e $RUN\",\"media\":[{\"media_id\":\"$MEDIA_ID\",\"kind\":\"image\",\"mime\":\"$MIME\",\"bytes\":$BYTES}]}]}")
[ "$code" = "200" ] && ok "media attached to draft -> 200" || bad "attach -> $code"

# --- schedule at an exact time; worker should publish via the simulator ---
RUN_AT=$(python3 -c "from datetime import datetime,timedelta,timezone;print((datetime.now(timezone.utc)+timedelta(seconds=4)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
JOB_ID=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/schedule" \
	-d "{\"post_id\":\"$POST_ID\",\"run_at\":\"$RUN_AT\"}" | jget data.jobs.0.id)
[ -n "$JOB_ID" ] && ok "scheduled job $JOB_ID at $RUN_AT" || bad "schedule failed"

# Poll with `from` in the past: the default window is [now, now+30d), so a
# just-executed job (run_at already behind us) would fall out of it.
FROM=$(python3 -c "from datetime import datetime,timedelta,timezone;print((datetime.now(timezone.utc)-timedelta(hours=1)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
status=""
for _ in $(seq 1 30); do
	status=$(curl -sS -b "$JAR" "$API/workspaces/$WS/calendar?from=$FROM" | python3 2>/dev/null -c "
import json,sys
jobs=json.load(sys.stdin)['data']['jobs']
print(next((j['status'] for j in jobs if j['id']=='$JOB_ID'),''))")
	[ "$status" = "published" ] && break
	sleep 3
done
[ "$status" = "published" ] && ok "worker published the job (status=published)" || bad "job status '$status' (want published)"

# --- slots + to_slots scheduling ---
SLOT_ID=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/slots/" \
	-d "{\"channel_id\":\"$CH\",\"day_of_week\":1,\"time_of_day\":\"09:00\",\"timezone\":\"UTC\"}" | jget data.id)
[ -n "$SLOT_ID" ] && ok "slot created ($SLOT_ID)" || bad "slot create failed"

POST2=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/posts/" \
	-d "{\"variants\":[{\"channel_id\":\"$CH\",\"body\":\"Slot-scheduled post $RUN\"}]}" | jget data.id)
njobs=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/schedule" \
	-d "{\"post_id\":\"$POST2\",\"to_slots\":true}" | python3 -c 'import json,sys;print(len(json.load(sys.stdin)["data"]["jobs"]))')
[ "$njobs" = "1" ] && ok "to_slots scheduling -> 1 job" || bad "to_slots -> $njobs jobs"

JOB2=$(curl -sS -b "$JAR" "$API/workspaces/$WS/calendar?to=$(python3 -c "from datetime import datetime,timedelta,timezone;print((datetime.now(timezone.utc)+timedelta(days=14)).strftime('%Y-%m-%dT%H:%M:%SZ'))")" \
	| python3 2>/dev/null -c "
import json,sys
jobs=json.load(sys.stdin)['data']['jobs']
print(next((j['id'] for j in jobs if j['post_id']=='$POST2'),''))")
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" -H "X-CSRF-Token: $CSRF" \
	-X DELETE "$API/workspaces/$WS/scheduled-jobs/$JOB2")
[ "$code" = "200" ] && ok "canceled the slot job -> 200" || bad "cancel -> $code"

# --- analytics endpoints respond for the web screens ---
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" "$API/workspaces/$WS/analytics/")
[ "$code" = "200" ] && ok "analytics overview -> 200" || bad "analytics -> $code"
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" "$API/workspaces/$WS/analytics/export.csv")
[ "$code" = "200" ] && ok "analytics CSV export -> 200" || bad "csv -> $code"

# --- failure paths the UI must surface ---
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/posts/" -d '{"variants":[]}')
[ "$code" = "403" ] && ok "mutation without CSRF -> 403" || bad "no-csrf -> $code (want 403)"
code=$(curl -sS -o /dev/null -w '%{http_code}' "$API/workspaces/$WS/channels/")
[ "$code" = "401" ] && ok "unauthenticated -> 401" || bad "unauth -> $code (want 401)"
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" "$API/channels/oauth/callback?state=bogus&code=x")
[ "$code" = "400" ] && ok "bogus oauth state -> 400" || bad "bogus state -> $code (want 400)"

# --- logout ends the session ---
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" -c "$JAR" -H "X-CSRF-Token: $CSRF" -X POST "$API/auth/logout")
[ "$code" = "200" ] && ok "logout -> 200" || bad "logout -> $code"
code=$(curl -sS -o /dev/null -w '%{http_code}' -b "$JAR" "$API/auth/me")
[ "$code" = "401" ] && ok "session gone after logout -> 401" || bad "post-logout me -> $code (want 401)"

rm -f "$JAR" "$PNG"
echo
echo "web-e2e: $pass passed, $fail failed"
[ "$fail" = "0" ]
