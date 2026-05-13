package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Server is the HTTP API server
type Server struct {
	addr     string
	webMode  bool
	sessions *SessionStore
	router   *mux.Router
}

// NewServer creates a Server listening on addr (e.g. ":8080")
func NewServer(addr string) *Server {
	s := &Server{
		addr:     addr,
		sessions: newSessionStore(),
		router:   mux.NewRouter(),
	}
	s.routes()
	return s
}

// NewWebServer creates a Server with the GoTTH web UI enabled
func NewWebServer(addr string) *Server {
	s := &Server{
		addr:     addr,
		webMode:  true,
		sessions: newSessionStore(),
		router:   mux.NewRouter(),
	}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler, forwarding to the router
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s)
}

func (s *Server) routes() {
	api := s.router.PathPrefix("/api").Subrouter()

	// Sessions
	api.HandleFunc("/sessions", s.handleCreateSession).Methods(http.MethodPost)
	api.HandleFunc("/sessions/{id}", s.handleGetSession).Methods(http.MethodGet)
	api.HandleFunc("/sessions/{id}", s.handleDeleteSession).Methods(http.MethodDelete)

	// Metadata
	api.HandleFunc("/sessions/{id}/fetch", s.handleFetchMetadata).Methods(http.MethodPost)

	// Pipeline control
	api.HandleFunc("/sessions/{id}/setlist", s.handleSetSetlist).Methods(http.MethodPost)
	api.HandleFunc("/sessions/{id}/start", s.handleStart).Methods(http.MethodPost)
	api.HandleFunc("/sessions/{id}/cancel", s.handleCancel).Methods(http.MethodPost)

	if s.webMode {
		s.router.HandleFunc("/", s.handleIndex).Methods(http.MethodGet)
		s.router.HandleFunc("/sessions", s.handleSessionsPage).Methods(http.MethodGet)
		s.router.HandleFunc("/reset", s.handleReset).Methods(http.MethodGet)
		s.router.HandleFunc("/sessions/{id}/load", s.handleWebLoadSession).Methods(http.MethodGet)
		s.router.HandleFunc("/sessions/{id}", s.handleWebDeleteSession).Methods(http.MethodDelete)
		s.router.HandleFunc("/fetch", s.handleWebFetch).Methods(http.MethodPost)
		s.router.HandleFunc("/start/{id}", s.handleWebStart).Methods(http.MethodPost)
		s.router.HandleFunc("/events/{id}", s.handleEvents).Methods(http.MethodGet)
		s.router.HandleFunc("/files/{sessionID}/{filename}", s.handleServeFile).Methods(http.MethodGet)
	}
}
