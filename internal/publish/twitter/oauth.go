package twitter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
)

// X OAuth / API paths (relative to the configured base URLs).
const (
	pathAuthorize = "/i/oauth2/authorize"
	// #nosec G101 -- OAuth endpoint URL path, not a hardcoded credential.
	pathToken   = "/2/oauth2/token"
	pathRevoke  = "/2/oauth2/revoke"
	pathUsersMe = "/2/users/me?user.fields=username,name"
)

// tokenResponse is the OAuth2 token endpoint payload.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// AuthURL builds the X authorize URL with PKCE S256 and CSRF state.
func (a *Adapter) AuthURL(state, codeChallenge, redirectURI string) string {
	if redirectURI == "" {
		redirectURI = a.cfg.RedirectURI
	}
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", a.cfg.ClientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("scope", oauthScopes)
	v.Set("state", state)
	v.Set("code_challenge", codeChallenge)
	v.Set("code_challenge_method", "S256")
	return a.cfg.AuthBaseURL + pathAuthorize + "?" + v.Encode()
}

// ExchangeCode swaps an authorization code (+ PKCE verifier) for tokens.
func (a *Adapter) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*channel.Token, error) {
	if redirectURI == "" {
		redirectURI = a.cfg.RedirectURI
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", codeVerifier)
	form.Set("client_id", a.cfg.ClientID)
	return a.tokenRequest(ctx, form)
}

// RefreshToken obtains a fresh token set from a refresh token.
func (a *Adapter) RefreshToken(ctx context.Context, refreshToken string) (*channel.Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", a.cfg.ClientID)
	return a.tokenRequest(ctx, form)
}

// tokenRequest performs a form-encoded POST to the token endpoint (Basic auth
// for confidential clients) and maps the response to a channel.Token.
func (a *Adapter) tokenRequest(ctx context.Context, form url.Values) (*channel.Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.APIBaseURL+pathToken, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, publish.Terminal("request_build_failed", "could not build token request", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if a.cfg.ClientSecret != "" {
		req.SetBasicAuth(a.cfg.ClientID, a.cfg.ClientSecret)
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, publish.Retryable("network_error", "token request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			// A transient rate-limit on refresh must NOT expire the channel.
			return nil, publish.RateLimited(retryAfterFromHeaders(resp.Header), errBody(data))
		case resp.StatusCode >= 500:
			return nil, publish.Retryable("token_server_error", "token endpoint server error", errBody(data))
		default:
			// 4xx (invalid_grant, invalid code/verifier) — terminal; the channel
			// refresh service marks the channel expired only on terminal errors.
			return nil, publish.Terminal("token_exchange_failed", "token endpoint rejected the request", errBody(data))
		}
	}

	var tr tokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, publish.Terminal("decode_failed", "could not decode token response", err)
	}
	tok := &channel.Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		Scopes:       strings.Fields(tr.Scope),
	}
	if tr.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return tok, nil
}

// Account resolves the connected account identity from an access token.
func (a *Adapter) Account(ctx context.Context, accessToken string) (*channel.Account, error) {
	var resp struct {
		Data struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"data"`
	}
	if err := a.getJSON(ctx, a.cfg.APIBaseURL+pathUsersMe, accessToken, &resp); err != nil {
		return nil, err
	}
	handle := resp.Data.Username
	if handle != "" {
		handle = "@" + handle
	}
	return &channel.Account{ID: resp.Data.ID, Handle: handle, DisplayName: resp.Data.Name}, nil
}

// Revoke best-effort revokes a token at X. Errors are returned but the caller
// (disconnect) treats revocation as best-effort.
func (a *Adapter) Revoke(ctx context.Context, token string) error {
	form := url.Values{}
	form.Set("token", token)
	form.Set("client_id", a.cfg.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.APIBaseURL+pathRevoke, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if a.cfg.ClientSecret != "" {
		req.SetBasicAuth(a.cfg.ClientID, a.cfg.ClientSecret)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
