package integration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/security"
)

// ErrNotConfigured is returned when an integration action needs a stored API
// key and none exists (or the integration is disabled).
var ErrNotConfigured = errors.New("integration is not configured")

// ErrBadKey is returned when the provider rejects a submitted API key.
var ErrBadKey = errors.New("the provider rejected that API key")

// Integration is the secret-free view of one workspace integration.
type Integration struct {
	Provider   string    `json:"provider"`
	Enabled    bool      `json:"enabled"`
	AutoApply  bool      `json:"auto_apply"`
	Configured bool      `json:"configured"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Service manages workspace integrations: secret-free listing, configuration
// (API keys sealed with the master key), and the OGShortener text action.
type Service struct {
	pool *db.Pool
	enc  *security.Encryptor
	og   *OGShortenerClient
}

// NewService builds the integration service. og may use a test base URL.
func NewService(pool *db.Pool, enc *security.Encryptor, og *OGShortenerClient) *Service {
	return &Service{pool: pool, enc: enc, og: og}
}

// List returns the workspace's integrations with secrets omitted. Providers
// never configured are still listed (as unconfigured) so the UI can offer them.
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID) ([]Integration, error) {
	rows, err := s.pool.Queries().ListWorkspaceIntegrations(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing integrations: %w", err)
	}
	byProvider := map[string]Integration{}
	for _, r := range rows {
		byProvider[r.Provider] = Integration{
			Provider: r.Provider, Enabled: r.Enabled, AutoApply: r.AutoApply,
			Configured: len(r.Credentials) > 0, UpdatedAt: r.UpdatedAt.Time,
		}
	}
	out := make([]Integration, 0, 1)
	for _, provider := range []string{ProviderOGShortener} {
		if it, ok := byProvider[provider]; ok {
			out = append(out, it)
		} else {
			out = append(out, Integration{Provider: provider})
		}
	}
	return out, nil
}

// Configure updates an integration. A non-nil apiKey is verified against the
// provider, then sealed; nil keeps any stored key. Enabling without a stored
// or submitted key fails with ErrNotConfigured.
func (s *Service) Configure(ctx context.Context, workspaceID uuid.UUID, provider string, enabled, autoApply bool, apiKey *string) (*Integration, error) {
	if provider != ProviderOGShortener {
		return nil, fmt.Errorf("unknown integration provider %q", provider)
	}
	var sealed []byte
	if apiKey != nil && *apiKey != "" {
		if err := s.og.Verify(ctx, *apiKey); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrBadKey, err)
		}
		var err error
		sealed, err = s.enc.Seal([]byte(*apiKey))
		if err != nil {
			return nil, fmt.Errorf("sealing API key: %w", err)
		}
	}
	if enabled && sealed == nil {
		existing, err := s.pool.Queries().GetWorkspaceIntegration(ctx, sqlc.GetWorkspaceIntegrationParams{
			WorkspaceID: workspaceID, Provider: provider,
		})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && len(existing.Credentials) == 0) {
			return nil, fmt.Errorf("enable needs an API key first: %w", ErrNotConfigured)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("loading integration: %w", err)
		}
	}
	row, err := s.pool.Queries().UpsertWorkspaceIntegration(ctx, sqlc.UpsertWorkspaceIntegrationParams{
		WorkspaceID: workspaceID, Provider: provider, Enabled: enabled, AutoApply: autoApply, Credentials: sealed,
	})
	if err != nil {
		return nil, fmt.Errorf("saving integration: %w", err)
	}
	return &Integration{
		Provider: row.Provider, Enabled: row.Enabled, AutoApply: row.AutoApply,
		Configured: len(row.Credentials) > 0, UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// ShortenText rewrites every link in text through the workspace's OGShortener
// account. Requires the integration enabled with a stored key.
func (s *Service) ShortenText(ctx context.Context, workspaceID uuid.UUID, text string) (string, error) {
	row, err := s.pool.Queries().GetWorkspaceIntegration(ctx, sqlc.GetWorkspaceIntegrationParams{
		WorkspaceID: workspaceID, Provider: ProviderOGShortener,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotConfigured
	}
	if err != nil {
		return "", fmt.Errorf("loading integration: %w", err)
	}
	if !row.Enabled || len(row.Credentials) == 0 {
		return "", ErrNotConfigured
	}
	key, err := s.enc.Open(row.Credentials)
	if err != nil {
		return "", fmt.Errorf("opening stored API key: %w", err)
	}
	return s.og.ShortenText(ctx, string(key), text)
}
