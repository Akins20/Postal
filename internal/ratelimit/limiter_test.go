package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestComputeBucket(t *testing.T) {
	rule := Rule{Capacity: 10, RefillRate: 2} // 2 tokens/sec, burst 10

	tests := []struct {
		name        string
		tokens      float64
		lastMs      int64
		nowMs       int64
		cost        int
		wantAllowed bool
		wantRemain  float64
	}{
		{name: "spend within budget", tokens: 5, lastMs: 0, nowMs: 0, cost: 3, wantAllowed: true, wantRemain: 2},
		{name: "exactly enough", tokens: 3, lastMs: 0, nowMs: 0, cost: 3, wantAllowed: true, wantRemain: 0},
		{name: "insufficient", tokens: 1, lastMs: 0, nowMs: 0, cost: 3, wantAllowed: false, wantRemain: 1},
		{name: "refill over 1s adds 2", tokens: 0, lastMs: 0, nowMs: 1000, cost: 2, wantAllowed: true, wantRemain: 0},
		{name: "refill capped at capacity", tokens: 0, lastMs: 0, nowMs: 100000, cost: 0, wantAllowed: true, wantRemain: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, remain, _ := computeBucket(tt.tokens, tt.lastMs, tt.nowMs, rule, tt.cost)
			if allowed != tt.wantAllowed {
				t.Errorf("allowed = %v, want %v", allowed, tt.wantAllowed)
			}
			if remain != tt.wantRemain {
				t.Errorf("remaining = %v, want %v", remain, tt.wantRemain)
			}
		})
	}
}

func TestComputeBucket_RetryAfter(t *testing.T) {
	rule := Rule{Capacity: 5, RefillRate: 1} // 1 token/sec
	// 0 tokens, need 2 -> must wait 2s.
	_, _, retry := computeBucket(0, 0, 0, rule, 2)
	if retry != 2*time.Second {
		t.Errorf("retryAfter = %v, want 2s", retry)
	}
}

func TestComputeBucket_ZeroRefillNeverRecovers(t *testing.T) {
	rule := Rule{Capacity: 1, RefillRate: 0}
	allowed, _, retry := computeBucket(0, 0, 100000, rule, 1)
	if allowed {
		t.Error("allowed with zero tokens and zero refill")
	}
	if retry != -1 {
		t.Errorf("retryAfter = %v, want -1 (never)", retry)
	}
}

// testRedis dials Redis from POSTAL_REDIS_ADDR, skipping the test if it is unset
// or unreachable. Integration tests use a real Redis (docker-compose).
func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("POSTAL_REDIS_ADDR")
	if addr == "" {
		t.Skip("POSTAL_REDIS_ADDR not set; skipping Redis integration test")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("POSTAL_REDIS_PASSWORD"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		t.Skipf("Redis not reachable at %s: %v", addr, err)
	}
	return rdb
}

func TestLimiter_Allow_Integration(t *testing.T) {
	rdb := testRedis(t)
	defer func() { _ = rdb.Close() }()

	ctx := context.Background()
	// Unique key per run; clean up after.
	key := "test:allow:" + t.Name()
	t.Cleanup(func() { rdb.Del(ctx, key) })

	now := time.UnixMilli(1_000_000)
	lim := NewLimiter(rdb, func() time.Time { return now })
	rule := Rule{Capacity: 2, RefillRate: 1} // burst 2, 1/sec

	// Two requests allowed from full bucket.
	for i := 0; i < 2; i++ {
		res, err := lim.Allow(ctx, key, rule, 1)
		if err != nil {
			t.Fatalf("Allow #%d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("request #%d should be allowed", i)
		}
	}

	// Third is denied with a retry hint.
	res, err := lim.Allow(ctx, key, rule, 1)
	if err != nil {
		t.Fatalf("Allow #3: %v", err)
	}
	if res.Allowed {
		t.Fatal("third request should be denied (bucket empty)")
	}
	if res.RetryAfter <= 0 || res.RetryAfter > time.Second {
		t.Errorf("RetryAfter = %v, want ~1s", res.RetryAfter)
	}

	// Advance the clock 2s -> bucket refills, request allowed again.
	now = now.Add(2 * time.Second)
	res, err = lim.Allow(ctx, key, rule, 1)
	if err != nil {
		t.Fatalf("Allow after refill: %v", err)
	}
	if !res.Allowed {
		t.Error("request after refill should be allowed")
	}
}
