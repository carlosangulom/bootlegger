package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LeftPanel is the container for INFO, SETLIST, and OUTPUT tabs
type LeftPanel struct {
	activeTab PreviewPanelTab
	tabs      []string
	width     int
	height    int

	// Tab components
	infoTab            InfoTab
	setlistTab         SetlistTab
	outputSettingsTab  OutputSettingsTab
}

// NewLeftPanel creates a new left panel
func NewLeftPanel() LeftPanel {
	return LeftPanel{
		activeTab:          PreviewTabInfo,
		tabs:               []string{"INFO", "SETLIST", "OUTPUT"},
		width:              65,
		height:             12,
		infoTab:            NewInfoTab(),
		setlistTab:         NewSetlistTab(),
		outputSettingsTab:  NewOutputSettingsTab(),
	}
}

// SetWidth sets the panel width
func (p *LeftPanel) SetWidth(w int) {
	p.width = w
	p.infoTab.SetWidth(w)
	p.setlistTab.SetWidth(w)
	p.outputSettingsTab.SetWidth(w)
}

// SetHeight sets the panel height
func (p *LeftPanel) SetHeight(h int) {
	p.height = h
	p.infoTab.SetHeight(h)
	p.setlistTab.SetHeight(h)
	p.outputSettingsTab.SetHeight(h)
}

// SetActiveTab sets the active tab
func (p *LeftPanel) SetActiveTab(tab PreviewPanelTab) {
	p.activeTab = tab
}

// GetActiveTab returns the active tab
func (p LeftPanel) GetActiveTab() PreviewPanelTab {
	return p.activeTab
}

// NextTab switches to the next tab
func (p *LeftPanel) NextTab() {
	p.activeTab = (p.activeTab + 1) % 3
}

// PrevTab switches to the previous tab
func (p *LeftPanel) PrevTab() {
	p.activeTab = (p.activeTab + 2) % 3
}

// InfoTab returns a pointer to the info tab
func (p *LeftPanel) InfoTab() *InfoTab {
	return &p.infoTab
}

// SetlistTab returns a pointer to the setlist tab
func (p *LeftPanel) SetlistTab() *SetlistTab {
	return &p.setlistTab
}

// OutputSettingsTab returns a pointer to the output settings tab
func (p *LeftPanel) OutputSettingsTab() *OutputSettingsTab {
	return &p.outputSettingsTab
}

// renderTabs renders the tab bar
func (p LeftPanel) renderTabs() string {
	var tabs []string

	for i, tabName := range p.tabs {
		var style lipgloss.Style
		if PreviewPanelTab(i) == p.activeTab {
			style = ActiveTabStyle
		} else {
			style = InactiveTabStyle
		}
		tabs = append(tabs, style.Render(tabName))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// Update handles messages for the active tab
func (p *LeftPanel) Update(msg tea.Msg) tea.Cmd {
	switch p.activeTab {
	case PreviewTabInfo:
		return p.infoTab.Update(msg)
	case PreviewTabSetlist:
		return p.setlistTab.Update(msg)
	case PreviewTabSettings:
		return p.outputSettingsTab.Update(msg)
	}
	return nil
}

// View renders the left panel with tabs
func (p LeftPanel) View() string {
	// Render tabs
	tabBar := p.renderTabs()

	// Render content based on active tab
	var content string
	switch p.activeTab {
	case PreviewTabInfo:
		content = p.infoTab.View()
	case PreviewTabSetlist:
		content = p.setlistTab.View()
	case PreviewTabSettings:
		content = p.outputSettingsTab.View()
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
