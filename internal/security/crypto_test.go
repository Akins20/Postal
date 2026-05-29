package security

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"
)

// newKey returns a random 32-byte key for tests.
func newKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		t.Fatalf("generating key: %v", err)
	}
	return k
}

func TestSealOpen_RoundTrip(t *testing.T) {
	enc, err := NewEncryptor(map[uint32][]byte{1: newKey(t)}, 1)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintexts := [][]byte{
		[]byte("an-oauth-access-token"),
		[]byte(""),
		bytes.Repeat([]byte("x"), 4096),
	}
	for _, pt := range plaintexts {
		blob, err := enc.Seal(pt)
		if err != nil {
			t.Fatalf("Seal: %v", err)
		}
		if bytes.Contains(blob, pt) && len(pt) > 0 {
			t.Error("ciphertext contains plaintext")
		}
		got, err := enc.Open(blob)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if !bytes.Equal(got, pt) {
			t.Errorf("round trip mismatch: got %q want %q", got, pt)
		}
	}
}

func TestSeal_ProducesDistinctCiphertexts(t *testing.T) {
	enc, _ := NewEncryptor(map[uint32][]byte{1: newKey(t)}, 1)
	pt := []byte("same-secret")

	a, _ := enc.Seal(pt)
	b, _ := enc.Seal(pt)
	if bytes.Equal(a, b) {
		t.Error("sealing identical plaintext twice produced identical ciphertext (nonce reuse?)")
	}
}

func TestOpen_TamperDetected(t *testing.T) {
	enc, _ := NewEncryptor(map[uint32][]byte{1: newKey(t)}, 1)
	blob, _ := enc.Seal([]byte("trusted-token"))

	// Flip a bit in the final byte (within the GCM-protected payload).
	tampered := bytes.Clone(blob)
	tampered[len(tampered)-1] ^= 0x01

	if _, err := enc.Open(tampered); err == nil {
		t.Fatal("Open accepted tampered ciphertext")
	}
}

func TestOpen_WrongKeyFails(t *testing.T) {
	encA, _ := NewEncryptor(map[uint32][]byte{1: newKey(t)}, 1)
	encB, _ := NewEncryptor(map[uint32][]byte{1: newKey(t)}, 1)

	blob, _ := encA.Seal([]byte("secret"))
	if _, err := encB.Open(blob); err == nil {
		t.Fatal("Open succeeded with the wrong master key")
	}
}

func TestKeyRotation(t *testing.T) {
	k1, k2 := newKey(t), newKey(t)

	// Old encryptor only knows v1; seal under v1.
	old, _ := NewEncryptor(map[uint32][]byte{1: k1}, 1)
	oldBlob, _ := old.Seal([]byte("legacy-token"))

	// Rotated encryptor: current is v2, but v1 retained for decryption.
	rotated, err := NewEncryptor(map[uint32][]byte{1: k1, 2: k2}, 2)
	if err != nil {
		t.Fatalf("NewEncryptor (rotated): %v", err)
	}
	if rotated.CurrentVersion() != 2 {
		t.Errorf("CurrentVersion = %d, want 2", rotated.CurrentVersion())
	}

	// New seals use v2.
	newBlob, _ := rotated.Seal([]byte("fresh-token"))
	if newBlob[3] != 2 { // last byte of the 4-byte big-endian version header
		t.Errorf("new ciphertext key version = %d, want 2", newBlob[3])
	}

	// Rotated encryptor opens BOTH old (v1) and new (v2) ciphertext.
	if got, err := rotated.Open(oldBlob); err != nil || string(got) != "legacy-token" {
		t.Errorf("opening legacy v1 blob after rotation: got %q err %v", got, err)
	}
	if got, err := rotated.Open(newBlob); err != nil || string(got) != "fresh-token" {
		t.Errorf("opening v2 blob: got %q err %v", got, err)
	}
}

func TestOpen_UnknownKeyVersion(t *testing.T) {
	enc, _ := NewEncryptor(map[uint32][]byte{2: newKey(t)}, 2)
	blob, _ := enc.Seal([]byte("data"))

	// An encryptor without version 2 cannot open it.
	other, _ := NewEncryptor(map[uint32][]byte{1: newKey(t)}, 1)
	_, err := other.Open(blob)
	if !errors.Is(err, ErrUnknownKeyVersion) {
		t.Errorf("err = %v, want ErrUnknownKeyVersion", err)
	}
}

func TestOpen_MalformedCiphertext(t *testing.T) {
	enc, _ := NewEncryptor(map[uint32][]byte{1: newKey(t)}, 1)
	for _, blob := range [][]byte{nil, {0x00}, []byte("too-short")} {
		if _, err := enc.Open(blob); err == nil {
			t.Errorf("Open(%v) = nil error, want malformed", blob)
		}
	}
}

func TestNewEncryptor_Validation(t *testing.T) {
	if _, err := NewEncryptor(nil, 1); !errors.Is(err, ErrNoKeys) {
		t.Errorf("empty keyring err = %v, want ErrNoKeys", err)
	}
	if _, err := NewEncryptor(map[uint32][]byte{1: make([]byte, 16)}, 1); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("short key err = %v, want ErrInvalidKey", err)
	}
	if _, err := NewEncryptor(map[uint32][]byte{1: make([]byte, keySize)}, 2); err == nil {
		t.Error("current version absent from keyring should error")
	}
}
