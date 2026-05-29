package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Akins20/postal/internal/auth"
	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/ratelimit"
)

// pingRateRule bounds the demo ping endpoint: a small burst with slow refill, so
// the rate-limit middleware is observable from a curl script.
var pingRateRule = ratelimit.Rule{Capacity: 5, RefillRate: 1}

// mountAPI wires the versioned API surface under /api/v1: a demo ping, the
// public auth routes, and an authenticated group (RequireUser + CSRF) hosting
// workspace endpoints.
func (s *Server) mountAPI(deps Deps) {
	s.mux.Route("/api/v1", func(r chi.Router) {
		s.mountPing(r, deps)

		if deps.AuthHandler != nil {
			r.Mount("/auth", deps.AuthHandler.Routes())
		}

		// Authenticated API: every route requires a valid access token, and
		// state-changing cookie-authenticated requests are CSRF-protected.
		if deps.Tokens != nil {
			r.Group(func(pr chi.Router) {
				pr.Use(auth.RequireUser(deps.Tokens, deps.Logger))
				pr.Use(auth.CSRFProtect(deps.Logger))
				if deps.WorkspaceHandler != nil {
					pr.Mount("/workspaces", deps.WorkspaceHandler.Routes())
				}
			})
		}
	})
}

// mountPing wires the demo /ping endpoint behind its own rate limiter.
func (s *Server) mountPing(r chi.Router, deps Deps) {
	if deps.Limiter != nil {
		r.With(deps.Limiter.Middleware(ratelimit.Config{
			Rule:   pingRateRule,
			Prefix: "rl:api:ping",
			Logger: deps.Logger,
		})).Get("/ping", handlePing)
		return
	}
	r.Get("/ping", handlePing)
}

// handlePing returns a trivial success envelope exercising the shared plumbing.
func handlePing(w http.ResponseWriter, _ *http.Request) {
	web.Respond(w, http.StatusOK, map[string]string{"message": "pong"})
}
