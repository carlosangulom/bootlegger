package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar displays session state and status information
type StatusBar struct {
	state        SessionState
	sessionID    string
	commitReady  bool
	sourceLocked bool
	width        int
}

// NewStatusBar creates a new status bar
func NewStatusBar() StatusBar {
	return StatusBar{
		state:       StateInit,
		sessionID:   time.Now().Format("20060102-150405"),
		commitReady: false,
		width:       80,
	}
}

// SetState sets the session state
func (s *StatusBar) SetState(state SessionState) {
	s.state = state
}

// SetSessionID sets the session ID
func (s *StatusBar) SetSessionID(id string) {
	s.sessionID = id
}

// SetCommitReady sets the commit ready flag
func (s *StatusBar) SetCommitReady(ready bool) {
	s.commitReady = ready
}

// SetSourceLocked sets the source locked flag
func (s *StatusBar) SetSourceLocked(locked bool) {
	s.sourceLocked = locked
}

// SetWidth sets the status bar width
func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// stateDisplayLabel returns a short display label for the state
func stateDisplayLabel(state SessionState) string {
	switch state {
	case StateInit:
		return "INIT"
	case StateBooting:
		return "BOOTING"
	case StateReady:
		return "READY"
	case StateFetching:
		return "FETCHING"
	case StatePreview:
		return "PREVIEW"
	case StateSetlistSplit:
		return "SPLIT"
	case StateSetlistInput:
		return "INPUT"
	case StateSettingsConfirm:
		return "CONFIRM"
	case StateDownloading:
		return "DOWNLOAD"
	case StateExtracting:
		return "EXTRACT"
	case StateSplitting:
		return "SPLITTING"
	case StateCommitted:
		return "COMPLETE"
	case StateError:
		return "ERROR"
	default:
		return string(state)
	}
}

// ViewWithBlink renders the status bar with blink state
func (s StatusBar) ViewWithBlink(blink bool) string {
	const (
		statusWidth = 68 // Left section width
		gapWidth    = 8  // Gap between sections
		hintsWidth  = 68 // Right section width for hints
	)

	// === LEFT SECTION: Status info (fixed 65 width) ===

	// State indicator with color based on state
	var stateStyle lipgloss.Style
	switch s.state {
	case StateCommitted:
		stateStyle = SuccessStyle
	case StateError:
		stateStyle = ErrorStyle
	case StateDownloading, StateExtracting, StateSplitting, StateFetching:
		stateStyle = AccentStyle
	default:
		stateStyle = TextStyle
	}

	stateLabel := StatusLabelStyle.Render("STATUS")
	stateValue := stateStyle.Bold(true).Render(stateDisplayLabel(s.state))

	// Session ID
	sessionLabel := StatusLabelStyle.Render("SESSION")
	sessionValue := MutedStyle.Render(s.sessionID)

	// Commit ready indicator
	commitLabel := StatusLabelStyle.Render("COMMIT")
	var commitValue string
	if s.commitReady {
		commitValue = SuccessStyle.Bold(true).Render("READY")
	} else {
		commitValue = MutedStyle.Render("PENDING")
	}

	// Build status section content
	statusParts := []string{
		fmt.Sprintf("%s: %s", stateLabel, stateValue),
		fmt.Sprintf("%s: %s", sessionLabel, sessionValue),
		fmt.Sprintf("%s: %s", commitLabel, commitValue),
	}

	statusContent := strings.Join(statusParts, "  |  ")

	// Fixed width status section
	statusSection := lipgloss.NewStyle().
		Width(statusWidth).
		Foreground(ColorText).
		Render(statusContent)

	// === MIDDLE SECTION: Gap ===
	gapSection := lipgloss.NewStyle().
		Width(gapWidth).
		Render("")

	// === RIGHT SECTION: Hints ===
	hintsContent := "[TAB] NAVIGATE LEFT TABS   [SHIFT+TAB] NAVIGATE RIGHT TABS "
	hintsSection := lipgloss.NewStyle().
		Width(hintsWidth).
		Foreground(ColorMuted).
		Render(hintsContent)

	// Combine all sections horizontally
	bar := lipgloss.JoinHorizontal(lipgloss.Bottom, statusSection, gapSection, hintsSection)

	return bar
}

// View renders the status bar (default non-blinking)
func (s StatusBar) View() string {
	return s.ViewWithBlink(true)
}
