package web

import (
	"context"

	"github.com/google/uuid"
)

// ctxKey is an unexported context key type to avoid collisions.
type ctxKey int

const userIDKey ctxKey = iota

// WithUserID returns a copy of ctx carrying the authenticated user's ID. The
// auth middleware sets this; downstream middleware/handlers read it via UserID.
// It lives in the neutral web package so both auth and domain packages can use
// it without an import cycle.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserID returns the authenticated user's ID from ctx, reporting whether present.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
