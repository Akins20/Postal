// Package security provides Postal's cryptographic primitives, token vault, and
// audit log. Its centerpiece is envelope encryption used to protect social
// OAuth tokens at rest: each secret is sealed with a fresh data key (DEK), and
// that DEK is wrapped by a versioned master key (KEK) loaded from config/KMS.
//
// Security invariants:
//   - Plaintext secrets and keys are never logged.
//   - Every seal uses a fresh random DEK and fresh random nonces (AES-256-GCM).
//   - Ciphertext is self-describing (carries its key version) so keys can be
//     rotated without a flag-day re-encryption.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// keySize is the AES-256 key length in bytes. Both KEKs and DEKs are 32 bytes.
const keySize = 32

// nonceSize is the AES-GCM standard nonce length in bytes.
const nonceSize = 12

// maxWrappedDEKLen bounds the wrapped data key so its length fits the uint16
// blob header. The actual wrapped DEK is nonce(12) + 32-byte key + 16-byte tag.
const maxWrappedDEKLen = 1<<16 - 1

// Sentinel errors for inspection via errors.Is.
var (
	// ErrInvalidKey indicates a master key of the wrong length.
	ErrInvalidKey = errors.New("security: master key must be 32 bytes")
	// ErrNoKeys indicates an Encryptor was constructed without any keys.
	ErrNoKeys = errors.New("security: keyring must contain at least one key")
	// ErrUnknownKeyVersion indicates ciphertext sealed under a key not in the ring.
	ErrUnknownKeyVersion = errors.New("security: unknown key version")
	// ErrMalformedCiphertext indicates a corrupt or truncated ciphertext blob.
	ErrMalformedCiphertext = errors.New("security: malformed ciphertext")
)

// Encryptor seals and opens secrets using envelope encryption. It holds a
// versioned keyring of master keys (KEKs); the current version is used for new
// seals, while older versions remain available to open previously sealed data
// during key rotation. It is safe for concurrent use (read-only after New).
type Encryptor struct {
	keks    map[uint32][]byte
	current uint32
}

// NewEncryptor builds an Encryptor from a versioned keyring. current must exist
// in keks, and every key must be 32 bytes. Keys are copied so callers may reuse
// their buffers.
func NewEncryptor(keks map[uint32][]byte, current uint32) (*Encryptor, error) {
	if len(keks) == 0 {
		return nil, ErrNoKeys
	}
	if _, ok := keks[current]; !ok {
		return nil, fmt.Errorf("security: current version %d not in keyring", current)
	}
	owned := make(map[uint32][]byte, len(keks))
	for v, k := range keks {
		if len(k) != keySize {
			return nil, fmt.Errorf("security: key version %d: %w", v, ErrInvalidKey)
		}
		buf := make([]byte, keySize)
		copy(buf, k)
		owned[v] = buf
	}
	return &Encryptor{keks: owned, current: current}, nil
}

// CurrentVersion returns the key version new seals are produced under. Callers
// persist this alongside the ciphertext (e.g. channel_credential.key_version)
// for operational queries such as "which rows still use an old key".
func (e *Encryptor) CurrentVersion() uint32 { return e.current }

// Seal encrypts plaintext and returns a self-describing ciphertext blob.
//
// Blob layout (all integers big-endian):
//
//	keyVersion (4) | wrappedDEKLen (2) | wrappedDEK | ciphertext
//
// where wrappedDEK = nonce(12) || GCM(KEK, DEK) and
// ciphertext = nonce(12) || GCM(DEK, plaintext).
func (e *Encryptor) Seal(plaintext []byte) ([]byte, error) {
	dek := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("generating data key: %w", err)
	}

	wrappedDEK, err := aesGCMSeal(e.keks[e.current], dek)
	if err != nil {
		return nil, fmt.Errorf("wrapping data key: %w", err)
	}
	data, err := aesGCMSeal(dek, plaintext)
	if err != nil {
		return nil, fmt.Errorf("sealing plaintext: %w", err)
	}

	// The wrapped DEK is a fixed ~60 bytes; the bound guards the uint16 length
	// prefix against overflow regardless.
	if len(wrappedDEK) > maxWrappedDEKLen {
		return nil, fmt.Errorf("security: wrapped data key too large: %d bytes", len(wrappedDEK))
	}

	out := make([]byte, 0, 4+2+len(wrappedDEK)+len(data))
	out = binary.BigEndian.AppendUint32(out, e.current)
	// #nosec G115 -- len(wrappedDEK) is bounded to <= maxWrappedDEKLen (65535) by the guard above, so it fits uint16.
	out = binary.BigEndian.AppendUint16(out, uint16(len(wrappedDEK)))
	out = append(out, wrappedDEK...)
	out = append(out, data...)
	return out, nil
}

// Open decrypts a blob produced by Seal. It selects the KEK by the version
// embedded in the blob, so rotation needs no caller changes. Tampering with any
// byte fails GCM authentication and returns an error.
func (e *Encryptor) Open(blob []byte) ([]byte, error) {
	const headerLen = 4 + 2
	if len(blob) < headerLen {
		return nil, ErrMalformedCiphertext
	}
	version := binary.BigEndian.Uint32(blob[0:4])
	wrappedLen := int(binary.BigEndian.Uint16(blob[4:6]))

	if len(blob) < headerLen+wrappedLen {
		return nil, ErrMalformedCiphertext
	}
	wrappedDEK := blob[headerLen : headerLen+wrappedLen]
	data := blob[headerLen+wrappedLen:]

	kek, ok := e.keks[version]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUnknownKeyVersion, version)
	}

	dek, err := aesGCMOpen(kek, wrappedDEK)
	if err != nil {
		return nil, fmt.Errorf("unwrapping data key: %w", err)
	}
	plaintext, err := aesGCMOpen(dek, data)
	if err != nil {
		return nil, fmt.Errorf("opening ciphertext: %w", err)
	}
	return plaintext, nil
}

// aesGCMSeal encrypts plaintext with key using AES-256-GCM, prefixing the
// returned slice with the random nonce: nonce(12) || ciphertext+tag.
func aesGCMSeal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// aesGCMOpen reverses aesGCMSeal, expecting nonce(12) || ciphertext+tag.
func aesGCMOpen(key, blob []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(blob) < nonceSize {
		return nil, ErrMalformedCiphertext
	}
	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm open: %w", err)
	}
	return plaintext, nil
}

// newGCM builds an AES-256-GCM AEAD from a 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != keySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating gcm: %w", err)
	}
	return gcm, nil
}
