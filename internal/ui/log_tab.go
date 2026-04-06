package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxLogEntries = 100

// LogTab displays logs and active operations
type LogTab struct {
	entries      []LogEntry
	progressLine string
	width        int
	height       int
	visible      int
}

// NewLogTab creates a new output tab
func NewLogTab() LogTab {
	return LogTab{
		entries: []LogEntry{},
		width:   65,
		height:  18,
		visible: 10,
	}
}

// SetWidth sets the tab width
func (t *LogTab) SetWidth(w int) {
	t.width = w
}

// SetHeight sets the tab height
func (t *LogTab) SetHeight(h int) {
	t.height = h
	t.visible = h - 7 // Account for tabs and borders
	if t.visible < 1 {
		t.visible = 1
	}
}

// AddEntry adds a new log entry
func (t *LogTab) AddEntry(entry LogEntry) {
	t.entries = append(t.entries, entry)
	// Keep only the last N entries
	if len(t.entries) > maxLogEntries {
		t.entries = t.entries[len(t.entries)-maxLogEntries:]
	}
}

// AddInfo adds an info log entry
func (t *LogTab) AddInfo(msg string) {
	t.AddEntry(LogEntry{
		Level:   LogInfo,
		Message: msg,
	})
}

// AddWarn adds a warning log entry
func (t *LogTab) AddWarn(msg string) {
	t.AddEntry(LogEntry{
		Level:   LogWarn,
		Message: msg,
	})
}

// AddError adds an error log entry
func (t *LogTab) AddError(msg string) {
	t.AddEntry(LogEntry{
		Level:   LogError,
		Message: msg,
	})
}

// SetProgressLine sets a progress line that appears at the top
func (t *LogTab) SetProgressLine(line string) {
	t.progressLine = line
}

// ClearProgressLine clears the progress line
func (t *LogTab) ClearProgressLine() {
	t.progressLine = ""
}

// Clear clears all log entries
func (t *LogTab) Clear() {
	t.entries = []LogEntry{}
}

// Update handles messages for this tab
func (t *LogTab) Update(msg tea.Msg) tea.Cmd {
	// Output tab doesn't need to handle updates directly
	return nil
}

// ViewWithBlink renders the output tab content with blink state
func (t LogTab) ViewWithBlink(blink bool) string {
	var sb strings.Builder

	// Show progress line at top if set
	if t.progressLine != "" {
		// LED indicator with blink
		var led string
		if blink {
			led = AccentStyle.Render("[●]")
		} else {
			led = MutedStyle.Render("[○]")
		}
		sb.WriteString(led + " " + AccentStyle.Render(t.progressLine) + "\n\n")
	}

	// Show logs below progress
	if len(t.entries) == 0 {
		if t.progressLine == "" {
			sb.WriteString(MutedStyle.Render("NO LOG ENTRIES"))
		}
	} else {
		// Show last N entries
		startIdx := len(t.entries) - t.visible
		if startIdx < 0 {
			startIdx = 0
		}

		for i := startIdx; i < len(t.entries); i++ {
			entry := t.entries[i]

			// Level prefix
			var levelStyle lipgloss.Style
			var prefix string
			switch entry.Level {
			case LogInfo:
				levelStyle = LogInfoStyle
				prefix = "INFO"
			case LogWarn:
				levelStyle = LogWarnStyle
				prefix = "WARN"
			case LogError:
				levelStyle = LogErrorStyle
				prefix = "ERR "
			default:
				levelStyle = LogInfoStyle
				prefix = "    "
			}

			// Truncate message if needed
			maxMsgLen := t.width - 12
			msg := entry.Message
			if len(msg) > maxMsgLen && maxMsgLen > 3 {
				msg = msg[:maxMsgLen-3] + "..."
			}

			line := fmt.Sprintf("%s %s",
				MutedStyle.Render("["+prefix+"]"),
				levelStyle.Render(msg),
			)

			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

// View renders the output tab content (default non-blinking)
func (t LogTab) View() string {
	return t.ViewWithBlink(true)
}
