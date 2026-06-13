package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResendMailerActionLink(t *testing.T) {
	t.Parallel()

	withBase := &ResendMailer{cfg: ResendConfig{AppBaseURL: "https://app.example.com/"}}
	got := withBase.actionLink("/verify-email", "a b/c") // token needs URL escaping
	want := "https://app.example.com/verify-email?token=a+b%2Fc"
	if got != want {
		t.Fatalf("actionLink = %q, want %q", got, want)
	}

	noBase := &ResendMailer{cfg: ResendConfig{}}
	if link := noBase.actionLink("/verify-email", "tok"); link != "" {
		t.Fatalf("actionLink with no base = %q, want empty", link)
	}
}

func TestRenderEmail(t *testing.T) {
	t.Parallel()

	// With a link: the button href is present and the bare token is not shown
	// as a fallback code block.
	withLink := renderEmail("Verify your email", "intro", "Verify email", "https://x/y?token=z", "TOKEN123")
	if !strings.Contains(withLink, "https://x/y?token=z") {
		t.Fatalf("renderEmail with link should include the link, got %q", withLink)
	}
	if !strings.Contains(withLink, "Verify email") {
		t.Fatalf("renderEmail should include the button label, got %q", withLink)
	}

	// Without a link: fall back to showing the token for manual entry.
	noLink := renderEmail("Verify your email", "intro", "Verify email", "", "TOKEN123")
	if !strings.Contains(noLink, "TOKEN123") {
		t.Fatalf("renderEmail without link should show the token, got %q", noLink)
	}
}

func TestResendMailerSend(t *testing.T) {
	t.Parallel()

	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))
	defer srv.Close()

	m := NewResendMailer(ResendConfig{APIKey: "re_test", From: "Postal <no-reply@example.com>"}, nil)
	m.client = srv.Client()
	// Point the package endpoint at the test server for this call.
	orig := resendURL
	resendURL = srv.URL
	defer func() { resendURL = orig }()

	if err := m.SendEmailVerification(context.Background(), "user@example.com", "vtok"); err != nil {
		t.Fatalf("SendEmailVerification: %v", err)
	}
	if gotAuth != "Bearer re_test" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if payload["from"] != "Postal <no-reply@example.com>" {
		t.Fatalf("from = %v", payload["from"])
	}
}
