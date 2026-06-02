# Postal

A free, no-paywall social media scheduling & publishing platform — a Buffer alternative.
Backend-first **modular monolith** in Go. See [`docs/MASTER_PLAN.md`](docs/MASTER_PLAN.md) for
*what* is built and in what order, [`CLAUDE.md`](CLAUDE.md) for *how* the project is run, and
[`docs/CODING_STANDARDS.md`](docs/CODING_STANDARDS.md) for the enforced code rules.

## Status

**Backend complete & frozen** (Phases 0–9 + 11). The `/api/v1` surface is final
and documented in [`docs/openapi.yaml`](docs/openapi.yaml) (OpenAPI 3.0). Frontend
planning begins next (Phase 12). Phase 10 (engagement/inbox) is optional/post-MVP.

### What's built

- **Auth & tenancy** — Argon2id passwords, JWT access + rotating/sliding refresh
  tokens, email verification, password reset; workspaces with a capability-based
  authorization model (role presets + per-user capabilities) and strict isolation.
- **Channels** — connect social accounts over OAuth 2.0 PKCE; tokens
  envelope-encrypted at rest (AES-256-GCM) in the vault.
- **Composer** — compose-once posts with per-channel variants, drafts,
  compose-time per-platform validation, UTM tagging.
- **Scheduling** — queue-based scheduling (specific time or posting slots) on an
  `asynq` worker; idempotent publishing (no double-posts under retries).
- **Publishing** — `PlatformAdapter` contract with the X/Twitter adapter and a
  faithful local simulator; retry-class error model with token refresh + backoff.
- **Media** — upload to S3-compatible storage (Cloudflare R2 / MinIO), per-workspace
  quota, chunked upload to the platform at publish.
- **Analytics** — a poller fetches post metrics into a time series; per-(post,
  channel) reporting, series, and CSV export.
- **Security & anti-abuse** — layered rate limits, per-workspace quotas, security
  headers, CORS allowlist, audit log, `govulncheck`-clean dependencies.

## Prerequisites

- **Go 1.23+**
- **Docker + Docker Compose** (local Postgres, Redis, MinIO)
- `make`, `curl`

## Quick start

```bash
cp .env.example .env          # local config (gitignored)
make up                       # start Postgres + Redis + MinIO
make migrate                  # apply DB migrations (proves the goose chain)
make run                      # start the API server on :8080
# in another shell:
./scripts/curl/health.sh      # smoke-test /healthz and /readyz
```

## The binary: two roles

One binary, selected by subcommand (both share the same image/config):

```bash
postal serve     # HTTP API server (/api/v1, health endpoints)
postal worker    # asynq job worker (scheduling/refresh/analytics — Phase 6+)
```

## Common make targets

| Target | What it does |
|--------|--------------|
| `make run` / `make run-worker` | run the API server / worker |
| `make up` / `make down` | start / stop local dependencies |
| `make migrate` / `make migrate-down` / `make migrate-status` | goose migrations |
| `make sqlc` | regenerate type-safe Go from `db/queries` |
| `make test` | `go test -race ./...` |
| `make lint` | golangci-lint |
| `make check` | full Definition-of-Done gate (fmt, vet, lint, ≤800-line check, race tests) |

## API

The full HTTP contract is in [`docs/openapi.yaml`](docs/openapi.yaml) — every
endpoint, request/response schema, auth scheme (Bearer JWT or cookie + CSRF),
capability requirement, and error shape. Load it into Swagger UI / Redoc, or
generate typed clients for web and mobile.

Two response conventions: JSON endpoints wrap payloads in `{ "data": ... }` and
errors in `{ "error": { code, message, fields? } }`; media download and the
analytics CSV export stream raw bodies.

## Testing

```bash
make check                    # full gate: fmt, vet, lint, ≤800-line, race tests (unit + integration)
# end-to-end HTTP checks against a running server (deps + `make run`):
for s in scripts/curl/*.sh; do bash "$s"; done
```

Integration tests run against the docker-compose Postgres/Redis/MinIO; social
publishing is exercised against the in-repo simulator (never the paid real API).

## Layout

See [`CLAUDE.md`](CLAUDE.md) §2. Each `internal/<domain>` package is self-contained
(handler → service → store) and talks to other domains through interfaces.
