package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// opaqueTokenBytes is the entropy of opaque tokens (refresh, email verification,
// password reset) before encoding.
const opaqueTokenBytes = 32

// newOpaqueToken returns a URL-safe random token with opaqueTokenBytes of entropy.
func newOpaqueToken() (string, error) {
	buf := make([]byte, opaqueTokenBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashToken returns the hex SHA-256 of a token. Only hashes are persisted, so a
// database or Redis dump cannot reveal usable tokens.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
