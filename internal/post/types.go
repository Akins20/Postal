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

// MediaMeta is media metadata attached to a variant for compose-time validation.
// Actual upload/storage is the Phase 7 media pipeline; here it's just the facts
// needed to validate counts/sizes against a platform.
type MediaMeta struct {
	Kind  string `json:"kind"` // image | gif | video
	MIME  string `json:"mime"`
	Bytes int64  `json:"bytes"`
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
