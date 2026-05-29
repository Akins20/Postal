package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Akins20/postal/internal/auth"
	"github.com/Akins20/postal/internal/config"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/metrics"
	"github.com/Akins20/postal/internal/platform/redis"
	"github.com/Akins20/postal/internal/ratelimit"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/server"
	"github.com/Akins20/postal/internal/workspace"
)

// runServe connects backing dependencies and runs the HTTP API server until the
// context is canceled, then shuts down gracefully.
func runServe(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := db.Connect(ctx, cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	cache, err := redis.Connect(ctx, redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	defer func() { _ = cache.Close() }()

	if err := initCrypto(cfg, log); err != nil {
		return err
	}

	limiter := ratelimit.NewLimiter(cache, nil)
	auditor := security.NewAuditor(pool.Queries(), log)

	deps := server.Deps{
		Logger:         log,
		DB:             pool,
		Redis:          cache,
		Metrics:        metrics.New(),
		Limiter:        limiter,
		RequestTimeout: cfg.HTTP.RequestTimeout,
	}
	if err := wireAuth(&deps, cfg, pool, cache, limiter, auditor, log); err != nil {
		return err
	}

	srv := server.New(cfg.HTTP.Addr, deps)
	log.Info("starting postal api server", slog.String("env", cfg.HTTP.Env))
	return srv.Start(ctx, cfg.HTTP.ShutdownTimeout)
}

// wireAuth constructs the auth and workspace stack and attaches it to deps. With
// no JWT secret configured, auth endpoints are disabled and the server still
// serves health, metrics, and ping.
func wireAuth(deps *server.Deps, cfg config.Config, pool *db.Pool, cache *redis.Client, limiter *ratelimit.Limiter, auditor *security.Auditor, log *slog.Logger) error {
	if cfg.Auth.JWTSecret == "" {
		log.Warn("POSTAL_JWT_SECRET not set; auth endpoints are disabled")
		return nil
	}

	tokens, err := auth.NewTokenIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL, nil)
	if err != nil {
		return fmt.Errorf("building token issuer: %w", err)
	}
	sessions, err := auth.NewSessionStore(cache, cfg.Auth.RefreshTokenTTL, cfg.Auth.RefreshTokenMaxTTL, nil)
	if err != nil {
		return fmt.Errorf("building session store: %w", err)
	}

	// The console mailer logs verification/reset tokens (single-use account
	// secrets), which is acceptable only for local development. Refuse to run it
	// in production until a real mailer is wired, rather than leaking tokens.
	if cfg.HTTP.IsProduction() {
		return fmt.Errorf("no production mailer configured: refusing to start auth with the dev console mailer in production")
	}
	authSvc := auth.NewService(pool, tokens, sessions, auth.NewConsoleMailer(log), auditor, nil)
	deps.Tokens = tokens
	deps.AuthHandler = auth.NewHandler(auth.HandlerConfig{
		Service:    authSvc,
		Tokens:     tokens,
		Limiter:    limiter,
		Cookies:    auth.CookieSettings{Domain: cfg.Auth.CookieDomain, Secure: cfg.Auth.CookieSecure},
		Logger:     log,
		AccessTTL:  cfg.Auth.AccessTokenTTL,
		RefreshTTL: cfg.Auth.RefreshTokenTTL,
	})
	deps.WorkspaceHandler = workspace.NewHandler(workspace.NewService(pool, auditor, nil), log)
	return nil
}

// initCrypto validates the configured master key. A present-but-invalid key is
// a fatal misconfiguration; an absent key only disables token features.
func initCrypto(cfg config.Config, log *slog.Logger) error {
	if cfg.Crypto.MasterKey == "" {
		log.Warn("POSTAL_MASTER_KEY not set; token-vault features are disabled until configured")
		return nil
	}
	enc, err := security.NewEncryptorFromSpec(cfg.Crypto.MasterKey)
	if err != nil {
		return fmt.Errorf("invalid POSTAL_MASTER_KEY: %w", err)
	}
	log.Info("encryption keyring loaded", slog.Int("current_key_version", int(enc.CurrentVersion())))
	return nil
}
