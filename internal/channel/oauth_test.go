package channel

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

func TestNewPKCE(t *testing.T) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		t.Fatalf("newPKCE: %v", err)
	}
	if verifier == "" || challenge == "" {
		t.Fatal("empty verifier or challenge")
	}
	// challenge must be the S256 (base64url, unpadded) of the verifier.
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Errorf("challenge = %q, want S256 %q", challenge, want)
	}

	// Two calls must produce different verifiers.
	v2, _, _ := newPKCE()
	if v2 == verifier {
		t.Error("PKCE verifier is not random across calls")
	}
}

// stubProvider is a minimal OAuthProvider for registry tests.
type stubProvider struct{ name string }

func (s stubProvider) Platform() string                          { return s.name }
func (s stubProvider) AuthURL(state, challenge, _ string) string { return "url?" + state + challenge }
func (s stubProvider) ExchangeCode(context.Context, string, string, string) (*Token, error) {
	return &Token{AccessToken: "a", ExpiresAt: time.Now()}, nil
}
func (s stubProvider) RefreshToken(context.Context, string) (*Token, error) {
	return &Token{AccessToken: "a"}, nil
}
func (s stubProvider) Account(context.Context, string) (*Account, error) {
	return &Account{ID: "id"}, nil
}
func (s stubProvider) Revoke(context.Context, string) error { return nil }

func TestRegistry(t *testing.T) {
	reg := NewRegistry(stubProvider{name: "twitter"})

	if _, ok := reg.Get("twitter"); !ok {
		t.Error("expected twitter provider to be registered")
	}
	if _, ok := reg.Get("facebook"); ok {
		t.Error("unregistered provider should not be found")
	}
}
