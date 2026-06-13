package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
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

// resendURL is the Resend transactional email API endpoint. It is a var (not a
// const) so tests can point it at a stub server.
var resendURL = "https://api.resend.com/emails"

// ResendConfig holds the settings a ResendMailer needs. APIKey and From are
// required. AppBaseURL is the public web origin used to build the action links;
// when empty the email falls back to presenting the raw token.
type ResendConfig struct {
	APIKey     string
	From       string
	AppBaseURL string
}

// ResendMailer sends transactional auth emails through the Resend HTTP API. It
// is Postal's production Mailer; construct it from configured Resend settings
// and pass it to NewService.
type ResendMailer struct {
	cfg    ResendConfig
	client *http.Client
	log    *slog.Logger
}

// NewResendMailer builds a ResendMailer. A nil logger falls back to slog.Default.
func NewResendMailer(cfg ResendConfig, log *slog.Logger) *ResendMailer {
	if log == nil {
		log = slog.Default()
	}
	return &ResendMailer{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
		log:    log,
	}
}

// SendEmailVerification emails the verification link/token.
func (m *ResendMailer) SendEmailVerification(ctx context.Context, email, token string) error {
	link := m.actionLink("/verify-email", token)
	subject := "Verify your Postal email"
	body := "Welcome to Postal. Confirm your email address to finish setting up your account." +
		actionHTML("Verify email", link, token)
	return m.send(ctx, email, subject, body)
}

// SendPasswordReset emails the password-reset link/token.
func (m *ResendMailer) SendPasswordReset(ctx context.Context, email, token string) error {
	link := m.actionLink("/reset/confirm", token)
	subject := "Reset your Postal password"
	body := "We received a request to reset your Postal password. If this was not you, you can ignore this email." +
		actionHTML("Reset password", link, token)
	return m.send(ctx, email, subject, body)
}

// actionLink builds a public web link with the token, or returns "" when no
// AppBaseURL is configured (the email then shows the raw token instead).
func (m *ResendMailer) actionLink(path, token string) string {
	if m.cfg.AppBaseURL == "" {
		return ""
	}
	return strings.TrimRight(m.cfg.AppBaseURL, "/") + path + "?token=" + url.QueryEscape(token)
}

// actionHTML renders the call-to-action: a link when one is available, otherwise
// the bare token for manual entry.
func actionHTML(label, link, token string) string {
	if link == "" {
		return fmt.Sprintf("<p>Your token: <code>%s</code></p>", token)
	}
	return fmt.Sprintf("<p><a href=%q>%s</a></p><p>Or paste this token: <code>%s</code></p>", link, label, token)
}

// send posts a single HTML email to the Resend API.
func (m *ResendMailer) send(ctx context.Context, to, subject, body string) error {
	payload, err := json.Marshal(map[string]any{
		"from":    m.cfg.From,
		"to":      []string{to},
		"subject": subject,
		"html":    "<html><body>" + body + "</body></html>",
	})
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("resend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	m.log.InfoContext(ctx, "transactional email sent via Resend",
		slog.String("to", to), slog.String("subject", subject))
	return nil
}
