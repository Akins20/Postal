package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenIssuer_IssueVerify(t *testing.T) {
	iss, err := NewTokenIssuer("super-secret", 15*time.Minute, nil)
	if err != nil {
		t.Fatalf("NewTokenIssuer: %v", err)
	}
	userID := uuid.New()

	token, err := iss.Issue(userID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	got, err := iss.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != userID {
		t.Errorf("subject = %v, want %v", got, userID)
	}
}

func TestTokenIssuer_Expired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	iss, _ := NewTokenIssuer("secret", time.Minute, func() time.Time { return now })
	token, _ := iss.Issue(uuid.New())

	// Advance past expiry.
	iss.clock = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := iss.Verify(token); err != ErrInvalidToken {
		t.Errorf("expired token err = %v, want ErrInvalidToken", err)
	}
}

func TestTokenIssuer_WrongSecret(t *testing.T) {
	a, _ := NewTokenIssuer("secret-a", time.Minute, nil)
	b, _ := NewTokenIssuer("secret-b", time.Minute, nil)
	token, _ := a.Issue(uuid.New())

	if _, err := b.Verify(token); err != ErrInvalidToken {
		t.Errorf("cross-secret verify err = %v, want ErrInvalidToken", err)
	}
}

func TestTokenIssuer_RejectsGarbage(t *testing.T) {
	iss, _ := NewTokenIssuer("secret", time.Minute, nil)
	for _, bad := range []string{"", "abc", "a.b.c"} {
		if _, err := iss.Verify(bad); err != ErrInvalidToken {
			t.Errorf("Verify(%q) err = %v, want ErrInvalidToken", bad, err)
		}
	}
}

func TestNewTokenIssuer_Validation(t *testing.T) {
	if _, err := NewTokenIssuer("", time.Minute, nil); err == nil {
		t.Error("empty secret should error")
	}
	if _, err := NewTokenIssuer("s", 0, nil); err == nil {
		t.Error("zero ttl should error")
	}
}
