// Package publish defines the platform-agnostic publishing pipeline: the
// PlatformAdapter contract every social network implements, the constraint and
// error model the pipeline reasons about, and the service that validates,
// publishes, retries, and records results. Per-platform adapters live in
// subpackages (e.g. publish/twitter); tests run against publish/simulator.
package publish

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Akins20/postal/internal/channel"
)

// MediaKind enumerates the kinds of media a post may carry.
type MediaKind string

const (
	// MediaImage is a still image (JPEG/PNG/WebP).
	MediaImage MediaKind = "image"
	// MediaGIF is an animated GIF.
	MediaGIF MediaKind = "gif"
	// MediaVideo is a video.
	MediaVideo MediaKind = "video"
)

// MediaRef describes one media item attached to a post. PlatformMediaID is set
// by the adapter after upload and referenced when creating the post.
type MediaRef struct {
	Kind            MediaKind
	MIME            string
	Bytes           int64
	Data            []byte // raw bytes for upload (omitted once uploaded)
	PlatformMediaID string
}

// PostVariant is the per-channel content to publish. IdempotencyKey makes
// publishing safe to retry: the pipeline never creates two posts for one key.
type PostVariant struct {
	Text           string
	Media          []MediaRef
	InReplyToID    string
	IdempotencyKey string
}

// Constraints declares a platform's publishing limits as data, so validation is
// table-driven rather than hard-coded per platform.
type Constraints struct {
	MaxWeightedTextLen int
	URLWeight          int // weighted length charged per URL
	MaxImages          int
	MaxGifs            int
	MaxVideos          int
	MaxImageBytes      int64
	MaxGIFBytes        int64
	MaxVideoBytes      int64
}

// Result is the outcome of a successful publish.
type Result struct {
	PlatformPostID string
	Raw            json.RawMessage
}

// Metric is a single analytics datapoint for a published post.
type Metric struct {
	Name  string
	Value int64
}

// Adapter is implemented per social network. It embeds the OAuth surface used
// to connect channels (Phase 3) and adds validation, publishing, and metrics.
// Every method must be exercisable against the platform simulator (the adapter
// takes an injectable base URL at construction).
type Adapter interface {
	channel.OAuthProvider

	// Constraints returns the platform's publishing limits.
	Constraints() Constraints
	// Validate checks a variant against Constraints before any API call.
	Validate(v PostVariant) error
	// Publish creates the post (uploading media first if present) and returns
	// the platform post ID. token is the decrypted access token.
	Publish(ctx context.Context, token channel.Token, v PostVariant) (*Result, error)
	// FetchMetrics returns analytics for a previously published post.
	FetchMetrics(ctx context.Context, token channel.Token, platformPostID string) ([]Metric, error)
}

// Class categorizes an adapter error so the pipeline knows how to react.
type Class int

const (
	// ClassTerminal must not be retried (validation, duplicate, forbidden, 400).
	ClassTerminal Class = iota
	// ClassRetryable should be retried with backoff (429, 5xx, network).
	ClassRetryable
	// ClassAuthExpired means refresh the token once, then retry (401).
	ClassAuthExpired
)

// Error is an adapter error carrying its retry class and platform context. The
// pipeline inspects Class (and RetryAfter for 429) to drive retry/backoff.
type Error struct {
	Class      Class
	Code       string
	Message    string
	RetryAfter time.Duration
	wrapped    error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.wrapped != nil {
		return e.Code + ": " + e.Message + ": " + e.wrapped.Error()
	}
	return e.Code + ": " + e.Message
}

// Unwrap exposes the wrapped cause.
func (e *Error) Unwrap() error { return e.wrapped }

// Retryable reports whether the error is transient (rate-limited or server
// error). Exposed as a method so other packages (e.g. channel) can detect
// transience structurally — via an interface{ Retryable() bool } assertion —
// without importing this package (avoiding an import cycle).
func (e *Error) Retryable() bool { return e.Class == ClassRetryable }

// Terminal builds a non-retryable adapter error.
func Terminal(code, message string, cause error) *Error {
	return &Error{Class: ClassTerminal, Code: code, Message: message, wrapped: cause}
}

// Retryable builds a retryable adapter error.
func Retryable(code, message string, cause error) *Error {
	return &Error{Class: ClassRetryable, Code: code, Message: message, wrapped: cause}
}

// RateLimited builds a retryable error carrying the backoff hint from the
// platform's rate-limit reset.
func RateLimited(retryAfter time.Duration, cause error) *Error {
	return &Error{Class: ClassRetryable, Code: "rate_limited", Message: "rate limited", RetryAfter: retryAfter, wrapped: cause}
}

// AuthExpired builds an error signaling the token should be refreshed and the
// publish retried once.
func AuthExpired(cause error) *Error {
	return &Error{Class: ClassAuthExpired, Code: "token_expired", Message: "access token expired or invalid", wrapped: cause}
}
