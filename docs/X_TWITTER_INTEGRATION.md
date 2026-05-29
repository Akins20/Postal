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

## 2. Access & cost reality
- X API v2 has tiered access: a very limited **Free** tier (tiny number of writes/month),
  **Basic** (~$100/mo, modest limits), and higher paid tiers. Confirm current tiers/limits at
  build time.
- **Implication:** Postal is free to end users, but the operator pays X. The system must be
  extremely conservative with quota — treat every write as expensive. The simulator must mirror
  the real (small) rate budgets so we never design something that only works at unrealistic scale.

## 3. OAuth 2.0 (Authorization Code with PKCE)
- App registered in X developer portal → Client ID/Secret, redirect URI, scopes.
- Scopes needed (confirm): `tweet.read`, `tweet.write`, `users.read`, `offline.access`
  (for refresh token), and media scopes as required for uploads.
- Flow:
  1. `AuthURL(state, codeChallenge)` → X authorize URL with PKCE `code_challenge` (S256),
     `state` (CSRF), scopes.
  2. User authorizes → callback with `code` + `state` (validate state).
  3. `ExchangeCode(code, codeVerifier)` → access token, refresh token, expiry.
  4. Store **encrypted** in `channel_credential`. Never log tokens.
  5. `RefreshToken(refresh)` before expiry (worker job). `offline.access` required for refresh.
- On `401`/invalid_grant: mark channel `expired`, notify user to reconnect.

## 4. Publishing
- **Create post:** POST to the v2 tweets endpoint (confirm path) with JSON body `{ "text": ... }`
  and optional `media.media_ids`.
- **Constraints to enforce in `Validate` (confirm current values):**
  - Text length: 280-char *weighted* count (URLs count as fixed length ~23; CJK chars count
    double). Implement weighted counting, not raw `len`.
  - Media: up to 4 images, OR 1 GIF, OR 1 video per post. Image/video format, size, and duration
    limits per X spec.
  - Duplicate content rejection: X rejects identical recent tweets → treat as terminal error.
- **Media upload** is a separate, multi-step (INIT → APPEND chunks → FINALIZE → STATUS for
  video) flow on the media endpoint, returning `media_id`s used in the create-post call. Adapter
  must implement chunked upload; simulator must mimic it.
- **Threads** (optional later): reply chains via `reply.in_reply_to_tweet_id`.

## 5. Rate limits & error handling
- Respect per-endpoint rate limits; read `x-rate-limit-remaining` / `x-rate-limit-reset`
  headers and back off proactively (don't just react to 429).
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

## 6. Analytics
- `FetchMetrics` pulls public metrics (likes, reposts, replies, impressions where available)
  for a `platform_post_id` via the tweet lookup endpoint with `tweet.fields=public_metrics`.
- Poll conservatively (rate budget!). Store as time series in `analytics_metric`.

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
1. Happy path: valid text-only tweet → 200 → `publish_result` recorded with platform id.
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
