// Package media implements the media pipeline: upload to S3-compatible object
// storage (Cloudflare R2 in prod, MinIO in dev), per-platform/size validation,
// per-workspace storage quota, and serving bytes into the publish path. Image
// dimensions are detected with the standard library; video transcode/probe
// (FFmpeg) and image resize (libvips) are pluggable and deferred.
package media

import (
	"time"

	"github.com/google/uuid"
)

// Media kinds.
const (
	KindImage = "image"
	KindGIF   = "gif"
	KindVideo = "video"
)

// Per-kind upload size caps, aligned with X's media rules. The overall
// per-file cap (config) still applies on top.
// NOTE: keep in sync with the X adapter's limits in
// internal/publish/twitter/adapter.go (maxImageBytes/maxGIFBytes/maxVideoBytes);
// upload-time and publish-time validation must agree.
const (
	maxImageBytes = 5 << 20   // 5 MiB
	maxGIFBytes   = 15 << 20  // 15 MiB
	maxVideoBytes = 512 << 20 // 512 MiB
)

// mimeKinds maps accepted content types to a media kind. Only formats the
// stdlib can decode dimensions for are accepted as images (WebP is omitted: no
// decoder is registered, so it would store zero dimensions).
var mimeKinds = map[string]string{
	"image/jpeg":      KindImage,
	"image/png":       KindImage,
	"image/gif":       KindGIF,
	"video/mp4":       KindVideo,
	"video/quicktime": KindVideo,
}

// Asset is an uploaded media asset's metadata (the bytes live in object storage).
type Asset struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Kind        string    `json:"kind"`
	MIME        string    `json:"mime"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	DurationMs  int       `json:"duration_ms"`
	Bytes       int64     `json:"bytes"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// kindForMIME returns the media kind for a content type and whether it's allowed.
func kindForMIME(mime string) (string, bool) {
	k, ok := mimeKinds[mime]
	return k, ok
}

// maxBytesForKind returns the per-kind size cap.
func maxBytesForKind(kind string) int64 {
	switch kind {
	case KindGIF:
		return maxGIFBytes
	case KindVideo:
		return maxVideoBytes
	default:
		return maxImageBytes
	}
}
