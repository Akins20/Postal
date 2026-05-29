package publish

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/channel"
)

// Default retry policy for the pipeline.
const (
	defaultMaxAttempts = 5
	baseBackoff        = time.Second
	maxBackoff         = 30 * time.Second
)

// Channels supplies the pipeline with a channel's platform + decrypted access
// token and a way to refresh it. The channel.Service satisfies this.
type Channels interface {
	PublishContext(ctx context.Context, channelID uuid.UUID) (platform string, token channel.Token, err error)
	Refresh(ctx context.Context, channelID uuid.UUID) (channel.Token, error)
}

// Results persists and looks up publish results for idempotency. Backed by
// publish_results.
type Results interface {
	Find(ctx context.Context, idempotencyKey string) (*Result, bool, error)
	Record(ctx context.Context, channelID uuid.UUID, idempotencyKey string, res *Result) error
}

// Pipeline validates and publishes a post variant to a channel, refreshing
// expired tokens, backing off on retryable failures, and never double-posting
// for a given idempotency key.
type Pipeline struct {
	adapters    map[string]Adapter
	channels    Channels
	results     Results
	maxAttempts int
	sleep       func(context.Context, time.Duration) error
}

// Option configures a Pipeline.
type Option func(*Pipeline)

// WithMaxAttempts overrides the retry attempt cap.
func WithMaxAttempts(n int) Option {
	return func(p *Pipeline) {
		if n > 0 {
			p.maxAttempts = n
		}
	}
}

// WithSleeper overrides the backoff sleeper (tests inject a no-op to avoid waits).
func WithSleeper(fn func(context.Context, time.Duration) error) Option {
	return func(p *Pipeline) {
		if fn != nil {
			p.sleep = fn
		}
	}
}

// NewPipeline builds a Pipeline from the channel provider, the results store,
// the platform adapters (keyed by Platform()), and optional overrides.
func NewPipeline(channels Channels, results Results, adapters []Adapter, opts ...Option) *Pipeline {
	m := make(map[string]Adapter, len(adapters))
	for _, a := range adapters {
		m[a.Platform()] = a
	}
	p := &Pipeline{
		adapters:    m,
		channels:    channels,
		results:     results,
		maxAttempts: defaultMaxAttempts,
		sleep:       sleepCtx,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Publish sends v to channelID. If v.IdempotencyKey was already published, the
// recorded result is returned without re-posting.
func (p *Pipeline) Publish(ctx context.Context, channelID uuid.UUID, v PostVariant) (*Result, error) {
	if v.IdempotencyKey != "" {
		if res, found, err := p.results.Find(ctx, v.IdempotencyKey); err != nil {
			return nil, err
		} else if found {
			return res, nil // already published — do not double-post
		}
	}

	platform, token, err := p.channels.PublishContext(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("loading publish context: %w", err)
	}
	adapter, ok := p.adapters[platform]
	if !ok {
		return nil, fmt.Errorf("no adapter for platform %q", platform)
	}
	// Pre-flight validation here fails fast (terminal) before any token load /
	// API call. The adapter also re-validates inside Publish so it stays correct
	// when invoked directly (e.g. a future worker bypassing the pipeline); the
	// duplicate cost is negligible versus the network round trip it prevents.
	if err := adapter.Validate(v); err != nil {
		return nil, err
	}

	return p.attemptPublish(ctx, channelID, adapter, token, v)
}

// attemptPublish runs the publish/retry loop, handling auth-expired (refresh
// once, without consuming a retry slot) and retryable (backoff) errors per the
// adapter's error classes.
func (p *Pipeline) attemptPublish(ctx context.Context, channelID uuid.UUID, adapter Adapter, token channel.Token, v PostVariant) (*Result, error) {
	refreshed := false

	for attempt := 1; ; attempt++ {
		res, err := adapter.Publish(ctx, token, v)
		if err == nil {
			if recErr := p.record(ctx, channelID, v, res); recErr != nil {
				return nil, recErr
			}
			return res, nil
		}

		var ae *Error
		if !errors.As(err, &ae) {
			return nil, err // unclassified — do not retry blindly
		}
		switch ae.Class {
		case ClassTerminal:
			return nil, err
		case ClassAuthExpired:
			// Refresh once and retry immediately; the refresh does not consume a
			// retry attempt, so the full backoff budget remains for transient
			// failures after re-auth.
			if refreshed {
				return nil, err
			}
			newToken, rErr := p.channels.Refresh(ctx, channelID)
			if rErr != nil {
				return nil, fmt.Errorf("refreshing token: %w", rErr)
			}
			token = newToken
			refreshed = true
			attempt-- // do not count the refresh as an attempt
		case ClassRetryable:
			if attempt >= p.maxAttempts {
				return nil, err
			}
			if sErr := p.sleep(ctx, backoff(attempt, ae.RetryAfter)); sErr != nil {
				return nil, sErr
			}
		}
	}
}

// record stores a successful publish under its idempotency key (best-effort: a
// missing key skips recording).
func (p *Pipeline) record(ctx context.Context, channelID uuid.UUID, v PostVariant, res *Result) error {
	if v.IdempotencyKey == "" {
		return nil
	}
	return p.results.Record(ctx, channelID, v.IdempotencyKey, res)
}

// backoff returns the wait before the next attempt: the platform's RetryAfter
// when provided, else exponential (capped).
func backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > maxBackoff {
			return maxBackoff
		}
		return retryAfter
	}
	d := baseBackoff << (attempt - 1)
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// sleepCtx waits for d or until ctx is canceled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
