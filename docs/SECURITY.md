# Postal — Security Checklist

> Created in Phase 1. This is the living security baseline. Every phase's
> Definition of Done re-checks the items relevant to that phase, and the full
> list is audited in Phase 9. Run `/security-review` after Phase 2 (auth),
> Phase 3 (token vault), Phase 4 (X adapter), and Phase 9 (hardening).

Legend: `[ ]` not yet · `[~]` partial · `[x]` in place & tested.

## 1. Secrets & key management
- [x] No secrets committed; `.env` gitignored, `.env.example` documents vars.
- [x] Master key (`POSTAL_MASTER_KEY`) loaded from env; never logged.
- [x] Token encryption: AES-256-GCM **envelope encryption** with per-secret data
      keys and a versioned master key (`internal/security`). Tamper-evident
      (GCM auth tag); ciphertext self-describes its key version for rotation.
- [x] Social OAuth tokens encrypted at rest (`channel_credentials` BYTEA via the
      vault, Phase 3); never returned to clients or logged; ciphertext-at-rest
      proven by integration test. PKCE S256 + single-use CSRF state (Redis GetDel)
      bound to the initiating user defeats OAuth code-injection.
- [ ] Key rotation runbook + re-encryption job (mechanism ready; ops doc TBD).
- [ ] Master key sourced from a KMS in production (env acceptable for dev).

## 2. Authentication & sessions (Phase 2 ✅)
- [x] Passwords hashed with Argon2id (per-password random salt, PHC-encoded params).
- [x] Sessions: short-lived JWT access + rotating refresh in Redis (sliding TTL,
      absolute cap, single-use rotation); JWT enforces HS256/issuer/expiry.
- [x] Login throttling (per-IP + per-email); generic auth-failure message; login
      timing equalized with a dummy Argon2id verify (enumeration defense).
- [x] Password reset tokens single-use, short-lived, hashed (SHA-256) at rest;
      email verification tokens likewise.
- [ ] Email verification required before publishing is enabled (flag set; gate
      enforced in the publish pipeline, Phase 4+).
- [ ] Revoke all sessions on password reset (needs session-versioning; deferred).

## 3. Authorization & tenancy (Phase 2 ✅)
- [x] Capability-based authz (`RequireCapability`); workspace isolation enforced —
      membership resolved per `{workspaceID}` for the authenticated user; non-members
      get 403 (existence not revealed). See MASTER_PLAN §5.1.
- [x] No privilege escalation: `CanGrant` blocks granting a capability the actor
      lacks; workspace owner membership is immutable.
- [x] Object-level checks: capability middleware gates each workspace route.

## 4. Input handling & output
- [x] Standard error envelope; internal causes never leak to clients
      (`internal/platform/web`).
- [x] JSON decoding bounded (max body size), strict (unknown fields rejected),
      content-type checked (`web.DecodeJSON`).
- [x] Validation on every endpoint; parameterized SQL only (sqlc-enforced).
- [x] `X-Content-Type-Options: nosniff` on JSON responses.
- [x] Full security headers (Phase 9): global middleware sets nosniff,
      `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, deny-all CSP
      (`default-src 'none'; frame-ancestors 'none'`), and HSTS in production.

## 5. Transport & CORS
- [x] Secure/httpOnly/SameSite cookies (Phase 2); CSRF double-submit on cookie
      flows; HSTS asserts HTTPS-only to browsers in production (TLS terminated at
      the edge/ingress — an ops concern, documented).
- [x] CORS locked to a configured exact-origin allowlist
      (`POSTAL_CORS_ALLOWED_ORIGINS`); never `*` with credentials; disabled when
      unset (same-origin / native clients).

## 6. Auditing & logging
- [x] Audit log writer for sensitive actions (`internal/security`, `audit_log`
      table). Records actor, workspace, action, target, metadata, IP.
- [x] Structured logs with request IDs; errors logged once at the boundary.
- [x] Log scrubbing verified (Phase 9 audit): the request logger logs path only
      (never query strings), adapters never log bearer tokens/Authorization, and
      the only token-logging path (dev console mailer) is hard-refused in
      production. No tokens/passwords/PII in prod logs.

## 7. Dependencies & supply chain
- [x] `golangci-lint` incl. `gosec` runs in `make check`.
- [x] `govulncheck` run in the Phase 9 pass: **0 vulnerabilities** on Postal's
      call paths (advisories exist in transitively-required modules but no
      vulnerable symbol is reachable). Re-run in CI.

## 8. Abuse-resistance (cross-ref `ANTI_ABUSE.md`)
- [x] Reusable Redis token-bucket rate limiter + middleware (`internal/ratelimit`).
- [x] Layered limits applied: per-IP buckets on every auth endpoint, a per-user
      catch-all on the whole authenticated API, plus per-workspace quotas.
- [~] Per-channel throttling: composed from the per-user catch-all + per-workspace
      pending-jobs quota + upstream-429 backoff + publish idempotency. A dedicated
      per-channel velocity limiter is a documented follow-up.
