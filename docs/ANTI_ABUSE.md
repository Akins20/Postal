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
- [ ] Global, per-IP, per-user, and per-endpoint buckets applied across the API.
- [ ] Tighter buckets on expensive/abuse-prone actions (signup, login, publish).

## 2. Quotas (durable, per workspace)
- [ ] Max scheduled posts, connected channels, and media storage per workspace.
- [ ] Quota checks enforced server-side with clear error codes.

## 3. Signup & account abuse (Phase 2)
- [ ] Signup velocity limits per IP; disposable-email domain blocking.
- [ ] Captcha hook on signup.
- [ ] Email verification required before publishing.
- [ ] Login brute-force throttling + lockout.

## 4. Content & media safety (Phases 5–7)
- [ ] Media size/type/dimension/duration validation before accept.
- [ ] Block known-malicious URLs in post content.
- [ ] Per-platform content validation at compose time and publish time.

## 5. Protecting shared upstream API keys (Phases 3–4)
- [ ] Per-channel / per-user throttling so one abuser cannot exhaust or get the
      shared X/Twitter app credentials rate-limited or banned.
- [ ] Respect upstream rate-limit headers; back off proactively.
- [ ] Idempotency on publish so retries never double-post.

## 6. Observability of abuse
- [x] Prometheus HTTP request/latency metrics (`internal/platform/metrics`).
- [ ] Counters for rate-limit rejections, quota hits, auth failures.
- [ ] Audit-log review surfaces for suspicious activity.
