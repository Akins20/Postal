package auth

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
)

// csrfHeaderName carries the double-submit token echoed from the CSRF cookie.
const csrfHeaderName = "X-CSRF-Token"

// CSRFProtect guards unsafe (state-changing) requests that authenticate via
// cookies, using the double-submit pattern: the X-CSRF-Token header must equal
// the CSRF cookie. Requests without a CSRF cookie are treated as bearer/API
// clients (cookies aren't auto-sent cross-site) and are exempt. Safe methods
// pass through.
func CSRFProtect(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(csrfCookieName)
			if err != nil {
				// No CSRF cookie -> not a cookie-authenticated browser flow.
				next.ServeHTTP(w, r)
				return
			}
			header := r.Header.Get(csrfHeaderName)
			if header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				web.Fail(w, r, log, apperr.Forbidden("csrf_failed", "CSRF validation failed"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isSafeMethod reports whether the HTTP method is read-only per RFC 7231.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
