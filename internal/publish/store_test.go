package publish_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/publish"
)

// seedChannel creates a user, workspace, and channel, returning the channel ID
// so publish_results' FK is satisfied.
func seedChannel(t *testing.T, pool *db.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	q := pool.Queries()
	user, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: "pub-" + uuid.NewString() + "@example.com", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: "Pub", OwnerUserID: user.ID})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	ch, err := q.CreateChannel(ctx, sqlc.CreateChannelParams{
		WorkspaceID: ws.ID, Platform: "twitter", PlatformAccountID: "acct-" + uuid.NewString(),
		Handle: "@x", DisplayName: "X", ConnectedBy: &user.ID,
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	return ch.ID
}

func TestStore_RecordFindIdempotency_Integration(t *testing.T) {
	dsn := os.Getenv("POSTAL_DATABASE_URL")
	if dsn == "" {
		t.Skip("POSTAL_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)

	channelID := seedChannel(t, pool)
	store := publish.NewStore(pool.Queries())
	key := "idem-" + uuid.NewString()

	// Not found initially.
	if _, found, err := store.Find(ctx, key); err != nil || found {
		t.Fatalf("Find before record: found=%v err=%v", found, err)
	}

	// Record then find.
	if err := store.Record(ctx, channelID, key, &publish.Result{PlatformPostID: "pp-1", Raw: json.RawMessage(`{"id":"pp-1"}`)}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	res, found, err := store.Find(ctx, key)
	if err != nil || !found || res.PlatformPostID != "pp-1" {
		t.Fatalf("Find after record: res=%+v found=%v err=%v", res, found, err)
	}

	// Re-record under the same key is a no-op (unique violation swallowed), and
	// the original result is preserved — the idempotency guarantee.
	if err := store.Record(ctx, channelID, key, &publish.Result{PlatformPostID: "pp-2"}); err != nil {
		t.Fatalf("idempotent re-record should not error: %v", err)
	}
	res2, _, _ := store.Find(ctx, key)
	if res2.PlatformPostID != "pp-1" {
		t.Errorf("re-record changed the stored result: got %s, want pp-1", res2.PlatformPostID)
	}
}
