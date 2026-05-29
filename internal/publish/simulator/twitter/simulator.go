// Package twittersim is a faithful local HTTP simulator of the X/Twitter API v2
// used to test the adapter and pipeline without touching the paid, rate-limited
// real API. It mirrors the endpoints, schemas, status codes (create -> 201),
// the media INIT/APPEND/FINALIZE/STATUS sequence, public_metrics, and supports
// controllable error injection (429, 401, 403 duplicate, 5xx, over-limit).
package twittersim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// maxText is the simulator's defense-in-depth text length cap (in runes).
const maxText = 280

// Server is the running simulator.
type Server struct {
	ts *httptest.Server

	mu          sync.Mutex
	validAccess map[string]bool   // access tokens currently accepted
	tweets      map[string]string // tweet id -> text
	seenText    map[string]string // text -> tweet id (duplicate detection)
	media       map[string]bool   // media id -> exists
	nextTok     int
	nextTweet   int
	nextMedia   int

	// injection (consumed once unless noted)
	forceCreateStatus int  // next create-tweet returns this status if != 0
	mediaProcessing   bool // when true, FINALIZE reports async processing (video/GIF)
}

// EnableMediaProcessing makes FINALIZE report async processing so the adapter's
// STATUS-poll path (used for video/GIF) is exercised.
func (s *Server) EnableMediaProcessing() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mediaProcessing = true
}

// New starts a simulator and returns it. Call Close when done.
func New() *Server {
	s := &Server{
		validAccess: map[string]bool{},
		tweets:      map[string]string{},
		seenText:    map[string]string{},
		media:       map[string]bool{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2/oauth2/token", s.handleToken)
	mux.HandleFunc("POST /2/oauth2/revoke", s.handleRevoke)
	mux.HandleFunc("GET /2/users/me", s.handleUsersMe)
	mux.HandleFunc("POST /2/tweets", s.handleCreateTweet)
	mux.HandleFunc("GET /2/tweets/{id}", s.handleTweetLookup)
	mux.HandleFunc("/2/media/upload", s.handleMedia)
	s.ts = httptest.NewServer(mux)
	return s
}

// URL returns the simulator base URL (use as the adapter APIBaseURL).
func (s *Server) URL() string { return s.ts.URL }

// Close shuts the simulator down.
func (s *Server) Close() { s.ts.Close() }

// --- injection knobs ---

// ExpireAccessTokens invalidates all currently-issued access tokens so API calls
// return 401 until a refresh issues a new one.
func (s *Server) ExpireAccessTokens() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.validAccess = map[string]bool{}
}

// ForceNextCreateStatus makes the next create-tweet call return the given HTTP
// status (e.g. 429, 500, 403). 403 yields a duplicate-shaped body.
func (s *Server) ForceNextCreateStatus(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forceCreateStatus = code
}

// TweetCount returns how many tweets have been created (for idempotency checks).
func (s *Server) TweetCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tweets)
}

// --- handlers ---

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	grant := r.Form.Get("grant_type")
	if grant == "refresh_token" && r.Form.Get("refresh_token") == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	s.mu.Lock()
	s.nextTok++
	access := fmt.Sprintf("at-%d", s.nextTok)
	refresh := fmt.Sprintf("rt-%d", s.nextTok)
	s.validAccess[access] = true
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"token_type":    "bearer",
		"expires_in":    7200,
		"access_token":  access,
		"refresh_token": refresh,
		"scope":         "tweet.read tweet.write users.read media.write offline.access",
	})
}

func (s *Server) handleRevoke(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})
}

func (s *Server) handleUsersMe(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeInvalidToken(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"id": "sim-user-1", "username": "simuser", "name": "Sim User"},
	})
}

func (s *Server) handleCreateTweet(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeInvalidToken(w)
		return
	}
	if status := s.takeForcedStatus(); status != 0 {
		s.writeForced(w, status)
		return
	}

	var body struct {
		Text  string `json:"text"`
		Media *struct {
			MediaIDs []string `json:"media_ids"`
		} `json:"media"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if utf8.RuneCountInString(body.Text) > maxText {
		writeError(w, http.StatusBadRequest, "Your post text is too long.")
		return
	}
	if body.Text == "" && (body.Media == nil || len(body.Media.MediaIDs) == 0) {
		writeError(w, http.StatusBadRequest, "post must contain text or media")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Duplicate-content rejection (text-only posts), mirroring X's 403.
	if body.Text != "" {
		if _, dup := s.seenText[body.Text]; dup {
			writeError(w, http.StatusForbidden, "You are not allowed to create a Post with duplicate content.")
			return
		}
	}
	s.nextTweet++
	id := fmt.Sprintf("%d", 1000000000000000000+int64(s.nextTweet))
	s.tweets[id] = body.Text
	if body.Text != "" {
		s.seenText[body.Text] = id
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]string{"id": id, "text": body.Text},
	})
}

func (s *Server) handleTweetLookup(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeInvalidToken(w)
		return
	}
	id := r.PathValue("id")
	s.mu.Lock()
	_, ok := s.tweets[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id": id,
			"public_metrics": map[string]int{
				"like_count": 5, "retweet_count": 2, "reply_count": 1,
				"quote_count": 0, "impression_count": 100, "bookmark_count": 3,
			},
		},
	})
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	if !s.authOK(r) {
		writeInvalidToken(w)
		return
	}
	// STATUS is a GET with query params; INIT/APPEND/FINALIZE are POSTs.
	command := r.URL.Query().Get("command")
	if command == "" {
		_ = r.ParseMultipartForm(8 << 20)
		if command = r.FormValue("command"); command == "" {
			_ = r.ParseForm()
			command = r.Form.Get("command")
		}
	}

	switch strings.ToUpper(command) {
	case "INIT":
		s.mu.Lock()
		s.nextMedia++
		id := fmt.Sprintf("media-%d", s.nextMedia)
		s.media[id] = true
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"id": id}})
	case "APPEND":
		w.WriteHeader(http.StatusNoContent)
	case "FINALIZE":
		id := r.FormValue("media_id")
		s.mu.Lock()
		processing := s.mediaProcessing
		s.mu.Unlock()
		if processing {
			// Video/GIF: report async processing so the client polls STATUS.
			writeJSON(w, http.StatusOK, map[string]any{
				"data": map[string]any{"id": id, "processing_info": map[string]string{"state": "in_progress"}},
			})
			return
		}
		// Images finalize synchronously (no processing_info -> succeeded).
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"id": id}})
	case "STATUS":
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"processing_info": map[string]string{"state": "succeeded"}},
		})
	default:
		writeError(w, http.StatusBadRequest, "unknown media command")
	}
}

// --- helpers ---

func (s *Server) authOK(r *http.Request) bool {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tok == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.validAccess[tok]
}

func (s *Server) takeForcedStatus() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.forceCreateStatus
	s.forceCreateStatus = 0
	return status
}

func (s *Server) writeForced(w http.ResponseWriter, status int) {
	switch status {
	case http.StatusTooManyRequests:
		w.Header().Set("x-rate-limit-limit", "50")
		w.Header().Set("x-rate-limit-remaining", "0")
		w.Header().Set("x-rate-limit-reset", strconv.FormatInt(time.Now().Add(2*time.Second).Unix(), 10))
		writeJSON(w, status, map[string]any{"errors": []map[string]any{{"code": 88, "message": "Rate limit exceeded"}}})
	case http.StatusForbidden:
		writeError(w, status, "You are not allowed to create a Post with duplicate content.")
	default:
		writeError(w, status, "server error")
	}
}

func writeInvalidToken(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"errors": []map[string]any{{"code": 89, "message": "Invalid or expired token"}},
	})
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]any{
		"title": http.StatusText(status), "detail": detail, "status": status,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
