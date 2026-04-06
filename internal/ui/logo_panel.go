package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LogoPanel displays the BOOTLEGGER logo
type LogoPanel struct {
	width     int
	height    int
	waveframe int
}

// NewLogoPanel creates a new logo panel
func NewLogoPanel() LogoPanel {
	return LogoPanel{
		width:     65,
		height:    10,
		waveframe: 0,
	}
}

// SetWidth sets the panel width
func (p *LogoPanel) SetWidth(w int) {
	p.width = w
}

// SetHeight sets the panel height
func (p *LogoPanel) SetHeight(h int) {
	p.height = h
}

// TickWave advances the waveform animation
func (p *LogoPanel) TickWave() {
	p.waveframe = (p.waveframe + 1) % 8
}

// View renders the logo panel
func (p LogoPanel) View() string {
	var sb strings.Builder

	// Waveform animation (2 lines)
	wave1, wave2, wave3, wave4 := p.getWaveform()

	sb.WriteString("\n")

	sb.WriteString(lipgloss.NewStyle().Foreground(ColorTitleBg).Render(wave2))
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorTitleBg).Render("   BTLGGR"))
	sb.WriteString("\n")

	// Line 4: Device with subtitle
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorTitleBg).Render(wave3))
	sb.WriteString(MutedStyle.Render("   audio archive terminal"))
	sb.WriteString("\n")

	sb.WriteString(lipgloss.NewStyle().Foreground(ColorTitleBg).Render(wave1))
	sb.WriteString("\n")

	sb.WriteString(lipgloss.NewStyle().Foreground(ColorTitleBg).Render(wave4))
	sb.WriteString(MutedStyle.Render("   VERSION: "))
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorText).Render("0.1.0"))
	sb.WriteString("\n")

	content := sb.String()

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1).
		Width(50).
		Render(content)

	return panel
}

// getWaveform returns the current waveform animation frame (4 lines)
// Simulates volume meters with gradient using block density characters
func (p LogoPanel) getWaveform() (string, string, string, string) {
	// Characters: ░ (light), ▒ (medium), ▓ (dense)
	frames := [][4]string{
		{
			"   ░░▒▒▓▓▓▓▓▒▒░░░░░░",
			"   ░░░▒▒▓▓▓▓▓▓▒▒░░░░",
			"   ░░░░▒▓▓▓▓▓▓▒▒░░░░",
			"   ░░▒▒▓▓▓▓▓▒░░░░░░░",
		},
		{
			"   ░▒▒▓▓▓▓▓▓▓▒▒░░░░░",
			"   ░░▒▒▓▓▓▓▓▓▓▓▒▒░░░",
			"   ░░░░▒▓▓▓▓▓▓▒░░░░░",
			"   ░░▒▒▓▓▓▓▓▒▒░░░░░░",
		},
		{
			"   ░░▒▒▓▓▓▓▓▓▓▓▒▒░░░",
			"   ░▒▒▓▓▓▓▓▓▓▓▓▒▒░░░",
			"   ░░░░▒▓▓▓▓▓▓▓▒▒░░░",
			"   ░░▒▒▓▓▓▓▓▓▒▒░░░░░",
		},
		{
			"   ░░░▒▒▓▓▓▓▓▓▓▒▒░░░",
			"   ░░▒▒▓▓▓▓▓▓▓▒▒░░░░",
			"   ░░░░▒▓▓▓▓▓▓▒▒░░░░",
			"   ░░▒▒▓▓▓▓▓▒▒░░░░░░",
		},
		{
			"   ░░▒▒▓▓▓▓▓▓▓▓▒▒░░░",
			"   ░░░▒▒▓▓▓▓▓▒▒░░░░░",
			"   ░░░▒▒▓▓▓▓▓▓▒▒░░░░",
			"   ░░▒▒▓▓▓▓▓▒▒░░░░░░",
		},
		{
			"   ░░▒▓▓▓▓▓▓▓▓▓▒▒░░░",
			"   ░░▒▒▓▓▓▓▓▓▒▒░░░░░",
			"   ░░░▒▒▓▓▓▓▓▓▓▒▒░░░",
			"   ░▒▒▓▓▓▓▓▒▒░░░░░░░",
		},
		{
			"   ░░▒▒▓▓▓▓▓▓▒▒░░░░░",
			"   ░▒▒▓▓▓▓▓▓▓▓▒▒░░░░",
			"   ░░░▒▒▓▓▓▓▓▓▓▒▒░░░",
			"   ░░▒▒▓▓▓▓▓▒▒░░░░░░",
		},
		{
			"   ░░░▒▒▓▓▓▓▓▓▓▒▒░░░",
			"   ░░▒▒▓▓▓▓▓▓▓▒▒░░░░",
			"   ░░░▒▒▓▓▓▓▓▓▒▒░░░░",
			"   ░░▒▒▓▓▓▓▓▒▒░░░░░░",
		},
	}

	return frames[p.waveframe][0], frames[p.waveframe][1], frames[p.waveframe][2], frames[p.waveframe][3]
}
