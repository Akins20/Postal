// Package tiktoksim is a faithful local HTTP simulator of TikTok's Content
// Posting API surface: OAuth authorize/token, user info, creator_info,
// video init + chunk upload, photo content init, publish status, and video
// metrics. Same shapes and error envelopes; unaudited-style accounts can be
// simulated by restricting privacy options to SELF_ONLY.
package tiktoksim

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
)

// Server is the running simulator.
type Server struct {
	ts *httptest.Server

	mu          sync.Mutex
	validTokens map[string]bool
	uploads     map[string]int64  // publish id -> uploaded byte count
	status      map[string]string // publish id -> status
	posts       map[string]bool
	nextID      int

	selfOnly        bool // unaudited app: only SELF_ONLY privacy
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
	s := &Server{
		validTokens: map[string]bool{}, uploads: map[string]int64{},
		status: map[string]string{}, posts: map[string]bool{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/auth/authorize/", s.handleAuthorize)
	mux.HandleFunc("POST /v2/oauth/token/", s.handleToken)
	mux.HandleFunc("POST /v2/oauth/revoke/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{})
	})
	mux.HandleFunc("GET /v2/user/info/", s.handleUserInfo)
	mux.HandleFunc("POST /v2/post/publish/creator_info/query/", s.handleCreatorInfo)
	mux.HandleFunc("POST /v2/post/publish/video/init/", s.handleVideoInit)
	mux.HandleFunc("POST /v2/post/publish/content/init/", s.handleContentInit)
	mux.HandleFunc("PUT /upload/{id}", s.handleUpload)
	mux.HandleFunc("POST /v2/post/publish/status/fetch/", s.handleStatus)
	mux.HandleFunc("POST /v2/video/query/", s.handleVideoQuery)
	return s, mux
}

// URL returns the simulator base URL (use as APIBaseURL and AuthBaseURL).
func (s *Server) URL() string { return s.ts.URL }

// Close shuts the simulator down.
func (s *Server) Close() { s.ts.Close() }

// SetSelfOnly restricts privacy options to SELF_ONLY (unaudited app).
func (s *Server) SetSelfOnly(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selfOnly = v
}

// EnableProcessing makes the next status polls report PROCESSING n times.
func (s *Server) EnableProcessing(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processingPolls = n
}

// PostCount returns how many posts completed publishing.
func (s *Server) PostCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.posts)
}

// UploadedBytes returns how many bytes were uploaded for a publish id.
func (s *Server) UploadedBytes(publishID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.uploads[publishID]
}

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirect, state := q.Get("redirect_uri"), q.Get("state")
	if redirect == "" || state == "" || q.Get("client_key") == "" || q.Get("code_challenge") == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing oauth parameters")
		return
	}
	u, err := url.Parse(redirect)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "bad redirect_uri")
		return
	}
	s.mu.Lock()
	s.nextID++
	code := fmt.Sprintf("ttcode-%d", s.nextID)
	s.mu.Unlock()
	v := u.Query()
	v.Set("state", state)
	v.Set("code", code)
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	grant := r.Form.Get("grant_type")
	if grant == "refresh_token" && r.Form.Get("refresh_token") == "" {
		writeJSON(w, http.StatusOK, map[string]string{"error": "invalid_request", "error_description": "missing refresh_token"})
		return
	}
	s.mu.Lock()
	s.nextID++
	access := fmt.Sprintf("tt-at-%d", s.nextID)
	refresh := fmt.Sprintf("tt-rt-%d", s.nextID)
	s.validTokens[access] = true
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access, "refresh_token": refresh, "expires_in": 86400,
		"scope": "user.info.basic,video.publish,video.list", "token_type": "Bearer",
	})
}

func (s *Server) authOK(r *http.Request) bool {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.validTokens[tok]
}

func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, "access_token_invalid", "The access token is invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"user": map[string]string{
		"open_id": "tt-open-1", "display_name": "Sim Tok", "username": "simtok",
	}}})
}

func (s *Server) handleCreatorInfo(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, "access_token_invalid", "The access token is invalid")
		return
	}
	s.mu.Lock()
	levels := []string{"PUBLIC_TO_EVERYONE", "MUTUAL_FOLLOW_FRIENDS", "SELF_ONLY"}
	if s.selfOnly {
		levels = []string{"SELF_ONLY"}
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"creator_username": "simtok", "privacy_level_options": levels,
	}})
}

func (s *Server) handleVideoInit(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, "access_token_invalid", "The access token is invalid")
		return
	}
	var in struct {
		SourceInfo struct {
			Source    string `json:"source"`
			VideoSize int64  `json:"video_size"`
		} `json:"source_info"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.SourceInfo.Source != "FILE_UPLOAD" || in.SourceInfo.VideoSize <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_params", "expected FILE_UPLOAD with video_size")
		return
	}
	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("ttpub-%d", s.nextID)
	s.status[id] = "PROCESSING_UPLOAD"
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{
		"publish_id": id, "upload_url": s.ts.URL + "/upload/" + id,
	}})
}

func (s *Server) handleContentInit(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, "access_token_invalid", "The access token is invalid")
		return
	}
	var in struct {
		MediaType  string `json:"media_type"`
		SourceInfo struct {
			Source      string   `json:"source"`
			PhotoImages []string `json:"photo_images"`
		} `json:"source_info"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.MediaType != "PHOTO" || in.SourceInfo.Source != "PULL_FROM_URL" || len(in.SourceInfo.PhotoImages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_params", "expected PHOTO via PULL_FROM_URL")
		return
	}
	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("ttpub-%d", s.nextID)
	s.status[id] = "PROCESSING_DOWNLOAD"
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"publish_id": id}})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, _ := io.Copy(io.Discard, r.Body)
	s.mu.Lock()
	if _, ok := s.status[id]; !ok {
		s.mu.Unlock()
		writeError(w, http.StatusNotFound, "invalid_params", "unknown publish id")
		return
	}
	s.uploads[id] += n
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, "access_token_invalid", "The access token is invalid")
		return
	}
	var in struct {
		PublishID string `json:"publish_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	s.mu.Lock()
	st, ok := s.status[in.PublishID]
	if !ok {
		s.mu.Unlock()
		writeError(w, http.StatusBadRequest, "invalid_params", "unknown publish id")
		return
	}
	if s.processingPolls > 0 {
		s.processingPolls--
	} else if st != "PUBLISH_COMPLETE" {
		s.status[in.PublishID] = "PUBLISH_COMPLETE"
		st = "PUBLISH_COMPLETE"
		s.posts[in.PublishID] = true
	}
	s.mu.Unlock()
	resp := map[string]any{"status": st}
	if st == "PUBLISH_COMPLETE" {
		// Field name matches TikTok's documented (mis)spelling.
		resp["publicaly_available_post_id"] = []int64{7421} //nolint:misspell
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": resp})
}

func (s *Server) handleVideoQuery(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeError(w, http.StatusUnauthorized, "access_token_invalid", "The access token is invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"videos": []map[string]int64{
		{"like_count": 11, "comment_count": 3, "share_count": 1, "view_count": 240},
	}}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
