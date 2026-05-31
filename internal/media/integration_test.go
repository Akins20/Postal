package media_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/media"
	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/platform/storage"
)

// harness bundles the media service with its workspace context for a test run.
type harness struct {
	svc   *media.Service
	pool  *db.Pool
	wsID  uuid.UUID
	store storage.Storage
}

// setup connects Postgres + MinIO from env, seeds a workspace, and builds a
// media service with small quotas so the quota path is exercisable. It skips
// (not fails) when the backing services are unavailable, so unit runs stay green.
func setup(t *testing.T) harness {
	t.Helper()
	dsn := os.Getenv("POSTAL_DATABASE_URL")
	endpoint := os.Getenv("POSTAL_STORAGE_ENDPOINT")
	if dsn == "" || endpoint == "" {
		t.Skip("POSTAL_DATABASE_URL / POSTAL_STORAGE_ENDPOINT not set; skipping media integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)

	store, err := storage.New(ctx, storage.Config{
		Endpoint:  endpoint,
		AccessKey: os.Getenv("POSTAL_STORAGE_ACCESS_KEY"),
		SecretKey: os.Getenv("POSTAL_STORAGE_SECRET_KEY"),
		Bucket:    envOr("POSTAL_STORAGE_BUCKET", "postal-media"),
		Region:    os.Getenv("POSTAL_STORAGE_REGION"),
		UseSSL:    os.Getenv("POSTAL_STORAGE_USE_SSL") == "true",
	})
	if err != nil {
		t.Skipf("minio unreachable: %v", err)
	}

	q := pool.Queries()
	user, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: "media-" + uuid.NewString() + "@example.com", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: "Media", OwnerUserID: user.ID})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	// 1 MiB workspace quota so a modest second upload trips it.
	svc := media.NewService(pool, store, nil, 1<<20, 1<<20, nil)
	return harness{svc: svc, pool: pool, wsID: ws.ID, store: store}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// pngBytes encodes a solid-color PNG of the given dimensions.
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestMedia_UploadDownloadDelete_Integration(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	raw := pngBytes(t, 64, 48)
	asset, err := h.svc.Upload(ctx, h.wsID, "image/png", bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if asset.Kind != "image" || asset.MIME != "image/png" {
		t.Fatalf("unexpected kind/mime: %q/%q", asset.Kind, asset.MIME)
	}
	if asset.Width != 64 || asset.Height != 48 {
		t.Fatalf("expected 64x48 dimensions, got %dx%d", asset.Width, asset.Height)
	}
	if asset.Bytes != int64(len(raw)) {
		t.Fatalf("expected %d bytes, got %d", len(raw), asset.Bytes)
	}

	// List returns the new asset.
	list, err := h.svc.List(ctx, h.wsID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) == 0 || list[0].ID != asset.ID {
		t.Fatalf("expected uploaded asset at head of list, got %+v", list)
	}

	// Download streams back the exact bytes stored.
	rc, got, err := h.svc.Download(ctx, h.wsID, asset.ID)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	gotBytes, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !bytes.Equal(gotBytes, raw) {
		t.Fatalf("download bytes differ: got %d want %d", len(gotBytes), len(raw))
	}
	if got.ID != asset.ID {
		t.Fatalf("download asset id mismatch")
	}

	// ResolveMedia (composer path) reports authoritative metadata.
	kind, mime, nbytes, err := h.svc.ResolveMedia(ctx, h.wsID, asset.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if kind != "image" || mime != "image/png" || nbytes != int64(len(raw)) {
		t.Fatalf("resolve mismatch: %q %q %d", kind, mime, nbytes)
	}

	// OpenMedia (worker publish path) returns the bytes.
	okind, omime, odata, err := h.svc.OpenMedia(ctx, asset.ID)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if okind != "image" || omime != "image/png" || !bytes.Equal(odata, raw) {
		t.Fatalf("open mismatch: %q %q %d bytes", okind, omime, len(odata))
	}

	// Delete removes it; a subsequent download is not found.
	if err := h.svc.Delete(ctx, h.wsID, asset.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, _, derr := h.svc.Download(ctx, h.wsID, asset.ID); apperr.KindOf(derr) != apperr.KindNotFound {
		t.Fatalf("expected not-found after delete, got %v", derr)
	}
}

func TestMedia_Failures_Integration(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	// Unsupported type.
	if _, err := h.svc.Upload(ctx, h.wsID, "application/zip", bytes.NewReader([]byte("x")), 1); apperr.KindOf(err) != apperr.KindValidation {
		t.Fatalf("expected validation for unsupported type, got %v", err)
	}

	// Empty file.
	if _, err := h.svc.Upload(ctx, h.wsID, "image/png", bytes.NewReader(nil), 0); apperr.KindOf(err) != apperr.KindValidation {
		t.Fatalf("expected validation for empty file, got %v", err)
	}

	// Cross-workspace access is hidden as not-found.
	raw := pngBytes(t, 8, 8)
	asset, err := h.svc.Upload(ctx, h.wsID, "image/png", bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	t.Cleanup(func() { _ = h.svc.Delete(ctx, h.wsID, asset.ID) })
	other := uuid.New()
	if _, _, derr := h.svc.Download(ctx, other, asset.ID); apperr.KindOf(derr) != apperr.KindNotFound {
		t.Fatalf("expected not-found for cross-workspace download, got %v", derr)
	}

	// Over-size: a payload larger than the per-upload cap is rejected.
	if _, err := h.svc.Upload(ctx, h.wsID, "image/png", bytes.NewReader(raw), (1<<20)+1); apperr.KindOf(err) != apperr.KindValidation {
		t.Fatalf("expected validation for over-size upload, got %v", err)
	}

	// Quota: a service whose workspace cap is just under one asset rejects the
	// first upload that would exceed it (isolated from the per-upload size cap).
	tight := media.NewService(h.pool, h.store, nil, 5<<20, int64(len(raw))-1, nil)
	if _, err := tight.Upload(ctx, h.wsID, "image/png", bytes.NewReader(raw), int64(len(raw))); apperr.KindOf(err) != apperr.KindValidation {
		t.Fatalf("expected validation for over-quota upload, got %v", err)
	}
}
