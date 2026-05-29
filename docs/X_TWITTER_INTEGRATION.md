# X / Twitter Integration Spec (FIRST social target)

> ⚠️ **Verify everything in this doc against the live X API docs before coding.** X's API
> endpoints, OAuth flow, rate limits, and pricing change frequently. Load the `deep-research`
> skill or use WebSearch/WebFetch on `developer.x.com` / `docs.x.com` to confirm current
> specifics at build time (Phase 4). This document captures *intent and structure*; the exact
> numbers must be confirmed live.

---

## 1. Why X is first
The user chose X/Twitter as the first integration. It is also the **hardest and most
constrained** (paid tiers, tight write limits, strict rules), so building it first forces the
publishing pipeline to be robust from the start. If the engine handles X cleanly, easier
platforms (Bluesky, Mastodon, Meta) slot in with minimal changes.

## 2. Access & cost reality — ⚠️ UPDATED (research verified 2026-05-29)

**Major change vs the old "Free/Basic/Pro tiers" model:** As of **Feb 6, 2026**, X
**discontinued the free tier and the fixed monthly subscription tiers for new developers** and
moved to a **default pay-per-use (metered) model — no subscriptions, pay only per request.**
- **Posting is PAID and metered**, not free. Creating a post is billed per request: launch rate
  $0.010, **raised to ~$0.015/post as of ~Apr 20, 2026, and ~$0.20 if the post contains a URL.**
  Reading a post ≈$0.005 (owned reads dropped to ≈$0.001); user-profile read ≈$0.010.
  *(Dollar figures are volatile — re-verify `docs.x.com/x-api/getting-started/pricing` at build
  time. Confidence: pay-per-use direction HIGH; exact prices MEDIUM.)*
- There is **no monthly write cap to design around** anymore — it is metered per request. Do NOT
  hardcode monthly caps. Existing free-tier devs were migrated with a one-time $10 voucher;
  whether legacy Basic/Pro subscriptions still work for grandfathered accounts is unclear.
- **Implication for Postal:** the app stays free to end users, but **every publish costs the
  operator real money per request.** This makes anti-abuse / quotas / spend-control first-class:
  abuse = direct $ cost. Treat publish + every read (incl. analytics polling) as billable. The
  simulator should model per-request cost hooks and the runtime rate-limit headers, and the paid
  surface must sit behind the **social feature toggles** (see MASTER_PLAN §6) so an operator can
  cap or disable costly features.
- **Open question (confirm at build):** exact billing mechanics — prepaid balance vs billing
  account, per-request cost in responses, spend caps, and the error/status when balance is
  exhausted.

## 3. OAuth 2.0 (Authorization Code with PKCE) — verified 2026-05-29
- **Authorize endpoint:** `https://x.com/i/oauth2/authorize` (params: `response_type=code`,
  `client_id`, `redirect_uri`, `state`, `code_challenge`, `code_challenge_method=S256`).
- **Token endpoint:** `https://api.x.com/2/oauth2/token` (initial exchange AND refresh).
- **Scopes:** `tweet.read tweet.write users.read media.write offline.access`
  (`media.write` is required for uploads; `offline.access` is required to receive a refresh
  token — without it none is issued).
- **Token lifetimes:** access token valid **2 hours**; refresh token issued only with
  `offline.access`, rotates on each use (community-reported ~6-month life). Adapter must refresh
  **proactively before the 2h expiry** and persist the rotated refresh token.
- Flow:
  1. `AuthURL(state, codeChallenge)` → X authorize URL with PKCE `code_challenge` (S256),
     `state` (CSRF), scopes.
  2. User authorizes → callback with `code` + `state` (validate state).
  3. `ExchangeCode(code, codeVerifier)` → access token, refresh token, expiry.
  4. Store **encrypted** in `channel_credential`. Never log tokens.
  5. `RefreshToken(refresh)` before expiry (worker job). `offline.access` required for refresh.
- On `401`/invalid_grant: mark channel `expired`, notify user to reconnect.

## 4. Publishing — verified 2026-05-29
- **Create post:** `POST https://api.x.com/2/tweets`. Success = **HTTP 201** with
  `{"data":{"id":"<id>","text":"<text>"}}` (note 201, not 200 — simulator must return 201).
- **Request body fields:** `text` (string); `reply.in_reply_to_tweet_id`; `media.media_ids`
  (array, 1–4); `poll.options` (2–4) + `poll.duration_minutes` (5–10080); `quote_tweet_id`
  (may require elevated/Enterprise — gate behind a feature flag); `reply_settings`
  (`following`/`mentionedUsers`/`subscribers`/`verified`). Other optional fields exist.
- **Constraints to enforce in `Validate`:**
  - Text length: 280-char *weighted* count (URLs count as fixed length ~23; CJK chars count
    double). Implement weighted counting, not raw `len`. ⚠️ **NOT freshly verified in research
    (weakest-covered area)** — re-confirm current weighted rules + premium/verified long-post
    limits against `twitter-text` / docs.x.com before finalizing validation.
  - Media (verified): **up to 4 photos, OR 1 animated GIF, OR 1 video** (mutually exclusive).
    Sizes: images ≤ **5 MB**, GIFs ≤ **15 MB** (a forum report notes non-chunked GIF may be
    enforced at 5 MB), videos ≤ **512 MB**. Premium/verified accounts exceed these.
  - Duplicate content rejection: X rejects identical recent tweets → treat as terminal error.
- **Media upload (verified):** `POST https://api.x.com/2/media/upload` (v2 — v1.1 sunset ~Jun 9,
  2025; build v2 only). Four-command chunked sequence: **INIT** (→ `media_id`) → **APPEND** (each
  chunk) → **FINALIZE** → **STATUS** (`GET ?command=STATUS&media_id=…`, only when FINALIZE
  returns `processing_info` for video/GIF; states `pending`→`in_progress`→`succeeded`/`failed`).
  Requires `media.write`. Images upload synchronously; video/GIF need STATUS polling.
- **Threads** (optional later): reply chains via `reply.in_reply_to_tweet_id`.

## 5. Rate limits & error handling — verified 2026-05-29
- Read the three response headers: `x-rate-limit-limit`, `x-rate-limit-remaining`,
  `x-rate-limit-reset` (Unix ts) and back off proactively (don't just react to 429). Headers may
  occasionally be absent on some v2 endpoints. **Do NOT hardcode per-endpoint posting limits**
  (the commonly-cited 100/15min + 10k/24h figures were *refuted* in research) — rely on the
  runtime headers. For the simulator, pick representative numbers but label them simulator-config.
- On exceed: **HTTP 429** with body `{"errors":[{"code":88,"message":"Rate limit exceeded"}]}`
  (code 88 is legacy but still documented).
- Error mapping:
  | HTTP / condition | Class | Action |
  |---|---|---|
  | 429 rate limited | retryable | backoff until reset header, then retry (respect attempt cap) |
  | 5xx | retryable | exponential backoff + jitter |
  | network timeout | retryable | backoff |
  | 401 invalid/expired token | terminal-for-now | refresh once; if still 401 → mark channel expired, notify |
  | 403 (suspended/duplicate/forbidden content) | terminal | fail post, surface reason, audit |
  | 400 invalid request/content | terminal | fail fast, show validation error |
- **Idempotency:** never double-post on retry. Use a per-job dedupe key; before publishing,
  check whether a `publish_result` already exists for that job.

## 6. Analytics — verified 2026-05-29
- `FetchMetrics`: `GET https://api.x.com/2/tweets/{id}?tweet.fields=public_metrics` (app-only
  Bearer token works). `public_metrics` = `like_count`, `retweet_count`, `quote_count`,
  `reply_count`, `impression_count`, `bookmark_count`.
- `non_public_metrics` / `organic_metrics` / `promoted_metrics` need **user-context** auth, are
  **owned posts only**, and only for posts **< 30 days** old → gate behind a feature flag.
- **Cost note:** under pay-per-use, **every metrics read is billable** (~$0.001 owned). Poll
  conservatively and make analytics polling a toggleable feature. Store as time series in
  `analytics_metric`.

## 7. The X Simulator (`internal/publish/simulator/twitter`)
A local HTTP server that stands in for X so tests never touch the paid/real API.

**Must faithfully mimic:**
- **Auth endpoints:** authorize redirect shape, token exchange (returns access+refresh+expiry),
  refresh, and `invalid_grant`/401 cases.
- **Create-tweet endpoint:** accepts `{text, media_ids}`, returns realistic `{data:{id, ...}}`.
- **Media upload:** INIT/APPEND/FINALIZE/STATUS sequence returning `media_id`s.
- **Metrics endpoint:** returns `public_metrics`.
- **Error injection (controllable via test setup):**
  - 280-char weighted limit → 400/422 with X-shaped error body
  - duplicate content → 403 duplicate
  - rate limit → 429 with `x-rate-limit-remaining: 0` and a `x-rate-limit-reset` timestamp
  - expired/invalid token → 401
  - server error → 500/503
  - network timeout / slow response
  - oversized/unsupported media → rejection at upload
- **Configurable rate budget** mirroring small real limits, so tests prove quota handling.

The adapter takes an **injectable base URL**; tests point it at the simulator. A handful of
**live smoke tests** may exist but are gated behind `POSTAL_LIVE_X=1` and excluded from default runs.

## 8. Phase-4 test matrix (all via simulator, all must pass)
1. Happy path: valid text-only tweet → **201** → `publish_result` recorded with platform id.
2. Text with 1–4 images: media upload sequence → create with media_ids → success.
3. Over-limit text (weighted) → rejected at `Validate` (no API call) AND at simulated API (defense in depth).
4. Unsupported / oversized media → rejected.
5. Expired access token → auto-refresh → retry → success.
6. Refresh fails (invalid_grant) → channel marked expired, user notified, no crash.
7. 429 rate limit → backoff to reset → retry → success (and attempt cap respected).
8. 5xx → exponential backoff retry → eventual success or terminal after cap.
9. Network timeout → retry path.
10. Duplicate content → terminal failure, clear reason, audited.
11. Idempotency: simulate retry after a success → NO second tweet created.
12. Concurrent publishes respect rate budget (no overshoot).

## 9. Security notes specific to X
- App Client Secret + user tokens are high-value: encrypted at rest, never logged, never
  returned to clients.
- Protect the shared app credentials — abusive users could get the whole app rate-limited or
  banned. Per-user/per-channel throttling protects the shared key (anti-abuse requirement).
- Validate `state` on OAuth callback (CSRF). Use PKCE S256.

## 10. References to confirm at build time
- X API v2 docs (developer.x.com / docs.x.com): tweets create, media upload, OAuth 2.0 PKCE,
  rate limits, pricing tiers. **Always confirm current — do not trust this doc's numbers blindly.**

## 11. Research provenance (2026-05-29) & open questions
Verified via multi-source deep research (21 sources, 25 claims adversarially verified, 22
confirmed / 3 refuted). Primary sources (docs.x.com): `/x-api/getting-started/pricing`,
`/fundamentals/authentication/oauth-2-0/authorization-code`, `/x-api/posts/create-post`,
`/x-api/media/quickstart/{best-practices,media-upload-chunked}`, `/x-api/fundamentals/{rate-limits,metrics}`.
Pricing corroborated by @XDevelopers announcement + press (medianama, TechCrunch).

**Refuted (do NOT use):** fixed monthly tiers with "no usage caps"; "$0.01/post + 2M reads/mo
cap"; hardcoded POST /2/tweets limit of 100/15min + 10k/24h.

**Open questions — re-verify before/at build:**
1. **Weighted character counting** (weakest-covered): confirm 280 standard limit, URL=23-char
   fixed weight, CJK double-weight, and premium/verified long-post limits (check `twitter-text`).
2. **Pay-per-use billing mechanics:** prepaid balance vs billing account; per-request cost in
   responses; spend caps; the error/status when balance is exhausted.
3. **Grandfathered access:** do legacy Basic/Pro/Enterprise subscriptions still work; does
   `quote_tweet_id` / Enterprise-gated functionality need a specific plan now.
4. **Real posting throughput limits** under pay-per-use (beyond per-request billing).
