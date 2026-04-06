package ui

import "time"

// SessionState represents the current state of a bootlegger session
type SessionState string

const (
	StateInit            SessionState = "INIT"
	StateBooting         SessionState = "BOOTING"
	StateReady           SessionState = "READY"
	StateFetching        SessionState = "FETCHING"
	StatePreview         SessionState = "PREVIEW"
	StateSetlistSplit    SessionState = "SETLIST_SPLIT"
	StateSetlistInput    SessionState = "SETLIST_INPUT"
	StateSettingsConfirm SessionState = "SETTINGS_CONFIRM"
	StateDownloading     SessionState = "DOWNLOADING"
	StateExtracting      SessionState = "EXTRACTING"
	StateSplitting       SessionState = "SPLITTING"
	StateCommitted       SessionState = "COMMITTED"
	StateError           SessionState = "ERROR"
)

// SourceType identifies the input source
type SourceType string

const (
	SourceNone    SourceType = "NONE"
	SourceYouTube SourceType = "YOUTUBE"
	SourceDropbox SourceType = "DROPBOX"
)

// Track represents a single track to be split
type Track struct {
	Number    int
	Title     string
	StartTime time.Duration
	EndTime   time.Duration
	Status    TrackStatus
	Selected  bool
}

// TrackStatus represents the processing state of a track
type TrackStatus string

const (
	TrackPending    TrackStatus = "PENDING"
	TrackProcessing TrackStatus = "PROCESSING"
	TrackComplete   TrackStatus = "COMPLETE"
	TrackError      TrackStatus = "ERROR"
)

// LogEntry represents a single log message
type LogEntry struct {
	Time    time.Time
	Level   LogLevel
	Message string
}

// LogLevel defines log severity
type LogLevel string

const (
	LogInfo  LogLevel = "INFO"
	LogWarn  LogLevel = "WARN"
	LogError LogLevel = "ERROR"
)

// Progress holds progress information for various operations
type Progress struct {
	Download   float64
	Extraction float64
	Split      float64
}

// LogPanelTab represents the active tab in the log panel
type LogPanelTab int

const (
	LogTabOutput LogPanelTab = iota
	LogTabFiles
)

// PreviewPanelTab represents the active tab in the preview panel
type PreviewPanelTab int

const (
	PreviewTabInfo PreviewPanelTab = iota
	PreviewTabSetlist
	PreviewTabSettings
)
