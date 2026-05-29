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
)

// envPrefix namespaces every Postal environment variable.
const envPrefix = "POSTAL_"

// Config is the fully resolved, validated runtime configuration.
type Config struct {
	HTTP   HTTP
	DB     DB
	Redis  Redis
	Crypto Crypto
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
