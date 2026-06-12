// Package server wires the HTTP API: router, middleware, and lifecycle
// (start/graceful-shutdown). Domain handlers mount their own sub-routers here;
// this package owns only cross-cutting concerns.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Akins20/postal/internal/analytics"
	"github.com/Akins20/postal/internal/auth"
	"github.com/Akins20/postal/internal/billing"
	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/media"
	"github.com/Akins20/postal/internal/platform/metrics"
	"github.com/Akins20/postal/internal/post"
	"github.com/Akins20/postal/internal/ratelimit"
	"github.com/Akins20/postal/internal/schedule"
	"github.com/Akins20/postal/internal/workspace"
)

// Pinger reports whether a backing dependency is reachable. Both the Postgres
// pool and the Redis client satisfy it; the server depends on this behavior
// rather than the concrete types.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps are the dependencies the server needs to wire its routes and readiness checks.
type Deps struct {
	Logger           *slog.Logger
	DB               Pinger
	Redis            Pinger
	Metrics          *metrics.Metrics
	Limiter          *ratelimit.Limiter
	Tokens           *auth.TokenIssuer
	AuthHandler      *auth.Handler
	WorkspaceHandler *workspace.Handler
	ChannelHandler   *channel.Handler
	PostHandler      *post.Handler
	ScheduleHandler  *schedule.Handler
	MediaHandler     *media.Handler
	AnalyticsHandler *analytics.Handler
	BillingHandler   *billing.Handler
	RequestTimeout   time.Duration
	// Production gates HSTS (only sent over the assumed-TLS production edge).
	Production bool
	// AllowedOrigins is the CORS allowlist; empty disables CORS.
	AllowedOrigins []string
}

// Server owns the HTTP router and underlying http.Server.
type Server struct {
	log  *slog.Logger
	mux  *chi.Mux
	http *http.Server
}

// New constructs a Server with cross-cutting middleware and the base routes
// (/healthz, /readyz) mounted. addr is the bind address (e.g. ":8080").
func New(addr string, deps Deps) *Server {
	mux := chi.NewRouter()

	// Order matters: assign a request ID first so every downstream log line and
	// panic recovery can reference it.
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	// Security headers + CORS run early so they cover every response (health,
	// metrics, recovered panics) and so preflight is answered before routing.
	mux.Use(securityHeaders(deps.Production))
	if len(deps.AllowedOrigins) > 0 {
		mux.Use(cors(deps.AllowedOrigins))
	}
	if deps.Metrics != nil {
		mux.Use(deps.Metrics.Middleware())
	}
	mux.Use(requestLogger(deps.Logger))
	mux.Use(recoverer(deps.Logger))
	if deps.RequestTimeout > 0 {
		mux.Use(middleware.Timeout(deps.RequestTimeout))
	}

	s := &Server{
		log: deps.Logger,
		mux: mux,
		http: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	s.mountHealth(deps.DB, deps.Redis)
	if deps.Metrics != nil {
		s.mux.Handle("/metrics", deps.Metrics.Handler())
	}
	s.mountAPI(deps)
	return s
}

// Router exposes the underlying chi router so domain packages can mount their
// own sub-routers (e.g. under /api/v1) during wiring.
func (s *Server) Router() chi.Router {
	return s.mux
}

// Start runs the HTTP server until the context is canceled, then shuts down
// gracefully within shutdownTimeout. It blocks until shutdown completes.
func (s *Server) Start(ctx context.Context, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http server listening", slog.String("addr", s.http.Addr))
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.http.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}
