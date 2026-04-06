package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// FilesTab displays file status information
type FilesTab struct {
	width  int
	height int
}

// NewFilesTab creates a new files tab
func NewFilesTab() FilesTab {
	return FilesTab{
		width:  65,
		height: 18,
	}
}

// SetWidth sets the tab width
func (t *FilesTab) SetWidth(w int) {
	t.width = w
}

// SetHeight sets the tab height
func (t *FilesTab) SetHeight(h int) {
	t.height = h
}

// Update handles messages for this tab
func (t *FilesTab) Update(msg tea.Msg) tea.Cmd {
	// Files tab doesn't need to handle updates
	return nil
}

// View renders the files tab content
func (t FilesTab) View() string {
	var sb strings.Builder

	sb.WriteString(MutedStyle.Render("FILES") + "\n\n")
	sb.WriteString(TextStyle.Render("• SOURCE: PENDING") + "\n")
	sb.WriteString(TextStyle.Render("• EXTRACTED: PENDING") + "\n")
	sb.WriteString(TextStyle.Render("• SPLITS: PENDING") + "\n")

	return sb.String()
}
