# Phase 13 — Wallet billing (X-exclusive pay-per-use)

> Status: IN PROGRESS (started 2026-06-12). Unfreezes the backend for one
> contained domain: `internal/billing`. Everything else stays frozen.

## 1. Why & what

X's API bills the **operator** per request — users can't bring their own dev
accounts. So workspaces pre-fund a **wallet** with Postal; each successful X
publish deducts credits. **Every other platform is and stays free** — billing
is exclusive to X, and the UI says so explicitly (wallet page, channels page,
schedule errors).

- Wallet holds **credits** (integer). Default pricing (all env-configurable):
  **1 credit = $0.01** (`CREDITS_PER_USD_CENT=1`), an X publish costs
  **25 credits** (`PUBLISH_COST_TWITTER=25`). Platforms without a configured
  cost are free and skip billing entirely.
- Payments: **Stripe** (cards, global) and **Paystack** (cards/bank, Africa) —
  the user picks at checkout. Both are redirect checkouts + signed webhooks.
  A **dev provider** (enabled only when `POSTAL_ENV=development`) credits
  instantly so the local stack works without real keys.

## 2. Data model (migration 10)

- `wallets` — `workspace_id PK/FK`, `balance` bigint ≥ 0, `updated_at`.
  Created lazily (first read/topup).
- `wallet_ledger` — append-only: `id`, `workspace_id`, `kind`
  (`topup|publish_charge|refund|adjustment`), `credits` (signed),
  `reference` (provider event id / job id), `note`, `created_at`.
  **Unique (workspace_id, kind, reference)** = idempotency for webhook
  retries and job re-claims.

## 3. Money flow

- **Top-up:** `POST /workspaces/{id}/billing/topup {provider, credits}` →
  create a Stripe Checkout Session / Paystack transaction (server-side; amount
  derived from credits + currency config) with `workspace_id`+`credits` in
  metadata → return the redirect URL. The browser pays on the provider's page
  and returns to `/wallet?status=…`.
- **Webhooks** (public, signature-verified, no session auth):
  `POST /api/v1/billing/webhooks/stripe` (HMAC `Stripe-Signature`,
  `checkout.session.completed`) and `/paystack` (HMAC-SHA512
  `x-paystack-signature`, `charge.success`) → credit the wallet, idempotent
  by provider event/reference id.
- **Charge:** the worker, after **claiming** an X job and before publishing,
  atomically deducts (`UPDATE … SET balance = balance - $cost WHERE balance >=
  $cost`). No funds → job fails `billing_insufficient` (permanent,
  user-actionable). A **permanent** publish failure after deduction refunds
  (ledger `refund`, reference = job id). Retries can't double-charge (single
  claim + unique ledger reference).
- **Soft gate:** scheduling X variants checks `balance ≥ cost × jobs` and
  rejects with `insufficient_credits` so the UI can prompt a top-up before
  anything is queued.

## 4. API (added to docs/openapi.yaml)

- `GET  /workspaces/{id}/billing/wallet` → `{balance, publish_costs: {twitter: 25}}` (read)
- `GET  /workspaces/{id}/billing/ledger` → entries, newest first (read)
- `POST /workspaces/{id}/billing/topup` → `{checkout_url}` (manage_workspace)
- `POST /billing/webhooks/stripe` / `…/paystack` (public; signature is the auth)

## 5. Config (env)

```
POSTAL_BILLING_CREDITS_PER_USD_CENT=1
POSTAL_BILLING_PUBLISH_COST_TWITTER=25
POSTAL_BILLING_MIN_TOPUP_CREDITS=500
POSTAL_STRIPE_SECRET_KEY=            # blank = provider disabled
POSTAL_STRIPE_WEBHOOK_SECRET=
POSTAL_PAYSTACK_SECRET_KEY=          # blank = provider disabled
POSTAL_PAYSTACK_NGN_PER_USD=1600     # display/charge rate for NGN
POSTAL_BILLING_RETURN_URL=http://localhost:3000/wallet
```

## 6. Frontend (web)

- **/wallet** page (feature shell): balance card + "X publishing is the only
  paid feature" explainer, top-up form (credit presets, Stripe/Paystack choice,
  Paystack labelled for African cards), redirect out, `?status=success|cancelled`
  return handling, ledger history list.
- **Channels page:** X row gets a `Pay-per-use` pill + hint.
- **Schedule/compose:** `insufficient_credits` errors deep-link to /wallet.
- Sidebar: Wallet entry under Manage.

## 7. Verification (DoD)

- [ ] Unit tests: ledger idempotency, atomic deduct, webhook signature
      verification (both providers), pricing math.
- [ ] Integration: topup→webhook→balance; schedule gate; charge+refund via
      worker against the X simulator.
- [ ] `scripts/curl/billing.sh`: wallet read, topup (dev provider), signed
      Stripe + Paystack webhooks (constructed with test secrets), soft-gate
      rejection, charge on publish, idempotent webhook replay.
- [ ] `make check` green; web `check` + Playwright green; live e2e extended.
- [ ] Security: webhook signatures mandatory, raw-body HMAC, no amounts
      trusted from the client (credits→amount computed server-side), ledger
      append-only, capability gates (read vs manage_workspace).
