package channel_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/workspace"
)

// fakeProvider is a deterministic in-memory OAuth provider for the round trip.
type fakeProvider struct{}

func (fakeProvider) Platform() string { return "fake" }
func (fakeProvider) AuthURL(state, challenge string) string {
	return "https://fake.test/authorize?state=" + url.QueryEscape(state) + "&code_challenge=" + url.QueryEscape(challenge)
}
func (fakeProvider) ExchangeCode(_ context.Context, code, _ string) (*channel.Token, error) {
	return &channel.Token{AccessToken: "access-" + code, RefreshToken: "refresh-" + code, Scopes: []string{"read", "write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (fakeProvider) RefreshToken(_ context.Context, _ string) (*channel.Token, error) {
	return &channel.Token{AccessToken: "access-refreshed", RefreshToken: "refresh-rotated", Scopes: []string{"read", "write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (fakeProvider) Account(_ context.Context, _ string) (*channel.Account, error) {
	return &channel.Account{ID: "acct-" + uuid.NewString(), Handle: "@fakeuser", DisplayName: "Fake User"}, nil
}
func (fakeProvider) Revoke(context.Context, string) error { return nil }

func setup(t *testing.T) (*db.Pool, *redis.Client, *security.Encryptor) {
	t.Helper()
	dsn := os.Getenv("POSTAL_DATABASE_URL")
	addr := os.Getenv("POSTAL_REDIS_ADDR")
	if dsn == "" || addr == "" {
		t.Skip("POSTAL_DATABASE_URL/POSTAL_REDIS_ADDR not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("POSTAL_REDIS_PASSWORD")})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unreachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	enc, err := security.NewEncryptorFromSpec(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	return pool, rdb, enc
}

// seedOwner creates a user + workspace + owner membership and returns their IDs.
func seedOwner(t *testing.T, pool *db.Pool) (workspaceID, userID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	q := pool.Queries()
	user, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: "chan-" + uuid.NewString() + "@example.com", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: "Test", OwnerUserID: user.ID})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := q.CreateMember(ctx, sqlc.CreateMemberParams{
		WorkspaceID: ws.ID, UserID: user.ID, Role: string(workspace.RoleOwner),
		Permissions: workspace.PresetCapabilities(workspace.RoleOwner),
	}); err != nil {
		t.Fatalf("create member: %v", err)
	}
	return ws.ID, user.ID
}

func TestChannelOAuthRoundTrip_Integration(t *testing.T) {
	pool, rdb, enc := setup(t)
	ctx := context.Background()
	wsID, userID := seedOwner(t, pool)

	wsSvc := workspace.NewService(pool, nil, nil)
	svc := channel.NewService(pool, channel.NewRegistry(fakeProvider{}), enc, rdb, wsSvc, nil, nil)

	// 1. StartConnect -> authorize URL carrying the state.
	authURL, err := svc.StartConnect(ctx, wsID, userID, "fake")
	if err != nil {
		t.Fatalf("StartConnect: %v", err)
	}
	parsed, _ := url.Parse(authURL)
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatal("authorize URL missing state")
	}

	// 2. CompleteConnect exchanges the code and stores the channel + credential.
	view, err := svc.CompleteConnect(ctx, userID, state, "auth-code-xyz")
	if err != nil {
		t.Fatalf("CompleteConnect: %v", err)
	}
	if view.Platform != "fake" || view.Handle != "@fakeuser" || view.Status != "active" {
		t.Errorf("unexpected channel view: %+v", view)
	}

	// 3. Credentials are ciphertext at rest, not plaintext.
	cred, err := pool.Queries().GetChannelCredential(ctx, view.ID)
	if err != nil {
		t.Fatalf("GetChannelCredential: %v", err)
	}
	if bytes.Contains(cred.EncryptedAccessToken, []byte("access-auth-code-xyz")) {
		t.Error("access token stored in plaintext")
	}
	plain, err := enc.Open(cred.EncryptedAccessToken)
	if err != nil || string(plain) != "access-auth-code-xyz" {
		t.Errorf("decrypt at rest = %q, %v", plain, err)
	}

	// 4. Reused state fails (single-use).
	if _, err := svc.CompleteConnect(ctx, userID, state, "auth-code-xyz"); err == nil {
		t.Error("reused oauth state should fail")
	}

	// 5. Refresh rotates the stored credential.
	if err := svc.RefreshChannel(ctx, view.ID); err != nil {
		t.Fatalf("RefreshChannel: %v", err)
	}
	cred2, _ := pool.Queries().GetChannelCredential(ctx, view.ID)
	plain2, _ := enc.Open(cred2.EncryptedAccessToken)
	if string(plain2) != "access-refreshed" {
		t.Errorf("after refresh, access = %q, want access-refreshed", plain2)
	}

	// 6. List shows the channel (no credentials in the view).
	list, err := svc.ListChannels(ctx, wsID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListChannels = %d items, %v", len(list), err)
	}

	// 7. Disconnect purges the credential and the channel.
	if err := svc.Disconnect(ctx, userID, wsID, view.ID); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if _, err := pool.Queries().GetChannelCredential(ctx, view.ID); err != pgx.ErrNoRows {
		t.Errorf("credential not purged after disconnect: %v", err)
	}
	if list, _ := svc.ListChannels(ctx, wsID); len(list) != 0 {
		t.Errorf("channel not removed after disconnect: %d remain", len(list))
	}
}

func TestCompleteConnect_RejectsForeignUser_Integration(t *testing.T) {
	pool, rdb, enc := setup(t)
	ctx := context.Background()
	wsID, userID := seedOwner(t, pool)
	_, otherUserID := seedOwner(t, pool)

	wsSvc := workspace.NewService(pool, nil, nil)
	svc := channel.NewService(pool, channel.NewRegistry(fakeProvider{}), enc, rdb, wsSvc, nil, nil)

	authURL, err := svc.StartConnect(ctx, wsID, userID, "fake")
	if err != nil {
		t.Fatalf("StartConnect: %v", err)
	}
	parsed, _ := url.Parse(authURL)
	state := parsed.Query().Get("state")

	// A different user completing the flow must be rejected.
	if _, err := svc.CompleteConnect(ctx, otherUserID, state, "code"); err == nil {
		t.Error("CompleteConnect by a different user should be forbidden")
	}
}

func TestStartConnect_UnsupportedPlatform_Integration(t *testing.T) {
	pool, rdb, enc := setup(t)
	ctx := context.Background()
	wsID, userID := seedOwner(t, pool)
	svc := channel.NewService(pool, channel.NewRegistry(fakeProvider{}), enc, rdb, workspace.NewService(pool, nil, nil), nil, nil)

	if _, err := svc.StartConnect(ctx, wsID, userID, "myspace"); err == nil {
		t.Error("unsupported platform should error")
	}
}
