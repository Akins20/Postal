package channel

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/security"
)

func testVault(t *testing.T) *Vault {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32))
	enc, err := security.NewEncryptorFromSpec(key)
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	return NewVault(enc)
}

func TestVault_SealOpenRoundTrip(t *testing.T) {
	v := testVault(t)
	tok := &Token{
		AccessToken:  "secret-access-token",
		RefreshToken: "secret-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	sealed, err := v.seal(tok)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if bytes.Contains(sealed.accessCipher, []byte(tok.AccessToken)) {
		t.Error("access ciphertext contains plaintext")
	}
	if bytes.Contains(sealed.refreshCipher, []byte(tok.RefreshToken)) {
		t.Error("refresh ciphertext contains plaintext")
	}

	cred := sqlc.ChannelCredential{
		EncryptedAccessToken:  sealed.accessCipher,
		EncryptedRefreshToken: sealed.refreshCipher,
	}
	gotAccess, err := v.openAccess(cred)
	if err != nil || gotAccess != tok.AccessToken {
		t.Errorf("openAccess = %q, %v", gotAccess, err)
	}
	gotRefresh, ok, err := v.openRefresh(cred)
	if err != nil || !ok || gotRefresh != tok.RefreshToken {
		t.Errorf("openRefresh = %q, ok=%v, %v", gotRefresh, ok, err)
	}
}

func TestVault_NoRefreshToken(t *testing.T) {
	v := testVault(t)
	sealed, err := v.seal(&Token{AccessToken: "only-access"})
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed.refreshCipher != nil {
		t.Error("expected nil refresh ciphertext when no refresh token")
	}
	_, ok, err := v.openRefresh(sqlc.ChannelCredential{EncryptedRefreshToken: sealed.refreshCipher})
	if err != nil || ok {
		t.Errorf("openRefresh with no token: ok=%v err=%v", ok, err)
	}
}

func TestVault_TamperDetected(t *testing.T) {
	v := testVault(t)
	sealed, _ := v.seal(&Token{AccessToken: "tamper-me"})
	sealed.accessCipher[len(sealed.accessCipher)-1] ^= 0x01

	if _, err := v.openAccess(sqlc.ChannelCredential{EncryptedAccessToken: sealed.accessCipher}); err == nil {
		t.Error("expected error opening tampered ciphertext")
	}
}
