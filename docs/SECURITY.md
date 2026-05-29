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
- [ ] Validation on every endpoint; parameterized SQL only (sqlc-enforced).
- [x] `X-Content-Type-Options: nosniff` on JSON responses.
- [ ] Full security headers (HSTS, frame options, referrer policy) — Phase 9.

## 5. Transport & CORS
- [ ] HTTPS only in production; secure, httpOnly, SameSite cookies for any
      cookie-based auth; CSRF protection for cookie flows.
- [ ] CORS locked to known origins.

## 6. Auditing & logging
- [x] Audit log writer for sensitive actions (`internal/security`, `audit_log`
      table). Records actor, workspace, action, target, metadata, IP.
- [x] Structured logs with request IDs; errors logged once at the boundary.
- [ ] Log scrubbing verified: no tokens/passwords/PII in logs — Phase 9.

## 7. Dependencies & supply chain
- [x] `golangci-lint` incl. `gosec` runs in `make check`.
- [ ] `govulncheck` in the security pass (Phase 9); dependencies pinned & tidy.

## 8. Abuse-resistance (cross-ref `ANTI_ABUSE.md`)
- [x] Reusable Redis token-bucket rate limiter + middleware (`internal/ratelimit`).
- [ ] Layered limits (global/IP/user/endpoint) applied on every public endpoint.
- [ ] Per-channel throttling to protect shared upstream API keys.
