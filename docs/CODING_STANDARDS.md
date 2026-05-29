# Postal — Coding Standards (enforced)

> These rules are **enforced by tooling**, not just convention. A phase is not "done" if the
> standards check (`make lint` / `scripts/dev/check.sh`) fails. See `CLAUDE.md` §4 Definition of Done.

---

## 1. File & function size limits (hard, enforced)

- **Max 800 lines per source file.** Enforced by a check in `scripts/dev/check.sh` (counts
  lines per `.go` file, fails CI/local check if any exceeds 800). If a file approaches the
  limit, split by responsibility (e.g. `handler.go`, `service.go`, `store.go`, `types.go`)
  rather than cramming.
- **Soft target ~500 lines/file.** Treat 800 as the ceiling, not the goal.
- **Max ~60 lines per function** (soft) / **80 hard.** Long functions get decomposed.
- **Cyclomatic complexity ≤ 15 per function** (enforced via `golangci-lint` `gocyclo`/`cyclop`).
- **Max ~5 parameters per function** — beyond that, pass a struct. (`context.Context` and the
  receiver don't count.)
- **Max nesting depth 4.** Prefer early returns / guard clauses over deep nesting.

## 2. Formatting & tooling (non-negotiable, automated)

- `gofmt` + `goimports` on every file — no unformatted code is committed.
- **`golangci-lint`** runs in `make lint` and the DoD check. Enabled linters (at minimum):
  `govet`, `staticcheck`, `errcheck`, `revive`, `gocyclo`/`cyclop`, `gosec`, `ineffassign`,
  `unconvert`, `unparam`, `misspell`, `bodyclose`, `noctx`, `sqlclosecheck`, `gocritic`.
- `go vet` clean. No `//nolint` without a one-line justification comment.
- A committed `.golangci.yml` defines the config; don't weaken it without user approval.

## 3. Naming conventions (Go idioms)

- **Packages:** short, lowercase, single word, no underscores or camelCase (`channel`,
  `publish`, not `channel_mgmt` or `publishService`). Package name = directory name.
- **Exported identifiers:** `PascalCase`. **Unexported:** `camelCase`.
- **Acronyms keep consistent case:** `ID`, `URL`, `API`, `HTTP`, `OAuth` → `userID`, `apiURL`,
  `parseHTTPResponse` (never `Url`, `Id`, `Http`).
- **Interfaces:** name for behavior. Single-method interfaces end in `-er` (`Publisher`,
  `TokenRefresher`). Don't prefix with `I`.
- **No stutter:** `channel.Service`, not `channel.ChannelService`. `publish.Result`, not
  `publish.PublishResult` (the package already says "publish").
- **Constants:** `PascalCase` (exported) or `camelCase`; group related ones; no magic numbers
  or strings in code — name them.
- **Errors:** sentinel errors prefixed `Err` (`ErrNotFound`, `ErrTokenExpired`). Error message
  text is lowercase, no trailing punctuation (`"token expired"`, not `"Token expired."`).
- **Test functions:** `TestXxx`; table-test cases have descriptive `name` fields.
- **Receivers:** short (1–2 chars), consistent per type (`func (s *Service)`, `func (a *Adapter)`).
- **Files:** lowercase, descriptive, snake-free where possible (`token_vault.go` is acceptable
  Go style; prefer `tokenvault.go` only if it reads well — be consistent within a package).

## 4. Documentation & comments

- **Every package has a package doc comment** in a `doc.go` or atop the primary file:
  `// Package channel manages connected social accounts and their encrypted credentials.`
- **Every exported symbol (type, func, const, var) has a godoc comment** starting with the
  symbol's name: `// Publish sends a post variant to the platform and returns the result.`
- **Comments explain WHY, not WHAT.** The code says what; comments justify non-obvious choices,
  document invariants, edge cases, and gotchas (especially platform quirks).
- **No commented-out code** committed. Delete it; git remembers.
- **TODOs are tracked:** `// TODO(postal-#NN): ...` referencing an issue/plan item, never a bare TODO.
- **Security-sensitive code is annotated** (why a check exists, what it defends against).
- Keep comments current — a wrong comment is worse than none. Update comments with the code.

## 5. Error handling

- Always check errors; never `_ =` an error without a justified comment.
- **Wrap with context** using `fmt.Errorf("doing X: %w", err)` so the chain is inspectable.
- Use `errors.Is` / `errors.As` for inspection, not string matching.
- **No `panic` in library/handler code.** Panics only for truly unrecoverable init failures in
  `main`. Recover at the server's top-level middleware to avoid crashing the process.
- Return errors up; log them once at the boundary (handler/worker), not at every level.
- Distinguish error classes (validation / not-found / unauthorized / retryable / terminal) so
  HTTP mapping and retry logic are consistent (see publish pipeline retry rules).

## 6. Code structure & architecture rules

- **Layered per domain:** `handler` (HTTP) → `service` (business logic) → `store` (DB). Handlers
  never touch the DB directly; stores never contain business rules.
- **Dependencies point inward via interfaces.** Domains depend on other domains' interfaces, not
  concrete types. No import cycles (enforced by Go; keep it clean).
- **`context.Context` is the first parameter** of any function doing I/O, and is propagated
  (never `context.Background()` deep in the call tree).
- **No global mutable state.** Dependencies are injected via constructors (`NewService(deps...)`).
- **Accept interfaces, return concrete types.**
- Keep `internal/` truly internal; shared infra lives in `internal/platform/`.
- Configuration is read once at startup into a typed struct and passed down — no `os.Getenv`
  scattered through the code.

## 7. Concurrency

- Every goroutine has a clear owner and a way to stop (context cancellation). No leaks.
- Protect shared state with mutexes or channels; run tests with `-race` (the test target uses
  `go test -race`).
- Respect context cancellation/deadlines on all external calls.

## 8. Testing standards

- **Table-driven tests** for multi-case logic.
- Test files: `xxx_test.go`, same package for white-box or `xxx_test` package for black-box API tests.
- Use `testify/require` for fatal assertions, `assert` for soft. Helpers marked `t.Helper()`.
- Tests are deterministic — inject clocks/IDs, no reliance on wall-clock or `Math.random`-style nondeterminism.
- Run with `-race`. Aim for meaningful coverage of logic and ALL failure paths (per `CLAUDE.md` §3).

## 9. Dependencies

- Prefer the standard library. Add a third-party dep only when it clearly earns its place.
- Pin versions in `go.mod`; run `go mod tidy`. `govulncheck` in the security pass (Phase 9).
- No abandoned/unmaintained libraries for anything security-sensitive.

## 10. Commits & SQL

- Conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `chore:`, `docs:`). One concern per commit.
- SQL lives in `db/queries/` for `sqlc`; migrations in `db/migrations/` via `goose`, always
  reversible (`up`/`down`). No raw string-concatenated SQL anywhere — parameterized only.

---

### Enforcement summary
`scripts/dev/check.sh` (run by `make check`, required for DoD) does:
1. `gofmt -l` (fail if any unformatted) 2. `goimports` check 3. `go vet ./...`
4. `golangci-lint run` 5. **file length check (≤800 lines)** 6. `go test -race ./...`
A phase cannot be declared complete while `make check` fails.
