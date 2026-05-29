package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/config"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/redis"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/schedule"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/worker"
	"github.com/Akins20/postal/internal/workspace"
)

// runWorker runs the asynq job processor: it executes scheduled publish jobs
// through the publish pipeline and periodically refreshes channel tokens. It
// requires the encryption key (to open stored credentials).
func runWorker(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := db.Connect(ctx, cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	cache, err := redis.Connect(ctx, redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	defer func() { _ = cache.Close() }()

	enc, err := initEncryptor(cfg, log)
	if err != nil {
		return err
	}
	if enc == nil {
		return fmt.Errorf("worker requires POSTAL_MASTER_KEY to open channel credentials")
	}

	auditor := security.NewAuditor(pool.Queries(), log)
	wsSvc := workspace.NewService(pool, auditor, nil)
	adapters := buildAdapters(cfg, log)
	channelSvc := channel.NewService(pool, channel.NewRegistry(toOAuthProviders(adapters)...), enc, cache, wsSvc, auditor, nil)

	pipeline := publish.NewPipeline(channelSvc, publish.NewStore(pool.Queries()), adapters)
	// The worker only processes jobs (claim/execute/mark) — it never enqueues —
	// so it needs no asynq client (nil Enqueuer).
	scheduleSvc := schedule.NewService(pool, channelSvc, nil, auditor, nil)

	processor := worker.NewProcessor(scheduleSvc, pipeline, channelSvc, log, nil)
	log.Info("starting postal worker", slog.String("env", cfg.HTTP.Env))
	return worker.Run(ctx, redisOpt(cfg), processor, log)
}
