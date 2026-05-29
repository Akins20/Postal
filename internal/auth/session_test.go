package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// testRedis dials Redis from POSTAL_REDIS_ADDR, skipping if unavailable.
func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("POSTAL_REDIS_ADDR")
	if addr == "" {
		t.Skip("POSTAL_REDIS_ADDR not set; skipping Redis integration test")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("POSTAL_REDIS_PASSWORD")})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		t.Skipf("Redis not reachable: %v", err)
	}
	return rdb
}

func TestSessionStore_CreateRotateRevoke(t *testing.T) {
	rdb := testRedis(t)
	defer func() { _ = rdb.Close() }()
	ctx := context.Background()

	now := time.Unix(1_700_000_000, 0)
	clock := func() time.Time { return now }
	store, err := NewSessionStore(rdb, time.Hour, 24*time.Hour, clock)
	if err != nil {
		t.Fatalf("NewSessionStore: %v", err)
	}
	userID := uuid.New()

	tok, err := store.Create(ctx, userID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { rdb.Del(ctx, refreshKey(tok)) })

	// Rotate: new token issued, same user; old token becomes invalid.
	newTok, gotUser, err := store.Rotate(ctx, tok)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	t.Cleanup(func() { rdb.Del(ctx, refreshKey(newTok)) })
	if gotUser != userID {
		t.Errorf("rotated user = %v, want %v", gotUser, userID)
	}
	if newTok == tok {
		t.Error("rotation must produce a new token")
	}

	// Reusing the old (already rotated) token must fail (reuse detection).
	if _, _, err := store.Rotate(ctx, tok); err != ErrInvalidSession {
		t.Errorf("reused token err = %v, want ErrInvalidSession", err)
	}

	// Revoke ends the session.
	if err := store.Revoke(ctx, newTok); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, _, err := store.Rotate(ctx, newTok); err != ErrInvalidSession {
		t.Errorf("revoked token err = %v, want ErrInvalidSession", err)
	}
}

func TestSessionStore_AbsoluteExpiryCap(t *testing.T) {
	rdb := testRedis(t)
	defer func() { _ = rdb.Close() }()
	ctx := context.Background()

	now := time.Unix(1_700_000_000, 0)
	clock := func() time.Time { return now }
	// Sliding 1h, absolute max 2h.
	store, _ := NewSessionStore(rdb, time.Hour, 2*time.Hour, clock)
	tok, _ := store.Create(ctx, uuid.New())
	t.Cleanup(func() { rdb.Del(ctx, refreshKey(tok)) })

	// Advance beyond the absolute cap; rotation must refuse even though the
	// sliding TTL alone might still be alive.
	now = now.Add(3 * time.Hour)
	if _, _, err := store.Rotate(ctx, tok); err != ErrInvalidSession {
		t.Errorf("past-absolute-cap err = %v, want ErrInvalidSession", err)
	}
}

func TestSessionStore_UnknownToken(t *testing.T) {
	rdb := testRedis(t)
	defer func() { _ = rdb.Close() }()
	store, _ := NewSessionStore(rdb, time.Hour, time.Hour, nil)

	if _, _, err := store.Rotate(context.Background(), "nope-not-a-real-token"); err != ErrInvalidSession {
		t.Errorf("unknown token err = %v, want ErrInvalidSession", err)
	}
}

func TestNewSessionStore_Validation(t *testing.T) {
	if _, err := NewSessionStore(nil, 0, time.Hour, nil); err == nil {
		t.Error("zero sliding ttl should error")
	}
	if _, err := NewSessionStore(nil, 2*time.Hour, time.Hour, nil); err == nil {
		t.Error("max < sliding should error")
	}
}
