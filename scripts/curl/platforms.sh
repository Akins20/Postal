#!/usr/bin/env bash
# Instagram + TikTok live e2e (Phase 14) against the running stack: tri-sim
# (postal sim), serve, worker, and the Next dev proxy. Covers OAuth connect,
# media upload, compose validation (text-only IG rejected), and a dual-
# platform publish executed by the worker (IG container flow via presigned
# URL; TikTok FILE_UPLOAD).
set -uo pipefail
API="http://localhost:3000/api/v1"
JAR="$(mktemp)"; RUN="$(date +%s)-$RANDOM"
pass=0; fail=0
ok(){ echo "PASS  $*"; pass=$((pass+1)); }; bad(){ echo "FAIL  $*"; fail=$((fail+1)); }
jget(){ python3 -c "import json,sys;d=json.load(sys.stdin)
for k in '$1'.split('.'):
    d=d[int(k)] if isinstance(d,list) else d[k]
print(d)" 2>/dev/null; }
csrf(){ awk '$6=="postal_csrf"{v=$7} END{print v}' "$JAR"; }

curl -sS -o /dev/null -c "$JAR" -H 'Content-Type: application/json' -X POST "$API/auth/signup" -d "{\"email\":\"igtt-$RUN@example.com\",\"password\":\"igtt-test-pw\"}"
curl -sS -o /dev/null -c "$JAR" -b "$JAR" -H 'Content-Type: application/json' -X POST "$API/auth/login" -d "{\"email\":\"igtt-$RUN@example.com\",\"password\":\"igtt-test-pw\"}"
CSRF="$(csrf)"; WS=$(curl -sS -b "$JAR" "$API/workspaces/" | jget data.0.id)
[ -n "$WS" ] && ok "workspace $WS" || bad "no workspace"

connect() { # platform -> channel id
  local AURL LOC STATE CODE
  AURL=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' -X POST "$API/workspaces/$WS/channels/connect" -d "{\"platform\":\"$1\"}" | jget data.authorize_url)
  LOC=$(curl -sS -o /dev/null -w '%{redirect_url}' "$AURL")
  STATE=$(printf '%s' "$LOC" | sed -n 's/.*[?&]state=\([^&]*\).*/\1/p'); CODE=$(printf '%s' "$LOC" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')
  curl -sS -b "$JAR" "$API/channels/oauth/callback?state=$STATE&code=$CODE" | jget data.id
}

IGCH=$(connect instagram); [ -n "$IGCH" ] && ok "instagram connected ($IGCH)" || bad "instagram connect failed"
TTCH=$(connect tiktok);    [ -n "$TTCH" ] && ok "tiktok connected ($TTCH)" || bad "tiktok connect failed"
H=$(curl -sS -b "$JAR" "$API/workspaces/$WS/channels/" | python3 -c 'import json,sys;print(",".join(sorted(c["handle"] for c in json.load(sys.stdin)["data"])))')
[ "$H" = "@simgram,@simtok" ] && ok "handles: $H" || bad "handles: $H"

# media upload (real multipart -> MinIO; presigned URL feeds IG, bytes feed TikTok)
PNG="$(mktemp --suffix=.png)"
python3 -c "
import struct, zlib
def c(t,d):
    x=t+d; return struct.pack('>I',len(d))+x+struct.pack('>I',zlib.crc32(x))
ihdr=struct.pack('>IIBBBBB',8,8,8,2,0,0,0)
raw=b''.join(b'\x00'+b'\x10\x80\xff'*8 for _ in range(8))
open('$PNG','wb').write(b'\x89PNG\r\n\x1a\n'+c(b'IHDR',ihdr)+c(b'IDAT',zlib.compress(raw))+c(b'IEND',b''))"
MB=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -X POST "$API/workspaces/$WS/media/" -F "file=@$PNG;type=image/png")
MID=$(printf '%s' "$MB" | jget data.id); BYTES=$(printf '%s' "$MB" | jget data.bytes)
[ -n "$MID" ] && ok "media uploaded ($MID)" || bad "upload failed: $MB"

# text-only to IG must be rejected at compose validation
PID=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' -X POST "$API/workspaces/$WS/posts/" \
  -d "{\"variants\":[{\"channel_id\":\"$IGCH\",\"body\":\"text only\"}]}" | jget data.id)
V=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -X POST "$API/workspaces/$WS/posts/$PID/validate" | jget data.variants.0.valid)
[ "$V" = "False" ] && ok "IG text-only post flagged invalid" || bad "IG text-only validate -> '$V'"

# media post to BOTH platforms, publish now, worker should publish both (free platforms, no wallet)
P2=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' -X POST "$API/workspaces/$WS/posts/" \
  -d "{\"variants\":[
    {\"channel_id\":\"$IGCH\",\"body\":\"ig pic $RUN\",\"media\":[{\"media_id\":\"$MID\",\"kind\":\"image\",\"mime\":\"image/png\",\"bytes\":$BYTES}]},
    {\"channel_id\":\"$TTCH\",\"body\":\"tt pic $RUN\",\"media\":[{\"media_id\":\"$MID\",\"kind\":\"image\",\"mime\":\"image/png\",\"bytes\":$BYTES}]}]}" | jget data.id)
[ -n "$P2" ] && ok "dual-platform media post drafted" || bad "draft failed"
RUN_AT=$(python3 -c "from datetime import datetime,timedelta,timezone;print((datetime.now(timezone.utc)+timedelta(seconds=3)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
JOBS=$(curl -sS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' -X POST "$API/workspaces/$WS/schedule" -d "{\"post_id\":\"$P2\",\"run_at\":\"$RUN_AT\"}" | python3 -c 'import json,sys;print(len(json.load(sys.stdin)["data"]["jobs"]))')
[ "$JOBS" = "2" ] && ok "2 jobs queued (free platforms, no wallet needed)" || bad "jobs: $JOBS"

FROM=$(python3 -c "from datetime import datetime,timedelta,timezone;print((datetime.now(timezone.utc)-timedelta(hours=1)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
done_count=""
for _ in $(seq 1 30); do
  done_count=$(curl -sS -b "$JAR" "$API/workspaces/$WS/calendar?from=$FROM" | python3 -c "
import json,sys
jobs=[j for j in json.load(sys.stdin)['data']['jobs'] if j['post_id']=='$P2']
print(sum(1 for j in jobs if j['status']=='published'))" 2>/dev/null)
  [ "$done_count" = "2" ] && break; sleep 3
done
[ "$done_count" = "2" ] && ok "worker published to Instagram AND TikTok" || bad "published count: $done_count"
rm -f "$JAR" "$PNG"
echo; echo "ig+tiktok e2e: $pass passed, $fail failed"; [ "$fail" = "0" ]
