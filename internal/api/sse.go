package api

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"bootlegger/internal/api/templates"
	"bootlegger/internal/core"
	gotempl "github.com/a-h/templ"
	"github.com/gorilla/mux"
)

// GET /events/{id}
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	rec, err := s.sessions.get(mux.Vars(r)["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	// Wait for pipeline to start
	var events chan core.Event
	deadline := time.Now().Add(30 * time.Second)
	for {
		rec.mu.RLock()
		events = rec.Events
		rec.mu.RUnlock()
		if events != nil {
			break
		}
		if time.Now().After(deadline) {
			writeSSE(w, "done", "<span class='text-danger'>TIMEOUT: pipeline not started</span>")
			flusher.Flush()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	currentStage := core.Stage("")

	for {
		select {
		case event, ok := <-events:
			if !ok {
				rec.mu.RLock()
				state := rec.State
				sessionID := rec.Session.ID
				splitsDir := rec.Session.SplitsDir
				tracks := make([]core.Track, len(rec.Tracks))
				copy(tracks, rec.Tracks)
				rec.mu.RUnlock()
				isErr := state == core.StateError
				msg := string(state)
				if !isErr {
					msg = "ALL TRACKS COMPLETE"
				}
				// Replace the live track list with inline players BEFORE sending
				// "done" — sse-close fires on "done" and would drop anything after it.
				if !isErr {
					playerTracks := scanSplitTracks(splitsDir)
					writeSSEComponent(w, r.Context(), "tracks", templates.TrackPlayerContent(sessionID, tracks, playerTracks))
				}
				writeSSEComponent(w, r.Context(), "done", templates.DoneContent(isErr, msg))
				flusher.Flush()
				return
			}
			handleEvent(w, r, rec, event, &currentStage)
			flusher.Flush()

		case <-keepalive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func handleEvent(w http.ResponseWriter, r *http.Request, rec *SessionRecord, event core.Event, currentStage *core.Stage) {
	switch event.Kind {
	case core.KindProgress:
		if event.Stage != *currentStage {
			*currentStage = event.Stage
			writeSSE(w, "status", fmt.Sprintf(
				`<span class="text-accent text-xs tracking-widest">%s</span>`,
				strings.ToUpper(string(event.Stage)),
			))
		}
		sseEvent := "progress-" + string(event.Stage)
		label := strings.ToUpper(string(event.Stage))
		writeSSEComponent(w, r.Context(), sseEvent, templates.ProgressBarContent(label, event.Percent))

	case core.KindLog:
		isErr := event.Error != nil
		msg := event.Message
		if isErr {
			msg = event.Error.Error()
		}
		writeSSEComponent(w, r.Context(), "log", templates.LogEntryContent(string(event.Stage), msg, isErr))

	case core.KindTrack:
		isProcessing := event.Message == "PROCESSING"
		isErr := event.Message == "ERROR"
		rec.mu.Lock()
		for i := range rec.Tracks {
			if rec.Tracks[i].Number == event.Track {
				switch {
				case isErr:
					rec.Tracks[i].Status = core.TrackError
				case isProcessing:
					rec.Tracks[i].Status = core.TrackProcessing
				default:
					rec.Tracks[i].Status = core.TrackComplete
				}
				break
			}
		}
		tracks := make([]core.Track, len(rec.Tracks))
		copy(tracks, rec.Tracks)
		rec.mu.Unlock()
		writeSSEComponent(w, r.Context(), "tracks", templates.TrackListContent(tracks))

	case core.KindError:
		msg := "unknown error"
		if event.Error != nil {
			msg = html.EscapeString(event.Error.Error())
		}
		writeSSEComponent(w, r.Context(), "done", templates.DoneContent(true, msg))
	}
}

func writeSSEComponent(w http.ResponseWriter, ctx context.Context, event string, component gotempl.Component) {
	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		return
	}
	writeSSE(w, event, buf.String())
}

func writeSSE(w http.ResponseWriter, event, data string) {
	fmt.Fprintf(w, "event: %s\n", event)
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
}
