package post_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/post"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/publish/twitter"
)

// fakeResolver maps a known channel to "twitter" within its workspace, and
// not-found otherwise (simulating channel ownership without the full service).
type fakeResolver struct {
	workspaceID uuid.UUID
	channelID   uuid.UUID
}

func (f fakeResolver) PlatformFor(_ context.Context, workspaceID, channelID uuid.UUID) (string, error) {
	if workspaceID == f.workspaceID && channelID == f.channelID {
		return "twitter", nil
	}
	return "", apperr.NotFound("channel_not_found", "channel not found")
}

func setup(t *testing.T) (*post.Service, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
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

	q := pool.Queries()
	user, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: "post-" + uuid.NewString() + "@example.com", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: "Composer", OwnerUserID: user.ID})
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

	resolver := fakeResolver{workspaceID: ws.ID, channelID: ch.ID}
	validator := publish.NewRegistry(twitter.New(twitter.Config{}))
	svc := post.NewService(pool, resolver, validator, nil, nil)
	return svc, ws.ID, user.ID, ch.ID
}

func TestComposer_CRUDAndValidate_Integration(t *testing.T) {
	svc, wsID, userID, chID := setup(t)
	ctx := context.Background()

	// Create a draft with one variant.
	created, err := svc.Create(ctx, wsID, userID, []post.VariantInput{
		{ChannelID: chID, Body: "hello world", PlatformOptions: map[string]any{"reply_settings": "everyone"}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Status != "draft" || len(created.Variants) != 1 {
		t.Fatalf("unexpected post: %+v", created)
	}

	// Get returns it with the variant + platform options round-tripped.
	got, err := svc.Get(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Variants) != 1 || got.Variants[0].Body != "hello world" {
		t.Errorf("Get variant mismatch: %+v", got.Variants)
	}
	if got.Variants[0].PlatformOptions["reply_settings"] != "everyone" {
		t.Errorf("platform options not round-tripped: %+v", got.Variants[0].PlatformOptions)
	}

	// List includes it.
	list, err := svc.List(ctx, wsID)
	if err != nil || len(list) == 0 {
		t.Fatalf("List = %d, %v", len(list), err)
	}

	// Validate: short body is valid.
	res, err := svc.Validate(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(res) != 1 || !res[0].Valid {
		t.Errorf("expected valid variant, got %+v", res)
	}

	// Cross-workspace isolation: Get with a different workspace -> not found.
	if _, err := svc.Get(ctx, uuid.New(), created.ID); err == nil {
		t.Error("cross-workspace Get should fail")
	}

	// Delete.
	if err := svc.Delete(ctx, wsID, userID, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(ctx, wsID, created.ID); err == nil {
		t.Error("post should be gone after delete")
	}
}

func TestComposer_ValidateRejectsOverLimit_Integration(t *testing.T) {
	svc, wsID, userID, chID := setup(t)
	ctx := context.Background()

	// A draft can hold over-limit content; Validate flags it per platform.
	over := strings.Repeat("a", 281)
	p, err := svc.Create(ctx, wsID, userID, []post.VariantInput{{ChannelID: chID, Body: over}})
	if err != nil {
		t.Fatalf("Create over-limit draft: %v", err)
	}
	res, err := svc.Validate(ctx, wsID, p.ID)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(res) != 1 || res[0].Valid || res[0].Code != "text_too_long" {
		t.Errorf("expected text_too_long invalid, got %+v", res)
	}
}

func TestComposer_ForeignChannelRejected_Integration(t *testing.T) {
	svc, wsID, userID, _ := setup(t)
	ctx := context.Background()

	// A channel not in the workspace must be rejected at create.
	if _, err := svc.Create(ctx, wsID, userID, []post.VariantInput{{ChannelID: uuid.New(), Body: "x"}}); err == nil {
		t.Error("creating a post for a foreign channel should fail")
	}

	// Duplicate channel in one post is rejected.
	if _, err := svc.Create(ctx, wsID, userID, nil); err == nil {
		t.Error("creating a post with no variants should fail")
	}
}
