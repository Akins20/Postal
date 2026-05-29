package auth

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
)

// accessCookieName is the cookie carrying the JWT access token for web clients
// that prefer cookies over the Authorization header.
const accessCookieName = "postal_access"

// RequireUser authenticates the request via a bearer access token or the access
// cookie, verifies it, and injects the user ID into the context. Unauthenticated
// requests get a 401 in the standard envelope.
func RequireUser(tokens *TokenIssuer, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := accessTokenFromRequest(r)
			if raw == "" {
				web.Fail(w, r, log, apperr.Unauthorized("missing_token", "authentication required"))
				return
			}
			userID, err := tokens.Verify(raw)
			if err != nil {
				web.Fail(w, r, log, apperr.Unauthorized("invalid_token", "authentication required"))
				return
			}
			next.ServeHTTP(w, r.WithContext(web.WithUserID(r.Context(), userID)))
		})
	}
}

// accessTokenFromRequest extracts the access token, preferring the Authorization
// bearer header (mobile/API clients) and falling back to the access cookie (web).
func accessTokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if after, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(after)
		}
	}
	if c, err := r.Cookie(accessCookieName); err == nil {
		return c.Value
	}
	return ""
}
