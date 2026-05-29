package security

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// emptyJSONObject is stored when an event carries no metadata, keeping the
// JSONB column a valid object rather than null.
var emptyJSONObject = []byte("{}")

// Event is a single auditable action. WorkspaceID and ActorUserID are optional
// (nil for system events or pre-account actions such as a failed login).
type Event struct {
	// WorkspaceID scopes the event to a workspace, when applicable.
	WorkspaceID *uuid.UUID
	// ActorUserID identifies who performed the action, when known.
	ActorUserID *uuid.UUID
	// Action is a stable verb describing what happened (e.g. "channel.connect").
	Action string
	// Target identifies the affected resource (e.g. a channel ID).
	Target string
	// Metadata holds extra structured context; serialized to JSONB.
	Metadata map[string]any
	// IP is the client address that initiated the action, when known.
	IP string
}

// Recorder persists audit events. Domains depend on this interface, not the
// concrete Auditor.
type Recorder interface {
	Record(ctx context.Context, e Event) error
}

// auditStore is the persistence dependency for the audit log. The sqlc-
// generated *sqlc.Queries satisfies it.
type auditStore interface {
	InsertAuditLog(ctx context.Context, arg sqlc.InsertAuditLogParams) (sqlc.InsertAuditLogRow, error)
}

// Auditor writes audit events to durable storage.
type Auditor struct {
	store auditStore
	log   *slog.Logger
}

// NewAuditor builds an Auditor backed by store. log may be nil.
func NewAuditor(store auditStore, log *slog.Logger) *Auditor {
	return &Auditor{store: store, log: log}
}

// Record persists e. A missing Action is rejected — every audit entry must say
// what happened. Metadata is marshaled to JSON; nil becomes an empty object.
func (a *Auditor) Record(ctx context.Context, e Event) error {
	if e.Action == "" {
		return fmt.Errorf("security: audit event requires an action")
	}

	meta := emptyJSONObject
	if len(e.Metadata) > 0 {
		encoded, err := json.Marshal(e.Metadata)
		if err != nil {
			return fmt.Errorf("marshaling audit metadata: %w", err)
		}
		meta = encoded
	}

	_, err := a.store.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		WorkspaceID: e.WorkspaceID,
		ActorUserID: e.ActorUserID,
		Action:      e.Action,
		Target:      e.Target,
		Metadata:    meta,
		Ip:          e.IP,
	})
	if err != nil {
		return fmt.Errorf("recording audit event %q: %w", e.Action, err)
	}
	return nil
}
