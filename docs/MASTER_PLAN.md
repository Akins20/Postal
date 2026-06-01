# Postal — Master Plan

> **Postal** is a free, no-paywall social media scheduling and publishing platform — a Buffer
> alternative. This document is the source of truth for *what* to build and *in what order*.
> For *how* Claude works (testing rules, security mandate, stack), read [`../CLAUDE.md`](../CLAUDE.md) first.
> For *how code is written* (≤800 lines/file, naming, godoc, error handling — all enforced),
> read [`CODING_STANDARDS.md`](CODING_STANDARDS.md).

---

## 1. Vision & principles

- **Free forever, no paywall.** No feature gating behind payment. (See §2 for the unavoidable
  caveat: the *platforms* charge for API access — that cost is external, not something Postal
  charges users for.)
- **Backend-first monolith.** A single Go binary, internally modular by domain, deployable
  cheaply on a small VPS. Designed to be split later only if ever necessary.
- **API for web AND mobile.** The backend is a clean, versioned JSON API. No HTML rendering
  coupling. Both a future web SPA and mobile apps consume the same endpoints.
- **Secure and abuse-resistant by default.** Free apps attract abuse; security and anti-abuse
  are first-class, built with each feature.
- **Everything tested locally before it's "done."** Real server, real curl/script tests, real
  output. Socials tested via faithful simulators.

---

## 2. Hard external constraints (must design around these)

| Platform | API reality | Implication for Postal |
|----------|-------------|------------------------|
| **X / Twitter** *(first target)* | API v2. OAuth 2.0 with PKCE. **Tiered paid access** (Free tier exists but heavily write-limited; Basic ~$100/mo; Pro much higher). Strict rate limits. | The app is free to users, but whoever runs Postal pays X. Build for tight rate limits and quota awareness from day one. Free X tier allows very few posts/day per app — design the simulator to mirror these limits. |
| Instagram / Facebook | Meta Graph API; Business/Creator only; requires Meta App Review. | Phase later; start App Review early. |
| LinkedIn | Requires Marketing Developer Platform partner approval. | Phase later. |
| TikTok *(2nd target, after X)* | Content Posting API; app audit required for public posts. | **Next integration after X.** Research in progress → `docs/TIKTOK_INTEGRATION.md`. |
| **Bluesky / Mastodon** | Open, free, no approval. | Ideal *secondary* targets to prove multi-platform abstraction cheaply. |

**Design rule:** the publishing layer must treat every platform's limits (char count, media
specs, rate limits, duplicate rejection, token expiry) as data-driven config, so adding a
platform is "implement the adapter + declare its constraints," not rewrite the engine.

**Feature toggles (planned, later phase):** Cost/availability varies by platform and API tier
(e.g. X is pay-per-use as of 2026). Postal must let the **operator (system-level)** and
**workspace admins (workspace-level)** independently enable/disable individual social features
(publish, media, analytics, …). Effective availability = system-enabled AND workspace-enabled
AND user-has-capability. Build adapter features as data-driven descriptors so a feature can be
turned off without code changes; dedicated feature-flag slice scheduled ~Phase 8/9.

**Billing & wallet (planned, future phase — reconciles pay-per-use with "no-paywall"):**
X bills via a **prepaid credit** system (operator buys credits in X's console; X deducts per
call; auto-recharge + spend cap). Real money reaches X only via the **operator's** prepaid
balance — Postal never pays X per request. **Decided model (2026-05-29):** the **operator
pre-funds X**, and Postal keeps an **internal per-workspace ledger that tracks usage and
enforces spend caps** so no workspace can burn the operator's X credits. No end-user payments /
Stripe for now (ledger built so pass-through top-ups could be added later). This does NOT
paywall any Postal feature — features stay free; the ledger only meters external API spend.
The publish pipeline (and analytics poller) debit the ledger per billable action via
per-platform/per-action cost descriptors, idempotently (reuse the publish idempotency key).
Ties into feature toggles (disable costly features when capped). Rates are config/data
(re-verify vs docs.x.com). Its own phase (~before backend freeze); Phase 4 `publish.Pipeline`
is the debit hook. See memory `billing-wallet-system`.

**Cross-platform publish & sync (planned):** (1) **compose-once, multi-publish** — one post
fanned out to many channels (already modeled via per-channel `post_variant`s; natural part of
Phases 5–6). (2) **import/mirror existing posts** across platforms (full/batch/single) — its own
later feature: needs per-adapter read/list methods, media re-hosting, a **content-type
compatibility matrix** (text↔video do NOT map — a tweet can't become a YouTube video), a
sync-mapping/dedupe table, and cost estimate+cap (reads are billed on X). See memory
`cross-platform-sync`.

---

## 3. Architecture overview

A **modular monolith**: one Go binary (`cmd/postal`), internally divided into domain packages
under `internal/` (see layout in `CLAUDE.md` §2). Two runtime roles in one binary, selected by
flag/subcommand:

- **API server** — serves `/api/v1/...` to clients.
- **Worker** — runs `asynq` workers that execute scheduled publish jobs, analytics polls, token
  refreshes, and abuse sweeps.

Run both as separate processes from the same image in production; run together locally.

```
            ┌─────────────┐     enqueue      ┌──────────┐
 clients ──►│ API server  │ ───────────────► │  Redis   │ ◄── rate-limit counters
 (web/app)  │  (chi)      │                  │ (asynq)  │
            └─────┬───────┘                  └────┬─────┘
                  │                               │ dequeue
                  ▼                               ▼
            ┌─────────────┐                 ┌──────────┐    ┌──────────────┐
            │ PostgreSQL  │ ◄────────────── │ Worker   │ ──►│ Social APIs  │
            │ (pgx/sqlc)  │                 │ (asynq)  │    │ (X first) /  │
            └─────────────┘                 └──────────┘    │ Simulator    │
                  ▲                                          └──────────────┘
                  │ media metadata
            ┌─────────────┐
            │ Obj storage │  (S3/R2/MinIO)
            └─────────────┘
```

---

## 4. Cross-cutting requirements (apply to every phase)

### Security (mandatory; full checklist created in Phase 1 → `docs/SECURITY.md`)
- All secrets via env/KMS; nothing in repo.
- Passwords: Argon2id (or bcrypt) hashing. Sessions: short-lived JWT access + rotating refresh, or opaque server-side sessions in Redis.
- **Social tokens encrypted at rest** with AES-256-GCM envelope encryption; never logged.
- Strict input validation + output encoding; parameterized queries only (sqlc enforces).
- HTTPS only; secure cookies; CSRF protection for cookie-based auth; CORS locked to known origins.
- Authorization checks on every resource (workspace isolation — a user can never touch another workspace's data).
- Audit log for sensitive actions (connect/disconnect channel, publish, delete, role change).
- Security headers (HSTS, X-Content-Type-Options, etc.). Dependency scanning.

### Anti-abuse (mandatory; checklist in §13)
- Global + per-user + per-IP + per-endpoint rate limiting (token-bucket in Redis).
- Quotas on posts scheduled, channels connected, media storage per account.
- Email verification before publishing is enabled.
- Bot/signup abuse defense (captcha hook, disposable-email blocking, signup velocity limits).
- Content safety hooks (block known-malicious URLs; size/type limits on media).
- Per-channel respect for platform rate limits (never get the shared app keys banned).

### Testing (see `CLAUDE.md` §3)
- Unit + integration + run-the-server curl/script tests + social simulators. Failure paths always.

### Observability
- Structured `slog` logs with request IDs; Prometheus metrics; health/readiness endpoints; Sentry-style error capture (optional self-host).

---

## 5. Core data model (initial; refined per phase)

Entities (PostgreSQL):
- **user** — id, email, password_hash, email_verified, status, created_at
- **workspace** — id, name, owner_user_id, plan(always "free"), created_at
- **workspace_member** — workspace_id, user_id, role(owner/admin/editor/viewer), permissions(text[] of capability flags) *(see §5.1 — role is a named preset; `permissions` is the authoritative grant and may diverge from the preset for fine-grained per-user access)*
- **channel** — id, workspace_id, platform(twitter/...), platform_account_id, handle, display_name, status(active/expired/revoked), connected_by, created_at
- **channel_credential** — channel_id, encrypted_access_token, encrypted_refresh_token, scopes, expires_at, key_version *(separate table, tightly access-controlled)*
- **post** — id, workspace_id, author_user_id, status(draft/scheduled/publishing/published/failed/canceled), created_at
- **post_variant** — post_id, channel_id, body, media_refs[], platform_options(jsonb) *(per-channel customization)*
- **schedule_slot** — channel_id, day_of_week, time_of_day, timezone *(queue definition)*
- **scheduled_job** — post_id, channel_id, run_at, status, attempts, last_error, asynq_task_id
- **publish_result** — post_id, channel_id, platform_post_id, published_at, raw_response(jsonb)
- **media_asset** — id, workspace_id, kind(image/video/gif), storage_key, mime, width, height, duration, bytes, status
- **analytics_metric** — channel_id, platform_post_id, metric, value, captured_at
- **audit_log** — id, workspace_id, actor_user_id, action, target, metadata(jsonb), ip, created_at
- **rate_counter** — (mostly in Redis; Postgres for durable quotas)

### 5.1 Authorization model — capability flags + role presets

Postal uses **capability-based authorization**, not a fixed role hierarchy. This lets an
admin grant a user *any combination* of abilities (e.g. "read + upload only", "read + delete",
"read-only") rather than forcing them into one of four buckets.

**Capabilities** (the authoritative permission registry; extend as features land):

| Capability | Grants the ability to |
|------------|------------------------|
| `read` | view workspace posts, channels, schedules, analytics |
| `create` | create posts / drafts |
| `update` | edit existing posts / drafts |
| `delete` | delete posts / drafts / media |
| `upload` | upload media assets |
| `publish` | schedule & publish posts to channels |
| `manage_channels` | connect / disconnect social channels (high-value — touches token vault) |
| `manage_members` | invite/remove members, change their capabilities |
| `manage_workspace` | rename/delete workspace, billing (owner-level) |

**Roles are named presets** over these capabilities — a convenience default, not a hard
boundary. On assignment a role expands to its capability set, written to
`workspace_member.permissions`; an admin may then add/remove individual capabilities for that
user. `permissions` is always the source of truth at authz time.

| Role (preset) | Capability set |
|---------------|----------------|
| `viewer` | `{read}` |
| `editor` | `{read, create, update, upload, publish}` |
| `admin` | editor + `{delete, manage_channels, manage_members}` |
| `owner` | all capabilities incl. `manage_workspace` |

**Enforcement:** authorization middleware checks **capabilities, not roles** —
`RequireCapability(cap)` (e.g. `RequireCapability("delete")`) gates each handler, layered after
`RequireUser` and workspace-membership resolution. Workspace isolation is still absolute: a
user with `delete` in workspace A has zero capabilities in workspace B. Invariants: only
`manage_members` holders can alter another member's capabilities; no member may grant a
capability they don't themselves hold (no privilege escalation); the workspace owner cannot be
stripped of `manage_workspace`. Capability changes are written to the `audit_log`.

---

## 6. The publishing pipeline (the core abstraction — get this right)

Define once, reuse for every platform.

```go
// PlatformAdapter is implemented per social network. Every method must be testable
// against the simulator (base URL is injectable).
type PlatformAdapter interface {
    Platform() string
    Constraints() PlatformConstraints // char limits, media specs, rate limits, dup rules

    // OAuth
    AuthURL(state, codeChallenge string) string
    ExchangeCode(ctx, code, codeVerifier string) (*Token, error)
    RefreshToken(ctx, refresh string) (*Token, error)

    // Publishing
    Validate(ctx, variant PostVariant) error            // pre-flight against Constraints
    Publish(ctx, cred Token, variant PostVariant) (*PublishResult, error)

    // Analytics
    FetchMetrics(ctx, cred Token, platformPostID string) ([]Metric, error)
}
```

Lifecycle of a scheduled post:
1. Validate against adapter constraints at **compose time** (fast feedback) and again at publish time.
2. Enqueue `scheduled_job` in asynq with `run_at`.
3. Worker dequeues → loads channel credential (decrypt) → refresh token if near expiry →
   check platform rate budget → `adapter.Publish` → record `publish_result` or retry/backoff →
   on permanent failure, mark failed + notify user + audit.
4. Later, analytics poller fetches metrics for published posts.

Retry policy: exponential backoff with jitter; cap attempts; distinguish retryable (429, 5xx,
network) from terminal (401 invalid token → mark channel expired & notify; 400 invalid content
→ fail fast). Idempotency: never double-publish (dedupe key per job; check before send).

---

## 7. Phased roadmap (build in this order)

Legend: `[ ]` todo · `[~]` in progress · `[x]` done. Update as you go. Each phase ends with the
full Definition of Done (`CLAUDE.md` §4).

### Phase 0 — Scaffolding & tooling ✅ DONE (2026-05-29)
**Goal:** a runnable empty monolith with all dev infrastructure.
- [x] `go mod init` (`github.com/Akins20/postal`), repo layout per `CLAUDE.md` §2
- [x] `docker-compose.yml` for Postgres + Redis (+ MinIO for later); host ports overridable
- [x] `Makefile`: `run`, `test`, `check`, `migrate`, `lint`, `sqlc`, `up`/`down` deps
- [x] Config loader (typed, env-based, stdlib-only) + `.env.example`
- [x] `cmd/postal` entry with `serve` and `worker` subcommands (graceful shutdown)
- [x] `chi` server with `/healthz`, `/readyz`, request-ID + slog logging middleware
- [x] sqlc + goose wired; one trivial migration (`00001_init.sql`) proves the chain
- [x] **`.golangci.yml`** + **`scripts/dev/check.sh`** enforcing `docs/CODING_STANDARDS.md`:
      gofmt/goimports, golangci-lint, **≤800-line/file check**, `go vet`, `go test -race`
**Tests/DoD:** ✅ `make run` serves `/healthz` 200 & `/readyz` 200 (real PG+Redis pings); deps
come up; `scripts/curl/health.sh` passes; `make check` runs green (full enforcement chain).

### Phase 1 — Foundation primitives ✅ DONE (2026-05-29)
**Goal:** shared building blocks every domain reuses.
- [x] Standard JSON response envelope + standard error type/codes (`platform/web` + `platform/apperr`)
- [x] Central error handling (`web.Handler`/`web.Fail` maps `apperr.Kind` → HTTP + safe messages; internal causes never leak)
- [x] Input validation helpers (`web.DecodeJSON`: bounded body, strict, content-type checked)
- [x] Crypto module: AES-256-GCM **envelope encryption** (per-secret DEK wrapped by versioned KEK) + key versioning/rotation (`internal/security`)
- [x] Audit log writer (`security.Auditor` + `audit_log` table, migration `00002`)
- [x] Rate-limit primitive (Redis token bucket, atomic Lua, clock-injected) + reusable middleware (`internal/ratelimit`)
- [x] `docs/SECURITY.md` + `docs/ANTI_ABUSE.md` checklists created
- [x] Metrics endpoint (`/metrics`) + base Prometheus counters + middleware (`platform/metrics`)
**Tests/DoD:** ✅ crypto unit tests (round-trip, tamper detection, key rotation, wrong key); rate-limit math + Redis-backed integration tests; `make check` green; curl test (`scripts/curl/ratelimit.sh`) proves 429 past threshold with standard envelope + `Retry-After`; `/metrics` exposed.

### Phase 2 — Auth, users, workspaces, roles ✅ DONE (2026-05-29)
**Goal:** identity and tenancy.
- [x] Signup (email + password, Argon2id), email verification flow (token issue/verify; console-sink mailer locally — guarded off in production)
- [x] Login, logout, refresh; **session strategy:** short-lived JWT access (HS256, 15m) + rotating refresh token in Redis with **sliding expiration** + absolute cap; **cookie delivery** (httpOnly+Secure+SameSite refresh cookie auth-path-scoped, access cookie + body token) and **CSRF double-submit** on refresh/logout; `Authorization: Bearer` also accepted. Logout revokes refresh + clears cookies
- [x] Password reset (request/confirm; single-use hashed tokens; no account enumeration)
- [x] Auth middleware (`RequireUser`) + current-user context (`web.UserID`)
- [x] Workspaces: list; **personal workspace auto-created on signup** (transactional with user + owner membership)
- [x] Membership + **capability-based permissions** (presets expand to capability sets; per-user capability toggling — §5.1). Member add is direct-add-existing-user-by-email (full email-invite-for-new-users deferred to a later phase)
- [x] Capability registry + preset→capability expansion; update-capabilities + add-member endpoints (guarded by `manage_members`, no privilege escalation, owner immutable)
- [x] **Workspace authorization** (`RequireUser` → `RequireCapability(cap)` resolving membership) — capability checks + workspace isolation enforced
- [x] Anti-abuse: signup velocity (per-IP), disposable-email block, password strength, login throttling (per-IP + per-email); timing-equalized login (enumeration defense)
**Tests/DoD:** ✅ `make check` green (unit + Redis/PG integration incl. full auth flow). Curl suites: `scripts/curl/auth.sh` (13/13: signup→verify→login→/me→refresh→logout + negatives: dup email 409, wrong pw 401, weak pw 400, disposable 400, CSRF-less refresh 403, post-logout refresh 401) and `scripts/curl/capabilities.sh` (12/12: capability gating, privilege-escalation blocked 403, owner immutable 403, workspace isolation 403). Argon2id hashes + hashed tokens verified at rest; audit log populated. **`/security-review` run — no must-fix findings; two sub-threshold items (login timing, prod mailer token logging) fixed.**

### Phase 3 — Channels & OAuth token vault ✅ DONE (2026-05-29)
**Goal:** connect social accounts securely (generic, X wired in Phase 4).
- [x] Channel CRUD (list/disconnect; status tracking active/expired) — `internal/channel`
- [x] OAuth connect flow (CSRF `state` + PKCE S256, Redis-backed single-use state, callback handler) — generic over `OAuthProvider`; `Registry` (empty until Phase 4)
- [x] `channel_credentials` storage: envelope-encrypted access+refresh tokens (BYTEA), scopes, expiry, key_version (migration `00004`)
- [x] Token refresh service (`RefreshChannel` + `DueForRefresh`; re-encrypts, marks channel expired on failure) — worker scheduling deferred to Phase 6
- [x] Disconnect = best-effort provider revoke + purge credential + delete channel + audit
- [x] Authorization: `manage_channels` capability for connect/disconnect, `read` for list; callback re-checks capability + binds to initiating user; tokens never returned or logged
**Tests/DoD:** ✅ Go integration test (fake provider) proves full OAuth round trip + **ciphertext at rest** (asserts plaintext absent) + refresh rotation + disconnect-purge + foreign-user rejection. `scripts/curl/channels.sh` 9/9 (capability gating, isolation, validation). `make check` green. **`/security-review` run — all 8 areas clean, no findings.**

### Phase 4 — Publishing pipeline + **X/Twitter adapter** + simulator ⭐ FIRST SOCIAL ✅ DONE (2026-05-29)
**Goal:** end-to-end publish to X (via simulator), proving the whole core.
- [x] `PlatformAdapter` interface + `Constraints` + retry-class error model (`internal/publish/adapter.go`); embeds the Phase 3 `OAuthProvider`
- [x] **X/Twitter adapter** (`internal/publish/twitter`): OAuth2 PKCE auth/exchange/refresh; `Validate` (weighted 280 count via twitter-text ranges, media exclusivity/size); `Publish` (create tweet + chunked media upload); `FetchMetrics` (`public_metrics`); error mapping (401→refresh / 403+duplicate→terminal / 429→backoff w/ reset header / 5xx→retry)
- [x] **X simulator** (`internal/publish/simulator/twitter`): faithful fake — token/users.me/create(201)/media INIT-APPEND-FINALIZE-STATUS/metrics + injectable 429/401/403-duplicate/5xx/over-limit and async-processing knob
- [x] Adapter base URL injectable → tests hit simulator (live `POSTAL_LIVE_X=1` path TBD when real creds exist)
- [x] Publish pipeline: validate → publish → record → retry/backoff + refresh-once + **idempotency** (`publish_results`, migration `00005`); debit-hook point for future billing
**Tests/DoD:** ✅ simulator matrix (happy, media image+video/STATUS-poll, >280→terminal-no-API, duplicate→terminal, 429→retry→success, 5xx→retry→success, expired→refresh→success incl. maxAttempts=1) + Store PG-integration idempotency. `make check` green. **`/security-review` clean; `/code-review` (high) run — fixed 5 issues** (auth-retry attempt accounting, token-429 wrongly expiring channel, URL-regex undercount, shared `db.IsUniqueViolation`, documented double-Validate). Real X creds + live smoke gated for when an X app exists.

### Phase 5 — Posts, drafts, composer API ✅ DONE (2026-05-29)
**Goal:** create/manage content (no schedule yet).
- [x] Post + post_variant CRUD (`internal/post`, migration `00006`); per-channel body + `platform_options` (jsonb); **compose-once multi-publish** = one post, many channel variants
- [x] Draft lifecycle (posts created as `draft`); full-replace update; workspace-isolated; capability-gated (read/create/update/delete). Templates deferred
- [x] Compose-time validation via the platform adapter (`publish.Registry.Validate`) — per-variant instant feedback (`POST /posts/{id}/validate`); drafts may hold invalid content
- [x] Link/UTM tagging (`ApplyUTM` + `/posts/utm-preview`); link shortening deferred
**Tests/DoD:** ✅ Go integration test (real PG, X adapter as validator): create/get/list/update/delete, validate (valid + over-limit→`text_too_long`), cross-workspace isolation, foreign-channel rejection. `scripts/curl/posts.sh` 10/10 (authz gating, validation, UTM, isolation, 401). `make check` green. **`/code-review` (high) run — fixed 5** (UTM punctuation-absorption, channel-deleted→404 not 500, variant-count cap, `updated_at` bump, shared `web.PathUUID`). Deferred (noted): list pagination, batch channel lookup (N+1).

### Phase 6 — Scheduling engine (queue + workers) ⭐ CORE FEATURE ✅ DONE (2026-05-29)
**Goal:** Buffer's signature queue-based scheduling.
- [x] `schedule_slots` model: per-channel days/times/timezone (migration `00007`); slot CRUD
- [x] Queue semantics: `NextOpenSlot` (next free recurring slot, multi-occurrence) + specific `run_at` scheduling; **compose-once multi-publish** schedules a job per channel variant
- [x] Calendar range endpoint (`GET /workspaces/{id}/calendar?from=&to=`)
- [x] **asynq** enqueue at `run_at`; `worker` subcommand runs the asynq server + executes jobs via the Phase 4 `publish.Pipeline` (idempotency key = job ID → no double-post)
- [x] Timezone correctness: slots computed in their IANA tz, stored/compared in UTC (DST handled by tz database; spring-forward gap noted)
- [x] Cancel + status transitions (`scheduled→publishing→published/failed`, guarded **Claim** so a canceled job is never published even if its task fires); periodic token-refresh job (reuses Phase 3 `DueForRefresh`)
- [ ] Bulk CSV import, evergreen re-queue, reschedule, user notifications — deferred (documented)
**Tests/DoD:** ✅ Go integration test: schedule → worker `ProcessPublish` → **simulator receives the tweet** → `publish_results` recorded + job `published`; idempotency (re-delivery skipped), **canceled job not published**, duplicate→terminal→failed, cancel, `NextOpenSlot`. `scripts/curl/scheduling.sh` 10/10; worker subcommand boots + graceful shutdown. `make check` green. **`/code-review` (high) — fixed 5** (canceled-job-published via guarded Claim, MarkRetry resurrection, queue single-occurrence, misleading cancel success, server enqueuer leak).

### Phase 7 — Media pipeline ✅ DONE (2026-05-31)
**Goal:** images/video/GIF handling.
- [x] Upload endpoint → object storage (R2 in prod / MinIO locally via one S3-compatible client); size/type validation, capability-gated, workspace-isolated
- [ ] Image processing (libvips: resize/format), video (FFmpeg: transcode/validate specs) — **deferred** (stdlib dimension detection only; transcode/probe/virus-scan pluggable later)
- [x] Per-platform media validation (X image/GIF/video byte caps + counts) at compose time via the adapter
- [x] Media attach to post_variant (resolved/owned at compose time); worker loads bytes and the X adapter chunk-uploads them at publish
- [x] Quota: per-workspace storage cap, enforced atomically (workspace row-lock, no TOCTOU)
**Tests/DoD:** ✅ upload valid/invalid/oversize/unsupported + cross-workspace isolation (Go integration vs MinIO); end-to-end media→schedule→worker→X simulator (asset bytes traverse INIT/APPEND/FINALIZE); `scripts/curl/media.sh` 14/14; media-load error classification + composer rejection unit tests; `/security-review` clean; `/code-review` (7 findings fixed); `make check` green.
**Notes:** media-load failures are retry-classified so a transient storage blip never permanently fails a scheduled post; media without an uploaded `media_id` is rejected at compose (validation matches publish). Deferred follow-ups: libvips/FFmpeg transcode + probe, virus scan, streaming (vs full-buffer) media in the publish path.

### Phase 8 — Analytics ingestion & reporting ✅ DONE (2026-05-31)
**Goal:** post performance.
- [x] Analytics poller job: per-platform `FetchMetrics` via the publish pipeline (token/refresh reuse); periodic asynq cron (@every 15m), bounded batch + bounded fetch concurrency; per-(channel, platform post) poll-state so deleted/failed posts aren't re-polled
- [x] Store time-series `metric_snapshots` (long format, platform-agnostic); workspace overview + per-post **per-channel** breakdown + per-channel metric series (compose-once posts fan out to many channels, reported separately); workspace-isolated, `CapRead`-gated
- [x] Export (CSV) with formula-injection guard
- [x] Retention prune (90d) bounds table growth; `publish_results.post_id` now populated to link snapshots to posts
**Tests/DoD:** ✅ simulator metrics knob; end-to-end publish→poll→overview/series/CSV (worker integration); per-channel breakout integration test; `Pipeline.FetchMetrics` happy-path + token-refresh unit tests; `scripts/curl/analytics.sh` 12/12; `/security-review` clean; `/code-review` (10 findings fixed); `make check` green.
**Notes:** reporting groups by (post, channel); poll dedup is channel-scoped via poll-state (fixes cross-channel suppression + dead-post re-poll); SQL-side LIMIT by recency; batched `:copyfrom` insert in a tx (atomic snapshots). Deferred: per-post metric rollups, alerting.

### Phase 9 — Security hardening & anti-abuse pass ⭐ GATE ✅ DONE (2026-06-01)
**Goal:** full audit before declaring backend done.
- [x] `docs/SECURITY.md` + `docs/ANTI_ABUSE.md` audited end-to-end and updated to true state (done/partial/deferred marked honestly)
- [x] Pen-test-style authz review: every endpoint inventoried (method × middleware); all workspace routes carry RequireUser + RequireCapability + workspace isolation; no cross-workspace or privilege-escalation path found
- [x] Rate-limit coverage: per-IP buckets on every auth endpoint (signup/login/refresh/logout/verify/reset/reset-confirm) + per-user catch-all on the whole authenticated API; per-workspace quotas (channels, pending jobs, media storage)
- [x] Secrets/token encryption/log scrubbing verified: no token/PII in prod logs (request logger = path only; adapters never log bearer; console mailer prod-refused)
- [x] Dependency scan (`govulncheck`: 0 reachable vulns); global security headers (HSTS prod-only, X-Frame-Options DENY, Referrer-Policy, deny-all CSP, nosniff); CORS exact-origin allowlist (never `*`+credentials)
- [~] Worker soak/idempotency: idempotency proven under retries (Phase 6 tests); a dedicated load/soak run is deferred to Phase 11
**Tests/DoD:** ✅ `/security-review` (full) clean; security-headers + CORS unit tests; schedule-quota integration test; `scripts/curl/hardening.sh` 8/8; `make check` green.
**Deferred (documented):** captcha hook, email-verification-gates-publish (feature-toggle pass), malicious-URL blocking, dedicated per-channel velocity limiter + abuse counters, KMS master key + key-rotation runbook, session revocation on password reset.

### Phase 10 — Engagement (optional, post-MVP backend)
- [ ] Webhook ingestion for comments/mentions; unified inbox API; reply via adapter.

### Phase 11 — Backend completion & freeze
- [ ] Full integration/e2e suite green; all curl scripts pass; docs complete; OpenAPI spec generated for clients.
- [ ] **Backend declared complete.** Only now does frontend planning begin.

### Phase 12+ — Frontend (planned only after backend complete)
- Web SPA (Next.js/React + TS) and mobile (to be decided) consuming the same `/api/v1`.
- Frontend gets its own master plan document when we reach it.

---

## 8. X/Twitter specifics
See [`docs/X_TWITTER_INTEGRATION.md`](X_TWITTER_INTEGRATION.md) for the detailed integration
spec, simulator behavior, and test matrix. Research current API details before coding — X's API
and pricing change frequently; do not trust memory.

## 9. Testing strategy summary
Unit (logic) → Integration (real PG/Redis via docker-compose) → Server+curl scripts (real
output) → Social simulators (faithful fakes, full input matrix) → optional gated live smoke
tests. Failure paths are mandatory, not optional. See `CLAUDE.md` §3.

## 10. Observability & ops
slog JSON logs + request IDs; Prometheus metrics; `/healthz` `/readyz`; graceful shutdown;
docker-compose for local; single binary, two roles (serve/worker) for prod.

## 11. Security checklist
Lives in `docs/SECURITY.md` (created Phase 1). Summarized in §4. Enforced at every phase DoD.

## 12. Anti-abuse checklist
Lives in `docs/ANTI_ABUSE.md` (created Phase 1). Summarized in §4. Key controls: layered rate
limiting, quotas, email verification before publish, signup abuse defense, content/media safety,
respecting platform limits to protect shared app keys.

## 13. Memory & skill usage
See `CLAUDE.md` §5–6. Write a memory entry after each phase; load `/security-review`,
`/code-review`, `deep-research`, and web search at the points called out in the phases.

## 14. Progress log
Keep a running note here of what phase we're in and what's done.
- 2026-05-29: Plan created. Next action: **Phase 0 — scaffolding.** First social = X/Twitter (Phase 4).
- 2026-05-29: Authorization model decided — **capability flags + role presets** (§5.1), not fixed-role hierarchy. Admin can grant any combination of `read/create/update/delete/upload/publish/manage_*`. Affects Phase 2 data model (`workspace_member.permissions`) and middleware (`RequireCapability`).
- 2026-05-29: **Phase 0 complete & verified.** Go 1.26.3 installed system-wide (`/usr/local/go`). Module `github.com/Akins20/postal`. Stdlib-only typed config, chi server + health/readyz + slog/request-ID middleware, two-role binary (serve/worker) with graceful shutdown, goose+sqlc chain proven, docker-compose deps, golangci-lint + ≤800-line check all green.
- 2026-05-29: **Phase 1 complete & verified.** Foundation primitives: response envelope + error taxonomy (`apperr`/`web`), central error handler, strict bounded JSON decoding, AES-256-GCM envelope encryption with key rotation (`security`), audit-log writer + `audit_log` table, Redis token-bucket rate limiter + middleware (`ratelimit`), Prometheus `/metrics` (`platform/metrics`), and SECURITY.md/ANTI_ABUSE.md. `make check` green; rate-limit curl proves 429.
- 2026-05-29: **Phase 2 complete & verified.** Auth/tenancy: Argon2id passwords, email verification, JWT access + sliding rotating refresh in Redis (cookies + Bearer, CSRF double-submit), password reset, `RequireUser`, auto personal workspace, capability-based membership (`internal/workspace`) with `RequireCapability`, add-member/update-capabilities (no escalation, owner immutable), layered anti-abuse. `/security-review` run (no must-fix; fixed login-timing enumeration + prod console-mailer guard). auth.sh 13/13, capabilities.sh 12/12, make check green.
- 2026-05-29: **Phase 3 complete & verified.** Channels + OAuth token vault (`internal/channel`): generic OAuthProvider + PKCE/state connect flow, envelope-encrypted credential storage (migration 00004), token refresh, disconnect-purge, capability-gated + workspace-isolated. `/security-review` clean (all 8 areas). Integration test proves OAuth round trip + ciphertext at rest; channels.sh 9/9; make check green.
- 2026-05-29: **X API research** (deep-research) → posting is PAY-PER-USE/metered (no free tier since Feb 2026); updated `docs/X_TWITTER_INTEGRATION.md`. Decisions: social feature toggles + billing/usage model (operator pre-funds X credits; Postal tracks+caps per workspace — see memories `social-feature-toggles`, `billing-wallet-system`, `cross-platform-sync`).
- 2026-05-29: **Phase 4 complete & verified.** Publishing pipeline + X adapter + X simulator (`internal/publish`). PlatformAdapter contract, weighted-280 validation, chunked media upload, metrics, retry/backoff/refresh/idempotency (`publish_results`, migration 00005). `/security-review` clean; `/code-review` (high) fixed 5 issues. Simulator matrix + PG idempotency green; make check green. TikTok research done (→ `docs/TIKTOK_INTEGRATION.md`).
- 2026-05-29: **Phase 5 complete & verified.** Composer (`internal/post`, migration 00006): post + per-channel variant CRUD (compose-once multi-publish), drafts, compose-time validation via `publish.Registry`, UTM tagging; capability-gated + workspace-isolated. `/code-review` (high) fixed 5. posts.sh 10/10 + PG integration green; make check green.
- 2026-05-29: **Phase 6 complete & verified** ⭐. Scheduling engine (`internal/schedule` + `internal/worker`, migration 00007, asynq): schedule_slots + NextOpenSlot queue + specific-time scheduling, calendar, cancel, guarded Claim status machine, asynq worker subcommand executing the Phase 4 pipeline (idempotent), periodic token refresh. `/code-review` (high) fixed 5 incl. a real "canceled job still publishes" bug. Worker integration test (schedule→fire→simulator→recorded), scheduling.sh 10/10, make check green. **Next: Phase 7 — Media pipeline** (MinIO upload, image/video validation + transcode, attach to variants, chunked X media — wires real media bytes into the publish path).
