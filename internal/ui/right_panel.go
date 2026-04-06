package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RightPanel is the container for LOG and FILES tabs
type RightPanel struct {
	activeTab LogPanelTab
	tabs      []string
	width     int
	height    int

	// Tab components
	logTab   LogTab
	filesTab FilesTab
}

// NewRightPanel creates a new right panel
func NewRightPanel() RightPanel {
	return RightPanel{
		activeTab: LogTabOutput,
		tabs:      []string{"LOG", "FILES"},
		width:     65,
		height:    18,
		logTab:    NewLogTab(),
		filesTab:  NewFilesTab(),
	}
}

// SetWidth sets the panel width
func (p *RightPanel) SetWidth(w int) {
	p.width = w
	p.logTab.SetWidth(w)
	p.filesTab.SetWidth(w)
}

// SetHeight sets the panel height
func (p *RightPanel) SetHeight(h int) {
	p.height = h
	p.logTab.SetHeight(h)
	p.filesTab.SetHeight(h)
}

// SetActiveTab sets the active tab
func (p *RightPanel) SetActiveTab(tab LogPanelTab) {
	p.activeTab = tab
}

// GetActiveTab returns the active tab
func (p RightPanel) GetActiveTab() LogPanelTab {
	return p.activeTab
}

// NextTab switches to the next tab
func (p *RightPanel) NextTab() {
	p.activeTab = (p.activeTab + 1) % 2
}

// PrevTab switches to the previous tab
func (p *RightPanel) PrevTab() {
	p.activeTab = (p.activeTab + 1) % 2
}

// LogTab returns a pointer to the log tab
func (p *RightPanel) LogTab() *LogTab {
	return &p.logTab
}

// FilesTab returns a pointer to the files tab
func (p *RightPanel) FilesTab() *FilesTab {
	return &p.filesTab
}

// renderTabs renders the tab bar
func (p RightPanel) renderTabs() string {
	var tabs []string

	for i, tabName := range p.tabs {
		var style lipgloss.Style
		if LogPanelTab(i) == p.activeTab {
			style = ActiveTabStyle
		} else {
			style = InactiveTabStyle
		}
		tabs = append(tabs, style.Render(tabName))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// Update handles messages for the active tab
func (p *RightPanel) Update(msg tea.Msg) tea.Cmd {
	switch p.activeTab {
	case LogTabOutput:
		return p.logTab.Update(msg)
	case LogTabFiles:
		return p.filesTab.Update(msg)
	}
	return nil
}

// ViewWithBlink renders the right panel with tabs and blink state
func (p RightPanel) ViewWithBlink(blink bool) string {
	// Render tabs
	tabBar := p.renderTabs()

	// Render content based on active tab
	var content string
	switch p.activeTab {
	case LogTabOutput:
		content = p.logTab.ViewWithBlink(blink)
	case LogTabFiles:
		content = p.filesTab.View()
	default:
		content = MutedStyle.Render("UNKNOWN TAB")
	}

	// Build panel using TabContentStyle with fixed height
	contentStyle := TabContentStyle.
		Width(65).
		Height(14).
		BorderTop(true)

	contentPanel := contentStyle.Render(content)

	// Combine tabs and content
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, contentPanel)
}

// View renders the right panel (default non-blinking)
func (p RightPanel) View() string {
	return p.ViewWithBlink(true)
}
