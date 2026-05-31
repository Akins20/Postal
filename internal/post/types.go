// Package post implements the composer: posts and per-channel content variants
// (compose-once, multi-channel), draft lifecycle, compose-time validation
// against each platform's adapter, and link/UTM tagging. Scheduling/publishing
// is Phase 6; here posts are created and edited as drafts.
package post

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/publish"
)

// Post statuses (Phase 5 creates drafts; later phases drive the rest).
const (
	statusDraft = "draft"
)

// MediaMeta is media attached to a variant. MediaID must reference an uploaded
// media asset (Phase 7); the composer resolves it and fills the authoritative
// kind/mime/bytes from the asset, so client-supplied values are advisory only.
// A media entry without a MediaID is rejected at compose time.
type MediaMeta struct {
	MediaID uuid.UUID `json:"media_id"`
	Kind    string    `json:"kind"` // image | gif | video
	MIME    string    `json:"mime"`
	Bytes   int64     `json:"bytes"`
}

// VariantInput is the client-supplied content for one channel.
type VariantInput struct {
	ChannelID       uuid.UUID      `json:"channel_id"`
	Body            string         `json:"body"`
	Media           []MediaMeta    `json:"media"`
	PlatformOptions map[string]any `json:"platform_options"`
}

// Variant is a stored per-channel content variant.
type Variant struct {
	ID              uuid.UUID      `json:"id"`
	ChannelID       uuid.UUID      `json:"channel_id"`
	Body            string         `json:"body"`
	Media           []MediaMeta    `json:"media"`
	PlatformOptions map[string]any `json:"platform_options"`
}

// Post is the logical post plus its per-channel variants.
type Post struct {
	ID           uuid.UUID  `json:"id"`
	WorkspaceID  uuid.UUID  `json:"workspace_id"`
	AuthorUserID *uuid.UUID `json:"author_user_id"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	Variants     []Variant  `json:"variants,omitempty"`
}

// VariantValidation is the compose-time validation result for one variant.
type VariantValidation struct {
	ChannelID uuid.UUID `json:"channel_id"`
	Valid     bool      `json:"valid"`
	Code      string    `json:"code,omitempty"`
	Message   string    `json:"message,omitempty"`
}

// ChannelResolver maps a channel to its platform, enforcing workspace ownership.
// The channel.Service satisfies it.
type ChannelResolver interface {
	PlatformFor(ctx context.Context, workspaceID, channelID uuid.UUID) (string, error)
}

// Validator checks a variant against a platform's constraints. The
// publish.Registry satisfies it.
type Validator interface {
	Validate(platform string, v publish.PostVariant) error
}

// MediaResolver verifies an attached media asset belongs to the workspace and
// returns its authoritative kind/mime/bytes. media.Service satisfies it. Nil
// when the media pipeline is disabled.
type MediaResolver interface {
	ResolveMedia(ctx context.Context, workspaceID, assetID uuid.UUID) (kind, mime string, bytes int64, err error)
}
