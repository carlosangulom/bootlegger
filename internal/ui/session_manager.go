package ui

import (
	"fmt"
	"strings"

	"bootlegger/internal/session"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const visibleSessionRows = 8

// SessionManagerModel is the TUI component for the startup session list.
type SessionManagerModel struct {
	records     []session.SessionRecord
	filtered    []session.SessionRecord
	cursor      int
	scrollOff   int
	filterText  string
	filterState session.FilterState
	filterDate  session.FilterDate
	filterMode  bool // user is typing a filter string
	confirmDel  bool // waiting for Y/N delete confirmation
	baseDir     string
}

// NewSessionManagerModel creates a new session manager for the given base directory.
func NewSessionManagerModel(baseDir string) SessionManagerModel {
	return SessionManagerModel{
		baseDir:     baseDir,
		filterState: session.FilterStateAll,
		filterDate:  session.FilterDateAll,
	}
}

// SetRecords replaces the session list and re-applies the current filter.
func (m *SessionManagerModel) SetRecords(records []session.SessionRecord) {
	m.records = records
	m.applyFilter()
	m.cursor = 0
	m.scrollOff = 0
}

func (m *SessionManagerModel) applyFilter() {
	m.filtered = session.FilterSessions(m.records, session.Filter{
		Text:  m.filterText,
		State: m.filterState,
		Date:  m.filterDate,
	})
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	// Clamp scroll
	if m.scrollOff > m.cursor {
		m.scrollOff = m.cursor
	}
}

// Update handles key messages for the session manager screen.
func (m SessionManagerModel) Update(msg tea.Msg) (SessionManagerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmDel {
			return m.handleConfirmKey(msg)
		}
		if m.filterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleListKey(msg)
	}
	return m, nil
}

func (m SessionManagerModel) handleConfirmKey(msg tea.KeyMsg) (SessionManagerModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.cursor < len(m.filtered) {
			dir := m.filtered[m.cursor].Dir
			_ = session.DeleteSession(dir)
			m.confirmDel = false
			return m, scanSessionsCmd(m.baseDir)
		}
		m.confirmDel = false
	case "n", "N", "esc":
		m.confirmDel = false
	}
	return m, nil
}

func (m SessionManagerModel) handleFilterKey(msg tea.KeyMsg) (SessionManagerModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterText = ""
		m.applyFilter()
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.applyFilter()
		}
	case "enter":
		m.filterMode = false
	default:
		key := msg.String()
		if len(key) == 1 {
			m.filterText += key
			m.applyFilter()
		}
	}
	return m, nil
}

func (m SessionManagerModel) handleListKey(msg tea.KeyMsg) (SessionManagerModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.scrollOff {
				m.scrollOff = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			if m.cursor >= m.scrollOff+visibleSessionRows {
				m.scrollOff = m.cursor - visibleSessionRows + 1
			}
		}
	case "n", "N":
		return m, func() tea.Msg { return NewSessionMsg{} }
	case "d", "D":
		if len(m.filtered) > 0 {
			m.confirmDel = true
		}
	case "/":
		m.filterMode = true
	case "tab":
		m.filterState = (m.filterState + 1) % 4
		m.applyFilter()
	case "f", "F":
		m.filterDate = (m.filterDate + 1) % 3
		m.applyFilter()
	case "ctrl+c", "q", "Q":
		return m, tea.Quit
	}
	return m, nil
}

// View renders the session manager, centered within the given terminal dimensions.
func (m SessionManagerModel) View(width, height int) string {
	var sb strings.Builder

	// Title
	sb.WriteString(AccentStyle.Bold(true).Render("BOOTLEGGER"))
	sb.WriteString(MutedStyle.Render(" — "))
	sb.WriteString(TextStyle.Bold(true).Render("SESSION MANAGER"))
	sb.WriteString("\n\n")

	// Filter bar
	filterInput := "[ " + m.filterText
	if m.filterMode {
		filterInput += "_"
	}
	filterInput += " ]"
	var filterInputRendered string
	if m.filterMode {
		filterInputRendered = AccentStyle.Render(filterInput)
	} else {
		filterInputRendered = TextStyle.Render(filterInput)
	}

	filterBar := MutedStyle.Render("FILTER:") + " " + filterInputRendered +
		"  " + MutedStyle.Render("STATE:") + " " + AccentStyle.Render(m.filterState.Label()) +
		"  " + MutedStyle.Render("DATE:") + " " + AccentStyle.Render(m.filterDate.Label())
	sb.WriteString(filterBar)
	sb.WriteString("\n\n")

	// Session list
	m.renderList(&sb)

	sb.WriteString("\n\n")

	// Bottom hints / delete confirmation
	if m.confirmDel && m.cursor < len(m.filtered) {
		r := m.filtered[m.cursor]
		displayTitle := r.Title
		if displayTitle == "" {
			displayTitle = r.ID
		}
		if len(displayTitle) > 30 {
			displayTitle = displayTitle[:27] + "..."
		}
		sb.WriteString(ErrorStyle.Render(
			fmt.Sprintf("DELETE \"%s\"? [Y] CONFIRM  [N] CANCEL", displayTitle),
		))
	} else {
		hints := strings.Join([]string{
			MutedStyle.Render("[/] FILTER"),
			MutedStyle.Render("[TAB] STATE"),
			MutedStyle.Render("[F] DATE"),
			AccentStyle.Render("[N] NEW"),
			ErrorStyle.Render("[D] DELETE"),
			MutedStyle.Render("[Q] QUIT"),
		}, "  ")
		sb.WriteString(hints)
	}

	inner := sb.String()

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 3).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m SessionManagerModel) renderList(sb *strings.Builder) {
	if len(m.filtered) == 0 {
		if len(m.records) == 0 {
			sb.WriteString(MutedStyle.Render("  NO SESSIONS FOUND. PRESS [N] TO START A NEW SESSION."))
		} else {
			sb.WriteString(MutedStyle.Render("  NO SESSIONS MATCH FILTER."))
		}
		sb.WriteString("\n")
		for i := 0; i < visibleSessionRows-1; i++ {
			sb.WriteString("\n")
		}
		return
	}

	end := m.scrollOff + visibleSessionRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	visible := m.filtered[m.scrollOff:end]

	for i, r := range visible {
		idx := m.scrollOff + i
		isCursor := idx == m.cursor

		cursor := "  "
		if isCursor {
			cursor = AccentStyle.Render("> ")
		}

		date := r.CreatedAt.Format("2006-01-02")

		displayTitle := r.Title
		if displayTitle == "" {
			displayTitle = r.ID
		}
		if len(displayTitle) > 42 {
			displayTitle = displayTitle[:39] + "..."
		}

		var stateRendered string
		switch r.State {
		case "COMMITTED":
			stateRendered = SuccessStyle.Render("COMPLETE")
		case "ERROR":
			stateRendered = ErrorStyle.Render("ERROR   ")
		default:
			stateRendered = MutedStyle.Render("PARTIAL ")
		}

		var trackStr string
		if r.TrackCount > 0 {
			trackStr = MutedStyle.Render(fmt.Sprintf("%2dtk", r.TrackCount))
		}

		var titleRendered string
		if isCursor {
			titleRendered = TextStyle.Render(fmt.Sprintf("%-44s", displayTitle))
		} else {
			titleRendered = MutedStyle.Render(fmt.Sprintf("%-44s", displayTitle))
		}

		line := cursor + MutedStyle.Render(date) + "  " + titleRendered + "  " + stateRendered + "  " + trackStr
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Pad remaining rows
	shown := end - m.scrollOff
	for i := shown; i < visibleSessionRows; i++ {
		sb.WriteString("\n")
	}

	// Scroll indicator
	if len(m.filtered) > visibleSessionRows {
		sb.WriteString(MutedStyle.Render(fmt.Sprintf("  %d/%d", m.cursor+1, len(m.filtered))))
	}
}

// scanSessionsCmd returns a tea.Cmd that scans sessions from disk.
func scanSessionsCmd(baseDir string) tea.Cmd {
	return func() tea.Msg {
		records, _ := session.ScanSessions(baseDir)
		return SessionsLoadedMsg{Records: records}
	}
}
