# Postal — Project Operating Rules (read this first, every session)

Postal is a **free, no-paywall social media scheduling & publishing platform** (a Buffer
alternative). This file defines *how* Claude works on this project. The *what* lives in
[`docs/MASTER_PLAN.md`](docs/MASTER_PLAN.md). Read both before starting any work.

---

## 0. Prime directives (non-negotiable)

1. **Backend first.** Build the backend as a **modular monolith** in Go. Do NOT start the
   frontend until the backend is declared complete (see Definition of Done in the plan).
   The backend is an API consumed by BOTH mobile and web clients — design every endpoint
   to be client-agnostic (no server-rendered HTML coupling, clean JSON contracts, versioned
   API `/api/v1`).

2. **Test everything, locally, before declaring anything complete.** No feature is "done"
   until it has been exercised by running the actual server and hitting it with real
   `curl`/script tests that pass. "It compiles" is not "it works." See §3.

3. **Security is not optional.** Every feature must be built secure-by-default. There is no
   "we'll add auth later." See `docs/SECURITY.md` checklist (created in Phase 1).

4. **Anti-abuse is not optional.** A free app is an abuse magnet. Rate limiting, quotas,
   and abuse controls are built alongside features, not bolted on. See anti-abuse checklist
   in the plan.

5. **One phase at a time, in order.** Finish and verify a phase before moving to the next.
   Update the plan's progress checkboxes as you go.

6. **First social integration is X/Twitter.** See [`docs/X_TWITTER_INTEGRATION.md`](docs/X_TWITTER_INTEGRATION.md).

7. **Coding standards are enforced, not optional.** Read and follow
   [`docs/CODING_STANDARDS.md`](docs/CODING_STANDARDS.md). Highlights enforced by `make check`:
   **≤ 800 lines per source file** (hard cap), function/complexity limits, `gofmt`+`goimports`,
   `golangci-lint` clean, Go naming idioms (`ID`/`URL`/`API` casing, no package stutter),
   **godoc on every exported symbol + package doc comments**, error wrapping with `%w`, no
   commented-out code, no magic numbers, layered handler→service→store, context-first, no global
   state, `go test -race`. A phase cannot be "done" while `make check` fails.

---

## 1. Tech stack (locked — do not change without explicit user approval)

- **Language:** Go (latest stable)
- **HTTP router:** `chi` (`github.com/go-chi/chi/v5`) — stdlib-close, middleware-friendly
- **Database:** PostgreSQL
- **DB access:** `sqlc` (type-safe Go generated from SQL) + `pgx` driver
- **Migrations:** `goose`
- **Job queue / scheduler:** `asynq` (Redis-backed) — the heart of the scheduling engine
- **Cache / rate-limit counters / broker:** Redis
- **Config:** env vars via a typed config loader (e.g. `envconfig` / `viper`); `.env` for local
- **Logging:** `log/slog` (structured JSON logs)
- **Secrets/token encryption:** envelope encryption (AES-256-GCM) with a master key from env/KMS
- **Testing:** Go's `testing` + `testify`; HTTP via `httptest`; integration via `testcontainers-go` or docker-compose
- **Media:** FFmpeg (video) + libvips (images), shelled out or via bindings
- **Containerization:** Docker + docker-compose for local Postgres/Redis

If a library choice turns out wrong, raise it with the user before swapping.

---

## 2. Repository layout (target)

```
Postal/
  cmd/postal/main.go         # single binary entrypoint (monolith)
  internal/
    config/                  # typed config loading
    server/                  # http server, router, middleware wiring
    auth/                    # users, sessions/JWT, password hashing
    workspace/               # workspaces/teams, membership, roles
    channel/                 # connected social accounts + OAuth token vault
    publish/                 # publishing pipeline + per-platform adapters
      adapter/               # adapter interface
      twitter/               # X/Twitter adapter
      simulator/             # fake social servers for testing
    post/                    # posts, drafts, per-channel content variants
    schedule/                # queue-based scheduling, calendar, slots
    worker/                  # asynq workers that execute publish jobs
    media/                   # upload, validation, transcode pipeline
    analytics/               # metrics ingestion & reporting
    ratelimit/               # anti-abuse / rate governor
    security/                # crypto, token vault, audit log
    platform/                # shared infra: db, redis, storage, mailer
  db/
    migrations/              # goose migrations
    queries/                 # sqlc query files
  scripts/
    curl/                    # manual curl test scripts per feature
    dev/                     # dev helpers (seed, reset)
  docs/                      # all planning & reference docs
  test/                      # integration & e2e tests
  docker-compose.yml
  Makefile
```

Each `internal/<domain>` package is self-contained (handlers, service, store, types) so the
monolith stays modular and could later be split if ever needed. Domains talk to each other
through interfaces, not by reaching into each other's internals.

---

## 3. Testing protocol (mandatory for every feature)

For each feature/endpoint you build:

1. **Unit tests** for business logic (services, validation, crypto, rate-limit math).
2. **Run the actual server** (`make run` / docker-compose up deps first).
3. **Manual curl/script test:** add a runnable script under `scripts/curl/<feature>.sh` that
   exercises the happy path AND failure paths (bad auth, bad input, rate-limit hit, etc.),
   and prints clear PASS/FAIL. These scripts are committed and re-runnable.
4. **Confirm with real output.** Paste/observe the actual responses. Never claim success
   without seeing the server's real response.
5. **Failure-path coverage:** every endpoint must be tested for: missing/invalid auth,
   malformed input, oversized input, unauthorized access (wrong workspace), and rate limits.

### Social integrations — simulated testing
Real social APIs (especially X/Twitter) are **paid, rate-limited, and require approval**, so
you must NOT depend on them for routine testing. Instead:

- Build a **social simulator** (`internal/publish/simulator/`) — a local HTTP server that
  **mimics each platform's real API**: same auth flow shape, same request/response schemas,
  same error codes (401, 403, 429 rate limit, 5xx), same quirks (char limits, media rules,
  duplicate-post rejection).
- Tests point the adapter's base URL at the simulator.
- Simulate diverse inputs: valid posts, too-long text, unsupported media, expired tokens,
  rate-limit responses, partial failures, network timeouts, duplicate content.
- A small set of **real-API smoke tests** can exist but must be gated behind an env flag
  (`POSTAL_LIVE_X=1`) and never run by default.

No social feature is complete until it passes the full simulator test matrix.

---

## 4. Definition of Done (applies to every phase)

A phase is complete ONLY when ALL are true:
- [ ] Code compiles and `go vet` / linter is clean
- [ ] `make check` passes — incl. **≤800 lines/file**, `golangci-lint`, formatting, `-race` (see `docs/CODING_STANDARDS.md`)
- [ ] Coding standards followed (naming, godoc on exports, error wrapping, layering)
- [ ] Unit tests pass
- [ ] Integration tests pass against real Postgres/Redis (docker-compose)
- [ ] The server runs and the feature works via committed curl/script tests (real output seen)
- [ ] Failure paths tested (auth, validation, rate limit, abuse)
- [ ] Security checklist items for that feature are satisfied
- [ ] Anti-abuse controls for that feature are in place and tested
- [ ] Docs updated (the plan's checkboxes + any new behavior documented)
- [ ] A memory entry is written capturing key patterns/decisions/gotchas (see §6)

Do not say "complete" until every box is checked.

---

## 5. When to load skills / agents / research

- **`/security-review`** — run before declaring ANY phase that touches auth, tokens, input
  handling, or external calls complete. Mandatory after Phase 2 (auth), Phase 3 (token vault),
  Phase 4 (X adapter), and Phase 9 (hardening).
- **`/code-review`** — run after each substantial phase, before marking it done.
- **`deep-research` skill** — load when integrating a NEW social platform (to pull current
  API docs, rate limits, auth flow, content rules) or when an external API's behavior is
  unclear. Always verify API details are current — they change often.
- **`WebSearch`/`WebFetch`** — use to confirm current X/Twitter API v2 endpoints, pricing
  tiers, OAuth 2.0 PKCE flow, and rate limits before/while building the adapter. Do not rely
  on memory for API specifics.
- **Subagents (Agent tool)** — use `Explore` for broad code search; spawn parallel agents
  when a phase has independent workstreams (e.g. building 3 unrelated endpoints). Use the
  `Plan` agent to design a phase's implementation before coding it.
- **`verify` skill** — use to confirm a change works in the running app.

Default to research-before-build for anything touching an external API.

---

## 6. Memory discipline

Maintain memory of what matters as the project proceeds. After each phase (and whenever a
non-obvious decision/gotcha appears), write a memory entry capturing:
- Architectural decisions and WHY (so they aren't relitigated)
- Patterns established (error handling, response envelope, auth middleware usage, adapter
  contract) so new code stays consistent
- Gotchas discovered (X API quirks, sqlc/pgx edge cases, asynq config)
- Current phase + what's done vs. pending

Memory lives in the Claude memory dir; keep `MEMORY.md` index updated. Also keep the plan's
progress checkboxes current as the in-repo source of truth.

---

## 7. House rules

- Conventional commits. Branch per phase; never commit straight to main unless told.
- Small, reviewable changes. One concern per commit.
- No secrets in the repo. `.env.example` documents required vars; real `.env` is gitignored.
- Every endpoint returns a consistent JSON envelope and consistent error shape (define once
  in Phase 1, reuse everywhere).
- Search scope: keep file searches focused to this project tree, never the whole filesystem.
