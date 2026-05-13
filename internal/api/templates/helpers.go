package templates

//go:generate templ generate

import (
	"encoding/json"
	"fmt"
	"net/url"

	"bootlegger/internal/core"
	"github.com/a-h/templ"
)

// PlayerTrack holds the display data for a single split track in the player.
type PlayerTrack struct {
	Number   int
	Title    string
	Filename string // base filename only, used to construct URL
}

// playerFileURL builds the URL used to serve a split FLAC file.
func playerFileURL(sessionID, filename string) string {
	return "/files/" + sessionID + "/" + url.PathEscape(filename)
}

func btlggrLogo() templ.Component {
	return templ.Raw(`<div class="border border-wire rounded p-3 font-mono text-sm leading-snug select-none inline-block" id="logo-panel">
<div><span class="text-accent" id="lw0"></span><span class="text-accent font-bold">   BTLGGR</span></div>
<div><span class="text-accent" id="lw1"></span><span class="text-muted">   audio archive terminal</span></div>
<div><span class="text-accent" id="lw2"></span></div>
<div><span class="text-accent" id="lw3"></span><span class="text-muted">   VERSION: </span><span class="text-gray-200">0.1.0</span></div>
</div>`)
}

func formatDuration(secs float64) string {
	h := int(secs) / 3600
	m := (int(secs) % 3600) / 60
	s := int(secs) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func logColor(isErr bool) string {
	if isErr {
		return "text-danger"
	}
	return "text-muted"
}

func trackColor(s core.TrackStatus) string {
	switch s {
	case core.TrackComplete:
		return "text-success"
	case core.TrackError:
		return "text-danger"
	case core.TrackProcessing:
		return "text-accent"
	default:
		return "text-muted"
	}
}

func sessionStateClass(state string) string {
	switch state {
	case "COMMITTED":
		return "text-success"
	case "ERROR":
		return "text-danger"
	default:
		return "text-muted"
	}
}

func sessionStateLabel(state string) string {
	switch state {
	case "COMMITTED":
		return "COMPLETE"
	case "ERROR":
		return "ERROR"
	default:
		return "PARTIAL"
	}
}

// audioURL returns the playback URL for a split track, or "" if not found.
func audioURL(sessionID string, playerTracks []PlayerTrack, trackNum int) string {
	for _, pt := range playerTracks {
		if pt.Number == trackNum {
			return playerFileURL(sessionID, pt.Filename)
		}
	}
	return ""
}

// playerTrackIdx returns the 0-based index of trackNum in playerTracks, or -1.
func playerTrackIdx(playerTracks []PlayerTrack, trackNum int) int {
	for i, pt := range playerTracks {
		if pt.Number == trackNum {
			return i
		}
	}
	return -1
}

// playerTracksJSON serialises the playlist to a JSON array read by btlggrPlayer.init().
func playerTracksJSON(sessionID string, playerTracks []PlayerTrack) string {
	type jt struct {
		Num   int    `json:"num"`
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	items := make([]jt, len(playerTracks))
	for i, pt := range playerTracks {
		items[i] = jt{pt.Number, pt.Title, playerFileURL(sessionID, pt.Filename)}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

// trackRowClass returns the non-active CSS classes for a track row in the player list.
// Playable rows get interactive styling; others fall back to the status colour.
func trackRowClass(playerTracks []PlayerTrack, trackNum int, status core.TrackStatus) string {
	if playerTrackIdx(playerTracks, trackNum) >= 0 {
		return "text-success cursor-pointer hover:bg-bg"
	}
	return trackColor(status)
}

func trackStatusIcon(s core.TrackStatus) string {
	switch s {
	case core.TrackComplete:
		return "✓"
	case core.TrackError:
		return "✗"
	case core.TrackProcessing:
		return "►"
	default:
		return "○"
	}
}
