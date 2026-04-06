package ui

import (
	"fmt"
	"strings"
	"time"

	"bootlegger/internal/source"

	tea "github.com/charmbracelet/bubbletea"
)

// InfoTab displays URL metadata and format selection
type InfoTab struct {
	metadata        *source.VideoMetadata
	formats         []source.AudioFormat
	selectedFormat  int
	width           int
	height          int
	loading         bool
	error           string
	showSplitPrompt bool
	splitSelection  int // 0 = YES, 1 = NO
}

// NewInfoTab creates a new info tab
func NewInfoTab() InfoTab {
	return InfoTab{
		selectedFormat: 0,
		width:          65,
		height:         20,
		loading:        false,
	}
}

// SetWidth sets the tab width
func (t *InfoTab) SetWidth(w int) {
	t.width = w
}

// SetHeight sets the tab height
func (t *InfoTab) SetHeight(h int) {
	t.height = h
}

// SetMetadata sets the video metadata to display
func (t *InfoTab) SetMetadata(meta *source.VideoMetadata) {
	t.metadata = meta
	t.loading = false
	t.error = ""

	// Filter to audio-only formats, limit to top options
	t.formats = filterAudioFormats(meta.Formats, 8)
	t.selectedFormat = 0
}

// SetLoading sets the loading state
func (t *InfoTab) SetLoading(loading bool) {
	t.loading = loading
	if loading {
		t.metadata = nil
		t.error = ""
	}
}

// SetError sets an error message
func (t *InfoTab) SetError(err string) {
	t.error = err
	t.loading = false
}

// Clear resets the tab
func (t *InfoTab) Clear() {
	t.metadata = nil
	t.formats = nil
	t.selectedFormat = 0
	t.loading = false
	t.error = ""
	t.showSplitPrompt = false
}

// SetSplitPrompt sets whether to show the split prompt
func (t *InfoTab) SetSplitPrompt(show bool) {
	t.showSplitPrompt = show
	if show {
		t.splitSelection = 0 // Default to YES
	}
}

// SelectSplitNext moves split selection down
func (t *InfoTab) SelectSplitNext() {
	t.splitSelection = (t.splitSelection + 1) % 2
}

// SelectSplitPrev moves split selection up
func (t *InfoTab) SelectSplitPrev() {
	t.splitSelection = (t.splitSelection + 1) % 2 // Only 2 options, so next = prev
}

// GetSplitSelection returns the current split selection (0 = YES, 1 = NO)
func (t *InfoTab) GetSplitSelection() int {
	return t.splitSelection
}

// SelectNext moves selection down
func (t *InfoTab) SelectNext() {
	if len(t.formats) > 0 {
		t.selectedFormat = (t.selectedFormat + 1) % len(t.formats)
	}
}

// SelectPrev moves selection up
func (t *InfoTab) SelectPrev() {
	if len(t.formats) > 0 {
		t.selectedFormat--
		if t.selectedFormat < 0 {
			t.selectedFormat = len(t.formats) - 1
		}
	}
}

// GetSelectedFormat returns the currently selected format
func (t *InfoTab) GetSelectedFormat() *source.AudioFormat {
	if len(t.formats) > 0 && t.selectedFormat < len(t.formats) {
		return &t.formats[t.selectedFormat]
	}
	return nil
}

// GetSelectedFormatID returns the format ID of the selected format
func (t *InfoTab) GetSelectedFormatID() string {
	if f := t.GetSelectedFormat(); f != nil {
		return f.FormatID
	}
	return ""
}

// HasMetadata returns true if metadata is loaded
func (t *InfoTab) HasMetadata() bool {
	return t.metadata != nil
}

// IsLoading returns true if loading
func (t *InfoTab) IsLoading() bool {
	return t.loading
}

// Update handles messages for this tab
func (t *InfoTab) Update(msg tea.Msg) tea.Cmd {
	// Info tab doesn't need to handle updates directly
	return nil
}

// View renders the info tab content
func (t InfoTab) View() string {
	var sb strings.Builder

	if t.loading {
		sb.WriteString(AccentStyle.Render("FETCHING METADATA..."))
		return sb.String()
	}

	if t.error != "" {
		sb.WriteString(ErrorStyle.Render("ERROR: " + t.error))
		return sb.String()
	}

	if t.metadata == nil {
		sb.WriteString(MutedStyle.Render("NO URL LOADED"))
		return sb.String()
	}

	// Show split prompt if requested
	if t.showSplitPrompt {
		sb.WriteString(TitleStyle.Render("SPLIT TRACKS?"))
		sb.WriteString("\n\n")
		sb.WriteString(TextStyle.Render("DO YOU WANT TO SPLIT THIS RECORDING"))
		sb.WriteString("\n")
		sb.WriteString(TextStyle.Render("INTO INDIVIDUAL TRACKS?"))
		sb.WriteString("\n\n")

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
		sb.WriteString("\n\n")

		sb.WriteString(MutedStyle.Render("[UP/DOWN] SELECT  [ENTER] CONFIRM  [ESC] CANCEL"))
		return sb.String()
	}

	meta := t.metadata

	// Title
	title := meta.Title
	maxTitleLen := t.width - 12
	if len(title) > maxTitleLen && maxTitleLen > 3 {
		title = title[:maxTitleLen-3] + "..."
	}
	sb.WriteString(AccentStyle.Bold(true).Render(title))
	sb.WriteString("\n\n")

	// Channel/Uploader
	channel := meta.Channel
	if channel == "" {
		channel = meta.Uploader
	}
	if channel != "" {
		sb.WriteString(StatusLabelStyle.Render("CHANNEL  "))
		sb.WriteString(TextStyle.Render(channel))
		sb.WriteString("\n")
	}

	// Duration
	if meta.Duration > 0 {
		sb.WriteString(StatusLabelStyle.Render("DURATION "))
		sb.WriteString(TextStyle.Render(formatDurationSeconds(meta.Duration)))
		sb.WriteString("\n")
	}

	// Upload date
	if meta.UploadDate != "" {
		sb.WriteString(StatusLabelStyle.Render("DATE     "))
		sb.WriteString(TextStyle.Render(formatUploadDate(meta.UploadDate)))
		sb.WriteString("\n")
	}

	// View count
	if meta.ViewCount > 0 {
		sb.WriteString(StatusLabelStyle.Render("VIEWS    "))
		sb.WriteString(TextStyle.Render(formatViewCount(meta.ViewCount)))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Format selection
	var indicator string
	indicator = AccentStyle.Render("[*]")
	sb.WriteString(indicator + " " + MutedStyle.Render("SELECT FORMAT"))
	sb.WriteString("\n")

	if len(t.formats) == 0 {
		sb.WriteString(MutedStyle.Render("  NO FORMATS AVAILABLE"))
	} else {
		for i, format := range t.formats {
			var line string
			if i == t.selectedFormat {
				// Selected format
				indicator := AccentStyle.Render("[>]")
				desc := AccentStyle.Render(format.FormatDescription())
				line = fmt.Sprintf("%s %s", indicator, desc)
			} else {
				// Unselected format
				indicator := MutedStyle.Render("[ ]")
				desc := TextStyle.Render(format.FormatDescription())
				line = fmt.Sprintf("%s %s", indicator, desc)
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(MutedStyle.Render("[UP/DOWN] SELECT  [ENTER] CONTINUE  [ESC] CANCEL"))

	return sb.String()
}

// Helper functions

// filterAudioFormats filters and limits audio formats
func filterAudioFormats(formats []source.AudioFormat, limit int) []source.AudioFormat {
	var audioOnly []source.AudioFormat
	var withVideo []source.AudioFormat

	// Separate audio-only and video+audio formats
	for _, f := range formats {
		if f.Acodec != "" || f.Ext != "" {
			if f.HasVideo {
				withVideo = append(withVideo, f)
			} else {
				audioOnly = append(audioOnly, f)
			}
		}
	}

	var result []source.AudioFormat

	// Prefer audio-only, but fall back to video+audio
	if len(audioOnly) > 0 {
		result = audioOnly
	} else if len(withVideo) > 0 {
		result = withVideo
	}

	// Limit results
	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

// formatDurationSeconds formats seconds as HH:MM:SS or MM:SS
func formatDurationSeconds(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// formatUploadDate formats YYYYMMDD as YYYY-MM-DD
func formatUploadDate(date string) string {
	if len(date) == 8 {
		return date[:4] + "-" + date[4:6] + "-" + date[6:]
	}
	return date
}

// formatViewCount formats view count with K/M suffix
func formatViewCount(count int64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}
