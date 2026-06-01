package server

import "net/http"

// hstsMaxAge is the Strict-Transport-Security max-age (2 years), sent only in
// production over the assumed-TLS edge.
const hstsMaxAge = "max-age=63072000; includeSubDomains; preload"

// corsPreflightMaxAge caches a successful preflight for 10 minutes.
const corsPreflightMaxAge = "600"

// securityHeaders sets defensive response headers on every request. The API
// returns only JSON, so a deny-all CSP and frame/sniff/referrer hardening cost
// nothing and shrink the attack surface. HSTS is production-only (dev is plain
// HTTP). Applied early so it covers health, metrics, and recovered panics too.
func securityHeaders(production bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			if production {
				h.Set("Strict-Transport-Security", hstsMaxAge)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// cors applies a strict, allowlist-based CORS policy for browser clients. Only
// exact configured origins are reflected (never "*", which is unsafe with
// credentialed requests), and credentials are allowed so the cookie-based auth
// flow works cross-origin. With no configured origins it is a no-op (same-origin
// or native clients need no CORS headers). Preflight (OPTIONS) is answered here,
// before routing, so unmatched-method routing never rejects it.
func cors(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
				h.Set("Access-Control-Allow-Credentials", "true")
				if r.Method == http.MethodOptions {
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token")
					h.Set("Access-Control-Max-Age", corsPreflightMaxAge)
					w.WriteHeader(http.StatusNoContent)
					return
				}
			} else if r.Method == http.MethodOptions {
				// Preflight from a disallowed (or absent) origin: answer without the
				// allow headers so the browser blocks the actual request.
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
