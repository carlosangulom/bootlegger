package ui

import "github.com/charmbracelet/lipgloss"

// Color palette - industrial engineering inspired
var (
	ColorBg      = lipgloss.Color("#0d0f11")
	ColorPanel   = lipgloss.Color("#14171a")
	ColorText    = lipgloss.Color("#e6e6e6")
	ColorMuted   = lipgloss.Color("#8a8f98")
	ColorAccent  = lipgloss.Color("#ffb000")
	ColorError   = lipgloss.Color("#ff4d4d")
	ColorGreen   = lipgloss.Color("#b6f6c8")
	ColorCyan    = lipgloss.Color("#defbe6")
	ColorBorder  = lipgloss.Color("#3a3f47")
	ColorTitleBg = lipgloss.Color("#24a148")
	ColorBlack   = lipgloss.Color("#121619")
)

// Layout constants
const (
	// TabContentHeight is the fixed height for all tab content areas
	TabContentHeight = 15
)

// Panel styles
var (
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorTitleBg).
			Bold(true).
			Padding(0, 1)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	TextStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorTitleBg).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)
)

// Status bar styles
var (
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Padding(0, 1)

	StatusLabelStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Bold(true)

	StatusValueStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)
)

// Track list styles
var (
	TrackStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	TrackActiveStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	TrackCompleteStyle = lipgloss.NewStyle().
				Foreground(ColorGreen)

	TrackNumberStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Width(4)
)

// Progress bar colors
var (
	ProgressFull  = ColorGreen
	ProgressEmpty = ColorBorder
)

// Log styles
var (
	LogInfoStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	LogWarnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffcc00"))

	LogErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	LogTimestampStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

// Boot animation style
var (
	BootTitleStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	BootTextStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	BootReadyStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)
)

// Tab styles
var (
	ActiveTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	InactiveTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	ActiveTabStyle = lipgloss.NewStyle().
			Border(ActiveTabBorder, true).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			Foreground(ColorGreen).
			Background(ColorTitleBg).
			Bold(true)

	InactiveTabStyle = lipgloss.NewStyle().
				Border(InactiveTabBorder, true).
				BorderForeground(ColorBorder).
				Padding(0, 1).
				Foreground(ColorMuted)

	TabContentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)
)

// RenderPanelTitle renders a panel title with full-width background
func RenderPanelTitle(title string, width int) string {
	style := lipgloss.NewStyle().
		Foreground(ColorGreen).
		Background(ColorTitleBg).
		Bold(true).
		Padding(0, 1).
		Align(lipgloss.Left)
	return style.Render(title)
}
