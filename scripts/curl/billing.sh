#!/usr/bin/env bash
# Wallet billing e2e (Phase 13) against a running server + worker + X simulator.
# Covers: wallet read, capability gates, dev top-up, Stripe/Paystack webhooks
# (signed with the configured test secrets), idempotent replay, the schedule
# soft gate, the worker's hard charge on publish, and the ledger.
#
# Stripe/Paystack webhook checks need the server started with:
#   POSTAL_STRIPE_SECRET_KEY + POSTAL_STRIPE_WEBHOOK_SECRET (any test values)
#   POSTAL_PAYSTACK_SECRET_KEY (any test value)
# They are skipped when WEBHOOK_STRIPE_SECRET / WEBHOOK_PAYSTACK_SECRET are unset.
set -uo pipefail

API="${POSTAL_BASE_URL:-http://localhost:8080}/api/v1"
PW="billing-test-pw"
RUN="$(date +%s)-$RANDOM"
pass=0; fail=0

ok()  { echo "PASS  $*"; pass=$((pass+1)); }
bad() { echo "FAIL  $*"; fail=$((fail+1)); }
jget() { python3 -c "import json,sys;d=json.load(sys.stdin)
for k in '$1'.split('.'):
    d=d[int(k)] if isinstance(d,list) else d[k]
print(d)" 2>/dev/null; }

signup_login() {
	curl -sS -o /dev/null -H 'Content-Type: application/json' -X POST "$API/auth/signup" -d "{\"email\":\"$1\",\"password\":\"$PW\"}"
	curl -sS -H 'Content-Type: application/json' -X POST "$API/auth/login" -d "{\"email\":\"$1\",\"password\":\"$PW\"}" | jget data.access_token
}

AT="$(signup_login "billing-owner-$RUN@example.com")"
VT="$(signup_login "billing-viewer-$RUN@example.com")"
[ -n "$AT" ] && ok "actors provisioned" || bad "provisioning failed"

WS=$(curl -sS "$API/workspaces" -H "Authorization: Bearer $AT" | jget data.0.id)
B="$API/workspaces/$WS/billing"

# 1. Wallet starts empty with the X price list visible.
bal=$(curl -sS -H "Authorization: Bearer $AT" "$B/wallet" | jget data.balance)
cost=$(curl -sS -H "Authorization: Bearer $AT" "$B/wallet" | jget data.publish_costs.twitter)
[ "$bal" = "0" ] && ok "fresh wallet -> balance 0" || bad "balance '$bal'"
[ -n "$cost" ] && [ "$cost" != "0" ] && ok "X publish cost advertised ($cost credits)" || bad "no X cost in wallet"

# 2. Capability gates: viewer (not a member) cannot read; member-viewer cannot top up.
code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $VT" "$B/wallet")
[ "$code" = "403" ] && ok "stranger wallet read -> 403" || bad "stranger read -> $code"
curl -sS -o /dev/null -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/members" -d "{\"email\":\"billing-viewer-$RUN@example.com\",\"role\":\"viewer\"}"
code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $VT" -H 'Content-Type: application/json' \
	-X POST "$B/topup" -d '{"provider":"dev","credits":1000}')
[ "$code" = "403" ] && ok "viewer top-up -> 403 (needs manage_workspace)" || bad "viewer topup -> $code"

# 3. Validation: below-minimum and unknown provider are rejected.
code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$B/topup" -d '{"provider":"dev","credits":1}')
[ "$code" = "400" ] && ok "below-minimum top-up -> 400" || bad "min topup -> $code"
code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$B/topup" -d '{"provider":"bitcoin","credits":1000}')
[ "$code" = "400" ] && ok "unknown provider -> 400" || bad "unknown provider -> $code"

# 4. Schedule soft gate: an X post with an empty wallet is refused up front.
AURL=$(curl -sS -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/channels/connect" -d '{"platform":"twitter"}' | jget data.authorize_url)
LOC=$(curl -sS -o /dev/null -w '%{redirect_url}' "$AURL")
STATE=$(printf '%s' "$LOC" | sed -n 's/.*[?&]state=\([^&]*\).*/\1/p')
CODE=$(printf '%s' "$LOC" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')
CH=$(curl -sS -H "Authorization: Bearer $AT" "$API/channels/oauth/callback?state=$STATE&code=$CODE" | jget data.id)
[ -n "$CH" ] && ok "X channel connected via simulator" || bad "channel connect failed"

POST_ID=$(curl -sS -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/posts/" \
	-d "{\"variants\":[{\"channel_id\":\"$CH\",\"body\":\"Billing e2e $RUN\"}]}" | jget data.id)
RUN_AT=$(python3 -c "from datetime import datetime,timedelta,timezone;print((datetime.now(timezone.utc)+timedelta(seconds=4)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
resp=$(curl -sS -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/schedule" -d "{\"post_id\":\"$POST_ID\",\"run_at\":\"$RUN_AT\"}")
echo "$resp" | grep -q insufficient_credits && ok "schedule soft gate -> insufficient_credits" || bad "soft gate response: $resp"

# 5. Dev top-up credits instantly (development-only provider).
url=$(curl -sS -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$B/topup" -d '{"provider":"dev","credits":1000}' | jget data.checkout_url)
case "$url" in *status=success*) ok "dev top-up checkout -> success url" ;; *) bad "dev topup url: '$url'" ;; esac
bal=$(curl -sS -H "Authorization: Bearer $AT" "$B/wallet" | jget data.balance)
[ "$bal" = "1000" ] && ok "wallet credited -> 1000" || bad "balance after dev topup: $bal"

# 6. Scheduling now passes; the worker charges exactly one publish cost.
RUN_AT=$(python3 -c "from datetime import datetime,timedelta,timezone;print((datetime.now(timezone.utc)+timedelta(seconds=4)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
JOB=$(curl -sS -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' \
	-X POST "$API/workspaces/$WS/schedule" -d "{\"post_id\":\"$POST_ID\",\"run_at\":\"$RUN_AT\"}" | jget data.jobs.0.id)
[ -n "$JOB" ] && ok "scheduled with funds ($JOB)" || bad "schedule failed with funds"
expected=$((1000 - cost))
got=""
for _ in $(seq 1 30); do
	got=$(curl -sS -H "Authorization: Bearer $AT" "$B/wallet" | jget data.balance)
	[ "$got" = "$expected" ] && break
	sleep 3
done
[ "$got" = "$expected" ] && ok "worker charged one publish -> balance $got" || bad "balance after publish: $got (want $expected)"

# 7. The ledger shows the movements (topup + charge).
kinds=$(curl -sS -H "Authorization: Bearer $AT" "$B/ledger" | python3 -c 'import json,sys;print(",".join(sorted({e["kind"] for e in json.load(sys.stdin)["data"]})))' 2>/dev/null)
case "$kinds" in *publish_charge*topup*) ok "ledger shows topup + publish_charge" ;; *) bad "ledger kinds: '$kinds'" ;; esac

# 8. Signed provider webhooks (skipped unless the secrets are exported).
if [ -n "${WEBHOOK_STRIPE_SECRET:-}" ]; then
	ts=$(date +%s)
	payload="{\"id\":\"evt_$RUN\",\"type\":\"checkout.session.completed\",\"data\":{\"object\":{\"payment_status\":\"paid\",\"metadata\":{\"workspace_id\":\"$WS\",\"credits\":\"700\"}}}}"
	sig=$(printf '%s.%s' "$ts" "$payload" | openssl dgst -sha256 -hmac "$WEBHOOK_STRIPE_SECRET" -hex | sed 's/^.* //')
	st=$(curl -sS -X POST "$API/billing/webhooks/stripe" -H "Stripe-Signature: t=$ts,v1=$sig" -d "$payload" | jget data.status)
	[ "$st" = "credited" ] && ok "stripe webhook credits" || bad "stripe webhook -> '$st'"
	st=$(curl -sS -X POST "$API/billing/webhooks/stripe" -H "Stripe-Signature: t=$ts,v1=$sig" -d "$payload" | jget data.status)
	[ "$st" = "duplicate" ] && ok "stripe webhook replay -> duplicate (no double credit)" || bad "stripe replay -> '$st'"
	code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$API/billing/webhooks/stripe" -H "Stripe-Signature: t=$ts,v1=deadbeef" -d "$payload")
	[ "$code" = "401" ] && ok "stripe bad signature -> 401" || bad "stripe bad sig -> $code"
else
	echo "SKIP  stripe webhook checks (WEBHOOK_STRIPE_SECRET unset)"
fi
if [ -n "${WEBHOOK_PAYSTACK_SECRET:-}" ]; then
	payload="{\"event\":\"charge.success\",\"data\":{\"reference\":\"ps_$RUN\",\"status\":\"success\",\"metadata\":{\"workspace_id\":\"$WS\",\"credits\":\"300\"}}}"
	sig=$(printf '%s' "$payload" | openssl dgst -sha512 -hmac "$WEBHOOK_PAYSTACK_SECRET" -hex | sed 's/^.* //')
	st=$(curl -sS -X POST "$API/billing/webhooks/paystack" -H "X-Paystack-Signature: $sig" -d "$payload" | jget data.status)
	[ "$st" = "credited" ] && ok "paystack webhook credits" || bad "paystack webhook -> '$st'"
	code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$API/billing/webhooks/paystack" -H "X-Paystack-Signature: 00" -d "$payload")
	[ "$code" = "401" ] && ok "paystack bad signature -> 401" || bad "paystack bad sig -> $code"
else
	echo "SKIP  paystack webhook checks (WEBHOOK_PAYSTACK_SECRET unset)"
fi

echo
echo "billing: $pass passed, $fail failed"
[ "$fail" = "0" ]
