package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"bootlegger/internal/core"

	"github.com/gorilla/mux"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// POST /api/sessions
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, "BOOTLEGGER")
	rec := s.sessions.create(baseDir)
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":  rec.Session.ID,
		"dir": rec.Session.Dir,
	})
}

// GET /api/sessions/{id}
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	rec, err := s.sessions.get(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	rec.mu.RLock()
	defer rec.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         rec.Session.ID,
		"state":      rec.State,
		"created_at": rec.CreatedAt,
		"tracks":     rec.Tracks,
	})
}

// DELETE /api/sessions/{id}
func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	s.sessions.delete(mux.Vars(r)["id"])
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/sessions/{id}/fetch  body: {"url":"..."}
func (s *Server) handleFetchMetadata(w http.ResponseWriter, r *http.Request) {
	rec, err := s.sessions.get(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		writeError(w, http.StatusBadRequest, "url required")
		return
	}

	meta, err := core.FetchMetadata(r.Context(), body.URL)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	rec.mu.Lock()
	rec.Metadata = meta
	rec.State = core.StatePreview
	rec.mu.Unlock()

	writeJSON(w, http.StatusOK, meta)
}

// POST /api/sessions/{id}/setlist  body: {"tracks":[...]}
func (s *Server) handleSetSetlist(w http.ResponseWriter, r *http.Request) {
	rec, err := s.sessions.get(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var body struct {
		Tracks []core.Track `json:"tracks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	rec.mu.Lock()
	rec.Tracks = body.Tracks
	rec.State = core.StateSetlistInput
	rec.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]int{"count": len(body.Tracks)})
}

// POST /api/sessions/{id}/start  body: {"url":"...","format_id":"..."}
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	rec, err := s.sessions.get(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var body struct {
		URL      string `json:"url"`
		FormatID string `json:"format_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		writeError(w, http.StatusBadRequest, "url required")
		return
	}

	rec.mu.Lock()
	if rec.cancel != nil {
		rec.mu.Unlock()
		writeError(w, http.StatusConflict, "pipeline already running")
		return
	}

	events := make(chan core.Event, 200)
	ctx, cancel := context.WithCancel(context.Background())
	rec.Events = events
	rec.cancel = cancel
	rec.State = core.StateDownloading
	rec.mu.Unlock()

	pipeline := core.NewPipeline(rec.Session)
	cfg := core.RunConfig{
		URL:      body.URL,
		FormatID: body.FormatID,
		Tracks:   rec.Tracks,
	}

	go func() {
		_, _, runErr := pipeline.Run(ctx, cfg, events)
		rec.mu.Lock()
		if runErr != nil {
			rec.State = core.StateError
		} else {
			rec.State = core.StateCommitted
		}
		rec.cancel = nil
		close(events)
		rec.mu.Unlock()
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

// POST /api/sessions/{id}/cancel
func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	rec, err := s.sessions.get(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	rec.mu.Lock()
	if rec.cancel != nil {
		rec.cancel()
		rec.cancel = nil
		rec.State = core.StateError
	}
	rec.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
