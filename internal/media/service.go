package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"  // register GIF decoder for DecodeConfig
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/platform/storage"
	"github.com/Akins20/postal/internal/security"
)

// listLimit caps how many assets a list query returns.
const listLimit = 200

// statusReady marks an asset whose bytes are stored and usable.
const statusReady = "ready"

// errQuotaExceeded is a sentinel returned from the upload transaction when the
// workspace storage quota would be exceeded; mapped to a validation error.
var errQuotaExceeded = errors.New("workspace storage quota exceeded")

// Service manages media uploads, retrieval, deletion, and quota over object
// storage + the media_assets table.
type Service struct {
	pool              *db.Pool
	storage           storage.Storage
	audit             security.Recorder
	maxUploadBytes    int64
	maxWorkspaceBytes int64
	clock             func() time.Time
}

// NewService builds a media Service. clock defaults to time.Now.
func NewService(pool *db.Pool, store storage.Storage, audit security.Recorder, maxUpload, maxWorkspace int64, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{pool: pool, storage: store, audit: audit, maxUploadBytes: maxUpload, maxWorkspaceBytes: maxWorkspace, clock: clock}
}

// Upload validates and stores an uploaded file, recording its metadata. Images
// and GIFs are buffered to detect dimensions; videos stream straight through.
func (s *Service) Upload(ctx context.Context, workspaceID uuid.UUID, contentType string, r io.Reader, size int64) (Asset, error) {
	kind, ok := kindForMIME(contentType)
	if !ok {
		return Asset{}, apperr.Validation("unsupported_media_type", "unsupported media type: "+contentType)
	}
	if size <= 0 {
		return Asset{}, apperr.Validation("empty_file", "uploaded file is empty")
	}
	if size > s.maxUploadBytes || size > maxBytesForKind(kind) {
		return Asset{}, apperr.Validation("file_too_large", "file exceeds the size limit for its type")
	}

	reader, putSize, width, height := r, size, 0, 0
	if kind == KindImage || kind == KindGIF {
		buf, w, h, err := bufferImage(r, kind)
		if err != nil {
			return Asset{}, err
		}
		reader, putSize, width, height = bytes.NewReader(buf), int64(len(buf)), w, h
	}

	key := workspaceID.String() + "/" + uuid.NewString()
	if err := s.storage.Put(ctx, key, reader, putSize, contentType); err != nil {
		return Asset{}, apperr.Internal(err)
	}

	row, err := s.insertWithinQuota(ctx, sqlc.CreateMediaAssetParams{
		WorkspaceID: workspaceID, Kind: kind, StorageKey: key, Mime: contentType,
		// #nosec G115 -- image dimensions are bounded well within int32.
		Width: int32(width), Height: int32(height), DurationMs: 0, Bytes: putSize, Status: statusReady,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, key) // don't leak an orphaned object
		if errors.Is(err, errQuotaExceeded) {
			return Asset{}, apperr.Validation("quota_exceeded", "workspace storage quota exceeded")
		}
		return Asset{}, apperr.Internal(err)
	}
	s.recordAudit(ctx, workspaceID, "media.upload", row.ID.String())
	return toAsset(row), nil
}

// bufferImage reads an image/GIF fully (re-enforcing the per-kind size cap on the
// actual bytes) and detects its dimensions; dimension-decode failures are
// non-fatal (width/height stay 0).
func bufferImage(r io.Reader, kind string) (buf []byte, width, height int, err error) {
	buf, err = io.ReadAll(io.LimitReader(r, maxBytesForKind(kind)+1))
	if err != nil {
		return nil, 0, 0, apperr.Internal(err)
	}
	if int64(len(buf)) > maxBytesForKind(kind) {
		return nil, 0, 0, apperr.Validation("file_too_large", "file exceeds the size limit for its type")
	}
	if cfg, _, derr := image.DecodeConfig(bytes.NewReader(buf)); derr == nil {
		width, height = cfg.Width, cfg.Height
	}
	return buf, width, height, nil
}

// insertWithinQuota inserts the asset row only if it keeps the workspace under
// its storage quota, atomically: a row lock on the workspace serializes
// concurrent uploads so they can't both read a stale total and overshoot the
// cap. Returns errQuotaExceeded (a sentinel) when the quota would be exceeded.
func (s *Service) insertWithinQuota(ctx context.Context, params sqlc.CreateMediaAssetParams) (sqlc.MediaAsset, error) {
	var row sqlc.MediaAsset
	err := s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.LockWorkspaceForUpdate(ctx, params.WorkspaceID); err != nil {
			return err
		}
		total, err := q.SumMediaBytesForWorkspace(ctx, params.WorkspaceID)
		if err != nil {
			return err
		}
		if total+params.Bytes > s.maxWorkspaceBytes {
			return errQuotaExceeded
		}
		row, err = q.CreateMediaAsset(ctx, params)
		return err
	})
	return row, err
}

// List returns a workspace's media assets, most recent first.
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID) ([]Asset, error) {
	rows, err := s.pool.Queries().ListMediaAssets(ctx, sqlc.ListMediaAssetsParams{
		WorkspaceID: workspaceID, Limit: listLimit, Offset: 0,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]Asset, len(rows))
	for i, r := range rows {
		out[i] = toAsset(r)
	}
	return out, nil
}

// Download streams an asset's bytes; the caller closes the reader.
func (s *Service) Download(ctx context.Context, workspaceID, assetID uuid.UUID) (io.ReadCloser, Asset, error) {
	row, err := s.ownedAsset(ctx, workspaceID, assetID)
	if err != nil {
		return nil, Asset{}, err
	}
	rc, err := s.storage.Get(ctx, row.StorageKey)
	if err != nil {
		return nil, Asset{}, apperr.Internal(err)
	}
	return rc, toAsset(row), nil
}

// Delete removes an asset's object and metadata (workspace-checked).
func (s *Service) Delete(ctx context.Context, workspaceID, assetID uuid.UUID) error {
	row, err := s.ownedAsset(ctx, workspaceID, assetID)
	if err != nil {
		return err
	}
	_ = s.storage.Delete(ctx, row.StorageKey) // best-effort; metadata is authoritative
	if err := s.pool.Queries().DeleteMediaAsset(ctx, assetID); err != nil {
		return apperr.Internal(err)
	}
	s.recordAudit(ctx, workspaceID, "media.delete", assetID.String())
	return nil
}

// ResolveMedia returns an asset's kind/mime/bytes if it belongs to the
// workspace, for the composer to validate attached media. Satisfies
// post.MediaResolver.
func (s *Service) ResolveMedia(ctx context.Context, workspaceID, assetID uuid.UUID) (kind, mime string, bytes int64, err error) {
	row, err := s.ownedAsset(ctx, workspaceID, assetID)
	if err != nil {
		return "", "", 0, err
	}
	return row.Kind, row.Mime, row.Bytes, nil
}

// OpenMedia downloads an asset's bytes by ID for the publish path. The caller
// (worker) has already authorized the owning post/workspace. A missing asset is
// a not-found (terminal for the publish); storage failures are internal (the
// scheduler treats them as retryable). The read is bounded to the asset's
// recorded size so a single job can't allocate without limit.
// MediaURL returns a short-lived presigned GET URL for an asset, for
// platforms that fetch media themselves (Instagram containers).
func (s *Service) MediaURL(ctx context.Context, assetID uuid.UUID, ttl time.Duration) (string, error) {
	row, err := s.pool.Queries().GetMediaAsset(ctx, assetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apperr.NotFound("media_not_found", "media not found")
		}
		return "", apperr.Internal(err)
	}
	url, err := s.storage.PresignGet(ctx, row.StorageKey, ttl)
	if err != nil {
		return "", apperr.Internal(err)
	}
	return url, nil
}

func (s *Service) OpenMedia(ctx context.Context, assetID uuid.UUID) (kind, mime string, data []byte, err error) {
	row, err := s.pool.Queries().GetMediaAsset(ctx, assetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil, apperr.NotFound("media_not_found", "media not found")
		}
		return "", "", nil, apperr.Internal(err)
	}
	if row.Bytes < 0 || row.Bytes > maxVideoBytes {
		return "", "", nil, apperr.Internal(errors.New("media asset has out-of-range size"))
	}
	rc, err := s.storage.Get(ctx, row.StorageKey)
	if err != nil {
		return "", "", nil, apperr.Internal(err)
	}
	defer func() { _ = rc.Close() }()
	buf := make([]byte, row.Bytes)
	if _, rerr := io.ReadFull(rc, buf); rerr != nil {
		return "", "", nil, apperr.Internal(rerr)
	}
	return row.Kind, row.Mime, buf, nil
}

// ownedAsset loads an asset and verifies it belongs to the workspace.
func (s *Service) ownedAsset(ctx context.Context, workspaceID, assetID uuid.UUID) (sqlc.MediaAsset, error) {
	row, err := s.pool.Queries().GetMediaAsset(ctx, assetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.MediaAsset{}, apperr.NotFound("media_not_found", "media not found")
		}
		return sqlc.MediaAsset{}, apperr.Internal(err)
	}
	if row.WorkspaceID != workspaceID {
		return sqlc.MediaAsset{}, apperr.NotFound("media_not_found", "media not found")
	}
	return row, nil
}

func (s *Service) recordAudit(ctx context.Context, workspaceID uuid.UUID, action, target string) {
	if s.audit == nil {
		return
	}
	ws := workspaceID
	_ = s.audit.Record(ctx, security.Event{WorkspaceID: &ws, Action: action, Target: target})
}

func toAsset(r sqlc.MediaAsset) Asset {
	return Asset{
		ID: r.ID, WorkspaceID: r.WorkspaceID, Kind: r.Kind, MIME: r.Mime,
		Width: int(r.Width), Height: int(r.Height), DurationMs: int(r.DurationMs),
		Bytes: r.Bytes, Status: r.Status, CreatedAt: r.CreatedAt.Time,
	}
}
