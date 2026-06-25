// Package facebooksim is a faithful local HTTP simulator of the Meta Graph API
// surface the Facebook adapter uses: OAuth dialog + token exchange, page
// resolution (with the page access token), and Page feed/photos/videos posting
// plus post insights. Same shapes, same error envelope
// ({"error":{"message","code"}}), same auth behavior (code 190).
package facebooksim

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"unicode/utf8"
)

// maxMessage mirrors Facebook's status length cap.
const maxMessage = 63206

const (
	pageID    = "fbpage-1"
	pageToken = "fb-page-token"
)

// Server is the running simulator.
type Server struct {
	ts *httptest.Server

	mu          sync.Mutex
	validTokens map[string]bool
	posts       map[string]bool
	nextID      int
}

// New starts a simulator on a random port. Call Close when done.
func New() *Server {
	s, mux := build()
	s.ts = httptest.NewServer(mux)
	return s
}

// NewAt starts a simulator on a fixed address for live dev (`postal sim`).
func NewAt(addr string) (*Server, error) {
	s, mux := build()
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", addr, err)
	}
	ts := httptest.NewUnstartedServer(mux)
	_ = ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	s.ts = ts
	return s, nil
}

func build() (*Server, *http.ServeMux) {
	s := &Server{
		validTokens: map[string]bool{pageToken: true}, // the page token is always valid
		posts:       map[string]bool{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v21.0/dialog/oauth", s.handleAuthorize)
	mux.HandleFunc("GET /v21.0/oauth/access_token", s.handleToken)
	mux.HandleFunc("GET /v21.0/me/accounts", s.handleAccounts)
	mux.HandleFunc("POST /v21.0/{id}/feed", s.handlePost("message"))
	mux.HandleFunc("POST /v21.0/{id}/photos", s.handlePost("url"))
	mux.HandleFunc("POST /v21.0/{id}/videos", s.handlePost("file_url"))
	mux.HandleFunc("GET /v21.0/{id}/insights", s.handleInsights)
	mux.HandleFunc("DELETE /v21.0/me/permissions", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	})
	return s, mux
}

// URL returns the simulator base URL (use as APIBaseURL and AuthBaseURL).
func (s *Server) URL() string { return s.ts.URL }

// Close shuts the simulator down.
func (s *Server) Close() { s.ts.Close() }

// PostCount returns how many posts were published.
func (s *Server) PostCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.posts)
}

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirect, state := q.Get("redirect_uri"), q.Get("state")
	if redirect == "" || state == "" || q.Get("client_id") == "" {
		writeError(w, http.StatusBadRequest, 100, "missing oauth parameters")
		return
	}
	u, err := url.Parse(redirect)
	if err != nil {
		writeError(w, http.StatusBadRequest, 100, "bad redirect_uri")
		return
	}
	s.mu.Lock()
	s.nextID++
	code := fmt.Sprintf("fbcode-%d", s.nextID)
	s.mu.Unlock()
	v := u.Query()
	v.Set("state", state)
	v.Set("code", code)
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("code") == "" && q.Get("fb_exchange_token") == "" {
		writeError(w, http.StatusBadRequest, 100, "missing code or fb_exchange_token")
		return
	}
	s.mu.Lock()
	s.nextID++
	token := fmt.Sprintf("fb-at-%d", s.nextID)
	s.validTokens[token] = true
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"access_token": token, "expires_in": 5184000, "token_type": "bearer"})
}

func (s *Server) authOK(r *http.Request) bool {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.validTokens[tok]
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": []map[string]string{
		{"id": pageID, "name": "Sim Page", "access_token": pageToken},
	}})
}

// handlePost handles feed/photos/videos: the requiredField names the content
// param each edge needs (message/url/file_url).
func (s *Server) handlePost(requiredField string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authOK(r) {
			writeError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token")
			return
		}
		if r.PathValue("id") != pageID {
			writeError(w, http.StatusBadRequest, 100, "unknown page")
			return
		}
		_ = r.ParseForm()
		msg := r.Form.Get("message") + r.Form.Get("caption") + r.Form.Get("description")
		if utf8.RuneCountInString(msg) > maxMessage {
			writeError(w, http.StatusBadRequest, 100, "message too long")
			return
		}
		if r.Form.Get(requiredField) == "" {
			writeError(w, http.StatusBadRequest, 100, requiredField+" is required")
			return
		}
		s.mu.Lock()
		s.nextID++
		postID := fmt.Sprintf("%s_%d", pageID, s.nextID)
		s.posts[postID] = true
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"id": postID, "post_id": postID})
	}
}

func (s *Server) handleInsights(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token")
		return
	}
	id := r.PathValue("id")
	s.mu.Lock()
	known := s.posts[id]
	s.mu.Unlock()
	if !known {
		writeError(w, http.StatusBadRequest, 100, "unknown post")
		return
	}
	point := func(name string, v int64) map[string]any {
		return map[string]any{"name": name, "values": []map[string]int64{{"value": v}}}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": []map[string]any{
		point("post_impressions", 320), point("post_engaged_users", 41),
	}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status, code int, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"message": msg, "code": code}})
}
