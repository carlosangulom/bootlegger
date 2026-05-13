package api

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bootlegger/internal/api/templates"
	"bootlegger/internal/core"
	"bootlegger/internal/session"
	"bootlegger/internal/source"
	"github.com/gorilla/mux"
)

// sessionInitData is marshalled to JSON and embedded in the LoadedIndexPage
// so the Alpine sessionForm component can pre-populate itself on init.
type sessionInitData struct {
	SetlistText      string             `json:"setlistText"`
	SelectedFormatID string             `json:"selectedFormatId"`
	Accepted         bool               `json:"accepted"`
	Tracks           []sessionInitTrack `json:"tracks"`
}

type sessionInitTrack struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	StartSec int    `json:"startSec"` // maps to Alpine `start` field
	Selected bool   `json:"selected"`
}

// GET /sessions — session manager page
func (s *Server) handleSessionsPage(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, "BOOTLEGGER")
	records, _ := session.ScanSessions(baseDir)
	w.Header().Set("Content-Type", "text/html")
	templates.SessionsPage(records).Render(r.Context(), w)
}

// GET / — new session form
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	templates.IndexPage().Render(r.Context(), w)
}

// GET /reset → returns SourceForm fragment to replace #content
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	templates.SourceForm().Render(r.Context(), w)
}

// DELETE /sessions/{id} — delete session from disk and in-memory store
func (s *Server) handleWebDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, "BOOTLEGGER")
	sessionDir := filepath.Join(baseDir, "sessions", id)

	// Remove from in-memory store (cancels pipeline if running)
	s.sessions.delete(id)

	// Remove from disk
	_ = session.DeleteSession(sessionDir)

	w.WriteHeader(http.StatusOK)
}

// GET /sessions/{id}/load — restore a previous session into a fresh form
func (s *Server) handleWebLoadSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, "BOOTLEGGER")
	sessionDir := filepath.Join(baseDir, "sessions", id)

	savedRec, err := session.Load(sessionDir)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Committed sessions: open the player directly — no need to re-run the pipeline.
	// Scan the splits directory so this also works for sessions predating SplitFiles.
	if savedRec.State == "COMMITTED" {
		splitsDir := filepath.Join(savedRec.Dir, "splits")
		playerTracks := scanSplitTracks(splitsDir)
		if len(playerTracks) > 0 {
			// Reconstruct a core.Track list from the files that actually exist.
			coreTracks := make([]core.Track, len(playerTracks))
			for i, pt := range playerTracks {
				coreTracks[i] = core.Track{
					Number: pt.Number,
					Title:  pt.Title,
					Status: core.TrackComplete,
				}
			}
			w.Header().Set("Content-Type", "text/html")
			templates.CommittedSessionPage(savedRec, coreTracks, playerTracks).Render(r.Context(), w)
			return
		}
	}

	// Legacy sessions (created before metadata persistence) can't be fully restored.
	if savedRec.Metadata == nil {
		w.Header().Set("Content-Type", "text/html")
		templates.LegacyLoadPage(savedRec.URL).Render(r.Context(), w)
		return
	}

	// Create a new in-memory session and pre-populate it with the saved metadata.
	newRec := s.sessions.create(baseDir)
	meta := sourceMetaFromSaved(savedRec.Metadata)
	newRec.mu.Lock()
	newRec.Metadata = meta
	newRec.State = core.StatePreview
	newRec.mu.Unlock()

	// Persist a PREVIEW record for the new session (includes metadata so it
	// can itself be loaded later if the user abandons before starting).
	writeSessionRecord(newRec.Session.Dir, newRec.Session.ID,
		savedRec.URL, savedRec.Title, "PREVIEW", 0, nil, savedRec.Metadata)

	// Build the init payload for Alpine.
	initTracks := make([]sessionInitTrack, len(savedRec.Tracks))
	for i, t := range savedRec.Tracks {
		initTracks[i] = sessionInitTrack{
			Number:   t.Number,
			Title:    t.Title,
			StartSec: t.StartSec,
			Selected: t.Selected,
		}
	}
	initData := sessionInitData{
		SetlistText:      savedRec.SetlistText,
		SelectedFormatID: savedRec.FormatID,
		Accepted:         len(savedRec.Tracks) > 0,
		Tracks:           initTracks,
	}
	initJSON, _ := json.Marshal(initData)

	w.Header().Set("Content-Type", "text/html")
	templates.LoadedIndexPage(newRec.Session.ID, meta, string(initJSON)).Render(r.Context(), w)
}

// POST /fetch  body: form field "url"
func (s *Server) handleWebFetch(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		http.Error(w, "url required", http.StatusBadRequest)
		return
	}

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, "BOOTLEGGER")
	rec := s.sessions.create(baseDir)

	meta, err := core.FetchMetadata(r.Context(), url)
	if err != nil {
		writeSessionRecord(rec.Session.Dir, rec.Session.ID, url, "", "ERROR", 0, nil, nil)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(errorPanel(err.Error())))
		return
	}

	rec.mu.Lock()
	rec.Metadata = meta
	rec.State = core.StatePreview
	rec.mu.Unlock()

	// Save metadata so this session can be restored later.
	writeSessionRecord(rec.Session.Dir, rec.Session.ID, url, meta.Title, "PREVIEW", 0, nil, savedMetaFromSource(meta))

	w.Header().Set("Content-Type", "text/html")
	templates.MetadataSection(rec.Session.ID, meta).Render(r.Context(), w)
}

// POST /start/{id}  body: form fields "format_id", "setlist"
func (s *Server) handleWebStart(w http.ResponseWriter, r *http.Request) {
	rec, err := s.sessions.get(mux.Vars(r)["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	formatID := r.FormValue("format_id")
	setlistText := r.FormValue("setlist")
	tracks := core.ParseSetlist(setlistText)

	// Apply client-side track selection if provided.
	if sel := r.FormValue("selected_tracks"); sel != "" {
		selected := map[int]bool{}
		for _, s := range strings.Split(sel, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n > 0 {
				selected[n] = true
			}
		}
		for i := range tracks {
			tracks[i].Selected = selected[tracks[i].Number]
		}
	}

	rec.mu.Lock()
	if rec.cancel != nil {
		rec.mu.Unlock()
		http.Error(w, "pipeline already running", http.StatusConflict)
		return
	}
	if len(tracks) > 0 {
		rec.Tracks = tracks
	}

	url := ""
	title := ""
	var savedMeta *session.SavedMetadata
	if rec.Metadata != nil {
		url = rec.Metadata.WebpageURL
		title = rec.Metadata.Title
		savedMeta = savedMetaFromSource(rec.Metadata)
	}

	events := make(chan core.Event, 500)
	ctx, cancel := context.WithCancel(context.Background())
	rec.Events = events
	rec.cancel = cancel
	rec.State = core.StateDownloading
	trackCount := len(tracks)
	sessionID := rec.Session.ID
	sessionDir := rec.Session.Dir
	createdAt := rec.Session.CreatedAt
	rec.mu.Unlock()

	pipeline := core.NewPipeline(rec.Session)
	cfg := core.RunConfig{
		URL:      url,
		FormatID: formatID,
		Tracks:   tracks,
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

		state := "COMMITTED"
		var completedAt *time.Time
		if runErr != nil {
			state = "ERROR"
		} else {
			now := time.Now()
			completedAt = &now
		}

		// Build saved track list for session restoration.
		savedTracks := make([]session.SavedTrack, len(tracks))
		for i, t := range tracks {
			savedTracks[i] = session.SavedTrack{
				Number:   t.Number,
				Title:    t.Title,
				StartSec: int(t.StartTime.Seconds()),
				Selected: t.Selected,
			}
		}

		// Record the basenames of every committed FLAC file so the load
		// handler can open this session directly without re-running the pipeline.
		var splitFiles []string
		if runErr == nil {
			if entries, err := os.ReadDir(filepath.Join(sessionDir, "splits")); err == nil {
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".flac") {
						splitFiles = append(splitFiles, e.Name())
					}
				}
			}
		}

		r := session.SessionRecord{
			ID:          sessionID,
			URL:         url,
			Title:       title,
			State:       state,
			TrackCount:  trackCount,
			CreatedAt:   createdAt,
			CompletedAt: completedAt,
			FormatID:    formatID,
			SetlistText: setlistText,
			Metadata:    savedMeta,
			Tracks:      savedTracks,
			SplitFiles:  splitFiles,
		}
		_ = r.Save(sessionDir)
	}()

	w.Header().Set("Content-Type", "text/html")
	templates.ProgressSection(rec.Session.ID, tracks).Render(r.Context(), w)
}

// writeSessionRecord is a fire-and-forget helper to persist a session.json to disk.
func writeSessionRecord(dir, id, url, title, state string, trackCount int, completedAt *time.Time, meta *session.SavedMetadata) {
	createdAt, err := time.ParseInLocation("20060102-150405", id, time.Local)
	if err != nil {
		createdAt = time.Now()
	}
	r := session.SessionRecord{
		ID:          id,
		URL:         url,
		Title:       title,
		State:       state,
		TrackCount:  trackCount,
		CreatedAt:   createdAt,
		CompletedAt: completedAt,
		Metadata:    meta,
	}
	_ = r.Save(dir)
}

// savedMetaFromSource converts a live source.VideoMetadata to the serialisable session type.
func savedMetaFromSource(meta *source.VideoMetadata) *session.SavedMetadata {
	if meta == nil {
		return nil
	}
	fmts := make([]session.SavedFormat, len(meta.Formats))
	for i, f := range meta.Formats {
		fmts[i] = session.SavedFormat{
			FormatID:   f.FormatID,
			Ext:        f.Ext,
			Acodec:     f.Acodec,
			Abr:        f.Abr,
			Asr:        f.Asr,
			Filesize:   f.Filesize,
			FormatNote: f.FormatNote,
			HasVideo:   f.HasVideo,
		}
	}
	return &session.SavedMetadata{
		Title:      meta.Title,
		Uploader:   meta.Uploader,
		Duration:   meta.Duration,
		Thumbnail:  meta.Thumbnail,
		WebpageURL: meta.WebpageURL,
		Formats:    fmts,
	}
}

// sourceMetaFromSaved reconstructs a source.VideoMetadata from the persisted session type.
func sourceMetaFromSaved(sm *session.SavedMetadata) *source.VideoMetadata {
	if sm == nil {
		return nil
	}
	fmts := make([]source.AudioFormat, len(sm.Formats))
	for i, f := range sm.Formats {
		fmts[i] = source.AudioFormat{
			FormatID:   f.FormatID,
			Ext:        f.Ext,
			Acodec:     f.Acodec,
			Abr:        f.Abr,
			Asr:        f.Asr,
			Filesize:   f.Filesize,
			FormatNote: f.FormatNote,
			HasVideo:   f.HasVideo,
		}
	}
	return &source.VideoMetadata{
		Title:      sm.Title,
		Uploader:   sm.Uploader,
		Duration:   sm.Duration,
		Thumbnail:  sm.Thumbnail,
		WebpageURL: sm.WebpageURL,
		Formats:    fmts,
	}
}

// GET /files/{sessionID}/{filename} — serve a split FLAC file
func (s *Server) handleServeFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]
	filename := vars["filename"]

	// Reject any path traversal attempts.
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.NotFound(w, r)
		return
	}

	home, _ := os.UserHomeDir()
	splitsDir := filepath.Join(home, "BOOTLEGGER", "sessions", sessionID, "splits")
	filePath := filepath.Join(splitsDir, filename)

	// Confirm resolved path stays within splits directory.
	absPath, err := filepath.Abs(filePath)
	if err != nil || !strings.HasPrefix(absPath, splitsDir+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}

// scanSplitTracks reads the splits directory and returns PlayerTrack entries sorted by filename.
func scanSplitTracks(splitsDir string) []templates.PlayerTrack {
	entries, err := os.ReadDir(splitsDir)
	if err != nil {
		return nil
	}
	var tracks []templates.PlayerTrack
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".flac") {
			continue
		}
		var num int
		title := strings.TrimSuffix(name, ".flac")
		// Parse "NN - Title.flac" → number + title
		if idx := strings.Index(name, " - "); idx >= 0 {
			n, err := strconv.Atoi(strings.TrimSpace(name[:idx]))
			if err == nil {
				num = n
			}
			title = strings.TrimSuffix(name[idx+3:], ".flac")
		}
		tracks = append(tracks, templates.PlayerTrack{
			Number:   num,
			Title:    title,
			Filename: name,
		})
	}
	return tracks
}

func errorPanel(msg string) string {
	return `<div class="bg-panel border border-danger p-4 mt-4">
		<div class="text-danger text-xs tracking-widest mb-2">ERROR</div>
		<div class="text-muted text-sm">` + html.EscapeString(msg) + `</div>
		<a href="/" class="inline-block mt-3 text-xs text-muted tracking-widest border border-wire px-3 py-1">BACK</a>
	</div>`
}
