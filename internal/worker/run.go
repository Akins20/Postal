package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// refreshCron is the periodic schedule for the token-refresh sweep.
const refreshCron = "@every 30m"

// workerConcurrency bounds concurrent task processing.
const workerConcurrency = 10

// Run starts the asynq server and the periodic scheduler, processing tasks until
// ctx is canceled, then shuts both down gracefully.
func Run(ctx context.Context, redis asynq.RedisClientOpt, proc *Processor, log *slog.Logger) error {
	srv := asynq.NewServer(redis, asynq.Config{Concurrency: workerConcurrency})
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypePublish, proc.ProcessPublish)
	mux.HandleFunc(TypeRefreshTokens, proc.ProcessRefreshTokens)

	if err := srv.Start(mux); err != nil {
		return fmt.Errorf("starting asynq server: %w", err)
	}

	scheduler := asynq.NewScheduler(redis, nil)
	if _, err := scheduler.Register(refreshCron, asynq.NewTask(TypeRefreshTokens, nil)); err != nil {
		srv.Shutdown()
		return fmt.Errorf("registering refresh schedule: %w", err)
	}
	if err := scheduler.Start(); err != nil {
		srv.Shutdown()
		return fmt.Errorf("starting scheduler: %w", err)
	}

	log.Info("worker started", slog.Int("concurrency", workerConcurrency))
	<-ctx.Done()
	log.Info("worker shutting down")
	scheduler.Shutdown()
	srv.Shutdown()
	return nil
}
