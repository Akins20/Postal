package auth

import (
	"context"
	"log/slog"
)

// Mailer delivers transactional auth emails. Production wires a real provider;
// local/dev uses ConsoleMailer, which logs the token so flows are testable
// without an SMTP server.
type Mailer interface {
	// SendEmailVerification delivers an email-verification token to the address.
	SendEmailVerification(ctx context.Context, email, token string) error
	// SendPasswordReset delivers a password-reset token to the address.
	SendPasswordReset(ctx context.Context, email, token string) error
}

// ConsoleMailer logs auth emails to the structured logger instead of sending
// them. It is for local development and tests only.
type ConsoleMailer struct {
	log *slog.Logger
}

// NewConsoleMailer builds a ConsoleMailer. A nil logger falls back to the
// default slog logger.
func NewConsoleMailer(log *slog.Logger) ConsoleMailer {
	if log == nil {
		log = slog.Default()
	}
	return ConsoleMailer{log: log}
}

// SendEmailVerification logs the verification token.
func (m ConsoleMailer) SendEmailVerification(ctx context.Context, email, token string) error {
	m.log.InfoContext(ctx, "DEV email verification (console mailer)",
		slog.String("to", email), slog.String("verification_token", token))
	return nil
}

// SendPasswordReset logs the reset token.
func (m ConsoleMailer) SendPasswordReset(ctx context.Context, email, token string) error {
	m.log.InfoContext(ctx, "DEV password reset (console mailer)",
		slog.String("to", email), slog.String("reset_token", token))
	return nil
}
