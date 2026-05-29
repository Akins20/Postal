package publish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// resultStore is the persistence dependency for publish results (idempotency).
// The sqlc-generated *sqlc.Queries satisfies it.
type resultStore interface {
	GetPublishResultByKey(ctx context.Context, idempotencyKey string) (sqlc.PublishResult, error)
	InsertPublishResult(ctx context.Context, arg sqlc.InsertPublishResultParams) (sqlc.PublishResult, error)
}

// Store implements Results over publish_results.
type Store struct {
	q resultStore
}

// NewStore builds a Store. Pass a *sqlc.Queries (e.g. pool.Queries()).
func NewStore(q resultStore) *Store {
	return &Store{q: q}
}

// Find returns the recorded result for an idempotency key, if any.
func (s *Store) Find(ctx context.Context, idempotencyKey string) (*Result, bool, error) {
	row, err := s.q.GetPublishResultByKey(ctx, idempotencyKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("finding publish result: %w", err)
	}
	return &Result{PlatformPostID: row.PlatformPostID, Raw: json.RawMessage(row.RawResponse)}, true, nil
}

// Record persists a successful publish under its idempotency key. A unique
// violation (concurrent record of the same key) is treated as success.
func (s *Store) Record(ctx context.Context, channelID uuid.UUID, idempotencyKey string, res *Result) error {
	raw := res.Raw
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	_, err := s.q.InsertPublishResult(ctx, sqlc.InsertPublishResultParams{
		ChannelID:      channelID,
		IdempotencyKey: idempotencyKey,
		PlatformPostID: res.PlatformPostID,
		RawResponse:    raw,
	})
	if err != nil && !db.IsUniqueViolation(err) {
		return fmt.Errorf("recording publish result: %w", err)
	}
	return nil
}
