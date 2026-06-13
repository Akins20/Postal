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

// SendEmailVerification emails a one-click verification button.
func (m *ResendMailer) SendEmailVerification(ctx context.Context, email, token string) error {
	link := m.actionLink("/verify-email", token)
	html := renderEmail(
		"Verify your email",
		"Welcome to Postal. Confirm your email address to finish setting up your account.",
		"Verify email", link, token)
	return m.send(ctx, email, "Verify your Postal email", html)
}

// SendPasswordReset emails a one-click password-reset button.
func (m *ResendMailer) SendPasswordReset(ctx context.Context, email, token string) error {
	link := m.actionLink("/reset/confirm", token)
	html := renderEmail(
		"Reset your password",
		"We received a request to reset your Postal password. If this was not you, you can ignore this email.",
		"Reset password", link, token)
	return m.send(ctx, email, "Reset your Postal password", html)
}

// actionLink builds a public web link with the token, or returns "" when no
// AppBaseURL is configured (the email then shows the raw token instead).
func (m *ResendMailer) actionLink(path, token string) string {
	if m.cfg.AppBaseURL == "" {
		return ""
	}
	return strings.TrimRight(m.cfg.AppBaseURL, "/") + path + "?token=" + url.QueryEscape(token)
}

// renderEmail builds a branded, table-based HTML email with a primary action
// button. When no link is available (AppBaseURL unset) it falls back to showing
// the token for manual entry, so the flow still works in pure dev setups.
func renderEmail(heading, intro, label, link, token string) string {
	var action string
	if link != "" {
		action = fmt.Sprintf(
			`<a href=%q style="display:inline-block;background:#2f6bef;color:#ffffff;text-decoration:none;font-weight:600;font-size:15px;padding:13px 26px;border-radius:10px">%s</a>`+
				`<p style="color:#8a8a8e;font-size:12px;line-height:1.5;margin:22px 0 0">If the button does not work, copy and paste this link into your browser:<br>`+
				`<a href=%q style="color:#2f6bef;word-break:break-all">%s</a></p>`,
			link, label, link, link)
	} else {
		action = fmt.Sprintf(
			`<p style="font-size:14px;color:#3a3a3c;margin:0 0 8px">Enter this code in the app:</p>`+
				`<p style="font-family:ui-monospace,Menlo,monospace;font-size:16px;background:#f2f2f7;color:#1c1c1e;padding:12px 16px;border-radius:10px;word-break:break-all;margin:0">%s</p>`,
			token)
	}
	return fmt.Sprintf(
		`<!doctype html><html><body style="margin:0;background:#f5f5f7;padding:32px 16px;`+
			`font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif">`+
			`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0"><tr><td align="center">`+
			`<table role="presentation" width="480" cellpadding="0" cellspacing="0" `+
			`style="background:#ffffff;border-radius:16px;padding:36px;text-align:left;max-width:480px">`+
			`<tr><td>`+
			`<div style="font-size:18px;font-weight:700;color:#1c1c1e">Postal</div>`+
			`<h1 style="font-size:21px;color:#1c1c1e;margin:18px 0 10px">%s</h1>`+
			`<p style="font-size:15px;color:#3a3a3c;line-height:1.55;margin:0 0 24px">%s</p>`+
			`%s`+
			`</td></tr></table>`+
			`<p style="color:#aeaeb2;font-size:11px;margin-top:18px">Postal. Free, no-paywall social media scheduling.</p>`+
			`</td></tr></table></body></html>`,
		heading, intro, action)
}

// send posts a single HTML email to the Resend API.
func (m *ResendMailer) send(ctx context.Context, to, subject, html string) error {
	payload, err := json.Marshal(map[string]any{
		"from":    m.cfg.From,
		"to":      []string{to},
		"subject": subject,
		"html":    html,
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
