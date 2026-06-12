# Phase 14 - Instagram & TikTok adapters

> Status: IN PROGRESS (started 2026-06-12). Research snapshot below was
> verified against current docs this date; re-verify before production
> credentials are issued (these APIs move).

## 1. API facts that shape the design (researched 2026-06)

### Instagram (Meta Graph API)
- OAuth 2.0 via Meta: authorize on facebook.com dialog, token exchange on
  graph.facebook.com, then a long-lived (60-day, refreshable) token.
  Identity resolution is multi-hop: `/me/accounts` -> page ->
  `?fields=instagram_business_account` -> IG user id -> `?fields=username`.
- Only **Business/Creator accounts** linked to a Facebook Page can publish.
  Production needs Meta App Review (`instagram_basic`,
  `instagram_content_publish`); pre-review apps are limited to test users.
- Publishing is a **container flow**: `POST /{ig-user}/media` with a caption
  plus `image_url`/`video_url` that **must be a PUBLIC URL** (Meta fetches
  it), poll `/{container}?fields=status_code` until `FINISHED`, then
  `POST /{ig-user}/media_publish?creation_id=...`.
- **No text-only posts.** Caption <= 2200 chars. ~100 API posts per rolling
  24h per account. Reels: 9:16, 5-90s. Metrics via `/{media-id}/insights`.

### TikTok (Content Posting API)
- OAuth 2.0 on open.tiktokapis.com (`client_key` + secret; refresh
  supported). Scopes: `user.info.basic` + `video.publish` (and photo
  posting via the content endpoint).
- Direct post: query `/v2/post/publish/creator_info/query/` FIRST, then
  `/v2/post/publish/video/init/` with `source=FILE_UPLOAD` -> chunked PUT to
  the returned `upload_url` -> poll `/v2/post/publish/status/fetch/`.
  (`PULL_FROM_URL` exists but requires domain verification - we hold the
  bytes already, so FILE_UPLOAD avoids that entirely.)
- Photos via `/v2/post/publish/content/init/`. **No text-only posts.**
- **Unaudited API clients can only post PRIVATE videos**; the audit takes
  2-4 weeks. The UI must say so until the operator's app passes audit.
- Metrics via `/v2/video/query/` (`video.list` scope): views/likes/
  comments/shares.

## 2. Design decisions

- **Public media URLs for IG:** storage gains `PresignGet` (minio-go
  presigned GET; works for both MinIO and R2). The schedule media loader
  attaches a short-lived presigned URL to each `publish.MediaRef` (new `URL`
  field) alongside the bytes; IG consumes the URL, X/TikTok the bytes.
- **Constraints gains `RequiresMedia`** so compose-time validation rejects
  text-only posts for IG/TikTok before any API call, and the composer can
  explain it.
- Both platforms are **free** to publish to (their APIs cost nothing), so
  billing stays X-exclusive automatically (no cost map entries).
- **Simulators first** (CLAUDE.md section 3): `postal sim` now hosts three
  fakes - X :10090, Instagram :10091, TikTok :10092 - each mimicking its
  platform's real endpoints, auth shape, and error codes. Adapters point at
  them via `POSTAL_IG_API_BASE_URL` etc. in dev; blank = real hosts.
- Visibility honesty: TikTok adapter posts with `privacy_level` from
  creator_info; until the app is audited everything lands private, and the
  channel card shows a hint.

## 3. Sub-phases

- [x] 14.0 Plumbing: storage.PresignGet (minio presigned GET), MediaRef.URL
      attached by the schedule media loader (2h TTL), Constraints.RequiresMedia,
      config structs + env, per-platform adapter registration (each platform
      registers only when its credentials are set). DONE 2026-06-12.
- [x] 14.1 Instagram adapter (OAuth long-lived token re-exchange as refresh,
      multi-hop identity, container flow with retryable IN_PROGRESS budget,
      insights metrics) + simulator + 7 tests. DONE 2026-06-12.
- [x] 14.2 TikTok adapter (PKCE OAuth, creator_info-first, FILE_UPLOAD video,
      PULL_FROM_URL photos, status polling, video.query metrics, SELF_ONLY
      tolerated for unaudited apps) + simulator + 8 tests. DONE 2026-06-12.
- [x] 14.3 Frontend: Instagram/TikTok registry entries (glyphs, 2200-char
      caption limits, requiresMedia composer gate with explanatory note,
      audit/business-account caveats on the channels page). Platform-authentic
      IG/TikTok preview cards remain a follow-up (X has one). DONE 2026-06-12.
- [x] 14.4 Live e2e: `postal sim` hosts X/IG/TikTok fakes (:10090-92);
      scripts/curl/platforms.sh drives connect -> upload -> compose
      (text-only IG rejected) -> dual-platform publish-now -> worker
      publishes both. 9/9 on the running stack. DONE 2026-06-12.

### Remaining before production
- Operator app registrations + reviews: Meta App Review and TikTok audit.
- Platform-authentic composer preview cards for IG/TikTok.
- Carousels (IG) beyond single image/video.

## 4. Config (env)

```
POSTAL_IG_CLIENT_ID=            # Meta app id (blank = IG disabled)
POSTAL_IG_CLIENT_SECRET=
POSTAL_IG_REDIRECT_URI=http://localhost:3000/oauth/callback
POSTAL_IG_API_BASE_URL=         # override -> simulator (dev)
POSTAL_IG_AUTH_BASE_URL=
POSTAL_TIKTOK_CLIENT_KEY=       # blank = TikTok disabled
POSTAL_TIKTOK_CLIENT_SECRET=
POSTAL_TIKTOK_REDIRECT_URI=http://localhost:3000/oauth/callback
POSTAL_TIKTOK_API_BASE_URL=
POSTAL_TIKTOK_AUTH_BASE_URL=
```
