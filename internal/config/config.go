// Package config loads Postal's runtime configuration from environment
// variables into a typed, validated struct read once at startup.
//
// Configuration is intentionally read in exactly one place (Load) and passed
// down by value/reference; no other package calls os.Getenv. This keeps
// dependencies explicit and the process free of hidden global state.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default configuration values. Named here rather than inlined so there are no
// magic literals scattered through the loader.
const (
	defaultHTTPAddr        = ":8080"
	defaultEnv             = "development"
	defaultLogLevel        = "info"
	defaultShutdownTimeout = 15 * time.Second
	defaultRequestTimeout  = 30 * time.Second
	defaultRedisAddr       = "localhost:6379"
	defaultRedisDB         = 0

	defaultAccessTokenTTL     = 15 * time.Minute
	defaultRefreshTokenTTL    = 30 * 24 * time.Hour // sliding window per use
	defaultRefreshTokenMaxTTL = 90 * 24 * time.Hour // absolute session cap
	defaultCookieSecure       = true

	defaultStorageBucket     = "postal-media"
	defaultMaxUploadBytes    = 512 << 20 // 512 MiB per file
	defaultMaxWorkspaceBytes = 2 << 30   // 2 GiB per workspace
)

// envPrefix namespaces every Postal environment variable.
const envPrefix = "POSTAL_"

// Config is the fully resolved, validated runtime configuration.
type Config struct {
	HTTP         HTTP
	DB           DB
	Redis        Redis
	Crypto       Crypto
	Auth         Auth
	Twitter      Twitter
	OAuth        OAuth
	Instagram    Instagram
	TikTok       TikTok
	Storage      Storage
	Billing      Billing
	Integrations Integrations
	Mail         Mail
}

// Mail holds transactional email settings. Postal sends auth mail (account
// verification, password reset) exclusively through Resend (resend.com). When
// APIKey or From is empty no mailer is built and production refuses to start
// auth (cmd/postal/serve.go); development falls back to the console mailer.
// AppBaseURL is the public web base used to build the action links inside those
// emails. APIKey is never logged.
type Mail struct {
	APIKey     string
	From       string
	AppBaseURL string
}

// Configured reports whether enough Resend settings are present to send mail.
func (m Mail) Configured() bool {
	return m.APIKey != "" && m.From != ""
}

// Integrations holds third-party integration settings. OGShortenerAPIBase
// overrides the live ogshortener.site host (tests/dev); blank = real service.
type Integrations struct {
	OGShortenerAPIBase string
}

// Billing holds the wallet economics and payment-provider credentials
// (Phase 13; see docs/BILLING_PLAN.md). A provider with a blank secret key is
// disabled. X/Twitter is the only platform with a publish cost.
type Billing struct {
	CreditsPerUSDCent       int64
	PublishCostTwitter      int64
	PublishCostTwitterMedia int64
	PublishCostTwitterURL   int64
	MinTopupCredits         int64
	NGNPerUSD               int64
	ReturnURL               string
	StripeSecretKey         string
	StripeWebhookSecret     string
	StripeAPIBase           string
	PaystackSecretKey       string
	PaystackAPIBase         string
}

// OAuth holds cross-provider OAuth settings. AllowedRedirects is the allowlist
// of client-supplied callback URIs (web page + native deep links); a connect
// request may only override the adapter default with a URI in this set.
type OAuth struct {
	AllowedRedirects []string
}

// Twitter holds the X/Twitter OAuth app credentials. When ClientID is empty the
// X adapter is not registered (channels for X are disabled). APIBaseURL and
// AuthBaseURL override the real X hosts so local dev/e2e can point the adapter
// at the simulator (`postal sim`); empty means the real API.
type Twitter struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
	AuthBaseURL  string
}

// Instagram holds the Meta app credentials for the Instagram adapter. When
// ClientID is empty the adapter is not registered. Base URLs override the
// real Meta hosts (dev/tests point at the simulator).
type Instagram struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
	AuthBaseURL  string
}

// TikTok holds the TikTok app credentials for the TikTok adapter. When
// ClientKey is empty the adapter is not registered.
type TikTok struct {
	ClientKey    string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
	AuthBaseURL  string
}

// Storage holds S3-compatible object-storage settings for the media pipeline.
// Production = Cloudflare R2 (endpoint <account>.r2.cloudflarestorage.com,
// Region "auto", UseSSL true); local dev = MinIO.
type Storage struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
	// MaxUploadBytes is the per-file upload cap.
	MaxUploadBytes int64
	// MaxWorkspaceBytes is the per-workspace storage quota.
	MaxWorkspaceBytes int64
}

// Auth holds authentication and session settings. JWTSecret is validated by the
// auth domain at construction (not here) so the server can still run scaffolding
// roles without it configured.
type Auth struct {
	// JWTSecret signs HS256 access tokens. Required before auth is used; never logged.
	JWTSecret string
	// AccessTokenTTL is the lifetime of a JWT access token.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is the sliding lifetime of a refresh token, extended on each use.
	RefreshTokenTTL time.Duration
	// RefreshTokenMaxTTL caps total session lifetime regardless of sliding.
	RefreshTokenMaxTTL time.Duration
	// CookieDomain scopes auth cookies (empty = host-only).
	CookieDomain string
	// CookieSecure sets the Secure flag on auth cookies (disable only for local http).
	CookieSecure bool
}

// HTTP holds API server settings.
type HTTP struct {
	// Addr is the bind address for the API server (e.g. ":8080").
	Addr string
	// Env is the deployment environment: development, staging, or production.
	Env string
	// LogLevel is the minimum slog level: debug, info, warn, or error.
	LogLevel string
	// ShutdownTimeout bounds graceful shutdown.
	ShutdownTimeout time.Duration
	// RequestTimeout bounds the lifetime of a single request handler.
	RequestTimeout time.Duration
	// AllowedOrigins is the CORS allowlist (exact origins). Empty disables CORS
	// (no cross-origin browser access), appropriate for same-origin or native
	// clients. "*" is intentionally NOT supported with credentialed requests.
	AllowedOrigins []string
}

// DB holds PostgreSQL connection settings.
type DB struct {
	// URL is the pgx-compatible connection DSN.
	URL string
}

// Redis holds Redis connection settings (asynq broker, rate-limit counters, cache).
type Redis struct {
	// Addr is the host:port of the Redis server.
	Addr string
	// Password is the Redis auth password (empty if unset).
	Password string
	// DB is the Redis logical database number.
	DB int
}

// Crypto holds secrets-handling configuration.
type Crypto struct {
	// MasterKey is the base64-encoded 32-byte key for AES-256-GCM envelope
	// encryption. It is optional during early scaffolding but REQUIRED before
	// any token-vault feature (Phase 1+). It is never logged.
	MasterKey string
}

// IsProduction reports whether the server is running in the production environment.
func (h HTTP) IsProduction() bool {
	return strings.EqualFold(h.Env, "production")
}

// Load reads configuration from the environment, applies defaults, and
// validates it. It returns an error (never panics) so main can decide how to
// fail. All variables use the POSTAL_ prefix.
func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTP{
			Addr:            getString("HTTP_ADDR", defaultHTTPAddr),
			Env:             getString("ENV", defaultEnv),
			LogLevel:        getString("LOG_LEVEL", defaultLogLevel),
			ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
			RequestTimeout:  getDuration("REQUEST_TIMEOUT", defaultRequestTimeout),
			AllowedOrigins:  getStringSlice("CORS_ALLOWED_ORIGINS"),
		},
		DB: DB{
			URL: getString("DATABASE_URL", ""),
		},
		Redis: Redis{
			Addr:     getString("REDIS_ADDR", defaultRedisAddr),
			Password: getString("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", defaultRedisDB),
		},
		Crypto: Crypto{
			MasterKey: getString("MASTER_KEY", ""),
		},
		Auth: Auth{
			JWTSecret:          getString("JWT_SECRET", ""),
			AccessTokenTTL:     getDuration("ACCESS_TOKEN_TTL", defaultAccessTokenTTL),
			RefreshTokenTTL:    getDuration("REFRESH_TOKEN_TTL", defaultRefreshTokenTTL),
			RefreshTokenMaxTTL: getDuration("REFRESH_TOKEN_MAX_TTL", defaultRefreshTokenMaxTTL),
			CookieDomain:       getString("COOKIE_DOMAIN", ""),
			CookieSecure:       getBool("COOKIE_SECURE", defaultCookieSecure),
		},
		OAuth: OAuth{
			AllowedRedirects: getStringSlice("OAUTH_ALLOWED_REDIRECTS"),
		},
		Twitter: Twitter{
			ClientID:     getString("X_CLIENT_ID", ""),
			ClientSecret: getString("X_CLIENT_SECRET", ""),
			RedirectURI:  getString("X_REDIRECT_URI", ""),
			APIBaseURL:   getString("X_API_BASE_URL", ""),
			AuthBaseURL:  getString("X_AUTH_BASE_URL", ""),
		},
		Instagram: Instagram{
			ClientID:     getString("IG_CLIENT_ID", ""),
			ClientSecret: getString("IG_CLIENT_SECRET", ""),
			RedirectURI:  getString("IG_REDIRECT_URI", ""),
			APIBaseURL:   getString("IG_API_BASE_URL", ""),
			AuthBaseURL:  getString("IG_AUTH_BASE_URL", ""),
		},
		TikTok: TikTok{
			ClientKey:    getString("TIKTOK_CLIENT_KEY", ""),
			ClientSecret: getString("TIKTOK_CLIENT_SECRET", ""),
			RedirectURI:  getString("TIKTOK_REDIRECT_URI", ""),
			APIBaseURL:   getString("TIKTOK_API_BASE_URL", ""),
			AuthBaseURL:  getString("TIKTOK_AUTH_BASE_URL", ""),
		},
		Integrations: Integrations{
			OGShortenerAPIBase: getString("OGSHORTENER_API_BASE", ""),
		},
		Mail: Mail{
			APIKey:     getString("RESEND_API_KEY", ""),
			From:       getString("MAIL_FROM", ""),
			AppBaseURL: getString("APP_BASE_URL", ""),
		},
		Billing: Billing{
			CreditsPerUSDCent:       getInt64("BILLING_CREDITS_PER_USD_CENT", 1),
			PublishCostTwitter:      getInt64("BILLING_PUBLISH_COST_TWITTER", 10),
			PublishCostTwitterMedia: getInt64("BILLING_PUBLISH_COST_TWITTER_MEDIA", 15),
			PublishCostTwitterURL:   getInt64("BILLING_PUBLISH_COST_TWITTER_URL", 25),
			MinTopupCredits:         getInt64("BILLING_MIN_TOPUP_CREDITS", 500),
			NGNPerUSD:               getInt64("PAYSTACK_NGN_PER_USD", 1600),
			ReturnURL:               getString("BILLING_RETURN_URL", "http://localhost:3000/wallet"),
			StripeSecretKey:         getString("STRIPE_SECRET_KEY", ""),
			StripeWebhookSecret:     getString("STRIPE_WEBHOOK_SECRET", ""),
			StripeAPIBase:           getString("STRIPE_API_BASE", ""),
			PaystackSecretKey:       getString("PAYSTACK_SECRET_KEY", ""),
			PaystackAPIBase:         getString("PAYSTACK_API_BASE", ""),
		},
		Storage: Storage{
			Endpoint:          getString("STORAGE_ENDPOINT", ""),
			AccessKey:         getString("STORAGE_ACCESS_KEY", ""),
			SecretKey:         getString("STORAGE_SECRET_KEY", ""),
			Bucket:            getString("STORAGE_BUCKET", defaultStorageBucket),
			Region:            getString("STORAGE_REGION", ""),
			UseSSL:            getBool("STORAGE_USE_SSL", false),
			MaxUploadBytes:    getInt64("STORAGE_MAX_UPLOAD_BYTES", defaultMaxUploadBytes),
			MaxWorkspaceBytes: getInt64("STORAGE_MAX_WORKSPACE_BYTES", defaultMaxWorkspaceBytes),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// validate enforces invariants that must hold for the server to start.
func (c Config) validate() error {
	if c.HTTP.Addr == "" {
		return fmt.Errorf("config: %sHTTP_ADDR must not be empty", envPrefix)
	}
	if c.DB.URL == "" {
		return fmt.Errorf("config: %sDATABASE_URL is required", envPrefix)
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("config: %sREDIS_ADDR is required", envPrefix)
	}
	switch c.HTTP.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("config: %sLOG_LEVEL %q is invalid (want debug|info|warn|error)", envPrefix, c.HTTP.LogLevel)
	}
	return nil
}

// getString returns the value of POSTAL_<key>, or def if unset/empty.
func getString(key, def string) string {
	if v, ok := os.LookupEnv(envPrefix + key); ok && v != "" {
		return v
	}
	return def
}

// getStringSlice returns POSTAL_<key> split on commas (trimmed, blanks dropped),
// or nil if unset/empty.
func getStringSlice(key string) []string {
	v, ok := os.LookupEnv(envPrefix + key)
	if !ok || v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// getBool returns the boolean value of POSTAL_<key>, or def if unset or unparsable.
func getBool(key string, def bool) bool {
	v, ok := os.LookupEnv(envPrefix + key)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// getInt64 returns the int64 value of POSTAL_<key>, or def if unset or unparsable.
func getInt64(key string, def int64) int64 {
	v, ok := os.LookupEnv(envPrefix + key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// getInt returns the integer value of POSTAL_<key>, or def if unset or unparsable.
func getInt(key string, def int) int {
	v, ok := os.LookupEnv(envPrefix + key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// getDuration returns the duration value of POSTAL_<key>, or def if unset or unparsable.
func getDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(envPrefix + key)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
