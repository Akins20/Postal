// Command postal is the single binary for the Postal platform. It runs in one
// of two roles selected by subcommand:
//
//	postal serve    # the HTTP API server
//	postal worker   # the asynq job worker (scheduling, refresh, analytics)
//
// Both roles share the same image and configuration; production runs them as
// separate processes.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Akins20/postal/internal/config"
	"github.com/Akins20/postal/internal/platform"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

// run parses the subcommand, loads config, and dispatches to the selected role.
// It returns an error rather than calling os.Exit so deferred cleanup runs.
func run() error {
	if len(os.Args) < 2 {
		usage()
		return fmt.Errorf("missing subcommand")
	}
	subcommand := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	log := platform.NewLogger(cfg.HTTP.LogLevel, cfg.HTTP.IsProduction())

	// Cancel the root context on SIGINT/SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch subcommand {
	case "serve":
		return runServe(ctx, cfg, log)
	case "worker":
		return runWorker(ctx, cfg, log)
	case "sim":
		return runSim(ctx, log)
	case "-h", "--help", "help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", subcommand)
	}
}

// usage prints the available subcommands to stderr.
func usage() {
	fmt.Fprint(os.Stderr, `postal — social media scheduling & publishing platform

usage:
  postal serve     run the HTTP API server
  postal worker    run the asynq job worker
  postal sim       run the X/Twitter API simulator (local dev/e2e)
  postal help      show this message
`)
}
