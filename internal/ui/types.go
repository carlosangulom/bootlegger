package ui

import "bootlegger/internal/core"

// Domain types — aliased from core so all ui/ files continue to compile unchanged
type SessionState = core.SessionState
type SourceType = core.SourceType
type Track = core.Track
type TrackStatus = core.TrackStatus
type LogLevel = core.LogLevel
type LogEntry = core.LogEntry

// SessionState constants
const (
	StateInit            = core.StateInit
	StateSessionManager  = core.StateSessionManager
	StateBooting         = core.StateBooting
	StateReady           = core.StateReady
	StateFetching        = core.StateFetching
	StatePreview         = core.StatePreview
	StateSetlistSplit    = core.StateSetlistSplit
	StateSetlistInput    = core.StateSetlistInput
	StateSettingsConfirm = core.StateSettingsConfirm
	StateDownloading     = core.StateDownloading
	StateExtracting      = core.StateExtracting
	StateSplitting       = core.StateSplitting
	StateCommitted       = core.StateCommitted
	StateError           = core.StateError
)

// SourceType constants
const (
	SourceNone    = core.SourceNone
	SourceYouTube = core.SourceYouTube
	SourceDropbox = core.SourceDropbox
)

// TrackStatus constants
const (
	TrackPending    = core.TrackPending
	TrackProcessing = core.TrackProcessing
	TrackComplete   = core.TrackComplete
	TrackError      = core.TrackError
)

// LogLevel constants
const (
	LogInfo  = core.LogInfo
	LogWarn  = core.LogWarn
	LogError = core.LogError
)

// Progress holds progress information for various operations
type Progress struct {
	Download   float64
	Extraction float64
	Split      float64
}

// LogPanelTab represents the active tab in the right panel
type LogPanelTab int

const (
	LogTabOutput LogPanelTab = iota
	LogTabFiles
)

// PreviewPanelTab represents the active tab in the left panel
type PreviewPanelTab int

const (
	PreviewTabInfo PreviewPanelTab = iota
	PreviewTabSetlist
	PreviewTabSettings
)
