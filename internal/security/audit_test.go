package security

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// fakeAuditStore captures the last insert for assertions.
type fakeAuditStore struct {
	last sqlc.InsertAuditLogParams
	err  error
}

func (f *fakeAuditStore) InsertAuditLog(_ context.Context, arg sqlc.InsertAuditLogParams) (sqlc.InsertAuditLogRow, error) {
	f.last = arg
	return sqlc.InsertAuditLogRow{ID: 1}, f.err
}

func TestAuditor_Record(t *testing.T) {
	store := &fakeAuditStore{}
	auditor := NewAuditor(store, nil)

	ws := uuid.New()
	actor := uuid.New()
	err := auditor.Record(context.Background(), Event{
		WorkspaceID: &ws,
		ActorUserID: &actor,
		Action:      "channel.connect",
		Target:      "channel-123",
		Metadata:    map[string]any{"platform": "twitter"},
		IP:          "203.0.113.7",
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	if store.last.Action != "channel.connect" {
		t.Errorf("Action = %q", store.last.Action)
	}
	if store.last.Ip != "203.0.113.7" {
		t.Errorf("IP = %q", store.last.Ip)
	}
	if store.last.WorkspaceID == nil || *store.last.WorkspaceID != ws {
		t.Errorf("WorkspaceID = %v, want %v", store.last.WorkspaceID, ws)
	}
	var meta map[string]any
	if err := json.Unmarshal(store.last.Metadata, &meta); err != nil {
		t.Fatalf("metadata not valid JSON: %v", err)
	}
	if meta["platform"] != "twitter" {
		t.Errorf("metadata.platform = %v, want twitter", meta["platform"])
	}
}

func TestAuditor_Record_EmptyMetadataIsObject(t *testing.T) {
	store := &fakeAuditStore{}
	auditor := NewAuditor(store, nil)

	if err := auditor.Record(context.Background(), Event{Action: "user.login"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if string(store.last.Metadata) != "{}" {
		t.Errorf("empty metadata = %q, want {}", store.last.Metadata)
	}
}

func TestAuditor_Record_RequiresAction(t *testing.T) {
	store := &fakeAuditStore{}
	auditor := NewAuditor(store, nil)

	if err := auditor.Record(context.Background(), Event{Action: ""}); err == nil {
		t.Error("Record with empty action should error")
	}
}

// Ensure *Auditor satisfies the Recorder interface.
var _ Recorder = (*Auditor)(nil)
