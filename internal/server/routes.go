package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/ratelimit"
)

// pingRateRule bounds the demo ping endpoint: a small burst with slow refill, so
// the rate-limit middleware is observable from a curl script. Domain endpoints
// define their own rules from Phase 2 on.
var pingRateRule = ratelimit.Rule{Capacity: 5, RefillRate: 1}

// mountAPI wires the versioned API surface under /api/v1. For Phase 1 this is a
// single rate-limited ping endpoint proving the envelope, rate limiting, and
// error handling compose correctly end to end.
func (s *Server) mountAPI(log *slog.Logger, limiter *ratelimit.Limiter) {
	s.mux.Route("/api/v1", func(r chi.Router) {
		if limiter != nil {
			r.Use(limiter.Middleware(ratelimit.Config{
				Rule:   pingRateRule,
				Prefix: "rl:api:ping",
				Logger: log,
			}))
		}
		r.Get("/ping", handlePing)
	})
}

// handlePing returns a trivial success envelope. It exists to exercise the
// shared HTTP plumbing until real endpoints arrive.
func handlePing(w http.ResponseWriter, _ *http.Request) {
	web.Respond(w, http.StatusOK, map[string]string{"message": "pong"})
}
