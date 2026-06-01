# Postal — Anti-Abuse Checklist

> Created in Phase 1. A free, no-paywall app is an abuse magnet, so abuse
> controls are built alongside each feature — never bolted on. Items are
> re-checked at each phase's Definition of Done and audited in Phase 9.

Legend: `[ ]` not yet · `[~]` partial · `[x]` in place & tested.

## 1. Rate limiting (layered)
- [x] Reusable token-bucket limiter in Redis, atomic via Lua, clock-injected
      for deterministic tests (`internal/ratelimit`).
- [x] Middleware returns `429` with the standard envelope + `Retry-After` and
      `X-RateLimit-*` headers; fail-closed by default, opt-in fail-open.
- [x] Per-IP buckets on every auth endpoint; a per-user catch-all bucket on the
      whole authenticated API (keyed by user ID after RequireUser).
- [~] Tighter buckets on signup/login (done, Phase 2). Publish velocity is bounded
      by the per-user cap + per-workspace pending-jobs quota; a dedicated publish
      bucket is a follow-up.

## 2. Quotas (durable, per workspace)
- [x] Connected channels (Phase 9), pending scheduled jobs (Phase 9), and media
      storage (Phase 7) all capped per workspace.
- [x] Quota checks enforced server-side with clear error codes
      (`channel_quota_exceeded`, `schedule_quota_exceeded`, `quota_exceeded`).

## 3. Signup & account abuse (Phase 2)
- [x] Signup velocity limits per IP; disposable-email domain blocking.
- [ ] Captcha hook on signup (provider integration deferred).
- [ ] Email verification required before publishing (flag exists; publish-gate
      deferred to a feature-toggle pass).
- [x] Login brute-force throttling (per-IP + per-email; timing-equalized).

## 4. Content & media safety (Phases 5–7)
- [x] Media size/type/dimension validation before accept (Phase 7); duration probe
      deferred with FFmpeg.
- [ ] Block known-malicious URLs in post content (URL reputation feed deferred).
- [x] Per-platform content validation at compose time and publish time (Phase 4–5).

## 5. Protecting shared upstream API keys (Phases 3–4)
- [~] Per-user throttling + per-workspace channel/job quotas bound one abuser's
      load on the shared X app; a per-channel velocity limiter is a follow-up.
- [x] Respect upstream rate-limit headers; back off proactively (adapter maps 429
      to retryable with RetryAfter; pipeline honors it).
- [x] Idempotency on publish so retries never double-post (job-ID key, Phase 6).

## 6. Observability of abuse
- [x] Prometheus HTTP request/latency metrics (`internal/platform/metrics`).
- [ ] Counters for rate-limit rejections, quota hits, auth failures (dedicated
      metrics deferred; rejections are logged and 429-observable today).
- [x] Audit-log writer records sensitive actions for review (`audit_log`).
