package channel

import (
	"fmt"

	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/security"
)

// sealedCredential is the encrypted form of a token set, ready for storage.
type sealedCredential struct {
	accessCipher  []byte
	refreshCipher []byte // nil when the token set has no refresh token
	keyVersion    int32
}

// Vault seals and opens channel OAuth tokens using envelope encryption. Tokens
// are only ever held in memory transiently; at rest they are AES-256-GCM
// ciphertext keyed by the current master key version.
type Vault struct {
	enc *security.Encryptor
}

// NewVault builds a Vault over the given encryptor.
func NewVault(enc *security.Encryptor) *Vault {
	return &Vault{enc: enc}
}

// seal encrypts a token's access and (optional) refresh secrets.
func (v *Vault) seal(t *Token) (sealedCredential, error) {
	accessCipher, err := v.enc.Seal([]byte(t.AccessToken))
	if err != nil {
		return sealedCredential{}, fmt.Errorf("sealing access token: %w", err)
	}
	var refreshCipher []byte
	if t.RefreshToken != "" {
		refreshCipher, err = v.enc.Seal([]byte(t.RefreshToken))
		if err != nil {
			return sealedCredential{}, fmt.Errorf("sealing refresh token: %w", err)
		}
	}
	// #nosec G115 -- key version is a small monotonic counter, far below int32 max.
	return sealedCredential{accessCipher: accessCipher, refreshCipher: refreshCipher, keyVersion: int32(v.enc.CurrentVersion())}, nil
}

// openAccess decrypts the access token from a stored credential.
func (v *Vault) openAccess(cred sqlc.ChannelCredential) (string, error) {
	plain, err := v.enc.Open(cred.EncryptedAccessToken)
	if err != nil {
		return "", fmt.Errorf("opening access token: %w", err)
	}
	return string(plain), nil
}

// openRefresh decrypts the refresh token, reporting false when none is stored.
func (v *Vault) openRefresh(cred sqlc.ChannelCredential) (string, bool, error) {
	if len(cred.EncryptedRefreshToken) == 0 {
		return "", false, nil
	}
	plain, err := v.enc.Open(cred.EncryptedRefreshToken)
	if err != nil {
		return "", false, fmt.Errorf("opening refresh token: %w", err)
	}
	return string(plain), true, nil
}
