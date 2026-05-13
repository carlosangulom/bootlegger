package core

import "time"

// SessionState represents the current state of a bootlegger session
type SessionState string

const (
	StateInit            SessionState = "INIT"
	StateSessionManager  SessionState = "SESSION_MANAGER"
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

// LogLevel defines log severity
type LogLevel string

const (
	LogInfo  LogLevel = "INFO"
	LogWarn  LogLevel = "WARN"
	LogError LogLevel = "ERROR"
)

// LogEntry represents a single log message
type LogEntry struct {
	Time    time.Time
	Level   LogLevel
	Message string
}

// Stage identifies the pipeline stage
type Stage string

const (
	StageDownload Stage = "download"
	StageExtract  Stage = "extract"
	StageSplit    Stage = "split"
)

// EventKind classifies a pipeline event
type EventKind string

const (
	KindProgress EventKind = "progress"
	KindLog      EventKind = "log"
	KindTrack    EventKind = "track"
	KindDone     EventKind = "done"
	KindError    EventKind = "error"
)

// Event is emitted by Pipeline.Run on the events channel
type Event struct {
	Kind    EventKind
	Stage   Stage
	Percent float64
	Message string
	Track   int // track number for KindTrack events
	Error   error
}
