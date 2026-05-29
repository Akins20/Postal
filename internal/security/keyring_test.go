package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"testing"
)

// b64Key returns a base64-encoded random 32-byte key.
func b64Key(t *testing.T) string {
	t.Helper()
	k := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return base64.StdEncoding.EncodeToString(k)
}

func TestParseKeyring_BareKeyIsVersion1(t *testing.T) {
	keys, current, err := ParseKeyring(b64Key(t))
	if err != nil {
		t.Fatalf("ParseKeyring: %v", err)
	}
	if current != 1 {
		t.Errorf("current = %d, want 1", current)
	}
	if len(keys) != 1 || len(keys[1]) != keySize {
		t.Errorf("unexpected keyring: %v", keys)
	}
}

func TestParseKeyring_VersionedHighestIsCurrent(t *testing.T) {
	spec := "1:" + b64Key(t) + ",3:" + b64Key(t) + ",2:" + b64Key(t)
	keys, current, err := ParseKeyring(spec)
	if err != nil {
		t.Fatalf("ParseKeyring: %v", err)
	}
	if current != 3 {
		t.Errorf("current = %d, want 3 (highest version)", current)
	}
	if len(keys) != 3 {
		t.Errorf("len(keys) = %d, want 3", len(keys))
	}
}

func TestParseKeyring_Errors(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{name: "empty", spec: ""},
		{name: "bad base64", spec: "1:not-base64!!!"},
		{name: "wrong length", spec: "1:" + base64.StdEncoding.EncodeToString([]byte("short"))},
		{name: "bad version", spec: "x:" + base64.StdEncoding.EncodeToString(make([]byte, keySize))},
		{name: "zero version", spec: "0:" + base64.StdEncoding.EncodeToString(make([]byte, keySize))},
		{name: "duplicate version", spec: "1:" + base64.StdEncoding.EncodeToString(make([]byte, keySize)) + ",1:" + base64.StdEncoding.EncodeToString(make([]byte, keySize))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := ParseKeyring(tt.spec); err == nil {
				t.Errorf("ParseKeyring(%q) = nil error, want error", tt.spec)
			}
		})
	}
}

func TestNewEncryptorFromSpec_EndToEnd(t *testing.T) {
	enc, err := NewEncryptorFromSpec(b64Key(t))
	if err != nil {
		t.Fatalf("NewEncryptorFromSpec: %v", err)
	}
	blob, err := enc.Seal([]byte("token"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	got, err := enc.Open(blob)
	if err != nil || string(got) != "token" {
		t.Errorf("round trip: got %q err %v", got, err)
	}
}

func TestNewEncryptorFromSpec_EmptyIsError(t *testing.T) {
	if _, err := NewEncryptorFromSpec(""); !errors.Is(err, ErrEmptyKeySpec) {
		t.Errorf("err = %v, want ErrEmptyKeySpec", err)
	}
}
