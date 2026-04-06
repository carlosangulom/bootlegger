package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Boot animation sequence
var bootSequence = []string{
	"BOOTLEGGER BL-01 v0.1",
	"INITIALIZING AUDIO ENGINE...",
	"HEADS CLEAN",
	"TAPES LOADED",
	"CHECKING SOURCES...",
	"SYNTHESIZING METADATA...",
	"READY",
}

// BootModel handles the boot animation
type BootModel struct {
	currentStep int
	done        bool
	width       int
}

// NewBootModel creates a new boot animation model
func NewBootModel() BootModel {
	return BootModel{
		currentStep: 0,
		done:        false,
		width:       60,
	}
}

// Init implements tea.Model
func (m BootModel) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return BootStepMsg{}
	})
}

// Update implements tea.Model
func (m BootModel) Update(msg tea.Msg) (BootModel, tea.Cmd) {
	switch msg.(type) {
	case BootStepMsg:
		if m.currentStep < len(bootSequence)-1 {
			m.currentStep++
			return m, tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
				return BootStepMsg{}
			})
		}
		m.done = true
		return m, func() tea.Msg {
			time.Sleep(time.Millisecond * 500)
			return BootCompleteMsg{}
		}
	}
	return m, nil
}

// View implements tea.Model
func (m BootModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")

	// ASCII art header
	header := `
    ____  ____  ____  ________    ________________  __________
   / __ )/ __ \/ __ \/_  __/ /   / ____/ ____/ __ \/ ____/ __ \
  / __  / / / / / / / / / / /   / __/ / / __/ / __/ __/ / /_/ /
 / /_/ / /_/ / /_/ / / / / /___/ /___/ /_/ / /_/ / /___/ _, _/
/_____/\____/\____/ /_/ /_____/_____/\____/\____/_____/_/ |_|

`
	sb.WriteString(BootTitleStyle.Render(header))
	sb.WriteString("\n\n")

	// Boot sequence lines
	for i := 0; i <= m.currentStep && i < len(bootSequence); i++ {
		line := bootSequence[i]
		if i == len(bootSequence)-1 {
			// READY line
			sb.WriteString(BootReadyStyle.Render(line))
		} else if i == 0 {
			// Title line
			sb.WriteString(BootTitleStyle.Render(line))
		} else {
			sb.WriteString(BootTextStyle.Render(line))
		}
		sb.WriteString("\n")
	}

	// Blinking cursor effect
	if !m.done && m.currentStep < len(bootSequence)-1 {
		sb.WriteString(AccentStyle.Render("_"))
	}

	return sb.String()
}

// IsDone returns whether boot animation is complete
func (m BootModel) IsDone() bool {
	return m.done
}

// SetWidth sets the width for rendering
func (m *BootModel) SetWidth(w int) {
	m.width = w
}
