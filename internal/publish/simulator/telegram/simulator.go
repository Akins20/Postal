// Package telegramsim is a local HTTP simulator of the Telegram Bot API surface
// the adapter uses: getMe, getChat, and sendMessage/sendPhoto/sendVideo. It
// mirrors the {"ok":bool,"result":...,"error_code":int,"description":string}
// envelope and rejects an unknown bot token with 401 (Unauthorized). The Bot
// API path is /bot<TOKEN>/<method>, so a single handler parses token + method.
package telegramsim

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// ValidToken is the bot token the simulator accepts; others get 401.
const ValidToken = "111:SIMBOTTOKEN"

// Server is the running simulator.
type Server struct {
	ts *httptest.Server

	mu     sync.Mutex
	posts  int
	nextID int
}

// New starts a simulator on a random port. Call Close when done.
func New() *Server {
	s := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.ts = httptest.NewServer(mux)
	return s
}

// NewAt starts a simulator on a fixed address for live dev (`postal sim`).
func NewAt(addr string) (*Server, error) {
	s := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
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

// URL returns the simulator base URL (use as APIBaseURL).
func (s *Server) URL() string { return s.ts.URL }

// Close shuts the simulator down.
func (s *Server) Close() { s.ts.Close() }

// PostCount returns how many messages were sent.
func (s *Server) PostCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.posts
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/bot")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		writeErr(w, http.StatusNotFound, 404, "Not Found")
		return
	}
	token, method := parts[0], parts[1]
	if token != ValidToken {
		writeErr(w, http.StatusUnauthorized, 401, "Unauthorized")
		return
	}
	switch method {
	case "getMe":
		writeOK(w, map[string]any{"id": 42, "is_bot": true, "username": "sim_bot", "first_name": "Sim Bot"})
	case "getChat":
		_ = r.ParseForm()
		if r.Form.Get("chat_id") == "" {
			writeErr(w, http.StatusBadRequest, 400, "Bad Request: chat_id is empty")
			return
		}
		writeOK(w, map[string]any{"id": -1001234567890, "title": "Sim Channel", "username": "simchannel", "type": "channel"})
	case "sendMessage", "sendPhoto", "sendVideo":
		_ = r.ParseForm()
		if r.Form.Get("chat_id") == "" {
			writeErr(w, http.StatusBadRequest, 400, "Bad Request: chat_id is empty")
			return
		}
		s.mu.Lock()
		s.nextID++
		s.posts++
		id := s.nextID
		s.mu.Unlock()
		writeOK(w, map[string]any{"message_id": id, "date": 0})
	default:
		writeErr(w, http.StatusNotFound, 404, "Not Found: method not found")
	}
}

func writeOK(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
}

func writeErr(w http.ResponseWriter, status, code int, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": code, "description": desc})
}
