package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Akins20/postal/internal/analytics"
	"github.com/Akins20/postal/internal/auth"
	"github.com/Akins20/postal/internal/billing"
	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/config"
	"github.com/Akins20/postal/internal/media"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/metrics"
	"github.com/Akins20/postal/internal/platform/redis"
	"github.com/Akins20/postal/internal/platform/storage"
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
	mediaSvc *media.Service // nil when storage is not configured
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
		Production:     cfg.HTTP.IsProduction(),
		AllowedOrigins: cfg.HTTP.AllowedOrigins,
	}
	mediaSvc, err := buildMedia(ctx, cfg, pool, w.auditor, log)
	if err != nil {
		return err
	}
	w.mediaSvc = mediaSvc

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

	// Analytics reporting is read-only on the API server (the worker owns the
	// poller), so it needs no metrics fetcher — only the pool + workspace service.
	deps.AnalyticsHandler = analytics.NewHandler(analytics.NewService(w.pool, nil, w.auditor, nil), w.wsSvc, w.log)
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

	// Keep media interfaces nil (not a typed-nil) when storage is disabled, so the
	// composer/scheduler correctly detect its absence.
	var mediaResolver post.MediaResolver
	var mediaLoader schedule.MediaLoader
	if w.mediaSvc != nil {
		mediaResolver, mediaLoader = w.mediaSvc, w.mediaSvc
		deps.MediaHandler = media.NewHandler(w.mediaSvc, w.wsSvc, w.log, w.cfg.Storage.MaxUploadBytes)
	}

	// The composer resolves channels via channelSvc, validates variants against
	// the platform adapters (publish.Registry), and resolves attached media.
	postSvc := post.NewService(w.pool, channelSvc, publish.NewRegistry(adapters...), mediaResolver, w.auditor, nil)
	deps.PostHandler = post.NewHandler(postSvc, w.wsSvc, w.log)

	// Wallet billing (Phase 13): X is pay-per-use; scheduling gets the
	// affordability soft gate, the worker the hard charge.
	billingSvc := buildBilling(w.cfg, w.pool, w.log)
	deps.BillingHandler = newBillingHandler(w.cfg, billingSvc, w.wsSvc, w.log)

	// Scheduling: enqueue publish tasks to asynq; the worker process executes
	// them. The asynq client satisfies schedule.Enqueuer; closed by runServe.
	w.enqueuer = worker.NewClient(redisOpt(w.cfg))
	scheduleSvc := schedule.NewService(w.pool, channelSvc, w.enqueuer, mediaLoader, w.auditor, billingSvc, nil)
	deps.ScheduleHandler = schedule.NewHandler(scheduleSvc, w.wsSvc, w.log)
}

// buildBilling constructs the wallet service with whichever payment providers
// are configured. The dev provider (instant free credits) registers ONLY in
// development. Shared by server and worker wiring.
func buildBilling(cfg config.Config, pool *db.Pool, log *slog.Logger) *billing.Service {
	pricing := billing.Pricing{
		CreditsPerUSDCent: cfg.Billing.CreditsPerUSDCent,
		PublishCosts:      map[string]int64{"twitter": cfg.Billing.PublishCostTwitter},
		MinTopupCredits:   cfg.Billing.MinTopupCredits,
		NGNPerUSD:         cfg.Billing.NGNPerUSD,
		ReturnURL:         cfg.Billing.ReturnURL,
	}
	providers := map[string]billing.Provider{}
	if cfg.Billing.StripeSecretKey != "" {
		providers["stripe"] = billing.NewStripeProvider(
			cfg.Billing.StripeSecretKey, cfg.Billing.StripeWebhookSecret, cfg.Billing.StripeAPIBase, nil, nil)
	}
	if cfg.Billing.PaystackSecretKey != "" {
		providers["paystack"] = billing.NewPaystackProvider(
			cfg.Billing.PaystackSecretKey, cfg.Billing.PaystackAPIBase, cfg.Billing.NGNPerUSD, nil)
	}
	svc := billing.NewService(pool, pricing, providers, log)
	if !cfg.HTTP.IsProduction() {
		providers["dev"] = billing.NewDevProvider(svc.Credit)
		log.Warn("dev payment provider enabled (instant credits) — development only")
	}
	return svc
}

// newBillingHandler builds the billing HTTP handler with the webhook verifiers
// for whichever providers are configured.
func newBillingHandler(cfg config.Config, svc *billing.Service, wsSvc *workspace.Service, log *slog.Logger) *billing.Handler {
	var stripe *billing.StripeProvider
	if cfg.Billing.StripeSecretKey != "" {
		stripe = billing.NewStripeProvider(
			cfg.Billing.StripeSecretKey, cfg.Billing.StripeWebhookSecret, cfg.Billing.StripeAPIBase, nil, nil)
	}
	var paystack *billing.PaystackProvider
	if cfg.Billing.PaystackSecretKey != "" {
		paystack = billing.NewPaystackProvider(
			cfg.Billing.PaystackSecretKey, cfg.Billing.PaystackAPIBase, cfg.Billing.NGNPerUSD, nil)
	}
	return billing.NewHandler(svc, wsSvc, stripe, paystack, log)
}

// buildMedia constructs the media service over object storage. When the storage
// endpoint is unset, media uploads are disabled (returns nil).
func buildMedia(ctx context.Context, cfg config.Config, pool *db.Pool, auditor *security.Auditor, log *slog.Logger) (*media.Service, error) {
	if cfg.Storage.Endpoint == "" {
		log.Warn("POSTAL_STORAGE_ENDPOINT not set; media uploads are disabled")
		return nil, nil
	}
	store, err := storage.New(ctx, storage.Config{
		Endpoint: cfg.Storage.Endpoint, AccessKey: cfg.Storage.AccessKey, SecretKey: cfg.Storage.SecretKey,
		Bucket: cfg.Storage.Bucket, Region: cfg.Storage.Region, UseSSL: cfg.Storage.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to object storage: %w", err)
	}
	return media.NewService(pool, store, auditor, cfg.Storage.MaxUploadBytes, cfg.Storage.MaxWorkspaceBytes, nil), nil
}

// buildAdapters constructs the platform adapters from config. The X adapter is
// included only when its app credentials are configured. Shared by server and
// worker wiring.
func buildAdapters(cfg config.Config, log *slog.Logger) []publish.Adapter {
	if cfg.Twitter.ClientID == "" {
		log.Warn("POSTAL_X_CLIENT_ID not set; X/Twitter is disabled")
		return nil
	}
	if cfg.Twitter.APIBaseURL != "" {
		log.Warn("X adapter base URLs overridden (simulator?)", "api", cfg.Twitter.APIBaseURL)
	}
	return []publish.Adapter{twitter.New(twitter.Config{
		ClientID:     cfg.Twitter.ClientID,
		ClientSecret: cfg.Twitter.ClientSecret,
		RedirectURI:  cfg.Twitter.RedirectURI,
		APIBaseURL:   cfg.Twitter.APIBaseURL,
		AuthBaseURL:  cfg.Twitter.AuthBaseURL,
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
