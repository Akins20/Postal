package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Akins20/postal/internal/config"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/metrics"
	"github.com/Akins20/postal/internal/platform/redis"
	"github.com/Akins20/postal/internal/ratelimit"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/server"
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

	// Validate the encryption keyring at startup so a misconfigured master key
	// fails fast rather than at first token write (Phase 3). Optional in early
	// dev: a missing key disables token features but still lets the server run.
	if err := initCrypto(cfg, log); err != nil {
		return err
	}

	srv := server.New(cfg.HTTP.Addr, server.Deps{
		Logger:         log,
		DB:             pool,
		Redis:          cache,
		Metrics:        metrics.New(),
		Limiter:        ratelimit.NewLimiter(cache, nil),
		RequestTimeout: cfg.HTTP.RequestTimeout,
	})

	log.Info("starting postal api server", slog.String("env", cfg.HTTP.Env))
	return srv.Start(ctx, cfg.HTTP.ShutdownTimeout)
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
