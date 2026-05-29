package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// tokenIssuer is the JWT "iss" claim value for Postal-minted access tokens.
const tokenIssuer = "postal"

// ErrInvalidToken indicates an access token that is malformed, expired, or
// signed with the wrong key.
var ErrInvalidToken = errors.New("auth: invalid access token")

// TokenIssuer mints and verifies short-lived HS256 JWT access tokens. The
// signing secret and TTL come from config; the clock is injectable for tests.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	clock  func() time.Time
}

// NewTokenIssuer builds a TokenIssuer. secret must be non-empty; ttl must be
// positive. clock defaults to time.Now when nil.
func NewTokenIssuer(secret string, ttl time.Duration, clock func() time.Time) (*TokenIssuer, error) {
	if secret == "" {
		return nil, errors.New("auth: JWT secret must not be empty")
	}
	if ttl <= 0 {
		return nil, errors.New("auth: access token TTL must be positive")
	}
	if clock == nil {
		clock = time.Now
	}
	return &TokenIssuer{secret: []byte(secret), ttl: ttl, clock: clock}, nil
}

// Issue mints a signed access token whose subject is the user ID.
func (t *TokenIssuer) Issue(userID uuid.UUID) (string, error) {
	now := t.clock()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return signed, nil
}

// Verify validates a signed access token and returns its subject (user ID).
// It enforces the HS256 algorithm, expiry, and issuer.
func (t *TokenIssuer) Verify(tokenStr string) (uuid.UUID, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
			}
			return t.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithTimeFunc(t.clock),
	)
	if err != nil || !parsed.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return userID, nil
}
