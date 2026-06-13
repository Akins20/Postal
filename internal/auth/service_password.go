package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// VerifyEmail consumes a single-use verification token and marks the user's
// email verified. Unknown, expired, or already-used tokens are rejected.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	q := s.pool.Queries()
	row, err := q.GetEmailVerificationToken(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Validation("invalid_token", "verification link is invalid")
		}
		return apperr.Internal(err)
	}
	if err := s.assertTokenUsable(row.ConsumedAt.Valid, row.ExpiresAt.Time.Before(s.clock())); err != nil {
		return err
	}

	err = s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.SetEmailVerified(ctx, row.UserID); err != nil {
			return err
		}
		return q.ConsumeEmailVerificationToken(ctx, row.ID)
	})
	if err != nil {
		return apperr.Internal(err)
	}
	s.recordAudit(ctx, &row.UserID, "user.email_verified", "", nil)
	return nil
}

// ResendEmailVerification re-issues a verification token for an unverified
// account and emails it. Like RequestPasswordReset it always reports success
// (no account enumeration); it is a no-op for unknown or already-verified
// addresses.
func (s *Service) ResendEmailVerification(ctx context.Context, email, ip string) error {
	email = normalizeEmail(email)

	user, err := s.pool.Queries().GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // do not disclose non-existence
		}
		return apperr.Internal(err)
	}
	if user.EmailVerified {
		return nil // already verified; nothing to send
	}

	token, err := newOpaqueToken()
	if err != nil {
		return apperr.Internal(err)
	}
	if _, err := s.pool.Queries().CreateEmailVerificationToken(ctx, sqlc.CreateEmailVerificationTokenParams{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		ExpiresAt: tsFromTime(s.clock().Add(emailVerifyTTL)),
	}); err != nil {
		return apperr.Internal(err)
	}

	if mailErr := s.mailer.SendEmailVerification(ctx, email, token); mailErr != nil {
		_ = mailErr // mail failures are non-fatal; the user can retry
	}
	s.recordAudit(ctx, &user.ID, "user.verification_resent", ip, nil)
	return nil
}

// RequestPasswordReset issues a reset token for the address if it exists. It
// always reports success to avoid revealing whether an account exists.
func (s *Service) RequestPasswordReset(ctx context.Context, email, ip string) error {
	email = normalizeEmail(email)

	user, err := s.pool.Queries().GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // do not disclose non-existence
		}
		return apperr.Internal(err)
	}

	token, err := newOpaqueToken()
	if err != nil {
		return apperr.Internal(err)
	}
	if _, err := s.pool.Queries().CreatePasswordResetToken(ctx, sqlc.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		ExpiresAt: tsFromTime(s.clock().Add(passwordResetTTL)),
	}); err != nil {
		return apperr.Internal(err)
	}

	if mailErr := s.mailer.SendPasswordReset(ctx, email, token); mailErr != nil {
		_ = mailErr
	}
	s.recordAudit(ctx, &user.ID, "user.password_reset_requested", ip, nil)
	return nil
}

// ResetPassword validates a reset token and sets a new password, consuming the
// token. The new password must satisfy the strength policy.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword, ip string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	q := s.pool.Queries()
	row, err := q.GetPasswordResetToken(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Validation("invalid_token", "reset link is invalid")
		}
		return apperr.Internal(err)
	}
	if err := s.assertTokenUsable(row.ConsumedAt.Valid, row.ExpiresAt.Time.Before(s.clock())); err != nil {
		return err
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return apperr.Internal(err)
	}
	err = s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.UpdatePasswordHash(ctx, sqlc.UpdatePasswordHashParams{ID: row.UserID, PasswordHash: hash}); err != nil {
			return err
		}
		return q.ConsumePasswordResetToken(ctx, row.ID)
	})
	if err != nil {
		return apperr.Internal(err)
	}
	s.recordAudit(ctx, &row.UserID, "user.password_reset", ip, nil)
	return nil
}

// assertTokenUsable rejects consumed or expired one-time tokens with a uniform
// invalid-token error (no distinction, to limit information leakage).
func (s *Service) assertTokenUsable(consumed, expired bool) error {
	if consumed || expired {
		return apperr.Validation("invalid_token", "this link is invalid or has expired")
	}
	return nil
}
