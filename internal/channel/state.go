package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// oauthStateTTL bounds how long an in-flight OAuth authorization may take.
const oauthStateTTL = 10 * time.Minute

// stateKeyPrefix namespaces OAuth state entries in Redis.
const stateKeyPrefix = "oauth:state:"

// ErrInvalidState indicates an unknown, expired, or already-used OAuth state.
var ErrInvalidState = errors.New("channel: invalid or expired oauth state")

// oauthState is the server-side context bound to an OAuth flow, looked up by the
// opaque state value on callback. It carries the PKCE verifier so the secret
// never leaves the server.
type oauthState struct {
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	UserID       uuid.UUID `json:"user_id"`
	Platform     string    `json:"platform"`
	CodeVerifier string    `json:"code_verifier"`
	// RedirectURI is the allowlisted callback chosen at connect time; reused at
	// exchange so authorize/exchange redirect_uri match. Empty = adapter default.
	RedirectURI string `json:"redirect_uri,omitempty"`
}

// stateRedis is the subset of the Redis client the state store needs.
type stateRedis interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) *redis.StatusCmd
	GetDel(ctx context.Context, key string) *redis.StringCmd
}

// stateStore persists short-lived OAuth flow state in Redis.
type stateStore struct {
	rdb stateRedis
}

// save stores st under a fresh random state token and returns the token.
func (s *stateStore) save(ctx context.Context, st oauthState) (string, error) {
	token, err := newStateToken()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(st)
	if err != nil {
		return "", fmt.Errorf("encoding oauth state: %w", err)
	}
	if err := s.rdb.Set(ctx, stateKeyPrefix+token, payload, oauthStateTTL).Err(); err != nil {
		return "", fmt.Errorf("storing oauth state: %w", err)
	}
	return token, nil
}

// consume atomically fetches and deletes the state for token (single use).
func (s *stateStore) consume(ctx context.Context, token string) (oauthState, error) {
	raw, err := s.rdb.GetDel(ctx, stateKeyPrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return oauthState{}, ErrInvalidState
	}
	if err != nil {
		return oauthState{}, fmt.Errorf("reading oauth state: %w", err)
	}
	var st oauthState
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return oauthState{}, ErrInvalidState
	}
	return st, nil
}
