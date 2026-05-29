package workspace

import (
	"context"

	"github.com/google/uuid"
)

// Member is a user's membership in a workspace, including the authoritative
// capability set.
type Member struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        string
	Permissions []string
}

// Has reports whether the member holds capability c.
func (m Member) Has(c Capability) bool {
	return Has(m.Permissions, c)
}

// ctxKey is an unexported context key type to avoid collisions.
type ctxKey int

const memberKey ctxKey = iota

// withMember returns a copy of ctx carrying the resolved membership.
func withMember(ctx context.Context, m Member) context.Context {
	return context.WithValue(ctx, memberKey, m)
}

// MemberFrom returns the membership resolved by RequireCapability, if present.
func MemberFrom(ctx context.Context) (Member, bool) {
	m, ok := ctx.Value(memberKey).(Member)
	return m, ok
}
