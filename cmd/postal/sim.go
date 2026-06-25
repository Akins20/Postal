package main

import (
	"context"
	"log/slog"
	"os"

	facebooksim "github.com/Akins20/postal/internal/publish/simulator/facebook"
	instagramsim "github.com/Akins20/postal/internal/publish/simulator/instagram"
	tiktoksim "github.com/Akins20/postal/internal/publish/simulator/tiktok"
	twittersim "github.com/Akins20/postal/internal/publish/simulator/twitter"
)

// Default simulator addresses; override with POSTAL_<X|IG|FB|TIKTOK>_SIM_ADDR.
const (
	defaultXSimAddr      = "127.0.0.1:10090"
	defaultIGSimAddr     = "127.0.0.1:10091"
	defaultTikTokSimAddr = "127.0.0.1:10092"
	defaultFBSimAddr     = "127.0.0.1:10093"
)

// runSim runs the platform API simulators as one process for local dev and
// e2e: X, Instagram (Meta Graph), and TikTok. Point the server at them with
// the matching POSTAL_*_API_BASE_URL / POSTAL_*_AUTH_BASE_URL overrides.
// Blocks until the context is canceled (Ctrl-C).
func runSim(ctx context.Context, log *slog.Logger) error {
	x, err := twittersim.NewAt(envOrDefault("POSTAL_X_SIM_ADDR", defaultXSimAddr))
	if err != nil {
		return err
	}
	defer x.Close()
	log.Info("x simulator listening", "url", x.URL())

	ig, err := instagramsim.NewAt(envOrDefault("POSTAL_IG_SIM_ADDR", defaultIGSimAddr))
	if err != nil {
		return err
	}
	defer ig.Close()
	log.Info("instagram simulator listening", "url", ig.URL())

	tt, err := tiktoksim.NewAt(envOrDefault("POSTAL_TIKTOK_SIM_ADDR", defaultTikTokSimAddr))
	if err != nil {
		return err
	}
	defer tt.Close()
	log.Info("tiktok simulator listening", "url", tt.URL())

	fb, err := facebooksim.NewAt(envOrDefault("POSTAL_FB_SIM_ADDR", defaultFBSimAddr))
	if err != nil {
		return err
	}
	defer fb.Close()
	log.Info("facebook simulator listening", "url", fb.URL())

	<-ctx.Done()
	log.Info("simulators shutting down")
	return nil
}

// envOrDefault returns the env value or def. Local to the sim command (the
// main config loader owns all server configuration).
func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
