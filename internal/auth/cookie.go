package auth

import (
	"net/http"
	"time"
)

// Cookie names and paths. The refresh cookie is scoped to the auth path so it is
// only ever sent to refresh/logout, limiting its exposure.
const (
	refreshCookieName = "postal_refresh"
	csrfCookieName    = "postal_csrf"
	authCookiePath    = "/api/v1/auth"
)

// CookieSettings configures auth cookie attributes from runtime config.
type CookieSettings struct {
	// Domain scopes the cookies (empty = host-only).
	Domain string
	// Secure sets the Secure flag (must be true over HTTPS; disable only for local http).
	Secure bool
}

// SetAuthCookies writes the access, refresh, and CSRF cookies. The access cookie
// (SameSite=Lax) authenticates API calls; the refresh cookie (httpOnly,
// SameSite=Strict, auth-path-scoped) drives refresh/logout; the CSRF cookie is
// readable by JS for the double-submit check.
func (cs CookieSettings) SetAuthCookies(w http.ResponseWriter, access string, accessTTL time.Duration, refresh, csrf string, refreshTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    access,
		Path:     "/",
		Domain:   cs.Domain,
		MaxAge:   int(accessTTL.Seconds()),
		HttpOnly: true,
		Secure:   cs.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refresh,
		Path:     authCookiePath,
		Domain:   cs.Domain,
		MaxAge:   int(refreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   cs.Secure,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrf,
		Path:     "/",
		Domain:   cs.Domain,
		MaxAge:   int(refreshTTL.Seconds()),
		HttpOnly: false, // readable by JS for the double-submit header
		Secure:   cs.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearAuthCookies expires the access, refresh, and CSRF cookies (logout).
func (cs CookieSettings) ClearAuthCookies(w http.ResponseWriter) {
	for _, c := range []struct{ name, path string }{
		{accessCookieName, "/"},
		{refreshCookieName, authCookiePath},
		{csrfCookieName, "/"},
	} {
		http.SetCookie(w, &http.Cookie{
			Name:     c.name,
			Value:    "",
			Path:     c.path,
			Domain:   cs.Domain,
			MaxAge:   -1,
			HttpOnly: c.name != csrfCookieName,
			Secure:   cs.Secure,
			SameSite: http.SameSiteStrictMode,
		})
	}
}
