package security

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrEmptyKeySpec indicates no master key was configured.
var ErrEmptyKeySpec = errors.New("security: master key spec is empty")

// NewEncryptorFromSpec builds an Encryptor from a configured master-key spec.
//
// The spec supports key rotation without code changes. It is a comma-separated
// list of "version:base64key" entries, e.g. "1:BASE64A,2:BASE64B". The highest
// version present becomes the current (seal) key; lower versions remain
// available to open previously sealed data. A bare base64 key with no version
// prefix is treated as version 1.
//
// Each decoded key must be exactly 32 bytes (AES-256). Keys are never logged.
func NewEncryptorFromSpec(spec string) (*Encryptor, error) {
	keys, current, err := ParseKeyring(spec)
	if err != nil {
		return nil, err
	}
	return NewEncryptor(keys, current)
}

// ParseKeyring decodes a master-key spec into a versioned keyring and reports
// the current (highest) version. See NewEncryptorFromSpec for the format.
func ParseKeyring(spec string) (keys map[uint32][]byte, current uint32, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, 0, ErrEmptyKeySpec
	}

	keys = make(map[uint32][]byte)
	for _, entry := range strings.Split(spec, ",") {
		version, b64, perr := splitKeyEntry(strings.TrimSpace(entry))
		if perr != nil {
			return nil, 0, perr
		}
		raw, derr := base64.StdEncoding.DecodeString(b64)
		if derr != nil {
			return nil, 0, fmt.Errorf("security: decoding key version %d: %w", version, derr)
		}
		if len(raw) != keySize {
			return nil, 0, fmt.Errorf("security: key version %d is %d bytes: %w", version, len(raw), ErrInvalidKey)
		}
		if _, dup := keys[version]; dup {
			return nil, 0, fmt.Errorf("security: duplicate key version %d", version)
		}
		keys[version] = raw
		if version > current {
			current = version
		}
	}
	return keys, current, nil
}

// splitKeyEntry parses one "version:base64" entry. A bare base64 value (no
// colon) is assigned version 1.
func splitKeyEntry(entry string) (version uint32, b64 string, err error) {
	prefix, rest, hasColon := strings.Cut(entry, ":")
	if !hasColon {
		return 1, entry, nil
	}
	n, perr := strconv.ParseUint(prefix, 10, 32)
	if perr != nil {
		return 0, "", fmt.Errorf("security: invalid key version %q: %w", prefix, perr)
	}
	if n == 0 {
		return 0, "", fmt.Errorf("security: key version must be >= 1")
	}
	return uint32(n), rest, nil
}
