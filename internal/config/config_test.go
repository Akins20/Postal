package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// setEnv sets a POSTAL_-prefixed variable for the duration of the test.
func setEnv(t *testing.T, key, val string) {
	t.Helper()
	t.Setenv(envPrefix+key, val)
}

// clearEnv neutralizes every POSTAL_-prefixed variable already present in the
// environment so the test runs hermetically regardless of the ambient shell
// (e.g. `make` exporting a developer's .env). The loader treats empty as unset,
// and t.Setenv restores originals at test end.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, _, ok := strings.Cut(kv, "="); ok && strings.HasPrefix(k, envPrefix) {
			t.Setenv(k, "")
		}
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	clearEnv(t)
	// Only the required vars are set; everything else should fall back to defaults.
	setEnv(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.HTTP.Addr != defaultHTTPAddr {
		t.Errorf("HTTP.Addr = %q, want default %q", cfg.HTTP.Addr, defaultHTTPAddr)
	}
	if cfg.HTTP.ShutdownTimeout != defaultShutdownTimeout {
		t.Errorf("HTTP.ShutdownTimeout = %v, want default %v", cfg.HTTP.ShutdownTimeout, defaultShutdownTimeout)
	}
	if cfg.Redis.Addr != defaultRedisAddr {
		t.Errorf("Redis.Addr = %q, want default %q", cfg.Redis.Addr, defaultRedisAddr)
	}
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://u:p@db:5432/postal")
	setEnv(t, "HTTP_ADDR", ":9090")
	setEnv(t, "LOG_LEVEL", "debug")
	setEnv(t, "SHUTDOWN_TIMEOUT", "5s")
	setEnv(t, "REDIS_DB", "3")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.HTTP.Addr != ":9090" {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":9090")
	}
	if cfg.HTTP.LogLevel != "debug" {
		t.Errorf("HTTP.LogLevel = %q, want %q", cfg.HTTP.LogLevel, "debug")
	}
	if cfg.HTTP.ShutdownTimeout != 5*time.Second {
		t.Errorf("HTTP.ShutdownTimeout = %v, want %v", cfg.HTTP.ShutdownTimeout, 5*time.Second)
	}
	if cfg.Redis.DB != 3 {
		t.Errorf("Redis.DB = %d, want %d", cfg.Redis.DB, 3)
	}
}

func TestLoad_MissingDatabaseURLFails(t *testing.T) {
	clearEnv(t)
	// DATABASE_URL is required; with it unset, Load must error.
	cfg, err := Load()
	if err == nil {
		t.Fatalf("Load() = %+v, want error for missing DATABASE_URL", cfg)
	}
}

func TestLoad_InvalidLogLevelFails(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db")
	setEnv(t, "LOG_LEVEL", "verbose")

	if _, err := Load(); err == nil {
		t.Fatal("Load() = nil error, want error for invalid LOG_LEVEL")
	}
}

func TestLoad_InvalidDurationFallsBackToDefault(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db")
	setEnv(t, "REQUEST_TIMEOUT", "not-a-duration")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.HTTP.RequestTimeout != defaultRequestTimeout {
		t.Errorf("RequestTimeout = %v, want default %v after unparsable value", cfg.HTTP.RequestTimeout, defaultRequestTimeout)
	}
}

func TestHTTP_IsProduction(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "production", env: "production", want: true},
		{name: "production mixed case", env: "Production", want: true},
		{name: "development", env: "development", want: false},
		{name: "empty", env: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HTTP{Env: tt.env}
			if got := h.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}
