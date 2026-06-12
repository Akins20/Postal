// Package instagramsim is a faithful local HTTP simulator of the Meta Graph
// API surface the Instagram adapter uses: OAuth dialog + token exchange,
// page -> IG business account resolution, the media container flow
// (create -> status -> publish), and media insights. Same shapes, same error
// envelope ({"error":{"message","code"}}), same auth behavior (code 190).
package instagramsim

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

// maxCaption mirrors Instagram's 2200-character caption cap.
const maxCaption = 2200

// Server is the running simulator.
type Server struct {
	ts *httptest.Server

	mu          sync.Mutex
	validTokens map[string]bool
	containers  map[string]string // container id -> status_code
	posts       map[string]bool
	nextID      int

	// processingPolls > 0 makes containers report IN_PROGRESS that many times
	// before FINISHED (exercises the adapter's poll/retry path).
	processingPolls int
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
	s := &Server{validTokens: map[string]bool{}, containers: map[string]string{}, posts: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v21.0/dialog/oauth", s.handleAuthorize)
	mux.HandleFunc("GET /v21.0/oauth/access_token", s.handleToken)
	mux.HandleFunc("GET /v21.0/me/accounts", s.handleAccounts)
	mux.HandleFunc("GET /v21.0/{id}", s.handleNode)
	mux.HandleFunc("POST /v21.0/{id}/media", s.handleCreateContainer)
	mux.HandleFunc("POST /v21.0/{id}/media_publish", s.handlePublish)
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

// EnableProcessing makes the next containers report IN_PROGRESS n times.
func (s *Server) EnableProcessing(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processingPolls = n
}

// PostCount returns how many media objects were published.
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
	code := fmt.Sprintf("igcode-%d", s.nextID)
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
	token := fmt.Sprintf("ig-at-%d", s.nextID)
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
	writeJSON(w, http.StatusOK, map[string]any{"data": []map[string]string{{"id": "page-1", "name": "Sim Page"}}})
}

// handleNode serves page lookups, IG-user lookups, and container status.
func (s *Server) handleNode(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token")
		return
	}
	id := r.PathValue("id")
	switch {
	case id == "page-1":
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "instagram_business_account": map[string]string{"id": "ig-1"}})
	case id == "ig-1":
		writeJSON(w, http.StatusOK, map[string]string{"id": "ig-1", "username": "simgram", "name": "Sim Gram"})
	case strings.HasPrefix(id, "igc-"):
		s.mu.Lock()
		status, ok := s.containers[id]
		if ok && status == "IN_PROGRESS" {
			if s.processingPolls > 0 {
				s.processingPolls--
			} else {
				s.containers[id] = "FINISHED"
				status = "FINISHED"
			}
		}
		s.mu.Unlock()
		if !ok {
			writeError(w, http.StatusBadRequest, 100, "unknown container")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": id, "status_code": status})
	default:
		writeError(w, http.StatusBadRequest, 100, "unknown node")
	}
}

func (s *Server) handleCreateContainer(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token")
		return
	}
	if r.PathValue("id") != "ig-1" {
		writeError(w, http.StatusBadRequest, 100, "unknown ig user")
		return
	}
	_ = r.ParseForm()
	caption := r.Form.Get("caption")
	if utf8.RuneCountInString(caption) > maxCaption {
		writeError(w, http.StatusBadRequest, 100, "caption too long")
		return
	}
	if r.Form.Get("image_url") == "" && r.Form.Get("video_url") == "" {
		writeError(w, http.StatusBadRequest, 100, "image_url or video_url required")
		return
	}
	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("igc-%d", s.nextID)
	if s.processingPolls > 0 {
		s.containers[id] = "IN_PROGRESS"
	} else {
		s.containers[id] = "FINISHED"
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token")
		return
	}
	_ = r.ParseForm()
	creation := r.Form.Get("creation_id")
	s.mu.Lock()
	status, ok := s.containers[creation]
	s.mu.Unlock()
	if !ok || status != "FINISHED" {
		writeError(w, http.StatusBadRequest, 100, "container not ready")
		return
	}
	s.mu.Lock()
	s.nextID++
	postID := fmt.Sprintf("igpost-%d", s.nextID)
	s.posts[postID] = true
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"id": postID})
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
		writeError(w, http.StatusBadRequest, 100, "unknown media")
		return
	}
	point := func(name string, v int64) map[string]any {
		return map[string]any{"name": name, "values": []map[string]int64{{"value": v}}}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": []map[string]any{
		point("likes", 7), point("comments", 2), point("reach", 150),
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
