package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SourcePanel displays source input and status
type SourcePanel struct {
	sourceType  SourceType
	url         string
	urlInput    string
	inputMode   bool
	locked      bool
	width       int
	height      int
	cursorBlink bool
}

// NewSourcePanel creates a new source panel
func NewSourcePanel() SourcePanel {
	return SourcePanel{
		sourceType: SourceNone,
		inputMode:  true,
		width:      65,
		height:     10,
	}
}

// SetSource sets the source type and URL (locks the source)
func (p *SourcePanel) SetSource(st SourceType, url string) {
	p.sourceType = st
	p.url = url
	p.locked = url != ""
}

// SetURLInput sets the current URL input text
func (p *SourcePanel) SetURLInput(input string) {
	p.urlInput = input
}

// SetInputMode sets whether input mode is active
func (p *SourcePanel) SetInputMode(mode bool) {
	p.inputMode = mode
}

// SetLocked sets the locked state
func (p *SourcePanel) SetLocked(locked bool) {
	p.locked = locked
}

// IsLocked returns whether the source is locked
func (p *SourcePanel) IsLocked() bool {
	return p.locked
}

// Clear resets the panel
func (p *SourcePanel) Clear() {
	p.sourceType = SourceNone
	p.url = ""
	p.urlInput = ""
	p.locked = false
	p.inputMode = true
}

// SetWidth sets the panel width
func (p *SourcePanel) SetWidth(w int) {
	p.width = w
}

// SetHeight sets the panel height
func (p *SourcePanel) SetHeight(h int) {
	p.height = h
}

// SetProgress is kept for compatibility but does nothing now
// Progress is shown in log panel instead
func (p *SourcePanel) SetProgress(percent float64) {
	// No-op - progress moved to log panel
}

// TickCursor toggles the cursor blink state
func (p *SourcePanel) TickCursor() {
	p.cursorBlink = !p.cursorBlink
}

// View renders the source panel
func (p SourcePanel) View() string {
	var sb strings.Builder

	if p.locked {

		// Show source type
		if p.sourceType != SourceNone {
			sb.WriteString(StatusLabelStyle.Render(string(p.sourceType)))
			sb.WriteString("\n")
		}

		// Show URL (truncated)
		url := p.url
		maxLen := p.width - 6
		if len(url) > maxLen && maxLen > 3 {
			url = url[:maxLen-3] + "..."
		}
		sb.WriteString(TextStyle.Render(url))
		sb.WriteString("\n\n")

		// Show reset hint
		sb.WriteString(MutedStyle.Render("[CTRL+R] RESET"))
	} else {
		// URL input field
		inputLabel := lipgloss.NewStyle().Foreground(ColorTitleBg).Render("> ")
		inputValue := TextStyle.Render(p.urlInput)
		cursor := ""
		if p.inputMode && p.cursorBlink {
			cursor = lipgloss.NewStyle().Foreground(ColorTitleBg).Render("_")
		}
		sb.WriteString(inputLabel + inputValue + cursor)
		sb.WriteString("\n\n\n")

		sb.WriteString(MutedStyle.Render("[CTRL+V] PASTE  [ENTER]  SUBMIT"))
	}

	return p.wrapInPanel(sb.String(), "SOURCE")
}

// wrapInPanel wraps content in a styled panel
func (p SourcePanel) wrapInPanel(content string, titleText string) string {
	// Add title inside the panel
	titleLine := RenderPanelTitle(titleText, p.width) + "\n\n"
	fullContent := titleLine + content

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1).
		Width(80).
		Render(fullContent)

	return panel
}
