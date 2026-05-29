package auth

import (
	"net/mail"
	"strings"

	"github.com/Akins20/postal/internal/platform/apperr"
)

// Credential policy bounds. The password floor balances usability with
// resistance to guessing; the ceiling bounds Argon2id work per attempt.
const (
	minPasswordLen = 10
	maxPasswordLen = 128
	maxEmailLen    = 254
)

// disposableDomains is a baseline blocklist of throwaway email providers, an
// anti-abuse measure for a free signup. Not exhaustive; extend or externalize
// as needed.
var disposableDomains = map[string]struct{}{
	"mailinator.com":    {},
	"guerrillamail.com": {},
	"10minutemail.com":  {},
	"tempmail.com":      {},
	"temp-mail.org":     {},
	"yopmail.com":       {},
	"throwawaymail.com": {},
	"trashmail.com":     {},
	"getnada.com":       {},
	"sharklasers.com":   {},
}

// normalizeEmail lowercases and trims an address for consistent storage and
// case-insensitive uniqueness.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validateEmail checks that email is a syntactically valid, reasonably sized
// address. It expects an already-normalized value.
func validateEmail(email string) error {
	if email == "" {
		return apperr.Validation("invalid_email", "email is required").WithField("email", "is required")
	}
	if len(email) > maxEmailLen {
		return apperr.Validation("invalid_email", "email is too long").WithField("email", "is too long")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return apperr.Validation("invalid_email", "email is not valid").WithField("email", "is not a valid address")
	}
	return nil
}

// validatePassword enforces the password length policy.
func validatePassword(password string) error {
	switch {
	case len(password) < minPasswordLen:
		return apperr.Validation("weak_password", "password is too short").
			WithField("password", "must be at least 10 characters")
	case len(password) > maxPasswordLen:
		return apperr.Validation("weak_password", "password is too long").
			WithField("password", "must be at most 128 characters")
	default:
		return nil
	}
}

// isDisposableEmail reports whether the address uses a known throwaway domain.
func isDisposableEmail(email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}
	_, blocked := disposableDomains[email[at+1:]]
	return blocked
}
