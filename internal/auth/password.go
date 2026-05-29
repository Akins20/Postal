// Package auth implements identity: signup, email verification, login/refresh
// with JWT access tokens and sliding rotating refresh tokens, password reset,
// and the RequireUser middleware. Passwords are hashed with Argon2id; refresh
// tokens live in Redis (see session.go).
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. Tuned for interactive logins: ~64 MiB, 3 passes. Encoded
// into each hash so they can be raised later without breaking existing hashes.
const (
	argonMemoryKiB  = 64 * 1024 // 64 MiB
	argonIterations = 3
	argonThreads    = 2
	argonSaltLen    = 16
	argonKeyLen     = 32
)

// ErrInvalidHash indicates a stored hash that is not a valid Argon2id PHC string.
var ErrInvalidHash = errors.New("auth: invalid password hash format")

// HashPassword derives an Argon2id hash of plain and returns it in PHC string
// format: $argon2id$v=19$m=<mem>,t=<iter>,p=<threads>$<b64salt>$<b64hash>.
func HashPassword(plain string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}
	hash := argon2.IDKey([]byte(plain), salt, argonIterations, argonMemoryKiB, argonThreads, argonKeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonIterations, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword reports whether plain matches the encoded Argon2id hash. The
// comparison is constant-time. A malformed hash returns ErrInvalidHash.
func VerifyPassword(plain, encoded string) (bool, error) {
	params, salt, want, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	// #nosec G115 -- len(want) is the stored digest length (32 bytes), far below uint32 max.
	keyLen := uint32(len(want))
	got := argon2.IDKey([]byte(plain), salt, params.iterations, params.memory, params.threads, keyLen)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// argonParams holds the cost parameters parsed from a PHC hash string.
type argonParams struct {
	memory     uint32
	iterations uint32
	threads    uint8
}

// decodeHash parses a PHC Argon2id string into its parameters, salt, and digest.
func decodeHash(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, nil, nil, ErrInvalidHash
	}

	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.threads); err != nil {
		return argonParams{}, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, ErrInvalidHash
	}
	digest, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, ErrInvalidHash
	}
	return p, salt, digest, nil
}
