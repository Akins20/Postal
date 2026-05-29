package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Akins20/postal/internal/auth"
	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/config"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/metrics"
	"github.com/Akins20/postal/internal/platform/redis"
	"github.com/Akins20/postal/internal/post"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/publish/twitter"
	"github.com/Akins20/postal/internal/ratelimit"
	"github.com/Akins20/postal/internal/schedule"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/server"
	"github.com/Akins20/postal/internal/worker"
	"github.com/Akins20/postal/internal/workspace"

	"github.com/hibiken/asynq"
)

// wiring holds the shared dependencies used to construct the domain stacks.
type wiring struct {
	cfg      config.Config
	pool     *db.Pool
	cache    *redis.Client
	limiter  *ratelimit.Limiter
	auditor  *security.Auditor
	enc      *security.Encryptor
	wsSvc    *workspace.Service
	enqueuer *worker.Client // asynq client; closed by runServe
	log      *slog.Logger
}

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

	enc, err := initEncryptor(cfg, log)
	if err != nil {
		return err
	}

	w := &wiring{
		cfg:     cfg,
		pool:    pool,
		cache:   cache,
		limiter: ratelimit.NewLimiter(cache, nil),
		auditor: security.NewAuditor(pool.Queries(), log),
		enc:     enc,
		wsSvc:   workspace.NewService(pool, security.NewAuditor(pool.Queries(), log), nil),
		log:     log,
	}

	deps := server.Deps{
		Logger:         log,
		DB:             pool,
		Redis:          cache,
		Metrics:        metrics.New(),
		Limiter:        w.limiter,
		RequestTimeout: cfg.HTTP.RequestTimeout,
	}
	if err := w.wireAuth(&deps); err != nil {
		return err
	}
	w.wireChannels(&deps)
	if w.enqueuer != nil {
		defer func() { _ = w.enqueuer.Close() }()
	}

	srv := server.New(cfg.HTTP.Addr, deps)
	log.Info("starting postal api server", slog.String("env", cfg.HTTP.Env))
	return srv.Start(ctx, cfg.HTTP.ShutdownTimeout)
}

// wireAuth constructs the auth and workspace stack and attaches it to deps. With
// no JWT secret configured, auth endpoints are disabled and the server still
// serves health, metrics, and ping.
func (w *wiring) wireAuth(deps *server.Deps) error {
	if w.cfg.Auth.JWTSecret == "" {
		w.log.Warn("POSTAL_JWT_SECRET not set; auth endpoints are disabled")
		return nil
	}

	tokens, err := auth.NewTokenIssuer(w.cfg.Auth.JWTSecret, w.cfg.Auth.AccessTokenTTL, nil)
	if err != nil {
		return fmt.Errorf("building token issuer: %w", err)
	}
	sessions, err := auth.NewSessionStore(w.cache, w.cfg.Auth.RefreshTokenTTL, w.cfg.Auth.RefreshTokenMaxTTL, nil)
	if err != nil {
		return fmt.Errorf("building session store: %w", err)
	}

	// The console mailer logs verification/reset tokens (single-use account
	// secrets), acceptable only for local development. Refuse it in production.
	if w.cfg.HTTP.IsProduction() {
		return fmt.Errorf("no production mailer configured: refusing to start auth with the dev console mailer in production")
	}
	authSvc := auth.NewService(w.pool, tokens, sessions, auth.NewConsoleMailer(w.log), w.auditor, nil)

	deps.Tokens = tokens
	deps.AuthHandler = auth.NewHandler(auth.HandlerConfig{
		Service:    authSvc,
		Tokens:     tokens,
		Limiter:    w.limiter,
		Cookies:    auth.CookieSettings{Domain: w.cfg.Auth.CookieDomain, Secure: w.cfg.Auth.CookieSecure},
		Logger:     w.log,
		AccessTTL:  w.cfg.Auth.AccessTokenTTL,
		RefreshTTL: w.cfg.Auth.RefreshTokenTTL,
	})
	deps.WorkspaceHandler = workspace.NewHandler(w.wsSvc, w.log)
	return nil
}

// wireChannels constructs the channel/OAuth-vault stack AND the composer (posts),
// which share the platform adapters and the workspace service. Both require the
// encryption key (to seal tokens) and the auth/workspace stack; absent either,
// they are disabled.
func (w *wiring) wireChannels(deps *server.Deps) {
	if w.enc == nil {
		w.log.Warn("POSTAL_MASTER_KEY not set; channel/composer disabled (token vault unavailable)")
		return
	}
	if deps.WorkspaceHandler == nil {
		w.log.Warn("auth/workspace disabled; channel/composer endpoints disabled")
		return
	}

	adapters := buildAdapters(w.cfg, w.log)
	channelSvc := channel.NewService(w.pool, channel.NewRegistry(toOAuthProviders(adapters)...), w.enc, w.cache, w.wsSvc, w.auditor, nil)
	deps.ChannelHandler = channel.NewHandler(channelSvc, w.wsSvc, w.log)

	// The composer resolves channels via channelSvc and validates variants
	// against the platform adapters (publish.Registry).
	postSvc := post.NewService(w.pool, channelSvc, publish.NewRegistry(adapters...), w.auditor, nil)
	deps.PostHandler = post.NewHandler(postSvc, w.wsSvc, w.log)

	// Scheduling: enqueue publish tasks to asynq; the worker process executes
	// them. The asynq client satisfies schedule.Enqueuer; closed by runServe.
	w.enqueuer = worker.NewClient(redisOpt(w.cfg))
	scheduleSvc := schedule.NewService(w.pool, channelSvc, w.enqueuer, w.auditor, nil)
	deps.ScheduleHandler = schedule.NewHandler(scheduleSvc, w.wsSvc, w.log)
}

// buildAdapters constructs the platform adapters from config. The X adapter is
// included only when its app credentials are configured. Shared by server and
// worker wiring.
func buildAdapters(cfg config.Config, log *slog.Logger) []publish.Adapter {
	if cfg.Twitter.ClientID == "" {
		log.Warn("POSTAL_X_CLIENT_ID not set; X/Twitter is disabled")
		return nil
	}
	return []publish.Adapter{twitter.New(twitter.Config{
		ClientID:     cfg.Twitter.ClientID,
		ClientSecret: cfg.Twitter.ClientSecret,
		RedirectURI:  cfg.Twitter.RedirectURI,
	})}
}

// redisOpt builds the asynq Redis options from config.
func redisOpt(cfg config.Config) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB}
}

// toOAuthProviders views the publish adapters as channel OAuth providers (each
// adapter embeds the OAuthProvider interface).
func toOAuthProviders(adapters []publish.Adapter) []channel.OAuthProvider {
	out := make([]channel.OAuthProvider, len(adapters))
	for i, a := range adapters {
		out[i] = a
	}
	return out
}

// initEncryptor validates the configured master key and builds the encryptor. A
// present-but-invalid key is fatal; an absent key disables token features.
func initEncryptor(cfg config.Config, log *slog.Logger) (*security.Encryptor, error) {
	if cfg.Crypto.MasterKey == "" {
		log.Warn("POSTAL_MASTER_KEY not set; token-vault features are disabled until configured")
		return nil, nil
	}
	enc, err := security.NewEncryptorFromSpec(cfg.Crypto.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("invalid POSTAL_MASTER_KEY: %w", err)
	}
	log.Info("encryption keyring loaded", slog.Int("current_key_version", int(enc.CurrentVersion())))
	return enc, nil
}
