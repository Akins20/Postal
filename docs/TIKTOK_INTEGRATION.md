# TikTok Integration Spec (SECOND social target, after X)

> ⚠️ **Verify against live TikTok docs before coding.** TikTok's API, scopes, and the
> audit process change often. Confirm at build time on `developers.tiktok.com`. This doc
> captures verified 2026 research (multi-source, adversarially checked) + flagged gaps.
> Built to mirror the structure of `X_TWITTER_INTEGRATION.md`. TikTok slots in behind the
> existing `publish.Adapter` / `channel.OAuthProvider` contracts (Phase 4) — same engine.

---

## 1. Why TikTok is second
After X proves the publishing engine, TikTok is the next platform (user direction, 2026-05-29).
It exercises the engine differently from X: **video/photo-first** (no text-only posts),
**asynchronous publish with status polling**, **chunked or pull-from-URL media**, and a hard
**app-audit gate** for public posting. If the adapter abstraction handles both X and TikTok,
it generalizes well.

## 2. ⚠️ The critical gate — app audit (materially different from X)
- **Unaudited apps can ONLY post `SELF_ONLY` (private) content.** Public posting requires
  passing a **TikTok app audit** (Terms-of-Service compliance review; secondary sources
  estimate ~2–4 weeks). Public attempts by an unaudited client are rejected:
  `unaudited_client_can_only_post_to_private_accounts` (HTTP 403).
- Unaudited phase also caps usage: **≤5 users posting per 24h**, and the target account must be
  private at post time.
- **Implication for Postal:** the TikTok adapter must support a `SELF_ONLY` privacy level and
  surface the audit requirement to operators. Before audit, only private/self posting + limited
  testing is possible. Treat "public posting enabled" as a per-deployment capability (ties into
  the **social feature toggles** — operator declares whether their TikTok client is audited).

## 3. Free vs paid — ⚠️ UNCONFIRMED (open question)
Research did **not** surface a verified answer on whether the Content Posting API is free or has
paid tiers/quotas (unlike X's confirmed pay-per-use). **Confirm at build time.** Working
assumption: TikTok content posting is free (no per-call billing like X), but do not hardcode
that — design cost descriptors as data (consistent with the billing/feature-toggle model) so
TikTok can be "free" while X is metered.

## 4. Content Posting API — two flows (verified)
Publishing is **asynchronous**: `init` returns a `publish_id`; then poll status.
- **Direct Post** (publish straight to the user's profile):
  - Video: `POST /v2/post/publish/video/init/` — scope **`video.publish`**.
  - Photo: `POST /v2/post/publish/content/init/` with `post_mode=DIRECT_POST`.
- **Upload / Inbox** (send a draft to the user's TikTok inbox; they finalize in-app):
  - Video: `POST /v2/post/publish/inbox/video/init/` — scope **`video.upload`**.
  - Photo: `POST /v2/post/publish/content/init/` with `post_mode=MEDIA_UPLOAD`.
- **Status poll:** `POST /v2/post/publish/status/fetch/` with the `publish_id`. Stages:
  `PROCESSING_UPLOAD` / `PROCESSING_DOWNLOAD` / `SEND_TO_USER_INBOX` / `PUBLISH_COMPLETE`.
  `publish_id` formats encode source/type: `v_pub_file~…` (file upload), `v_pub_url~…`
  (pull/direct), `p_pub_url~…` (photos), `v_inbox_file~…` (inbox). "A time limit is not
  guaranteed" → must poll (fire-then-poll async job, like a worker task).

## 5. Media transfer (verified)
- **`FILE_UPLOAD`** — `init` returns an `upload_url`; client PUTs chunks with
  `Content-Range: bytes {first}-{last}/{total}`. Each chunk **5 MB–64 MB** (final chunk may
  exceed up to **128 MB**); **1–1000 chunks**; **max 4 GB**; files < 5 MB uploaded whole;
  `upload_url` valid ~1h. `source_info` carries `source`, `video_size`, `chunk_size`,
  `total_chunk_count`.
- **`PULL_FROM_URL`** — TikTok downloads from a URL on a **domain/URL-prefix you've verified**
  in the developer portal (`url_ownership_unverified` otherwise; 3xx redirects rejected).
- **Photos support `PULL_FROM_URL` ONLY** (no `FILE_UPLOAD` for photos — a refuted claim
  wrongly said otherwise). `photo_images` is an array of public, app-verified URLs.
- *(Postal's `MediaRef` model maps to `FILE_UPLOAD`; the cross-platform-sync "mirror" feature
  would lean on `PULL_FROM_URL` from re-hosted media.)*

## 6. OAuth (TikTok Login Kit v2) — verified
- **Authorize:** `https://www.tiktok.com/v2/auth/authorize/` (v1 deprecated). Params:
  `client_key`, `response_type=code`, `scope` (comma-separated), `redirect_uri`, `state`, and
  for desktop/mobile `code_challenge` + `code_challenge_method=S256`.
- **Token:** `POST https://open.tiktokapis.com/v2/oauth/token/` (form-encoded) for BOTH
  `authorization_code` and `refresh_token` grants. Returns `access_token`, `expires_in`,
  `open_id`, `refresh_token`, `refresh_expires_in`, `scope`, `token_type=Bearer`.
  Revoke at `/v2/oauth/revoke/`.
- **Scopes:** `video.publish` (direct post), `video.upload` (inbox), `user.info.basic`
  (identity), plus a read scope for listing existing videos (likely `video.list` — **unconfirmed**, see §9).
- **Token lifetimes:** access **24h** (`86400s`); refresh **365 days** (`31536000s`), rolls
  over on each refresh. Much longer-lived than typical — refresh well before 24h, persist the
  rolled refresh token. (Our channel refresh worker handles this; expiry/maxTTL differ from X.)
- **PKCE:** mandatory for desktop/mobile; S256. ⚠️ **GOTCHA:** TikTok docs show **hex**
  encoding of the SHA-256 challenge, deviating from RFC 7636's base64url — our existing PKCE
  helper uses base64url (for X). The TikTok adapter must implement TikTok's encoding; **do not
  reuse the X PKCE helper blindly.** Confirm at build.

## 7. Content constraints — ⚠️ mostly UNCONFIRMED (open question)
Research verified only the media size/chunk limits (§5). NOT verified: supported video
codecs/resolution/duration, caption character limit, photo-carousel min/max image count.
**Confirm before writing `Validate`.** `client_key`/uniqueness, privacy_level, and
`disable_comment/duet/stitch` flags exist on the init request — model them as platform_options.

## 8. Rate limits & errors — ⚠️ partial
Only an unverified mention of ~6 req/min/user for video init surfaced; no full rate-limit table
or complete error catalog. Known error codes: `unaudited_client_can_only_post_to_private_accounts`,
`url_ownership_unverified`, `scope_not_authorized`. **Confirm the full catalog + map to our
retry classes (Terminal/Retryable/AuthExpired)** at build. Simulator should mirror the async
status-poll + the audit/private restriction.

## 9. Reading existing content (for cross-platform sync) — ⚠️ UNCONFIRMED
Research did **not** confirm a Video List / read API or the `video.list` scope. This **blocks
the content-sync (import/mirror) feasibility** for TikTok — needs dedicated follow-up research
before committing to that feature (see memory `cross-platform-sync`). Likely exists (TikTok
Display API has a video list), but verify scope, fields, and history/restrictions.

## 10. Adapter mapping (how it fits Postal)
- Implements `publish.Adapter` + `channel.OAuthProvider` (same as X), with TikTok specifics:
  - `Publish` is **async**: init → (chunked upload or pull) → poll `status/fetch` until
    `PUBLISH_COMPLETE` (reuse the X media STATUS-poll pattern, generalized).
  - `Constraints`: video/photo (no text-only); privacy levels incl. `SELF_ONLY`; sizes per §5.
  - `Account` via `user.info.basic` (`open_id` as `platform_account_id`).
  - Separate base URLs: authorize `tiktok.com`, API `open.tiktokapis.com`.
  - Privacy + audit state surfaced; default `SELF_ONLY` until the operator's client is audited.
- A **TikTok simulator** (`internal/publish/simulator/tiktok`) mirrors init/upload/pull/status,
  the audit/private restriction, and error injection — same approach as the X simulator.

## 11. Research provenance (2026-05-29) & open questions
Verified via multi-source deep research (primary: developers.tiktok.com docs for content-posting,
media-transfer, login-kit, oauth token management). **High-confidence:** the two flows, media
transfer modes/limits, the audit/private gate, OAuth endpoints/scopes/lifetimes, PKCE-S256 +
hex-encoding caveat, async status polling.

**Open questions to resolve before building (do NOT assume):**
1. Free vs paid / quotas (§3).
2. Video List / read API + `video.list` scope for sync (§9).
3. Content constraints: codecs, resolution, duration, caption limit, carousel counts (§7).
4. Full per-endpoint rate limits + complete error-code catalog (§8).
5. Sandbox specifics for pre-audit testing.
6. 4 GB (official) vs 1 GB (a third-party blog) max file size — trust 4 GB; reconfirm.
