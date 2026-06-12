package main

import (
	"context"
	"log/slog"
	"os"

	twittersim "github.com/Akins20/postal/internal/publish/simulator/twitter"
)

// defaultSimAddr is where `postal sim` listens unless POSTAL_X_SIM_ADDR is set.
const defaultSimAddr = "127.0.0.1:10090"

// runSim runs the X/Twitter API simulator as a standalone process for local
// dev and e2e runs. Point the server at it with:
//
//	POSTAL_X_API_BASE_URL=http://127.0.0.1:10090
//	POSTAL_X_AUTH_BASE_URL=http://127.0.0.1:10090
//
// It serves the OAuth authorize/token endpoints plus tweets/users/media, and
// blocks until the context is canceled (Ctrl-C).
func runSim(ctx context.Context, log *slog.Logger) error {
	addr := os.Getenv("POSTAL_X_SIM_ADDR")
	if addr == "" {
		addr = defaultSimAddr
	}
	sim, err := twittersim.NewAt(addr)
	if err != nil {
		return err
	}
	defer sim.Close()
	log.Info("x simulator listening", "url", sim.URL())
	<-ctx.Done()
	log.Info("x simulator shutting down")
	return nil
}
