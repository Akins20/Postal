# Postal

A free, no-paywall social media scheduling & publishing platform — a Buffer alternative.
Backend-first **modular monolith** in Go. See [`docs/MASTER_PLAN.md`](docs/MASTER_PLAN.md) for
*what* is built and in what order, [`CLAUDE.md`](CLAUDE.md) for *how* the project is run, and
[`docs/CODING_STANDARDS.md`](docs/CODING_STANDARDS.md) for the enforced code rules.

## Status

**Phase 0 — scaffolding.** A runnable empty monolith with dev tooling. No domain features yet.

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

## Layout

See [`CLAUDE.md`](CLAUDE.md) §2. Each `internal/<domain>` package is self-contained
(handler → service → store) and talks to other domains through interfaces.
