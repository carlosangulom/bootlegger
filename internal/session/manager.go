package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FilterState controls which session states are shown.
type FilterState int

const (
	FilterStateAll FilterState = iota
	FilterStateComplete
	FilterStateError
	FilterStatePartial
)

// FilterStateLabel returns a display label for a FilterState.
func (f FilterState) Label() string {
	switch f {
	case FilterStateComplete:
		return "COMPLETE"
	case FilterStateError:
		return "ERROR"
	case FilterStatePartial:
		return "PARTIAL"
	default:
		return "ALL"
	}
}

// FilterDate controls the date range of sessions shown.
type FilterDate int

const (
	FilterDateAll FilterDate = iota
	FilterDateLast7Days
	FilterDateLast30Days
)

// Label returns a display label for a FilterDate.
func (f FilterDate) Label() string {
	switch f {
	case FilterDateLast7Days:
		return "LAST 7D"
	case FilterDateLast30Days:
		return "LAST 30D"
	default:
		return "ALL"
	}
}

// Filter specifies criteria for FilterSessions.
type Filter struct {
	Text  string
	State FilterState
	Date  FilterDate
}

// ScanSessions reads all session directories under baseDir/sessions/ and
// returns the parsed records sorted by creation time descending (newest first).
// Directories without a valid session.json are silently skipped.
func ScanSessions(baseDir string) ([]SessionRecord, error) {
	sessionsDir := filepath.Join(baseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var records []SessionRecord
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(sessionsDir, entry.Name())
		r, err := Load(dir)
		if err != nil {
			continue
		}
		records = append(records, *r)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})

	return records, nil
}

// FilterSessions returns the subset of records that match f.
func FilterSessions(records []SessionRecord, f Filter) []SessionRecord {
	now := time.Now()
	var result []SessionRecord

	for _, r := range records {
		if f.Text != "" && !strings.Contains(strings.ToLower(r.Title), strings.ToLower(f.Text)) {
			continue
		}

		switch f.State {
		case FilterStateComplete:
			if r.State != "COMMITTED" {
				continue
			}
		case FilterStateError:
			if r.State != "ERROR" {
				continue
			}
		case FilterStatePartial:
			if !IsPartial(r.State) {
				continue
			}
		}

		switch f.Date {
		case FilterDateLast7Days:
			if now.Sub(r.CreatedAt) > 7*24*time.Hour {
				continue
			}
		case FilterDateLast30Days:
			if now.Sub(r.CreatedAt) > 30*24*time.Hour {
				continue
			}
		}

		result = append(result, r)
	}

	return result
}

// DeleteSession removes the session directory and all its contents from disk.
func DeleteSession(dir string) error {
	return os.RemoveAll(dir)
}
