package auth_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Akins20/postal/internal/auth"
	"github.com/Akins20/postal/internal/platform/db"
)

// capturingMailer records the last token so the test can complete email flows.
type capturingMailer struct{ verifyToken, resetToken string }

func (m *capturingMailer) SendEmailVerification(_ context.Context, _, token string) error {
	m.verifyToken = token
	return nil
}

func (m *capturingMailer) SendPasswordReset(_ context.Context, _, token string) error {
	m.resetToken = token
	return nil
}

// newService wires a real auth.Service against Postgres + Redis, skipping when
// either is unconfigured/unreachable.
func newService(t *testing.T) (*auth.Service, *capturingMailer) {
	t.Helper()
	dsn := os.Getenv("POSTAL_DATABASE_URL")
	addr := os.Getenv("POSTAL_REDIS_ADDR")
	if dsn == "" || addr == "" {
		t.Skip("POSTAL_DATABASE_URL/POSTAL_REDIS_ADDR not set; skipping integration test")
	}
	ctx := context.Background()

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)

	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("POSTAL_REDIS_PASSWORD")})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unreachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	tokens, err := auth.NewTokenIssuer("integration-test-secret", 15*time.Minute, nil)
	if err != nil {
		t.Fatalf("token issuer: %v", err)
	}
	sessions, err := auth.NewSessionStore(rdb, time.Hour, 24*time.Hour, nil)
	if err != nil {
		t.Fatalf("session store: %v", err)
	}
	mailer := &capturingMailer{}
	return auth.NewService(pool, tokens, sessions, mailer, nil, nil), mailer
}

func TestAuthFlow_Integration(t *testing.T) {
	svc, mailer := newService(t)
	ctx := context.Background()
	email := "it-" + uuid.NewString() + "@example.com"
	const password = "integration-pw-123"

	// Signup creates the user (and personal workspace + owner membership).
	user, err := svc.Signup(ctx, email, password, "127.0.0.1")
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if user.EmailVerified {
		t.Error("new user should start unverified")
	}

	// Duplicate signup is a conflict.
	if _, err := svc.Signup(ctx, email, password, "127.0.0.1"); err == nil {
		t.Error("duplicate signup should fail")
	}

	// Verify email using the captured token.
	if mailer.verifyToken == "" {
		t.Fatal("no verification token captured")
	}
	if err := svc.VerifyEmail(ctx, mailer.verifyToken); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	// Wrong password is rejected; correct password logs in.
	if _, err := svc.Login(ctx, email, "wrong", "127.0.0.1"); err == nil {
		t.Error("login with wrong password should fail")
	}
	login, err := svc.Login(ctx, email, password, "127.0.0.1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if login.AccessToken == "" || login.RefreshToken == "" {
		t.Fatal("login did not issue tokens")
	}

	// Refresh rotates the refresh token.
	refreshed, err := svc.Refresh(ctx, login.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshed.RefreshToken == login.RefreshToken {
		t.Error("refresh should rotate the token")
	}
	if refreshed.UserID != user.ID {
		t.Error("refreshed session has wrong user")
	}

	// Old refresh token is now invalid (rotation).
	if _, err := svc.Refresh(ctx, login.RefreshToken); err == nil {
		t.Error("reused old refresh token should fail")
	}

	// Logout revokes the current refresh token.
	if err := svc.Logout(ctx, refreshed.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, refreshed.RefreshToken); err == nil {
		t.Error("refresh after logout should fail")
	}
}
