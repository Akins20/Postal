package channel

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/workspace"
)

// maxChannelsPerWorkspace caps connected social accounts per workspace
// (anti-abuse: each channel consumes shared upstream API quota and a refresh
// slot). Generous for real teams; blocks scripted mass-connect. A var so tests
// can lower it.
var maxChannelsPerWorkspace int64 = 25

// errChannelQuota is a sentinel returned from the connect transaction when the
// per-workspace channel limit is reached; mapped to a validation error.
var errChannelQuota = errors.New("channel quota exceeded")

// Authorizer resolves a user's workspace membership for capability checks. The
// workspace.Service satisfies it; defining the interface here avoids a hard
// dependency and keeps the channel domain testable.
type Authorizer interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (workspace.Member, error)
}

// Service manages channel connection, listing, and disconnection over the OAuth
// providers, credential vault, OAuth-state store, and audit log.
type Service struct {
	pool     *db.Pool
	registry *Registry
	vault    *Vault
	states   *stateStore
	authz    Authorizer
	audit    security.Recorder
	clock    func() time.Time
	// allowedRedirects bounds client-supplied OAuth redirect_uris (anti-open-redirect).
	allowedRedirects map[string]struct{}
}

// NewService builds a channel Service. clock defaults to time.Now.
func NewService(pool *db.Pool, registry *Registry, enc *security.Encryptor, rdb stateRedis, authz Authorizer, audit security.Recorder, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{
		pool:     pool,
		registry: registry,
		vault:    NewVault(enc),
		states:   &stateStore{rdb: rdb},
		authz:    authz,
		audit:    audit,
		clock:    clock,
	}
}

// View is the client-safe representation of a channel. It never includes
// credentials.
type View struct {
	ID                uuid.UUID  `json:"id"`
	Platform          string     `json:"platform"`
	PlatformAccountID string     `json:"platform_account_id"`
	Handle            string     `json:"handle"`
	DisplayName       string     `json:"display_name"`
	Status            string     `json:"status"`
	ConnectedBy       *uuid.UUID `json:"connected_by"`
	CreatedAt         time.Time  `json:"created_at"`
}

// StartConnect begins an OAuth flow: it generates PKCE + CSRF state, stores the
// flow context, and returns the provider authorize URL. The caller's
// manage_channels capability is enforced by route middleware.
// AllowRedirects sets the allowlist of client-supplied OAuth redirect URIs
// (e.g. the web callback page and a native deep link). Clients may only
// override the adapter default with a URI in this set. Returns s for chaining.
func (s *Service) AllowRedirects(uris []string) *Service {
	s.allowedRedirects = make(map[string]struct{}, len(uris))
	for _, u := range uris {
		if u != "" {
			s.allowedRedirects[u] = struct{}{}
		}
	}
	return s
}

func (s *Service) StartConnect(ctx context.Context, workspaceID, userID uuid.UUID, platform, redirectURI string) (string, error) {
	provider, ok := s.registry.Get(platform)
	if !ok {
		return "", apperr.Validation("unsupported_platform", "unsupported platform: "+platform).
			WithField("platform", "unsupported")
	}
	if redirectURI != "" {
		if _, ok := s.allowedRedirects[redirectURI]; !ok {
			return "", apperr.Validation("invalid_redirect", "redirect_uri is not allowed").
				WithField("redirect_uri", "not allowed")
		}
	}
	verifier, challenge, err := newPKCE()
	if err != nil {
		return "", apperr.Internal(err)
	}
	state, err := s.states.save(ctx, oauthState{
		WorkspaceID:  workspaceID,
		UserID:       userID,
		Platform:     platform,
		CodeVerifier: verifier,
		RedirectURI:  redirectURI,
	})
	if err != nil {
		return "", apperr.Internal(err)
	}
	return provider.AuthURL(state, challenge, redirectURI), nil
}

// CompleteConnect finishes an OAuth flow: it validates the single-use state,
// re-verifies the caller initiated it and still holds manage_channels, exchanges
// the code, resolves the account, and stores the encrypted credentials.
func (s *Service) CompleteConnect(ctx context.Context, callerUserID uuid.UUID, state, code string) (View, error) {
	st, err := s.states.consume(ctx, state)
	if err != nil {
		return View{}, apperr.Validation("invalid_state", "the authorization link is invalid or has expired")
	}
	if st.UserID != callerUserID {
		return View{}, apperr.Forbidden("state_mismatch", "this authorization was started by a different user")
	}
	if err := s.requireManageChannels(ctx, st.WorkspaceID, callerUserID); err != nil {
		return View{}, err
	}

	provider, ok := s.registry.Get(st.Platform)
	if !ok {
		return View{}, apperr.Validation("unsupported_platform", "unsupported platform")
	}

	token, err := provider.ExchangeCode(ctx, code, st.CodeVerifier, st.RedirectURI)
	if err != nil {
		return View{}, apperr.Validation("oauth_exchange_failed", "could not complete authorization with the provider")
	}
	account, err := provider.Account(ctx, token.AccessToken)
	if err != nil {
		return View{}, apperr.Internal(err)
	}
	sealed, err := s.vault.seal(token)
	if err != nil {
		return View{}, apperr.Internal(err)
	}

	ch, err := s.persistChannel(ctx, st, callerUserID, account, sealed, token.Scopes, token.ExpiresAt)
	if err != nil {
		return View{}, err
	}
	s.recordAudit(ctx, st.WorkspaceID, callerUserID, "channel.connect", ch.ID.String(),
		map[string]any{"platform": st.Platform, "handle": account.Handle})
	return toView(ch), nil
}

// persistChannel upserts the channel row and its credential atomically.
func (s *Service) persistChannel(ctx context.Context, st oauthState, connectedBy uuid.UUID, account *Account, sealed sealedCredential, scopes []string, expiresAt time.Time) (sqlc.Channel, error) {
	var ch sqlc.Channel
	err := s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		existing, getErr := q.GetChannelByAccount(ctx, sqlc.GetChannelByAccountParams{
			WorkspaceID: st.WorkspaceID, Platform: st.Platform, PlatformAccountID: account.ID,
		})
		switch {
		case getErr == nil:
			ch = existing
			if err := q.UpdateChannelIdentity(ctx, sqlc.UpdateChannelIdentityParams{ID: ch.ID, Handle: account.Handle, DisplayName: account.DisplayName}); err != nil {
				return err
			}
			if err := q.UpdateChannelStatus(ctx, sqlc.UpdateChannelStatusParams{ID: ch.ID, Status: statusActive}); err != nil {
				return err
			}
			ch.Status = statusActive
			ch.Handle = account.Handle
			ch.DisplayName = account.DisplayName
		case errors.Is(getErr, pgx.ErrNoRows):
			// Anti-abuse: cap connected channels per workspace. Counted inside the
			// tx so concurrent connects can't both slip past the limit.
			count, cErr := q.CountActiveChannelsForWorkspace(ctx, st.WorkspaceID)
			if cErr != nil {
				return cErr
			}
			if count >= maxChannelsPerWorkspace {
				return errChannelQuota
			}
			created, err := q.CreateChannel(ctx, sqlc.CreateChannelParams{
				WorkspaceID:       st.WorkspaceID,
				Platform:          st.Platform,
				PlatformAccountID: account.ID,
				Handle:            account.Handle,
				DisplayName:       account.DisplayName,
				ConnectedBy:       &connectedBy,
			})
			if err != nil {
				return err
			}
			ch = created
		default:
			return getErr
		}

		return q.UpsertChannelCredential(ctx, sqlc.UpsertChannelCredentialParams{
			ChannelID:             ch.ID,
			EncryptedAccessToken:  sealed.accessCipher,
			EncryptedRefreshToken: sealed.refreshCipher,
			Scopes:                scopes,
			ExpiresAt:             expiryTS(expiresAt),
			KeyVersion:            sealed.keyVersion,
		})
	})
	if err != nil {
		if errors.Is(err, errChannelQuota) {
			return sqlc.Channel{}, apperr.Validation("channel_quota_exceeded",
				"this workspace has reached its connected-channel limit")
		}
		return sqlc.Channel{}, apperr.Internal(err)
	}
	return ch, nil
}

// ListChannels returns the workspace's channels without any credential data.
func (s *Service) ListChannels(ctx context.Context, workspaceID uuid.UUID) ([]View, error) {
	rows, err := s.pool.Queries().ListChannels(ctx, workspaceID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]View, len(rows))
	for i, r := range rows {
		out[i] = toView(r)
	}
	return out, nil
}

// Disconnect best-effort revokes the token at the provider, purges the stored
// credential, deletes the channel, and audits the action. The channel must
// belong to the given workspace (isolation).
func (s *Service) Disconnect(ctx context.Context, actorID, workspaceID, channelID uuid.UUID) error {
	ch, err := s.pool.Queries().GetChannel(ctx, channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("channel_not_found", "channel not found")
		}
		return apperr.Internal(err)
	}
	if ch.WorkspaceID != workspaceID {
		return apperr.NotFound("channel_not_found", "channel not found")
	}

	s.bestEffortRevoke(ctx, ch)

	err = s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.DeleteChannelCredential(ctx, channelID); err != nil {
			return err
		}
		return q.DeleteChannel(ctx, channelID)
	})
	if err != nil {
		return apperr.Internal(err)
	}
	s.recordAudit(ctx, workspaceID, actorID, "channel.disconnect", channelID.String(),
		map[string]any{"platform": ch.Platform})
	return nil
}

// bestEffortRevoke attempts to revoke the access token at the provider. Failures
// are non-fatal: the local credential is purged regardless.
func (s *Service) bestEffortRevoke(ctx context.Context, ch sqlc.Channel) {
	provider, ok := s.registry.Get(ch.Platform)
	if !ok {
		return
	}
	cred, err := s.pool.Queries().GetChannelCredential(ctx, ch.ID)
	if err != nil {
		return
	}
	access, err := s.vault.openAccess(cred)
	if err != nil {
		return
	}
	_ = provider.Revoke(ctx, access)
}

// requireManageChannels enforces the manage_channels capability for callbacks,
// which are not workspace-path-scoped and so bypass RequireCapability middleware.
func (s *Service) requireManageChannels(ctx context.Context, workspaceID, userID uuid.UUID) error {
	member, err := s.authz.Membership(ctx, workspaceID, userID)
	if err != nil || !member.Has(workspace.CapManageChannels) {
		return apperr.Forbidden("forbidden", "you lack the required capability: manage_channels")
	}
	return nil
}

// recordAudit best-effort writes an audit entry.
func (s *Service) recordAudit(ctx context.Context, workspaceID, actorID uuid.UUID, action, target string, meta map[string]any) {
	if s.audit == nil {
		return
	}
	ws := workspaceID
	_ = s.audit.Record(ctx, security.Event{
		WorkspaceID: &ws, ActorUserID: &actorID, Action: action, Target: target, Metadata: meta,
	})
}

// expiryTS converts a token expiry to a nullable timestamp (zero -> NULL).
func expiryTS(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// toView maps a stored channel to its client-safe view.
func toView(c sqlc.Channel) View {
	return View{
		ID:                c.ID,
		Platform:          c.Platform,
		PlatformAccountID: c.PlatformAccountID,
		Handle:            c.Handle,
		DisplayName:       c.DisplayName,
		Status:            c.Status,
		ConnectedBy:       c.ConnectedBy,
		CreatedAt:         c.CreatedAt.Time,
	}
}
