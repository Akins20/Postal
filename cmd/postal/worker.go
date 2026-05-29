package main

import (
	"context"
	"log/slog"

	"github.com/Akins20/postal/internal/config"
)

// runWorker is the entrypoint for the asynq job worker role. The job-processing
// engine (publish, token refresh, analytics polls, abuse sweeps) is built in
// Phase 6; for now the role connects nothing and idles until canceled so the
// subcommand and its lifecycle are wired and testable from Phase 0.
//
// TODO(postal-phase6): wire asynq server, Redis broker, and task handlers here.
//
//nolint:unparam // error return is part of the run* dispatch contract (mirrors runServe) and becomes non-nil once Phase 6 wires asynq.
func runWorker(ctx context.Context, _ config.Config, log *slog.Logger) error {
	log.Info("worker role started (no handlers until Phase 6); waiting for shutdown signal")
	<-ctx.Done()
	log.Info("worker shutting down")
	return nil
}
