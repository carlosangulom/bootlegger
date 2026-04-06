package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SetlistTab displays track list input/editing and splitting progress
type SetlistTab struct {
	tracks           []Track
	currentTrack     int
	scrollOffset     int
	inputMode        bool
	showTextInput    bool // True when showing textarea (whether focused or blurred)
	textarea         textarea.Model
	width            int
	height           int
	showSplitPrompt  bool
	splitSelection   int  // 0 = YES, 1 = NO
	selectedTrack    int  // For track selection/deselection
	selectionMode    bool // True when actively selecting tracks with cursor
	showSelection    bool // True when showing selection checkboxes (even if not actively selecting)
	showConfirmation bool // True when showing confirmation prompt after track input
}

// NewSetlistTab creates a new setlist tab
func NewSetlistTab() SetlistTab {
	ta := textarea.New()
	ta.Placeholder = "ENTER SETLIST (FORMAT: HH:MM:SS TITLE)"
	ta.ShowLineNumbers = true
	ta.CharLimit = 0

	// Style the textarea with industrial theme
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(ColorPanel)
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(ColorMuted)
	ta.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().Foreground(ColorTitleBg).Bold(true)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(ColorMuted)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(ColorText)
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(ColorTitleBg)

	ta.BlurredStyle.LineNumber = lipgloss.NewStyle().Foreground(ColorMuted)
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(ColorMuted)
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(ColorMuted)

	// Set initial size - make it scrollable with at least 12 visible lines
	ta.SetHeight(12)
	ta.SetWidth(55)

	ta.Blur()

	return SetlistTab{
		tracks:       []Track{},
		currentTrack: -1,
		scrollOffset: 0,
		textarea:     ta,
		width:        65,
		height:       16,
	}
}

// SetWidth sets the tab width
func (t *SetlistTab) SetWidth(w int) {
	t.width = w
	// Account for borders and padding
	t.textarea.SetWidth(w - 8)
}

// SetHeight sets the tab height
func (t *SetlistTab) SetHeight(h int) {
	t.height = h
	// Account for tabs (3 lines), borders (4 lines), hints (2 lines) = 9 lines
	// Ensure minimum of 12 lines for textarea
	textareaHeight := h - 9
	if textareaHeight < 12 {
		textareaHeight = 12
	}
	t.textarea.SetHeight(textareaHeight)
}

// SetInputMode sets whether the tab is in input mode
func (t *SetlistTab) SetInputMode(mode bool) {
	t.inputMode = mode
	t.showTextInput = mode // Show textarea when in input mode
	if mode {
		t.textarea.Focus()
	} else {
		t.textarea.Blur()
	}
}

// SetTextareaFocus sets whether the textarea is focused (for toggling with CTRL+E)
func (t *SetlistTab) SetTextareaFocus(focused bool) {
	if focused {
		t.textarea.Focus()
	} else {
		t.textarea.Blur()
	}
}

// SetTextInput sets the text input content
func (t *SetlistTab) SetTextInput(text string) {
	// Clear existing content first
	t.textarea.Reset()
	// Insert the new text
	t.textarea.InsertString(text)
}

// GetTextInput returns the current text input
func (t *SetlistTab) GetTextInput() string {
	return t.textarea.Value()
}

// IsFocused returns whether the textarea is focused
func (t *SetlistTab) IsFocused() bool {
	return t.textarea.Focused()
}

// SetTracks sets the track list
func (t *SetlistTab) SetTracks(tracks []Track) {
	t.tracks = tracks
}

// GetTracks returns the track list
func (t *SetlistTab) GetTracks() []Track {
	return t.tracks
}

// SetCurrentTrack sets the currently processing track
func (t *SetlistTab) SetCurrentTrack(n int) {
	t.currentTrack = n
	// Auto-scroll to keep current track visible
	visibleTracks := 10
	if n >= t.scrollOffset+visibleTracks {
		t.scrollOffset = n - visibleTracks + 1
	} else if n < t.scrollOffset {
		t.scrollOffset = n
	}
}

// UpdateTrackStatus updates a track's status
func (t *SetlistTab) UpdateTrackStatus(trackNum int, status TrackStatus) {
	for i := range t.tracks {
		if t.tracks[i].Number == trackNum {
			t.tracks[i].Status = status
			break
		}
	}
}

// SetSplitPrompt sets whether to show the split prompt
func (t *SetlistTab) SetSplitPrompt(show bool) {
	t.showSplitPrompt = show
	if show {
		t.splitSelection = 0 // Default to YES
	}
}

// SelectSplitNext moves split selection down
func (t *SetlistTab) SelectSplitNext() {
	t.splitSelection = (t.splitSelection + 1) % 2
}

// SelectSplitPrev moves split selection up
func (t *SetlistTab) SelectSplitPrev() {
	t.splitSelection = (t.splitSelection + 1) % 2 // Only 2 options, so next = prev
}

// GetSplitSelection returns the current split selection (0 = YES, 1 = NO)
func (t *SetlistTab) GetSplitSelection() int {
	return t.splitSelection
}

// SelectNextTrack moves track selection down
func (t *SetlistTab) SelectNextTrack() {
	if len(t.tracks) > 0 {
		t.selectedTrack = (t.selectedTrack + 1) % len(t.tracks)
		// Auto-scroll to keep selected track visible
		visibleTracks := 10

		// If selected track is below visible window, scroll down
		if t.selectedTrack >= t.scrollOffset+visibleTracks {
			t.scrollOffset = t.selectedTrack - visibleTracks + 1
		}
		// If selected track is above visible window (wrapped around), scroll to top
		if t.selectedTrack < t.scrollOffset {
			t.scrollOffset = t.selectedTrack
		}
	}
}

// SelectPrevTrack moves track selection up
func (t *SetlistTab) SelectPrevTrack() {
	if len(t.tracks) > 0 {
		t.selectedTrack--
		if t.selectedTrack < 0 {
			t.selectedTrack = len(t.tracks) - 1
		}
		// Auto-scroll to keep selected track visible
		visibleTracks := 10

		// If selected track is above visible window, scroll up
		if t.selectedTrack < t.scrollOffset {
			t.scrollOffset = t.selectedTrack
		}
		// If selected track is below visible window (wrapped to bottom), scroll down
		if t.selectedTrack >= t.scrollOffset+visibleTracks {
			t.scrollOffset = t.selectedTrack - visibleTracks + 1
		}
	}
}

// ToggleSelectedTrack toggles the selection state of the currently selected track
func (t *SetlistTab) ToggleSelectedTrack() {
	if t.selectedTrack >= 0 && t.selectedTrack < len(t.tracks) {
		t.tracks[t.selectedTrack].Selected = !t.tracks[t.selectedTrack].Selected
	}
}

// GetSelectedTracks returns only the tracks that are selected
func (t *SetlistTab) GetSelectedTracks() []Track {
	var selected []Track
	for _, track := range t.tracks {
		if track.Selected {
			selected = append(selected, track)
		}
	}
	return selected
}

// SetSelectionMode sets whether to show track selection UI with cursor
func (t *SetlistTab) SetSelectionMode(mode bool) {
	t.selectionMode = mode
	t.showSelection = mode // When in selection mode, also show selections
	if mode {
		t.selectedTrack = 0
		t.showTextInput = false // Hide textarea when showing track selection
		t.textarea.Blur()       // Ensure textarea is blurred so it doesn't capture input
		// Select all tracks by default
		for i := range t.tracks {
			t.tracks[i].Selected = true
		}
	}
	// Note: when exiting selection mode, we don't automatically show the textarea
	// The caller should explicitly call SetInputMode(true) if they want to re-enable editing
}

// SetShowSelection sets whether to show selection checkboxes (without cursor)
func (t *SetlistTab) SetShowSelection(show bool) {
	t.showSelection = show
}

// GetShowSelection returns whether selection checkboxes are visible
func (t *SetlistTab) GetShowSelection() bool {
	return t.showSelection
}

// ScrollUp scrolls the track list up by one line
func (t *SetlistTab) ScrollUp() {
	if t.scrollOffset > 0 {
		t.scrollOffset--
	}
}

// ScrollDown scrolls the track list down by one line
func (t *SetlistTab) ScrollDown() {
	visibleTracks := 10
	maxOffset := len(t.tracks) - visibleTracks
	if maxOffset < 0 {
		maxOffset = 0
	}
	if t.scrollOffset < maxOffset {
		t.scrollOffset++
	}
}

// SetShowConfirmation sets whether to show confirmation prompt
func (t *SetlistTab) SetShowConfirmation(show bool) {
	t.showConfirmation = show
}

// Update handles messages for this tab
func (t *SetlistTab) Update(msg tea.Msg) tea.Cmd {
	// Only forward to textarea if it's focused
	if t.textarea.Focused() {
		var cmd tea.Cmd
		t.textarea, cmd = t.textarea.Update(msg)
		return cmd
	}
	return nil
}

// View renders the setlist tab content
func (t SetlistTab) View() string {
	var sb strings.Builder

	// Show split prompt if requested
	if t.showSplitPrompt {
		sb.WriteString(TitleStyle.Render("SPLIT TRACKS?"))
		sb.WriteString("\n\n")
		sb.WriteString(TextStyle.Render("DO YOU WANT TO SPLIT THIS RECORDING"))
		sb.WriteString("\n")
		sb.WriteString(TextStyle.Render("INTO INDIVIDUAL TRACKS?"))
		sb.WriteString("\n\n\n")

		// Option 0: YES
		var yesLine string
		if t.splitSelection == 0 {
			indicator := AccentStyle.Render("[>]")
			text := AccentStyle.Render("YES - ENTER TRACK LIST")
			yesLine = fmt.Sprintf("%s %s", indicator, text)
		} else {
			indicator := MutedStyle.Render("[ ]")
			text := TextStyle.Render("YES - ENTER TRACK LIST")
			yesLine = fmt.Sprintf("%s %s", indicator, text)
		}
		sb.WriteString(yesLine)
		sb.WriteString("\n")

		// Option 1: NO
		var noLine string
		if t.splitSelection == 1 {
			indicator := AccentStyle.Render("[>]")
			text := AccentStyle.Render("NO - DOWNLOAD AS SINGLE FILE")
			noLine = fmt.Sprintf("%s %s", indicator, text)
		} else {
			indicator := MutedStyle.Render("[ ]")
			text := TextStyle.Render("NO - DOWNLOAD AS SINGLE FILE")
			noLine = fmt.Sprintf("%s %s", indicator, text)
		}
		sb.WriteString(noLine)
		sb.WriteString("\n\n\n\n\n\n")

		sb.WriteString(MutedStyle.Render("[UP/DOWN] SELECT  [ENTER] CONFIRM  [ESC] CANCEL"))
		return sb.String()
	}

	if t.showTextInput {
		// Render the textarea component
		sb.WriteString(t.textarea.View())
		sb.WriteString("\n\n")

		// Show hints based on focus state
		if t.textarea.Focused() {
			sb.WriteString(MutedStyle.Render("[CTRL+V] PASTE  [CTRL+E] DISABLE INPUT  [ESC] CANCEL"))
		} else {
			sb.WriteString(MutedStyle.Render("[CTRL+E] ENABLE INPUT  [ENTER] CONFIRM  [ESC] CANCEL"))
		}
	} else if t.showConfirmation {
		// Show confirmation prompt after track selection
		sb.WriteString(TitleStyle.Render("CONFIRM TRACK SELECTION"))
		sb.WriteString("\n\n")

		selectedCount := len(t.GetSelectedTracks())
		totalCount := len(t.tracks)
		sb.WriteString(TextStyle.Render(fmt.Sprintf("%d OF %d TRACKS SELECTED", selectedCount, totalCount)))
		sb.WriteString("\n\n")
		sb.WriteString(AccentStyle.Render("PRESS ENTER TO PROCEED TO SETTINGS"))
		sb.WriteString("\n\n")
		sb.WriteString(MutedStyle.Render("[ENTER] CONFIRM  [ESC] BACK"))
	} else if len(t.tracks) == 0 {
		sb.WriteString(MutedStyle.Render("ENTER TRACK MARKERS"))
	} else {
		// Fixed visible tracks: show 10 tracks at a time
		visibleTracks := 10

		endIdx := t.scrollOffset + visibleTracks
		if endIdx > len(t.tracks) {
			endIdx = len(t.tracks)
		}

		// Build track list
		var trackLines []string
		for i := t.scrollOffset; i < endIdx; i++ {
			track := t.tracks[i]

			// Determine indicator and style based on mode
			var indicator string
			var style lipgloss.Style

			if t.showSelection {
				// Show selection checkboxes (with or without cursor)
				if t.selectionMode && i == t.selectedTrack {
					// Active selection mode with cursor
					if track.Selected {
						indicator = "[X]"
						style = AccentStyle
					} else {
						indicator = "[ ]"
						style = AccentStyle
					}
				} else {
					// Just showing selections (no cursor)
					if track.Selected {
						indicator = "[X]"
						style = TrackStyle
					} else {
						indicator = "[ ]"
						style = MutedStyle
					}
				}
			} else {
				// Status mode: show processing status
				switch track.Status {
				case TrackPending:
					indicator = "[ ]"
					style = TrackStyle
				case TrackProcessing:
					indicator = "[>]"
					style = TrackActiveStyle
				case TrackComplete:
					indicator = "[+]"
					style = TrackCompleteStyle
				case TrackError:
					indicator = "[!]"
					style = ErrorStyle
				default:
					indicator = "[ ]"
					style = TrackStyle
				}
			}

			// Track number
			numStr := TrackNumberStyle.Render(fmt.Sprintf("%02d", track.Number))

			// Track title (truncated if needed)
			maxTitleLen := t.width - 20
			title := track.Title
			if len(title) > maxTitleLen && maxTitleLen > 3 {
				title = title[:maxTitleLen-3] + "..."
			}

			// Time
			timeStr := MutedStyle.Render(formatDuration(track.StartTime))

			line := fmt.Sprintf("%s %s %s %s",
				style.Render(indicator),
				numStr,
				style.Render(title),
				timeStr,
			)

			trackLines = append(trackLines, line)
		}

		// Render track lines
		sb.WriteString(strings.Join(trackLines, "\n"))

		// Add scroll indicator on new line below tracks if needed
		if len(t.tracks) > visibleTracks {
			sb.WriteString("\n")
			scrollInfo := fmt.Sprintf("[%d-%d/%d]", t.scrollOffset+1, endIdx, len(t.tracks))
			sb.WriteString(MutedStyle.Render(scrollInfo))
		}

		// Show hints at bottom based on mode
		if t.selectionMode {
			sb.WriteString("\n\n\n")
			sb.WriteString(MutedStyle.Render("[UP/DOWN] NAVIGATE  [SPACE] TOGGLE  [ENTER] CONFIRM"))
		}
	}

	return sb.String()
}

// formatDuration formats a duration as MM:SS
func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
