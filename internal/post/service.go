package post

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/security"
)

// listLimit caps how many posts a list query returns (pagination is a later
// enhancement; for now the most-recent listLimit posts are returned).
const listLimit = 100

// maxVariantsPerPost bounds variants per post (anti-abuse: each variant costs a
// channel lookup + an insert). A real post fans out to a handful of connected
// channels, so this is generous.
const maxVariantsPerPost = 50

// Service implements the composer over posts/post_variants, resolving channels
// and validating variants against platform adapters.
type Service struct {
	pool      *db.Pool
	channels  ChannelResolver
	validator Validator
	media     MediaResolver
	audit     security.Recorder
	clock     func() time.Time
}

// NewService builds a post Service. media may be nil (media pipeline disabled);
// clock defaults to time.Now.
func NewService(pool *db.Pool, channels ChannelResolver, validator Validator, media MediaResolver, audit security.Recorder, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{pool: pool, channels: channels, validator: validator, media: media, audit: audit, clock: clock}
}

// Create creates a draft post with one variant per input channel. Each channel
// must belong to the workspace, and a post may have at most one variant per
// channel. Drafts may hold not-yet-valid content; use Validate for feedback.
func (s *Service) Create(ctx context.Context, workspaceID, authorID uuid.UUID, inputs []VariantInput) (Post, error) {
	encoded, err := s.prepareVariants(ctx, workspaceID, inputs)
	if err != nil {
		return Post{}, err
	}

	var row sqlc.Post
	variants := make([]Variant, 0, len(encoded))
	err = s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		p, err := q.CreatePost(ctx, sqlc.CreatePostParams{
			WorkspaceID: workspaceID, AuthorUserID: &authorID, Status: statusDraft,
		})
		if err != nil {
			return err
		}
		row = p
		for _, e := range encoded {
			v, err := q.CreatePostVariant(ctx, sqlc.CreatePostVariantParams{
				PostID: p.ID, ChannelID: e.channelID, Body: e.body,
				MediaRefs: e.media, PlatformOptions: e.opts,
			})
			if err != nil {
				return err
			}
			variants = append(variants, toVariant(v))
		}
		return nil
	})
	if err != nil {
		return Post{}, mapVariantTxErr(err)
	}
	s.recordAudit(ctx, workspaceID, authorID, "post.create", row.ID.String())
	return toPost(row, variants), nil
}

// Get returns a post with its variants, enforcing workspace ownership.
func (s *Service) Get(ctx context.Context, workspaceID, postID uuid.UUID) (Post, error) {
	row, err := s.loadOwnedPost(ctx, workspaceID, postID)
	if err != nil {
		return Post{}, err
	}
	vrows, err := s.pool.Queries().ListVariantsByPost(ctx, postID)
	if err != nil {
		return Post{}, apperr.Internal(err)
	}
	variants := make([]Variant, len(vrows))
	for i, v := range vrows {
		variants[i] = toVariant(v)
	}
	return toPost(row, variants), nil
}

// List returns the workspace's posts (without variants), most recent first.
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID) ([]Post, error) {
	rows, err := s.pool.Queries().ListPostsByWorkspace(ctx, sqlc.ListPostsByWorkspaceParams{
		WorkspaceID: workspaceID, Limit: listLimit, Offset: 0,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	posts := make([]Post, len(rows))
	for i, r := range rows {
		posts[i] = toPost(r, nil)
	}
	return posts, nil
}

// Update replaces a post's variants with the given inputs (full replace).
func (s *Service) Update(ctx context.Context, workspaceID, authorID, postID uuid.UUID, inputs []VariantInput) (Post, error) {
	row, err := s.loadOwnedPost(ctx, workspaceID, postID)
	if err != nil {
		return Post{}, err
	}
	encoded, err := s.prepareVariants(ctx, workspaceID, inputs)
	if err != nil {
		return Post{}, err
	}

	// Full replace: delete then recreate variants. NOTE: this regenerates variant
	// IDs on every edit; if a later phase (scheduling/analytics) references a
	// variant ID, switch to an upsert keyed on (post_id, channel_id).
	variants := make([]Variant, 0, len(encoded))
	err = s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.DeleteVariantsForPost(ctx, postID); err != nil {
			return err
		}
		for _, e := range encoded {
			v, err := q.CreatePostVariant(ctx, sqlc.CreatePostVariantParams{
				PostID: postID, ChannelID: e.channelID, Body: e.body,
				MediaRefs: e.media, PlatformOptions: e.opts,
			})
			if err != nil {
				return err
			}
			variants = append(variants, toVariant(v))
		}
		return q.TouchPost(ctx, postID) // bump updated_at
	})
	if err != nil {
		return Post{}, mapVariantTxErr(err)
	}
	s.recordAudit(ctx, workspaceID, authorID, "post.update", postID.String())
	return toPost(row, variants), nil
}

// mapVariantTxErr maps a variant-insert transaction error: a foreign-key
// violation means a referenced channel was removed concurrently (a clean
// not-found, not a 500); anything else is internal.
func mapVariantTxErr(err error) error {
	if db.IsForeignKeyViolation(err) {
		return apperr.NotFound("channel_not_found", "a target channel no longer exists")
	}
	return apperr.Internal(err)
}

// Delete removes a post (and its variants via cascade), enforcing ownership.
func (s *Service) Delete(ctx context.Context, workspaceID, authorID, postID uuid.UUID) error {
	if _, err := s.loadOwnedPost(ctx, workspaceID, postID); err != nil {
		return err
	}
	if err := s.pool.Queries().DeletePost(ctx, postID); err != nil {
		return apperr.Internal(err)
	}
	s.recordAudit(ctx, workspaceID, authorID, "post.delete", postID.String())
	return nil
}

// Validate runs compose-time validation of every variant against its channel's
// platform adapter, returning per-variant results (instant composer feedback).
func (s *Service) Validate(ctx context.Context, workspaceID, postID uuid.UUID) ([]VariantValidation, error) {
	p, err := s.Get(ctx, workspaceID, postID)
	if err != nil {
		return nil, err
	}
	results := make([]VariantValidation, 0, len(p.Variants))
	for _, v := range p.Variants {
		results = append(results, s.validateVariant(ctx, workspaceID, v))
	}
	return results, nil
}

// validateVariant resolves the platform and validates one variant.
func (s *Service) validateVariant(ctx context.Context, workspaceID uuid.UUID, v Variant) VariantValidation {
	platform, err := s.channels.PlatformFor(ctx, workspaceID, v.ChannelID)
	if err != nil {
		return VariantValidation{ChannelID: v.ChannelID, Valid: false, Code: "channel_unavailable", Message: "channel not found or not in this workspace"}
	}
	pv := publish.PostVariant{Text: v.Body, Media: toPublishMedia(v.Media)}
	if verr := s.validator.Validate(platform, pv); verr != nil {
		var pe *publish.Error
		if errors.As(verr, &pe) {
			return VariantValidation{ChannelID: v.ChannelID, Valid: false, Code: pe.Code, Message: pe.Message}
		}
		return VariantValidation{ChannelID: v.ChannelID, Valid: false, Code: "invalid", Message: verr.Error()}
	}
	return VariantValidation{ChannelID: v.ChannelID, Valid: true}
}

// encodedVariant is a validated, JSON-encoded variant ready for insertion.
type encodedVariant struct {
	channelID   uuid.UUID
	body        string
	media, opts []byte
}

// prepareVariants validates channel ownership + uniqueness and JSON-encodes the
// media/options for storage. Returns a validation error on empty input,
// duplicate channels, or a foreign channel.
func (s *Service) prepareVariants(ctx context.Context, workspaceID uuid.UUID, inputs []VariantInput) ([]encodedVariant, error) {
	if len(inputs) == 0 {
		return nil, apperr.Validation("no_variants", "a post needs at least one channel variant")
	}
	if len(inputs) > maxVariantsPerPost {
		return nil, apperr.Validation("too_many_variants", "too many channel variants in one post")
	}
	seen := make(map[uuid.UUID]struct{}, len(inputs))
	out := make([]encodedVariant, 0, len(inputs))
	for _, in := range inputs {
		if _, dup := seen[in.ChannelID]; dup {
			return nil, apperr.Validation("duplicate_channel", "only one variant per channel is allowed")
		}
		seen[in.ChannelID] = struct{}{}
		if _, err := s.channels.PlatformFor(ctx, workspaceID, in.ChannelID); err != nil {
			return nil, err // not-found for a foreign/unknown channel
		}
		media, err := s.resolveMedia(ctx, workspaceID, in.Media)
		if err != nil {
			return nil, err
		}
		out = append(out, encodedVariant{
			channelID: in.ChannelID, body: in.Body,
			media: marshalJSON(media, "[]"), opts: marshalJSON(in.PlatformOptions, "{}"),
		})
	}
	return out, nil
}

// resolveMedia validates each media reference: a referenced asset (MediaID set)
// must belong to the workspace, and its authoritative kind/mime/bytes replace
// any client-supplied values.
func (s *Service) resolveMedia(ctx context.Context, workspaceID uuid.UUID, media []MediaMeta) ([]MediaMeta, error) {
	if len(media) == 0 {
		return nil, nil
	}
	if s.media == nil {
		return nil, apperr.Validation("media_unavailable", "media uploads are not configured")
	}
	out := make([]MediaMeta, len(media))
	copy(out, media)
	for i := range out {
		// Every attached media must reference an uploaded asset. Otherwise it
		// would pass compose-time validation (counting toward platform limits)
		// but be dropped at publish — a green validation that fails when run.
		if out[i].MediaID == uuid.Nil {
			return nil, apperr.Validation("media_unresolved", "attached media must reference an uploaded asset")
		}
		kind, mime, bytes, err := s.media.ResolveMedia(ctx, workspaceID, out[i].MediaID)
		if err != nil {
			return nil, err // not-found for a foreign/unknown asset
		}
		out[i].Kind, out[i].MIME, out[i].Bytes = kind, mime, bytes
	}
	return out, nil
}

// loadOwnedPost loads a post and verifies it belongs to the workspace.
func (s *Service) loadOwnedPost(ctx context.Context, workspaceID, postID uuid.UUID) (sqlc.Post, error) {
	row, err := s.pool.Queries().GetPost(ctx, postID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Post{}, apperr.NotFound("post_not_found", "post not found")
		}
		return sqlc.Post{}, apperr.Internal(err)
	}
	if row.WorkspaceID != workspaceID {
		return sqlc.Post{}, apperr.NotFound("post_not_found", "post not found")
	}
	return row, nil
}

func (s *Service) recordAudit(ctx context.Context, workspaceID, actorID uuid.UUID, action, target string) {
	if s.audit == nil {
		return
	}
	ws := workspaceID
	_ = s.audit.Record(ctx, security.Event{WorkspaceID: &ws, ActorUserID: &actorID, Action: action, Target: target})
}

// --- mapping helpers ---

func toVariant(v sqlc.PostVariant) Variant {
	return Variant{
		ID: v.ID, ChannelID: v.ChannelID, Body: v.Body,
		Media: unmarshalMedia(v.MediaRefs), PlatformOptions: unmarshalOpts(v.PlatformOptions),
	}
}

func toPost(p sqlc.Post, variants []Variant) Post {
	return Post{
		ID: p.ID, WorkspaceID: p.WorkspaceID, AuthorUserID: p.AuthorUserID,
		Status: p.Status, CreatedAt: p.CreatedAt.Time, Variants: variants,
	}
}

func toPublishMedia(media []MediaMeta) []publish.MediaRef {
	if len(media) == 0 {
		return nil
	}
	out := make([]publish.MediaRef, len(media))
	for i, m := range media {
		out[i] = publish.MediaRef{Kind: publish.MediaKind(m.Kind), MIME: m.MIME, Bytes: m.Bytes}
	}
	return out
}

// marshalJSON encodes v to JSON, returning fallback for nil/empty.
func marshalJSON(v any, fallback string) []byte {
	if v == nil {
		return []byte(fallback)
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return []byte(fallback)
	}
	return b
}

func unmarshalMedia(b []byte) []MediaMeta {
	var m []MediaMeta
	_ = json.Unmarshal(b, &m)
	return m
}

func unmarshalOpts(b []byte) map[string]any {
	var o map[string]any
	_ = json.Unmarshal(b, &o)
	return o
}
