package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Akins20/postal/internal/config"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/redis"
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

	srv := server.New(cfg.HTTP.Addr, server.Deps{
		Logger:         log,
		DB:             pool,
		Redis:          cache,
		RequestTimeout: cfg.HTTP.RequestTimeout,
	})

	log.Info("starting postal api server", slog.String("env", cfg.HTTP.Env))
	return srv.Start(ctx, cfg.HTTP.ShutdownTimeout)
}
