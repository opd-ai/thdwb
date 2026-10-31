// Package webdriver implements a minimal WebDriver server for automating
// thdwb browser instances, primarily for extension testing.
package webdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Session represents a WebDriver session.
type Session struct {
	ID           string        `json:"id"`
	URL          string        `json:"url,omitempty"`
	Title        string        `json:"title,omitempty"`
	Capabilities Capabilities  `json:"capabilities"`
}

// Capabilities describes browser capabilities.
type Capabilities struct {
	BrowserName    string `json:"browserName"`
	BrowserVersion string `json:"browserVersion"`
	PlatformName   string `json:"platformName"`
}

// Server is a WebDriver HTTP server.
type Server struct {
	mu       sync.Mutex
	sessions map[string]*Session
	nextID   int
	handler  http.Handler
}

// NewServer creates a new WebDriver server.
func NewServer() *Server {
	s := &Server{
		sessions: make(map[string]*Session),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/session", s.handleSession)
	mux.HandleFunc("/session/", s.handleSessionID)
	s.handler = mux
	return s
}

// Handler returns the HTTP handler for the WebDriver server.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// handleSession handles POST /session (create) and GET /session (list).
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createSession(w, r)
	case http.MethodGet:
		s.listSessions(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSessionID handles operations on a specific session.
func (s *Server) handleSessionID(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path: /session/{id}/...
	path := r.URL.Path
	id := path[len("/session/"):]
	// Remove trailing slash and any sub-path.
	for i, c := range id {
		if c == '/' {
			id = id[:i]
			break
		}
	}

	switch r.Method {
	case http.MethodDelete:
		s.deleteSession(w, r, id)
	case http.MethodGet:
		s.getSession(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// createSession creates a new WebDriver session.
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Capabilities         Capabilities `json:"capabilities"`
		DesiredCapabilities Capabilities `json:"desiredCapabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	id := fmt.Sprintf("session-%d", s.nextID)
	sess := &Session{
		ID:           id,
		Capabilities: req.Capabilities,
	}
	if sess.Capabilities.BrowserName == "" {
		sess.Capabilities.BrowserName = "thdwb"
	}
	if sess.Capabilities.PlatformName == "" {
		sess.Capabilities.PlatformName = "linux"
	}
	s.sessions[id] = sess

	resp := map[string]interface{}{
		"value": map[string]interface{}{
			"sessionId":   id,
			"capabilities": sess.Capabilities,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// listSessions lists all active sessions.
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := make([]map[string]interface{}, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, map[string]interface{}{
			"sessionId":   sess.ID,
			"capabilities": sess.Capabilities,
		})
	}

	resp := map[string]interface{}{
		"value": sessions,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// getSession returns session details.
func (s *Server) getSession(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	s.mu.Unlock()

	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"value": map[string]interface{}{
			"sessionId":   sess.ID,
			"capabilities": sess.Capabilities,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// deleteSession ends a session.
func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	_, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"value": nil,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// Start starts the WebDriver server on the given address.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.handler)
}

// StartContext starts the server and blocks until ctx is done.
func (s *Server) StartContext(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.handler,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	return srv.ListenAndServe()
}

// SessionCount returns the number of active sessions.
func (s *Server) SessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// GetSession returns a session by ID.
func (s *Server) GetSession(id string) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	return sess, ok
}
