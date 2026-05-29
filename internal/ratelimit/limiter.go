// Package ratelimit provides Postal's anti-abuse rate governor: a Redis-backed
// token-bucket limiter and reusable HTTP middleware. Token buckets allow short
// bursts up to a capacity while bounding the sustained rate, which fits a free
// app that must protect both itself and shared upstream API keys.
//
// The bucket is evaluated atomically in a single Lua script so concurrent
// requests for the same key cannot race. Time is injected (the current
// timestamp is passed into the script) so behavior is deterministic in tests.
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// Rule defines a token bucket: a burst Capacity and a sustained RefillRate in
// tokens per second.
type Rule struct {
	// Capacity is the maximum number of tokens (the burst ceiling).
	Capacity int
	// RefillRate is how many tokens are added per second.
	RefillRate float64
}

// Result describes the outcome of an Allow check.
type Result struct {
	// Allowed reports whether the request may proceed.
	Allowed bool
	// Remaining is the whole tokens left after this check.
	Remaining int
	// RetryAfter is how long until enough tokens refill, when not allowed.
	RetryAfter time.Duration
}

// bucketScript implements a token bucket atomically.
//
// KEYS[1] = bucket key
// ARGV: capacity, refillRate(tokens/sec), nowMillis, cost, ttlSeconds
// Returns: { allowed(0|1), remaining, retryAfterMillis }
var bucketScript = redis.NewScript(`
local key        = KEYS[1]
local capacity   = tonumber(ARGV[1])
local refill     = tonumber(ARGV[2])
local now        = tonumber(ARGV[3])
local cost       = tonumber(ARGV[4])
local ttl        = tonumber(ARGV[5])

local data    = redis.call('HMGET', key, 'tokens', 'ts')
local tokens  = tonumber(data[1])
local ts      = tonumber(data[2])
if tokens == nil then
  tokens = capacity
  ts = now
end

-- Refill based on elapsed time since last update.
local elapsed = math.max(0, now - ts) / 1000.0
tokens = math.min(capacity, tokens + elapsed * refill)

local allowed = 0
local retry = 0
if tokens >= cost then
  allowed = 1
  tokens = tokens - cost
else
  if refill > 0 then
    retry = math.ceil(((cost - tokens) / refill) * 1000)
  else
    retry = -1
  end
end

redis.call('HSET', key, 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', key, ttl * 1000)

return { allowed, math.floor(tokens), retry }
`)

// Scripter is the subset of the Redis client the limiter needs. *redis.Client
// (and Postal's redis.Client wrapper) satisfy it.
type Scripter = redis.Scripter

// Limiter evaluates token buckets against Redis.
type Limiter struct {
	rdb   Scripter
	clock func() time.Time
}

// NewLimiter builds a Limiter. If clock is nil, time.Now is used. Injecting the
// clock keeps Allow deterministic in tests.
func NewLimiter(rdb Scripter, clock func() time.Time) *Limiter {
	if clock == nil {
		clock = time.Now
	}
	return &Limiter{rdb: rdb, clock: clock}
}

// Allow consumes cost tokens from the bucket identified by key under rule,
// reporting whether the request is permitted. It is atomic and safe for
// concurrent callers sharing a key.
func (l *Limiter) Allow(ctx context.Context, key string, rule Rule, cost int) (Result, error) {
	ttl := bucketTTL(rule)
	out, err := bucketScript.Run(ctx, l.rdb, []string{key},
		rule.Capacity, rule.RefillRate, l.clock().UnixMilli(), cost, int64(ttl.Seconds()),
	).Int64Slice()
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit: evaluating bucket: %w", err)
	}
	if len(out) != 3 {
		return Result{}, fmt.Errorf("ratelimit: unexpected script result length %d", len(out))
	}

	res := Result{Allowed: out[0] == 1, Remaining: int(out[1])}
	if !res.Allowed && out[2] >= 0 {
		res.RetryAfter = time.Duration(out[2]) * time.Millisecond
	}
	return res, nil
}

// bucketTTL returns how long an idle bucket key should live: long enough to
// fully refill, with a floor so very fast buckets still persist briefly.
func bucketTTL(rule Rule) time.Duration {
	const floor = time.Minute
	if rule.RefillRate <= 0 {
		return floor
	}
	refillTime := time.Duration(float64(rule.Capacity)/rule.RefillRate*float64(time.Second)) + floor
	return refillTime
}

// computeBucket is the pure-Go reference implementation of the bucket math,
// used in unit tests to validate the algorithm independently of Redis. The Lua
// script above must stay behaviorally identical.
func computeBucket(tokens float64, lastMillis, nowMillis int64, rule Rule, cost int) (allowed bool, remaining float64, retryAfter time.Duration) {
	elapsed := math.Max(0, float64(nowMillis-lastMillis)) / 1000.0
	tokens = math.Min(float64(rule.Capacity), tokens+elapsed*rule.RefillRate)

	if tokens >= float64(cost) {
		return true, tokens - float64(cost), 0
	}
	if rule.RefillRate <= 0 {
		return false, tokens, -1
	}
	ms := math.Ceil((float64(cost) - tokens) / rule.RefillRate * 1000)
	return false, tokens, time.Duration(ms) * time.Millisecond
}
