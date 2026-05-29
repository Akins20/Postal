package ratelimit

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
)

// KeyFunc derives the bucket key for a request. Returning the same key for
// different requests makes them share a budget (e.g. per-IP, per-user).
type KeyFunc func(r *http.Request) string

// Config configures the rate-limit middleware.
type Config struct {
	// Rule is the token bucket applied per key.
	Rule Rule
	// Key derives the bucket key; defaults to per-client-IP when nil.
	Key KeyFunc
	// Prefix namespaces keys in Redis (e.g. "rl:ping"). Distinct prefixes keep
	// independent limiters from colliding.
	Prefix string
	// Logger records limiter backend failures; may be nil.
	Logger *slog.Logger
	// FailOpen, when true, allows requests if Redis is unavailable (availability
	// over strictness). Defaults to fail-closed for abuse-sensitive endpoints.
	FailOpen bool
}

// Middleware returns chi-compatible middleware enforcing cfg's token bucket. On
// rejection it responds 429 with the standard error envelope and a Retry-After
// header. Each allowed request consumes one token.
func (l *Limiter) Middleware(cfg Config) func(http.Handler) http.Handler {
	key := cfg.Key
	if key == nil {
		key = ClientIPKey
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bucketKey := cfg.Prefix + ":" + key(r)

			res, err := l.Allow(r.Context(), bucketKey, cfg.Rule, 1)
			if err != nil {
				l.handleBackendError(w, r, cfg, next, err)
				return
			}

			setRateLimitHeaders(w, cfg.Rule, res)
			if !res.Allowed {
				if res.RetryAfter > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(int(res.RetryAfter.Round(time.Second).Seconds())))
				}
				web.Fail(w, r, cfg.Logger, apperr.RateLimited("rate_limited", "too many requests; please slow down"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// handleBackendError applies the fail-open/closed policy when Redis errors.
func (l *Limiter) handleBackendError(w http.ResponseWriter, r *http.Request, cfg Config, next http.Handler, err error) {
	if cfg.Logger != nil {
		cfg.Logger.LogAttrs(r.Context(), slog.LevelError, "rate limiter backend error",
			slog.String("error", err.Error()), slog.Bool("fail_open", cfg.FailOpen))
	}
	if cfg.FailOpen {
		next.ServeHTTP(w, r)
		return
	}
	web.Fail(w, r, cfg.Logger, apperr.RateLimited("rate_limiter_unavailable", "service is busy; please retry shortly"))
}

// setRateLimitHeaders advertises the limit and remaining budget to clients.
func setRateLimitHeaders(w http.ResponseWriter, rule Rule, res Result) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rule.Capacity))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
}

// ClientIPKey keys the bucket by client IP. RemoteAddr is expected to already
// reflect the real client (chi's RealIP middleware runs earlier).
func ClientIPKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
