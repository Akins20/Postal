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
| TikTok | Content Posting API; app audit required. | Phase later. |
| **Bluesky / Mastodon** | Open, free, no approval. | Ideal *secondary* targets to prove multi-platform abstraction cheaply. |

**Design rule:** the publishing layer must treat every platform's limits (char count, media
specs, rate limits, duplicate rejection, token expiry) as data-driven config, so adding a
platform is "implement the adapter + declare its constraints," not rewrite the engine.

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

### Phase 4 — Publishing pipeline + **X/Twitter adapter** + simulator ⭐ FIRST SOCIAL
**Goal:** end-to-end publish to X (via simulator), proving the whole core.
- [ ] `PlatformAdapter` interface + `PlatformConstraints` (§6)
- [ ] **X/Twitter adapter**: OAuth2 PKCE auth URL / code exchange / refresh; `Validate` (280-char weighted count, media rules); `Publish` (create Tweet, with media upload sequence); `FetchMetrics`; error mapping (401/403/429/duplicate/5xx). See `docs/X_TWITTER_INTEGRATION.md`.
- [ ] **X simulator** (`internal/publish/simulator/twitter`): faithful HTTP fake — same endpoints, schemas, error codes, rate-limit (429 + reset headers), duplicate-tweet rejection, char-limit rejection, media upload chunking, expired/invalid token responses.
- [ ] Adapter base URL injectable → tests hit simulator; live tests gated by `POSTAL_LIVE_X=1`.
- [ ] Publish service: validate → publish via adapter → record result → retry/backoff/idempotency.
**Tests/DoD:** simulator test matrix (valid post, >280, bad media, expired token→refresh, 429→backoff, duplicate→terminal, 5xx→retry, timeout). Real output captured. **Run `/security-review` + `/code-review`.** Research X API current specs before building (`deep-research`/WebSearch).

### Phase 5 — Posts, drafts, composer API
**Goal:** create/manage content (no schedule yet).
- [ ] Post + post_variant CRUD; per-channel body customization; platform_options
- [ ] Draft lifecycle; templates
- [ ] Compose-time validation via adapter `Validate` (instant feedback)
- [ ] Link handling + UTM tagging; (link shortening optional/self-host later)
**Tests/DoD:** curl scripts for create/edit/draft/validate; rejects over-limit content per platform; authz enforced.

### Phase 6 — Scheduling engine (queue + workers) ⭐ CORE FEATURE
**Goal:** Buffer's signature queue-based scheduling.
- [ ] `schedule_slot` model: per-channel posting schedule (days/times/timezone)
- [ ] Queue semantics: drop a post into the next open slot; reorder; specific date/time scheduling too
- [ ] Calendar data endpoints (range query of scheduled posts)
- [ ] asynq enqueue at `run_at`; worker executes via Phase 4 pipeline
- [ ] Timezone correctness (store UTC, compute per channel tz); DST handling
- [ ] Bulk scheduling (CSV import) + re-queue evergreen
- [ ] Cancel/reschedule; status transitions; user notifications on publish/fail
**Tests/DoD:** schedule a post → worker fires at time (use injectable clock / short delays) → simulator receives it → result recorded. Test reorder, cancel, tz edge cases, retry on simulated 429. Run `/code-review`.

### Phase 7 — Media pipeline
**Goal:** images/video/GIF handling.
- [ ] Upload endpoint → object storage (MinIO locally); virus/size/type validation
- [ ] Image processing (libvips: resize/format), video (FFmpeg: transcode/validate specs)
- [ ] Per-platform media validation (X: image/video/GIF specs, counts, durations)
- [ ] Media attach to post_variant; chunked upload to X in adapter
- [ ] Quota: storage per workspace
**Tests/DoD:** upload valid/invalid media; oversize rejected; X media-upload simulated end to end; storage quota enforced.

### Phase 8 — Analytics ingestion & reporting
**Goal:** post performance.
- [ ] Analytics poller job: fetch metrics for published posts (adapter `FetchMetrics`)
- [ ] Store time-series `analytics_metric`; aggregate endpoints (per post, per channel, ranges)
- [ ] Export (CSV)
**Tests/DoD:** simulator returns metrics; poller stores them; aggregation endpoints correct; respects rate limits.

### Phase 9 — Security hardening & anti-abuse pass ⭐ GATE
**Goal:** full audit before declaring backend done.
- [ ] Complete `docs/SECURITY.md` and `docs/ANTI_ABUSE.md` checklists end-to-end
- [ ] Pen-test-style review of authz on every endpoint (cross-workspace, privilege escalation)
- [ ] Rate-limit/quota coverage on every public endpoint; abuse simulation tests
- [ ] Secrets handling, token encryption, log scrubbing verified
- [ ] Dependency vulnerability scan; security headers; CORS final
- [ ] Load/soak test the scheduling worker; idempotency under retries verified
**Tests/DoD:** **Run `/security-review` (full)**; all checklist boxes ticked; abuse tests pass.

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
- 2026-05-29: **Phase 3 complete & verified.** Channels + OAuth token vault (`internal/channel`): generic OAuthProvider + PKCE/state connect flow, envelope-encrypted credential storage (migration 00004), token refresh, disconnect-purge, capability-gated + workspace-isolated. `/security-review` clean (all 8 areas). Integration test proves OAuth round trip + ciphertext at rest; channels.sh 9/9; make check green. **Next: Phase 4 — Publishing pipeline + X/Twitter adapter + X simulator** (first social). Run `/security-review` + `/code-review` after Phase 4; research current X API specs first.
