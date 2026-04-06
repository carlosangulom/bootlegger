package ui

import "time"

// TickMsg is sent periodically to update animations and progress
type TickMsg time.Time

// BootStepMsg advances the boot animation
type BootStepMsg struct{}

// BootCompleteMsg signals boot animation is done
type BootCompleteMsg struct{}

// URLSubmitMsg is sent when user submits a URL
type URLSubmitMsg struct {
	URL string
}

// ClipboardPasteMsg contains text pasted from clipboard
type ClipboardPasteMsg struct {
	Text  string
	Error string
}

// MetadataFetchMsg initiates metadata fetching
type MetadataFetchMsg struct {
	URL string
}

// MetadataResultMsg contains fetched metadata
type MetadataResultMsg struct {
	URL      string
	Metadata interface{} // *source.VideoMetadata
	Error    error
}

// DownloadStartMsg initiates a download
type DownloadStartMsg struct {
	URL      string
	FormatID string
}

// DownloadProgressMsg reports download progress
type DownloadProgressMsg struct {
	Percent float64
	Status  string
}

// DownloadCompleteMsg signals download completion
type DownloadCompleteMsg struct {
	FilePath string
	Title    string
	Error    error
}

// ExtractionStartMsg initiates audio extraction
type ExtractionStartMsg struct {
	InputPath string
}

// ExtractionProgressMsg reports extraction progress
type ExtractionProgressMsg struct {
	Percent float64
	Status  string
}

// ExtractionCompleteMsg signals extraction completion
type ExtractionCompleteMsg struct {
	OutputPath string
	Duration   time.Duration
	Error      error
}

// ProgressMsg updates progress for an operation
type ProgressMsg struct {
	Type  ProgressType
	Value float64
}

type ProgressType string

const (
	ProgressDownload   ProgressType = "download"
	ProgressExtraction ProgressType = "extraction"
	ProgressSplit      ProgressType = "split"
)

// TrackUpdateMsg updates a track's status
type TrackUpdateMsg struct {
	TrackNumber int
	Status      TrackStatus
}

// LogMsg adds a new log entry
type LogMsg struct {
	Level   LogLevel
	Message string
}

// StateChangeMsg changes the session state
type StateChangeMsg struct {
	State SessionState
}

// SetTracksMsg sets the track list
type SetTracksMsg struct {
	Tracks []Track
}

// ErrorMsg reports an error
type ErrorMsg struct {
	Err error
}

// WindowSizeMsg reports terminal size changes
type WindowSizeMsg struct {
	Width  int
	Height int
}

// LEDBlinkMsg triggers LED blink toggle
type LEDBlinkMsg struct{}

// TabSwitchMsg switches the active tab in the log panel
type TabSwitchMsg struct {
	Tab LogPanelTab
}
