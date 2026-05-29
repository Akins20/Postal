package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// refreshKeyPrefix namespaces refresh-token entries in Redis.
const refreshKeyPrefix = "auth:refresh:"

// ErrInvalidSession indicates a refresh token that is unknown, already rotated,
// revoked, or past its absolute lifetime.
var ErrInvalidSession = errors.New("auth: invalid or expired session")

// sessionRedis is the subset of the Redis client the session store needs.
type sessionRedis interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, ttl time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// sessionData is the Redis-stored payload behind a refresh token. AbsoluteExpiry
// caps the total session lifetime regardless of sliding renewals.
type sessionData struct {
	UserID         uuid.UUID `json:"user_id"`
	AbsoluteExpiry time.Time `json:"absolute_expiry"`
}

// SessionStore issues, rotates, and revokes opaque refresh tokens in Redis.
// Tokens slide (TTL extends on each rotation) up to an absolute maximum.
// Only a hash of each token is stored, so a Redis dump cannot reveal live
// tokens. Safe for concurrent use.
type SessionStore struct {
	rdb        sessionRedis
	slidingTTL time.Duration
	maxTTL     time.Duration
	clock      func() time.Time
}

// NewSessionStore builds a SessionStore. slidingTTL is the per-use lifetime;
// maxTTL is the absolute session cap (must be >= slidingTTL). clock defaults to
// time.Now.
func NewSessionStore(rdb sessionRedis, slidingTTL, maxTTL time.Duration, clock func() time.Time) (*SessionStore, error) {
	if slidingTTL <= 0 || maxTTL <= 0 {
		return nil, errors.New("auth: session TTLs must be positive")
	}
	if maxTTL < slidingTTL {
		return nil, errors.New("auth: max session TTL must be >= sliding TTL")
	}
	if clock == nil {
		clock = time.Now
	}
	return &SessionStore{rdb: rdb, slidingTTL: slidingTTL, maxTTL: maxTTL, clock: clock}, nil
}

// Create issues a new refresh token for userID, establishing the session's
// absolute expiry at now + maxTTL.
func (s *SessionStore) Create(ctx context.Context, userID uuid.UUID) (string, error) {
	now := s.clock()
	return s.store(ctx, userID, now.Add(s.maxTTL), now)
}

// Rotate validates oldToken, invalidates it, and issues a fresh token for the
// same session — sliding its TTL forward while preserving the absolute expiry.
// Reusing an already-rotated token fails with ErrInvalidSession.
func (s *SessionStore) Rotate(ctx context.Context, oldToken string) (newToken string, userID uuid.UUID, err error) {
	now := s.clock()
	key := refreshKey(oldToken)

	raw, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", uuid.Nil, ErrInvalidSession
	}
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("reading session: %w", err)
	}

	var data sessionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "", uuid.Nil, ErrInvalidSession
	}

	// Invalidate the old token immediately (single-use rotation; reuse-evident).
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return "", uuid.Nil, fmt.Errorf("invalidating old session: %w", err)
	}
	if !now.Before(data.AbsoluteExpiry) {
		return "", uuid.Nil, ErrInvalidSession
	}

	newToken, err = s.store(ctx, data.UserID, data.AbsoluteExpiry, now)
	if err != nil {
		return "", uuid.Nil, err
	}
	return newToken, data.UserID, nil
}

// Revoke deletes a refresh token, ending the session (logout). Revoking an
// unknown token is a no-op.
func (s *SessionStore) Revoke(ctx context.Context, token string) error {
	if err := s.rdb.Del(ctx, refreshKey(token)).Err(); err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}
	return nil
}

// store generates a token and writes the session, choosing a TTL that slides
// but never outlives the absolute expiry.
func (s *SessionStore) store(ctx context.Context, userID uuid.UUID, absoluteExpiry, now time.Time) (string, error) {
	token, err := newOpaqueToken()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(sessionData{UserID: userID, AbsoluteExpiry: absoluteExpiry})
	if err != nil {
		return "", fmt.Errorf("encoding session: %w", err)
	}

	ttl := s.slidingTTL
	if remaining := absoluteExpiry.Sub(now); remaining < ttl {
		ttl = remaining
	}
	if err := s.rdb.Set(ctx, refreshKey(token), payload, ttl).Err(); err != nil {
		return "", fmt.Errorf("storing session: %w", err)
	}
	return token, nil
}

// refreshKey maps a token to its Redis key via SHA-256 so raw tokens are never
// stored at rest.
func refreshKey(token string) string {
	return refreshKeyPrefix + hashToken(token)
}
