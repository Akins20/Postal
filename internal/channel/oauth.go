// Package channel manages connected social accounts and their OAuth credentials.
// Credentials are stored only as envelope-encrypted ciphertext (see vault.go)
// and are never returned to clients or logged. The OAuth flow is generic over a
// per-platform OAuthProvider; the X/Twitter provider is wired in Phase 4.
package channel

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"time"
)

// pkceVerifierBytes is the entropy of a PKCE code verifier before encoding.
const pkceVerifierBytes = 32

// Token is an OAuth token set returned by a provider. ExpiresAt is the zero time
// when the provider does not supply an expiry.
type Token struct {
	AccessToken  string
	RefreshToken string
	Scopes       []string
	ExpiresAt    time.Time
}

// Account identifies the connected account on the platform.
type Account struct {
	ID          string // platform_account_id (stable platform identifier)
	Handle      string // e.g. @username
	DisplayName string
}

// OAuthProvider is implemented per platform. It covers only the OAuth surface
// needed in Phase 3; the full publishing adapter (Phase 4) embeds this.
type OAuthProvider interface {
	// Platform returns the provider's platform key (e.g. "twitter").
	Platform() string
	// AuthURL builds the provider authorize URL with PKCE S256 and CSRF state.
	// redirectURI overrides the adapter's configured callback (empty = default);
	// it lets web and native clients use distinct, allowlisted redirects.
	AuthURL(state, codeChallenge, redirectURI string) string
	// ExchangeCode swaps an authorization code (+ PKCE verifier) for tokens.
	// redirectURI must match the one used in AuthURL (empty = adapter default).
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*Token, error)
	// RefreshToken obtains a fresh token set from a refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
	// Account resolves the connected account's identity from an access token.
	Account(ctx context.Context, accessToken string) (*Account, error)
	// Revoke best-effort revokes a token at the provider; may be a no-op.
	Revoke(ctx context.Context, token string) error
}

// ManualConnector is implemented by providers that connect via user-supplied
// credentials (e.g. a Telegram bot token + chat id) rather than an OAuth
// redirect. ConnectManual validates the credentials and returns the channel
// token plus the resolved account identity.
type ManualConnector interface {
	ConnectManual(ctx context.Context, creds map[string]string) (*Token, *Account, error)
}

// Registry resolves OAuthProviders by platform key.
type Registry struct {
	providers map[string]OAuthProvider
}

// NewRegistry builds a registry from the given providers, keyed by Platform().
func NewRegistry(providers ...OAuthProvider) *Registry {
	m := make(map[string]OAuthProvider, len(providers))
	for _, p := range providers {
		m[p.Platform()] = p
	}
	return &Registry{providers: m}
}

// Get returns the provider for a platform, reporting whether it is registered.
func (r *Registry) Get(platform string) (OAuthProvider, bool) {
	p, ok := r.providers[platform]
	return p, ok
}

// newPKCE generates a PKCE code verifier and its S256 challenge.
func newPKCE() (verifier, challenge string, err error) {
	buf := make([]byte, pkceVerifierBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", "", fmt.Errorf("generating pkce verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// newStateToken returns an unguessable CSRF state value for the OAuth flow.
func newStateToken() (string, error) {
	buf := make([]byte, pkceVerifierBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
