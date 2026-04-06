package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// OutputSettingsTab displays audio configuration settings
type OutputSettingsTab struct {
	width             int
	height            int
	showConfirmation  bool
	willSplit         bool
	trackCount        int
	formatDescription string
}

// NewOutputSettingsTab creates a new output settings tab
func NewOutputSettingsTab() OutputSettingsTab {
	return OutputSettingsTab{
		width:  65,
		height: 20,
	}
}

// SetWidth sets the tab width
func (t *OutputSettingsTab) SetWidth(w int) {
	t.width = w
}

// SetHeight sets the tab height
func (t *OutputSettingsTab) SetHeight(h int) {
	t.height = h
}

// SetShowConfirmation sets whether to show download confirmation
func (t *OutputSettingsTab) SetShowConfirmation(show bool) {
	t.showConfirmation = show
}

// SetDownloadInfo sets download information for the confirmation screen
func (t *OutputSettingsTab) SetDownloadInfo(willSplit bool, trackCount int, formatDesc string) {
	t.willSplit = willSplit
	t.trackCount = trackCount
	t.formatDescription = formatDesc
}

// Update handles messages for this tab
func (t *OutputSettingsTab) Update(msg tea.Msg) tea.Cmd {
	// Output settings tab doesn't need to handle updates
	return nil
}

// View renders the output settings tab content
func (t OutputSettingsTab) View() string {
	var sb strings.Builder

	if t.showConfirmation {
		// Show download confirmation screen
		sb.WriteString(TitleStyle.Render("CONFIRM DOWNLOAD"))
		sb.WriteString("\n\n")

		// Download details
		sb.WriteString(StatusLabelStyle.Render("FORMAT     "))
		sb.WriteString(TextStyle.Render(t.formatDescription))
		sb.WriteString("\n")

		sb.WriteString(StatusLabelStyle.Render("SPLIT      "))
		if t.willSplit {
			sb.WriteString(AccentStyle.Render(fmt.Sprintf("YES (%d TRACKS)", t.trackCount)))
		} else {
			sb.WriteString(TextStyle.Render("NO (SINGLE FILE)"))
		}
		sb.WriteString("\n\n")

		// Output settings
		sb.WriteString(MutedStyle.Render("OUTPUT SETTINGS:"))
		sb.WriteString("\n")
		sb.WriteString(TextStyle.Render("• OUTPUT FORMAT: FLAC"))
		sb.WriteString("\n")
		sb.WriteString(TextStyle.Render("• SAMPLE RATE: 48KHZ"))
		sb.WriteString("\n")
		sb.WriteString(TextStyle.Render("• BIT DEPTH: 16-BIT"))
		sb.WriteString("\n")
		sb.WriteString(TextStyle.Render("• CHANNELS: STEREO"))
		sb.WriteString("\n\n")

		sb.WriteString(AccentStyle.Render("PRESS ENTER TO START DOWNLOAD"))
		sb.WriteString("\n\n")
		sb.WriteString(MutedStyle.Render("[ENTER] START  [ESC] CANCEL"))
	} else {
		// Show settings (non-confirmation mode)
		sb.WriteString(MutedStyle.Render("OUTPUT SETTINGS") + "\n\n")
		sb.WriteString(TextStyle.Render("• OUTPUT FORMAT: FLAC") + "\n")
		sb.WriteString(TextStyle.Render("• SAMPLE RATE: 48KHZ") + "\n")
		sb.WriteString(TextStyle.Render("• BIT DEPTH: 16-BIT") + "\n")
		sb.WriteString(TextStyle.Render("• CHANNELS: STEREO") + "\n")
	}

	return sb.String()
}
