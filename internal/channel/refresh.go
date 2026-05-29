package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// ErrNoRefreshToken indicates a channel credential has no refresh token to use.
var ErrNoRefreshToken = errors.New("channel: no refresh token available")

// RefreshChannel obtains a new token set for a channel using its stored refresh
// token, re-encrypts it, and updates the credential. On provider failure the
// channel is marked expired so the user is prompted to reconnect. This is
// invoked by the refresh worker (Phase 6) and is safe to call directly in tests.
func (s *Service) RefreshChannel(ctx context.Context, channelID uuid.UUID) error {
	ch, err := s.pool.Queries().GetChannel(ctx, channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("refresh channel %s: %w", channelID, pgx.ErrNoRows)
		}
		return fmt.Errorf("loading channel: %w", err)
	}
	cred, err := s.pool.Queries().GetChannelCredential(ctx, channelID)
	if err != nil {
		return fmt.Errorf("loading credential: %w", err)
	}

	refreshToken, ok, err := s.vault.openRefresh(cred)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoRefreshToken
	}

	provider, ok := s.registry.Get(ch.Platform)
	if !ok {
		return fmt.Errorf("no provider for platform %q", ch.Platform)
	}

	token, err := provider.RefreshToken(ctx, refreshToken)
	if err != nil {
		// Only a terminal failure (e.g. invalid_grant) means the user must
		// reconnect; a transient error (429/5xx/network) leaves the channel
		// active so a later refresh can succeed.
		if !isRetryable(err) {
			s.markExpired(ctx, ch, err)
		}
		return fmt.Errorf("refreshing token: %w", err)
	}

	sealed, err := s.vault.seal(token)
	if err != nil {
		return err
	}
	if err := s.pool.Queries().UpsertChannelCredential(ctx, sqlc.UpsertChannelCredentialParams{
		ChannelID:             ch.ID,
		EncryptedAccessToken:  sealed.accessCipher,
		EncryptedRefreshToken: sealed.refreshCipher,
		Scopes:                token.Scopes,
		ExpiresAt:             expiryTS(token.ExpiresAt),
		KeyVersion:            sealed.keyVersion,
	}); err != nil {
		return fmt.Errorf("storing refreshed credential: %w", err)
	}
	s.recordAudit(ctx, ch.WorkspaceID, uuid.Nil, "channel.token_refreshed", ch.ID.String(),
		map[string]any{"platform": ch.Platform})
	return nil
}

// PublishContext returns a channel's platform and decrypted access token for
// the publish pipeline. It errors if the channel is missing or not active.
func (s *Service) PublishContext(ctx context.Context, channelID uuid.UUID) (string, Token, error) {
	ch, err := s.pool.Queries().GetChannel(ctx, channelID)
	if err != nil {
		return "", Token{}, fmt.Errorf("loading channel: %w", err)
	}
	if ch.Status != statusActive {
		return "", Token{}, fmt.Errorf("channel %s is not active (status %q)", channelID, ch.Status)
	}
	cred, err := s.pool.Queries().GetChannelCredential(ctx, channelID)
	if err != nil {
		return "", Token{}, fmt.Errorf("loading credential: %w", err)
	}
	access, err := s.vault.openAccess(cred)
	if err != nil {
		return "", Token{}, err
	}
	return ch.Platform, Token{AccessToken: access}, nil
}

// Refresh refreshes a channel's token (persisting it) and returns the new
// decrypted access token, for the pipeline's auth-expired retry path.
func (s *Service) Refresh(ctx context.Context, channelID uuid.UUID) (Token, error) {
	if err := s.RefreshChannel(ctx, channelID); err != nil {
		return Token{}, err
	}
	_, tok, err := s.PublishContext(ctx, channelID)
	return tok, err
}

// DueForRefresh returns channels whose credentials expire before `before`, for
// the refresh worker to process (Phase 6).
func (s *Service) DueForRefresh(ctx context.Context, before time.Time, limit int32) ([]uuid.UUID, error) {
	rows, err := s.pool.Queries().ListChannelsDueForRefresh(ctx, sqlc.ListChannelsDueForRefreshParams{
		ExpiresAt: expiryTS(before),
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing channels due for refresh: %w", err)
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, nil
}

// isRetryable reports whether err signals a transient failure, detected
// structurally (the provider's publish.Error exposes Retryable) so channel need
// not import the publish package.
func isRetryable(err error) bool {
	var r interface{ Retryable() bool }
	return errors.As(err, &r) && r.Retryable()
}

// markExpired flags a channel expired after a failed refresh and audits it.
func (s *Service) markExpired(ctx context.Context, ch sqlc.Channel, cause error) {
	_ = s.pool.Queries().UpdateChannelStatus(ctx, sqlc.UpdateChannelStatusParams{ID: ch.ID, Status: statusExpired})
	s.recordAudit(ctx, ch.WorkspaceID, uuid.Nil, "channel.token_refresh_failed", ch.ID.String(),
		map[string]any{"platform": ch.Platform, "error": cause.Error()})
}
