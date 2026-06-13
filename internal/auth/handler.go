package auth

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/ratelimit"
)

// Anti-abuse rate rules for auth endpoints (token buckets). Refill rates are
// per second; the small sustained rates make brute force and signup spam
// impractical while allowing legitimate bursts.
var (
	signupIPRule = ratelimit.Rule{Capacity: 10, RefillRate: 10.0 / 3600} // ~10/hour per IP
	loginIPRule  = ratelimit.Rule{Capacity: 10, RefillRate: 0.05}        // ~3/min per IP
	loginEmail   = ratelimit.Rule{Capacity: 8, RefillRate: 8.0 / 3600}   // ~8/hour per email
	resetIPRule  = ratelimit.Rule{Capacity: 3, RefillRate: 3.0 / 3600}   // ~3/hour per IP
	// tokenIPRule throttles opaque-token submissions (email verify, reset
	// confirm) to defeat token brute-forcing; legitimate users submit once.
	tokenIPRule = ratelimit.Rule{Capacity: 10, RefillRate: 10.0 / 3600} // ~10/hour per IP
	// refreshIPRule bounds refresh/logout churn (a valid session refreshes ~every
	// access-token TTL); generous so real clients behind NAT aren't throttled.
	refreshIPRule = ratelimit.Rule{Capacity: 60, RefillRate: 0.2} // ~12/min per IP
)

// RelaxAuthLimitsForDevelopment multiplies the auth buckets so local e2e
// suites (each run signs up several throwaway accounts) don't starve the
// per-IP signup bucket. Wiring calls it ONLY when the environment is not
// production; the production rules above are the contract.
func RelaxAuthLimitsForDevelopment() {
	for _, r := range []*ratelimit.Rule{
		&signupIPRule, &loginIPRule, &loginEmail, &resetIPRule, &tokenIPRule, &refreshIPRule,
	} {
		r.Capacity *= 100
		r.RefillRate *= 100
	}
}

// Handler serves the /api/v1/auth endpoints.
type Handler struct {
	svc        *Service
	tokens     *TokenIssuer
	limiter    *ratelimit.Limiter
	cookies    CookieSettings
	log        *slog.Logger
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// HandlerConfig bundles the Handler's dependencies.
type HandlerConfig struct {
	Service    *Service
	Tokens     *TokenIssuer
	Limiter    *ratelimit.Limiter
	Cookies    CookieSettings
	Logger     *slog.Logger
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// NewHandler builds the auth HTTP handler.
func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{
		svc:        cfg.Service,
		tokens:     cfg.Tokens,
		limiter:    cfg.Limiter,
		cookies:    cfg.Cookies,
		log:        cfg.Logger,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
	}
}

// Routes returns the auth subrouter to mount under /api/v1/auth. CSRF protection
// guards refresh/logout (which consume the session cookie); per-endpoint rate
// limits guard abuse-prone routes. Login/signup are CSRF-exempt — they don't
// authenticate via an existing session cookie.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	csrf := CSRFProtect(h.log)

	r.With(h.rateLimit("rl:auth:signup", signupIPRule)).Post("/signup", web.Handler(h.log, h.signup))
	r.With(h.rateLimit("rl:auth:login", loginIPRule)).Post("/login", web.Handler(h.log, h.login))
	r.With(h.rateLimit("rl:auth:refresh", refreshIPRule), csrf).Post("/refresh", web.Handler(h.log, h.refresh))
	r.With(h.rateLimit("rl:auth:logout", refreshIPRule), csrf).Post("/logout", web.Handler(h.log, h.logout))
	r.With(h.rateLimit("rl:auth:verify", tokenIPRule)).Post("/verify-email", web.Handler(h.log, h.verifyEmail))
	r.With(h.rateLimit("rl:auth:verify-resend", resetIPRule)).Post("/verify-email/resend", web.Handler(h.log, h.resendVerification))
	r.With(h.rateLimit("rl:auth:reset", resetIPRule)).Post("/password-reset/request", web.Handler(h.log, h.requestReset))
	r.With(h.rateLimit("rl:auth:reset-confirm", tokenIPRule)).Post("/password-reset/confirm", web.Handler(h.log, h.confirmReset))

	r.With(RequireUser(h.tokens, h.log)).Get("/me", web.Handler(h.log, h.me))
	return r
}

// rateLimit builds per-IP rate-limit middleware for a route.
func (h *Handler) rateLimit(prefix string, rule ratelimit.Rule) func(http.Handler) http.Handler {
	return h.limiter.Middleware(ratelimit.Config{Rule: rule, Prefix: prefix, Logger: h.log})
}

// --- request/response DTOs ---

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type tokenResponse struct {
	AccessToken  string        `json:"access_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int           `json:"expires_in"`
	CSRFToken    string        `json:"csrf_token"`
	RefreshToken string        `json:"refresh_token,omitempty"`
	User         *userResponse `json:"user,omitempty"`
}

// --- handlers ---

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) error {
	var req credentialsRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	user, err := h.svc.Signup(r.Context(), req.Email, req.Password, ratelimit.ClientIPKey(r))
	if err != nil {
		return err
	}
	resp := toUserResponse(user)
	web.Respond(w, http.StatusCreated, resp)
	return nil
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) error {
	var req credentialsRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	// Per-email throttle complements the per-IP middleware (distributed brute force).
	if err := h.throttleByEmail(r, req.Email); err != nil {
		return err
	}

	res, err := h.svc.Login(r.Context(), req.Email, req.Password, ratelimit.ClientIPKey(r))
	if err != nil {
		return err
	}
	u := toUserResponse(res.User)
	return h.writeSession(w, res.AccessToken, res.RefreshToken, &u)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) error {
	token := h.refreshTokenFromRequest(r)
	if token == "" {
		return apperr.Unauthorized("missing_refresh_token", "no refresh token provided")
	}
	res, err := h.svc.Refresh(r.Context(), token)
	if err != nil {
		return err
	}
	return h.writeSession(w, res.AccessToken, res.RefreshToken, nil)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	token := h.refreshTokenFromRequest(r)
	if token != "" {
		if err := h.svc.Logout(r.Context(), token); err != nil {
			return err
		}
	}
	h.cookies.ClearAuthCookies(w)
	web.Respond(w, http.StatusOK, map[string]string{"message": "logged out"})
	return nil
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) error {
	userID, ok := web.UserID(r.Context())
	if !ok {
		return apperr.Unauthorized("missing_token", "authentication required")
	}
	user, err := h.svc.GetUser(r.Context(), userID)
	if err != nil {
		return err
	}
	resp := toUserResponse(user)
	web.Respond(w, http.StatusOK, resp)
	return nil
}

// --- helpers ---

// throttleByEmail consumes from a per-email login bucket, returning a rate-limit
// error when exhausted.
func (h *Handler) throttleByEmail(r *http.Request, email string) error {
	key := "rl:auth:login:email:" + normalizeEmail(email)
	res, err := h.limiter.Allow(r.Context(), key, loginEmail, 1)
	if err != nil {
		return nil // fail open on limiter backend errors for the per-email layer; per-IP still applies
	}
	if !res.Allowed {
		return apperr.RateLimited("too_many_attempts", "too many login attempts; please try again later")
	}
	return nil
}

// refreshTokenFromRequest reads the refresh token from the cookie (web) or, if
// absent, the JSON body (mobile/API). A missing or malformed body is tolerated.
func (h *Handler) refreshTokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if r.Body == nil {
		return ""
	}
	var body tokenRequest
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	return body.RefreshToken
}

// writeSession sets auth cookies and writes the token response body. user is
// included on login and omitted on refresh.
func (h *Handler) writeSession(w http.ResponseWriter, access, refresh string, user *userResponse) error {
	csrf, err := newOpaqueToken()
	if err != nil {
		return apperr.Internal(err)
	}
	h.cookies.SetAuthCookies(w, access, h.accessTTL, refresh, csrf, h.refreshTTL)
	web.Respond(w, http.StatusOK, tokenResponse{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.accessTTL.Seconds()),
		CSRFToken:    csrf,
		RefreshToken: refresh,
		User:         user,
	})
	return nil
}

// toUserResponse maps a stored user to its safe API representation (no hash).
func toUserResponse(u sqlc.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Status:        u.Status,
		CreatedAt:     u.CreatedAt.Time,
	}
}
