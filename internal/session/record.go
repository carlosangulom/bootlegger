package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SavedFormat is a serialisable mirror of source.AudioFormat.
// Kept in the session package to avoid a circular import.
type SavedFormat struct {
	FormatID   string  `json:"formatId"`
	Ext        string  `json:"ext"`
	Acodec     string  `json:"acodec"`
	Abr        float64 `json:"abr"`
	Asr        int     `json:"asr"`
	Filesize   int64   `json:"filesize,omitempty"`
	FormatNote string  `json:"formatNote,omitempty"`
	HasVideo   bool    `json:"hasVideo,omitempty"`
}

// SavedMetadata holds enough of source.VideoMetadata to reconstruct the
// MetadataSection form when restoring a session.
type SavedMetadata struct {
	Title      string        `json:"title"`
	Uploader   string        `json:"uploader"`
	Duration   float64       `json:"duration"`
	Thumbnail  string        `json:"thumbnail"`
	WebpageURL string        `json:"webpageUrl"`
	Formats    []SavedFormat `json:"formats"`
}

// SavedTrack stores the minimal per-track data needed for session restoration.
type SavedTrack struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	StartSec int    `json:"startSec"`
	Selected bool   `json:"selected"`
}

// SessionRecord holds the persisted metadata for a single bootlegger session.
// It is written to <sessionDir>/session.json at key pipeline milestones.
type SessionRecord struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	State       string     `json:"state"`
	TrackCount  int        `json:"trackCount,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Restoration fields — all omitempty for backward-compatibility with old records.
	FormatID    string         `json:"formatId,omitempty"`
	SetlistText string         `json:"setlistText,omitempty"`
	Metadata    *SavedMetadata `json:"metadata,omitempty"`
	Tracks      []SavedTrack   `json:"tracks,omitempty"`
	SplitFiles  []string       `json:"splitFiles,omitempty"` // basenames of committed FLAC files

	// Dir is the session directory path. Not persisted to JSON.
	Dir string `json:"-"`
}

// Load reads session.json from dir and returns the parsed record.
func Load(dir string) (*SessionRecord, error) {
	path := filepath.Join(dir, "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r SessionRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	r.Dir = dir
	return &r, nil
}

// Save writes the record as session.json inside dir, creating dir if needed.
func (r *SessionRecord) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "session.json"), data, 0o644)
}

// IsPartial reports whether a state string represents an incomplete (non-terminal) session.
func IsPartial(state string) bool {
	return state != "COMMITTED" && state != "ERROR"
}
