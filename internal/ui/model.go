package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bootlegger/internal/source"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the main TUI model
type Model struct {
	// Panels
	boot       BootModel
	source     SourcePanel
	leftPanel  LeftPanel  // Contains INFO, SETLIST, OUTPUT tabs
	rightPanel RightPanel // Contains LOG, FILES tabs
	logo       LogoPanel
	statusBar  StatusBar

	// State
	state     SessionState
	showBoot  bool
	urlInput  string
	inputMode bool

	// Current URL being processed
	currentURL string

	// Session
	sessionID  string
	sessionDir string

	// Download
	downloader   *source.Downloader
	downloadFile string

	// Extraction
	extractedFile     string
	extractedDuration time.Duration

	// Layout
	width  int
	height int

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Quitting
	quitting bool

	// LED blink state
	ledBlink bool
}

// New creates a new Model
func New() Model {
	ctx, cancel := context.WithCancel(context.Background())

	sessionID := time.Now().Format("20060102-150405")
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, "BOOTLEGGER")
	sessionDir := filepath.Join(baseDir, "sessions", sessionID)

	m := Model{
		boot:       NewBootModel(),
		source:     NewSourcePanel(),
		leftPanel:  NewLeftPanel(),
		rightPanel: NewRightPanel(),
		logo:       NewLogoPanel(),
		statusBar:  NewStatusBar(),
		state:      StateBooting,
		showBoot:   true,
		width:      80,
		height:     24,
		sessionID:  sessionID,
		sessionDir: sessionDir,
		ctx:        ctx,
		cancel:     cancel,
	}

	m.statusBar.SetSessionID(sessionID)

	return m
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.boot.Init(),
		tea.EnableMouseCellMotion, // Enable mouse support
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			m.quitting = true
			return m, tea.Quit
		case "q":
			if !m.inputMode && m.state != StatePreview && m.state != StateSetlistSplit && m.state != StateSetlistInput && m.state != StateSettingsConfirm {
				m.cancel()
				m.quitting = true
				return m, tea.Quit
			}
			if m.inputMode {
				m.urlInput += "q"
				m.source.SetURLInput(m.urlInput)
			}
		case "ctrl+r":
			// Reset session - clear everything and start over
			if m.state != StateBooting && m.state != StateReady {
				m.state = StateReady
				m.statusBar.SetState(StateReady)
				m.statusBar.SetSourceLocked(false)
				m.inputMode = true
				m.source.SetInputMode(true)
				m.source.Clear()
				m.leftPanel.InfoTab().Clear()
				m.leftPanel.SetlistTab().SetInputMode(false)
				m.leftPanel.SetlistTab().SetTextInput("")
				m.leftPanel.SetlistTab().SetSplitPrompt(false)
				m.leftPanel.SetlistTab().SetSelectionMode(false)
				m.leftPanel.SetlistTab().SetShowSelection(false)
				m.leftPanel.OutputSettingsTab().SetShowConfirmation(false)
				m.leftPanel.SetActiveTab(PreviewTabInfo)
				m.urlInput = ""
				m.currentURL = ""
				m.downloadFile = ""
				m.extractedFile = ""
				m.rightPanel.LogTab().AddInfo("SESSION RESET")
				m.rightPanel.LogTab().AddInfo("ENTER URL OR CTRL+V TO PASTE")
			}
		case "ctrl+e":
			// Toggle setlist textarea focus
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().showTextInput {
				if m.leftPanel.SetlistTab().IsFocused() {
					m.leftPanel.SetlistTab().SetTextareaFocus(false)
					m.rightPanel.LogTab().AddInfo("TEXTAREA DISABLED - PRESS ENTER TO CONFIRM")
				} else if !m.leftPanel.SetlistTab().selectionMode {
					m.leftPanel.SetlistTab().SetTextareaFocus(true)
					m.rightPanel.LogTab().AddInfo("TEXTAREA ENABLED - CTRL+E TO DISABLE")
				}
			}
		case "ctrl+v":
			// Paste from clipboard
			if m.inputMode {
				cmds = append(cmds, pasteFromClipboard())
			} else if m.state == StateSetlistInput {
				cmds = append(cmds, pasteFromClipboard())
			}
		case " ":
			// Toggle track selection in track selection mode
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().selectionMode {
				m.leftPanel.SetlistTab().ToggleSelectedTrack()
			} else if m.inputMode {
				m.urlInput += " "
				m.source.SetURLInput(m.urlInput)
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			}
		case "enter":
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				// Pass ENTER to textarea for newline when focused
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			} else if m.inputMode && m.urlInput != "" {
				// Fetch metadata instead of downloading immediately
				m.inputMode = false
				m.source.SetInputMode(false)
				url := strings.TrimSpace(m.urlInput)
				m.currentURL = url
				sourceType := detectSourceType(url)
				m.source.SetSource(sourceType, url)
				m.statusBar.SetSourceLocked(true)
				m.rightPanel.LogTab().AddInfo("FETCHING METADATA...")
				m.statusBar.SetState(StateFetching)
				m.state = StateFetching
				m.leftPanel.InfoTab().SetLoading(true)
				cmds = append(cmds, fetchMetadata(m.ctx, url))
				cmds = append(cmds, ledBlinkTick())
			} else if m.state == StatePreview {
				// Automatically switch to setlist tab and show split prompt
				m.state = StateSetlistSplit
				m.statusBar.SetState(StateSetlistSplit)
				m.leftPanel.SetActiveTab(PreviewTabSetlist)
				m.leftPanel.SetlistTab().SetSplitPrompt(true)
				m.rightPanel.LogTab().AddInfo("FORMAT SELECTED")
			} else if m.state == StateSetlistSplit {
				// Confirm split selection
				if m.leftPanel.SetlistTab().GetSplitSelection() == 0 {
					// User wants to split - show track input
					m.state = StateSetlistInput
					m.statusBar.SetState(StateSetlistInput)
					m.leftPanel.SetlistTab().SetInputMode(true)
					m.leftPanel.SetlistTab().SetSplitPrompt(false)
					m.rightPanel.LogTab().AddInfo("ENTER SETLIST")
				} else {
					// User doesn't want to split - go to settings confirmation
					format := m.leftPanel.InfoTab().GetSelectedFormat()
					formatDesc := "UNKNOWN"
					if format != nil {
						formatDesc = format.FormatDescription()
					}
					m.state = StateSettingsConfirm
					m.statusBar.SetState(StateSettingsConfirm)
					m.leftPanel.SetlistTab().SetSplitPrompt(false)
					m.leftPanel.SetActiveTab(PreviewTabSettings)
					m.leftPanel.OutputSettingsTab().SetDownloadInfo(false, 0, formatDesc)
					m.leftPanel.OutputSettingsTab().SetShowConfirmation(true)
					m.rightPanel.LogTab().AddInfo("SWITCHED TO SETTINGS")
				}
			} else if m.state == StateSetlistInput && !m.leftPanel.SetlistTab().IsFocused() && !m.leftPanel.SetlistTab().selectionMode {
				// User finished entering track list (textarea is blurred) - parse and show for selection
				trackText := m.leftPanel.SetlistTab().GetTextInput()
				if trackText != "" {
					m.rightPanel.LogTab().AddInfo("PROCESSING TRACK LIST...")
					tracks, err := parseSetlist(trackText)
					if err != nil || len(tracks) == 0 {
						m.rightPanel.LogTab().AddError("FAILED TO PARSE SETLIST - CHECK FORMAT (HH:MM:SS TITLE)")
						// Re-enable textarea so user can fix it
						m.leftPanel.SetlistTab().SetTextareaFocus(true)
					} else {
						m.leftPanel.SetlistTab().SetTracks(tracks)
						m.leftPanel.SetlistTab().SetSelectionMode(true)
						m.rightPanel.LogTab().AddInfo(fmt.Sprintf("PARSED %d TRACKS", len(tracks)))
					}
				}
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().selectionMode {
				// User confirmed track selection - go directly to settings
				selectedTracks := m.leftPanel.SetlistTab().GetSelectedTracks()
				format := m.leftPanel.InfoTab().GetSelectedFormat()
				formatDesc := "UNKNOWN"
				if format != nil {
					formatDesc = format.FormatDescription()
				}
				// Exit selection mode but keep showing selections (don't re-enable textarea)
				m.leftPanel.SetlistTab().selectionMode = false
				m.leftPanel.SetlistTab().SetShowSelection(true) // Keep showing checkboxes
				m.state = StateSettingsConfirm
				m.statusBar.SetState(StateSettingsConfirm)
				m.leftPanel.SetActiveTab(PreviewTabSettings)
				m.leftPanel.OutputSettingsTab().SetDownloadInfo(true, len(selectedTracks), formatDesc)
				m.leftPanel.OutputSettingsTab().SetShowConfirmation(true)
				m.rightPanel.LogTab().AddInfo("SWITCHED TO SETTINGS")
			} else if m.state == StateSettingsConfirm {
				// User confirmed download - start download
				formatID := m.leftPanel.InfoTab().GetSelectedFormatID()
				m.state = StateDownloading
				m.statusBar.SetState(StateDownloading)
				m.leftPanel.OutputSettingsTab().SetShowConfirmation(false)
				InitDownloadChannels()
				cmds = append(cmds, StartDownloadAsync(m.ctx, m.currentURL, m.sessionDir, formatID))
				cmds = append(cmds, ledBlinkTick())
				m.rightPanel.LogTab().AddInfo("DOWNLOAD STARTED")
			}
		case "up", "k":
			if m.state == StatePreview {
				m.leftPanel.InfoTab().SelectPrev()
			} else if m.state == StateSetlistSplit {
				m.leftPanel.SetlistTab().SelectSplitPrev()
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().selectionMode {
				m.leftPanel.SetlistTab().SelectPrevTrack()
			} else if m.state == StateSettingsConfirm && m.leftPanel.GetActiveTab() == PreviewTabSetlist && m.leftPanel.SetlistTab().GetShowSelection() {
				m.leftPanel.SetlistTab().ScrollUp()
			}
		case "down", "j":
			if m.state == StatePreview {
				m.leftPanel.InfoTab().SelectNext()
			} else if m.state == StateSetlistSplit {
				m.leftPanel.SetlistTab().SelectSplitNext()
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().selectionMode {
				m.leftPanel.SetlistTab().SelectNextTrack()
			} else if m.state == StateSettingsConfirm && m.leftPanel.GetActiveTab() == PreviewTabSetlist && m.leftPanel.SetlistTab().GetShowSelection() {
				m.leftPanel.SetlistTab().ScrollDown()
			}
		case "left":
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			}
		case "right":
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			}
		case "tab":
			// Navigate between left panel tabs (unless textarea is focused)
			if !(m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused()) {
				m.leftPanel.NextTab()
			} else {
				// Let textarea handle tab
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			}
		case "shift+tab":
			// Navigate between right panel tabs (unless textarea is focused)
			if !(m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused()) {
				m.rightPanel.NextTab()
			}
		case "1":
			// Jump to INFO tab in left panel (unless textarea is focused or URL input mode)
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			} else if m.inputMode {
				m.urlInput += "1"
				m.source.SetURLInput(m.urlInput)
			} else {
				m.leftPanel.SetActiveTab(PreviewTabInfo)
			}
		case "2":
			// Jump to SETLIST tab in left panel (unless textarea is focused or URL input mode)
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			} else if m.inputMode {
				m.urlInput += "2"
				m.source.SetURLInput(m.urlInput)
			} else {
				m.leftPanel.SetActiveTab(PreviewTabSetlist)
			}
		case "3":
			// Jump to OUTPUT tab in left panel (unless textarea is focused or URL input mode)
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			} else if m.inputMode {
				m.urlInput += "3"
				m.source.SetURLInput(m.urlInput)
			} else {
				m.leftPanel.SetActiveTab(PreviewTabSettings)
			}
		case "!":
			// SHIFT+1: Jump to OUTPUT tab in right panel
			if !(m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused()) && !m.inputMode {
				m.rightPanel.SetActiveTab(LogTabOutput)
			}
		case "@":
			// SHIFT+2: Jump to FILES tab in right panel
			if !(m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused()) && !m.inputMode {
				m.rightPanel.SetActiveTab(LogTabFiles)
			}
		case "4", "5", "6", "7", "8", "9", "0":
			// Handle other number keys in input modes
			if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			} else if m.inputMode {
				m.urlInput += msg.String()
				m.source.SetURLInput(m.urlInput)
			}
		case "backspace":
			if m.inputMode && len(m.urlInput) > 0 {
				m.urlInput = m.urlInput[:len(m.urlInput)-1]
				m.source.SetURLInput(m.urlInput)
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			}
		case "esc":
			if m.inputMode {
				m.urlInput = ""
				m.source.SetURLInput("")
			} else if m.state == StateSetlistSplit {
				// Cancel split prompt, go back to preview
				m.state = StatePreview
				m.statusBar.SetState(StatePreview)
				m.leftPanel.SetActiveTab(PreviewTabInfo)
				m.leftPanel.SetlistTab().SetSplitPrompt(false)
				m.rightPanel.LogTab().AddInfo("BACK TO FORMAT SELECTION")
			} else if m.state == StatePreview {
				// Cancel preview, go back to input mode
				m.state = StateReady
				m.statusBar.SetState(StateReady)
				m.statusBar.SetSourceLocked(false)
				m.inputMode = true
				m.source.SetInputMode(true)
				m.urlInput = ""
				m.currentURL = ""
				m.leftPanel.InfoTab().Clear()
				m.source.Clear()
				m.rightPanel.LogTab().AddInfo("CANCELLED")
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().selectionMode {
				// Cancel track selection, go back to track input
				m.leftPanel.SetlistTab().SetSelectionMode(false)
				m.leftPanel.SetlistTab().SetShowSelection(false) // Hide selections
				m.leftPanel.SetlistTab().SetInputMode(true)
				m.rightPanel.LogTab().AddInfo("BACK TO SETLIST INPUT")
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				// Cancel track input, go back to split prompt
				m.state = StateSetlistSplit
				m.statusBar.SetState(StateSetlistSplit)
				m.leftPanel.SetlistTab().SetSplitPrompt(true)
				m.leftPanel.SetlistTab().SetInputMode(false)
				m.leftPanel.SetlistTab().SetTextInput("")
				m.rightPanel.LogTab().AddInfo("BACK TO SPLIT SELECTION")
			} else if m.state == StateSettingsConfirm {
				// Cancel settings confirmation, go back to track selection
				m.state = StateSetlistInput
				m.statusBar.SetState(StateSetlistInput)
				m.leftPanel.SetActiveTab(PreviewTabSetlist)
				m.leftPanel.SetlistTab().selectionMode = true
				m.leftPanel.OutputSettingsTab().SetShowConfirmation(false)
				m.rightPanel.LogTab().AddInfo("BACK TO TRACK SELECTION")
			}
		default:
			if m.inputMode {
				// Handle regular character input
				key := msg.String()
				if len(key) == 1 || key == " " {
					m.urlInput += key
					m.source.SetURLInput(m.urlInput)
				}
			} else if m.state == StateSetlistInput && m.leftPanel.SetlistTab().IsFocused() {
				// Route all other keys to textarea when focused
				cmd := m.leftPanel.SetlistTab().Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case BootStepMsg:
		var cmd tea.Cmd
		m.boot, cmd = m.boot.Update(msg)
		cmds = append(cmds, cmd)

	case BootCompleteMsg:
		m.showBoot = false
		m.state = StateReady
		m.statusBar.SetState(StateReady)
		m.inputMode = true
		m.rightPanel.LogTab().AddInfo("SYSTEM READY")
		m.rightPanel.LogTab().AddInfo("ENTER URL OR CTRL+V TO PASTE")
		// Start waveform animation
		cmds = append(cmds, ledBlinkTick())

	case ClipboardPasteMsg:
		if msg.Error != "" {
			m.rightPanel.LogTab().AddError(msg.Error)
		} else if m.inputMode && msg.Text != "" {
			// For URL input, take only the first line
			urlText := msg.Text
			if idx := strings.Index(urlText, "\n"); idx > 0 {
				urlText = urlText[:idx]
			}
			m.urlInput = urlText
			m.source.SetURLInput(m.urlInput)
			m.rightPanel.LogTab().AddInfo("URL PASTED FROM CLIPBOARD")
		} else if m.state == StateSetlistInput && msg.Text != "" {
			// For track input, preserve all lines
			m.leftPanel.SetlistTab().SetTextInput(msg.Text)
			// Auto-enable textarea when pasting
			if !m.leftPanel.SetlistTab().IsFocused() {
				m.leftPanel.SetlistTab().SetInputMode(true)
			}
			m.rightPanel.LogTab().AddInfo(fmt.Sprintf("SETLIST PASTED FROM CLIPBOARD (%d chars)", len(msg.Text)))
		}

	case MetadataResultMsg:
		if msg.Error != nil {
			m.rightPanel.LogTab().AddError(fmt.Sprintf("METADATA FETCH FAILED: %v", msg.Error))
			m.state = StateError
			m.statusBar.SetState(StateError)
			m.statusBar.SetSourceLocked(false)
			m.leftPanel.InfoTab().SetError(msg.Error.Error())
			m.inputMode = true
			m.source.SetInputMode(true)
			m.source.Clear()
			m.urlInput = ""
		} else {
			meta := msg.Metadata.(*source.VideoMetadata)
			m.leftPanel.InfoTab().SetMetadata(meta)
			m.state = StatePreview
			m.statusBar.SetState(StatePreview)
			m.rightPanel.LogTab().AddInfo("METADATA LOADED: " + meta.Title)
		}

	case DownloadProgressMsg:
		// Show progress in log with bar
		m.rightPanel.LogTab().SetProgressLine(formatProgressBar("INGESTING SOURCE", msg.Percent))
		// Continue polling for more progress
		if m.state == StateDownloading {
			cmds = append(cmds, WaitForDownloadProgress())
		}

	case DownloadTickMsg:
		// Poll for more progress
		if m.state == StateDownloading {
			cmds = append(cmds, WaitForDownloadProgress())
		}

	case DownloadCompleteMsg:
		m.rightPanel.LogTab().ClearProgressLine()
		if msg.Error != nil {
			m.rightPanel.LogTab().AddError(fmt.Sprintf("DOWNLOAD FAILED: %v", msg.Error))
			m.state = StateError
			m.statusBar.SetState(StateError)
			m.statusBar.SetSourceLocked(false)
			m.inputMode = true
			m.source.SetInputMode(true)
			m.source.Clear()
			m.urlInput = ""
		} else {
			m.downloadFile = msg.FilePath
			m.rightPanel.LogTab().AddInfo("SOURCE INGESTED: " + msg.Title)
			m.state = StateExtracting
			m.statusBar.SetState(StateExtracting)
			// Start real audio extraction
			InitExtractionChannels()
			cmds = append(cmds, StartExtractionAsync(m.ctx, msg.FilePath, m.sessionDir))
			cmds = append(cmds, ledBlinkTick())
		}

	case ExtractionProgressMsg:
		// Show extraction progress in log with bar
		m.rightPanel.LogTab().SetProgressLine(formatProgressBar("EXTRACTING AUDIO", msg.Percent))
		if msg.Status != "" && msg.Status != "EXTRACTING AUDIO" {
			m.rightPanel.LogTab().AddInfo(msg.Status)
		}
		// Continue polling for more progress
		if m.state == StateExtracting {
			cmds = append(cmds, WaitForExtractionProgress())
		}

	case ExtractionTickMsg:
		// Poll for more extraction progress
		if m.state == StateExtracting {
			cmds = append(cmds, WaitForExtractionProgress())
		}

	case ExtractionCompleteMsg:
		m.rightPanel.LogTab().ClearProgressLine()
		if msg.Error != nil {
			m.rightPanel.LogTab().AddError(fmt.Sprintf("EXTRACTION FAILED: %v", msg.Error))
			m.state = StateError
			m.statusBar.SetState(StateError)
			m.statusBar.SetSourceLocked(false)
			m.inputMode = true
			m.source.SetInputMode(true)
			m.source.Clear()
			m.urlInput = ""
		} else {
			m.extractedFile = msg.OutputPath
			m.extractedDuration = msg.Duration
			m.rightPanel.LogTab().AddInfo("EXTRACTION COMPLETE")
			m.rightPanel.LogTab().AddInfo("OUTPUT: " + filepath.Base(msg.OutputPath))
			m.state = StateSplitting
			m.statusBar.SetState(StateSplitting)
			// Get selected tracks from setlist tab
			tracks := m.leftPanel.SetlistTab().GetSelectedTracks()
			if len(tracks) == 0 {
				// No tracks selected, use all tracks
				tracks = m.leftPanel.SetlistTab().GetTracks()
			}
			m.leftPanel.SetActiveTab(PreviewTabSetlist)
			// Start actual splitting with real tracks
			cmds = append(cmds, StartSplittingAsync(m.ctx, msg.OutputPath, tracks, m.sessionDir, msg.Duration))
			cmds = append(cmds, WaitForTrackUpdate())
			cmds = append(cmds, ledBlinkTick())
		}

	case ProgressMsg:
		switch msg.Type {
		case ProgressSplit:
			// Track splitting progress handled by TrackUpdateMsg
		}

	case TrackUpdateMsg:
		if msg.TrackNumber > 0 {
			m.leftPanel.SetlistTab().UpdateTrackStatus(msg.TrackNumber, msg.Status)
		}
		if msg.Status == TrackProcessing {
			m.leftPanel.SetlistTab().SetCurrentTrack(msg.TrackNumber - 1)
			m.rightPanel.LogTab().AddInfo(fmt.Sprintf("SPLITTING TRACK %02d", msg.TrackNumber))
		} else if msg.Status == TrackComplete {
			m.rightPanel.LogTab().AddInfo(fmt.Sprintf("TRACK %02d COMPLETE", msg.TrackNumber))
		} else if msg.Status == TrackError {
			m.rightPanel.LogTab().AddError(fmt.Sprintf("TRACK %02d FAILED", msg.TrackNumber))
		}
		// Wait for next track update
		if m.state == StateSplitting {
			cmds = append(cmds, WaitForTrackUpdate())
		}

	case SplittingCompleteMsg:
		m.state = StateCommitted
		m.statusBar.SetState(StateCommitted)
		m.statusBar.SetCommitReady(true)
		m.rightPanel.LogTab().AddInfo("ALL TRACKS SPLIT")
		m.rightPanel.LogTab().AddInfo("SESSION COMMITTED")

	case LogMsg:
		m.rightPanel.LogTab().AddEntry(LogEntry{Level: msg.Level, Message: msg.Message})

	case StateChangeMsg:
		m.state = msg.State
		m.statusBar.SetState(msg.State)

	case LEDBlinkMsg:
		// Toggle LED blink state
		m.ledBlink = !m.ledBlink
		// Tick waveform animation
		m.logo.TickWave()
		// Tick cursor animation
		m.source.TickCursor()
		// Continue blinking if in active state or always for waveform
		if m.state == StateDownloading || m.state == StateExtracting ||
			m.state == StateSplitting || m.state == StateFetching || !m.showBoot {
			cmds = append(cmds, ledBlinkTick())
		}

	case tea.MouseMsg:
		// Handle mouse clicks on tabs
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// The tabs for bottom panels are at a fixed Y position
			// Based on the layout: top panels + spacing
			// From empirical testing, tabs appear around Y=10
			// Allow a range to account for different terminal sizes
			bottomTabsY := 10

			// Check if click is on the tab row (give 1 line tolerance)
			if msg.Y >= bottomTabsY && msg.Y <= bottomTabsY+1 {
				availableWidth := m.width - 4
				leftWidth := availableWidth / 2
				rightPanelX := leftWidth + 3 // Left panel width + spacing

				// Left panel tabs (INFO, SETLIST, OUTPUT)
				if msg.X < rightPanelX-2 {
					// Clicked in left panel area
					// INFO tab: ~0-10 chars
					if msg.X < 10 {
						m.leftPanel.SetActiveTab(PreviewTabInfo)
					// SETLIST tab: ~10-22 chars
					} else if msg.X < 22 {
						m.leftPanel.SetActiveTab(PreviewTabSetlist)
					// OUTPUT tab: ~22+ chars
					} else {
						m.leftPanel.SetActiveTab(PreviewTabSettings)
					}
				} else if msg.X >= rightPanelX {
					// Right panel tabs (LOG, FILES)
					relX := msg.X - rightPanelX
					// LOG tab: ~0-10 chars
					if relX < 10 {
						m.rightPanel.SetActiveTab(LogTabOutput)
					// FILES tab: ~10+ chars
					} else {
						m.rightPanel.SetActiveTab(LogTabFiles)
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return "BOOTLEGGER SHUTDOWN\n"
	}

	if m.showBoot {
		return m.boot.View()
	}

	var sb strings.Builder

	// Account for borders and spacing
	availableWidth := m.width - 4
	availableHeight := m.height - 4

	// Two column layout (50% width each)
	leftWidth := availableWidth / 2
	rightWidth := availableWidth - leftWidth - 2
	panelHeight := availableHeight / 2

	m.logo.SetWidth(leftWidth)
	m.logo.SetHeight(panelHeight)
	m.source.SetWidth(leftWidth)
	m.source.SetHeight(panelHeight)
	m.rightPanel.SetWidth(rightWidth)
	m.rightPanel.SetHeight(panelHeight)
	m.statusBar.SetWidth(m.width)

	// Top row: Logo panel (left) and source panel (right)
	logoView := m.logo.View()
	sourceView := m.source.View()
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, logoView, " ", sourceView)
	sb.WriteString(topRow)
	sb.WriteString("\n\n")

	// Bottom row: Left panel (with tabs for INFO/SETLIST/OUTPUT) (left) and right panel (with tabs for LOG/FILES) (right)
	m.leftPanel.SetWidth(rightWidth)
	m.leftPanel.SetHeight(panelHeight)
	bottomLeftView := m.leftPanel.View()

	bottomRightView := m.rightPanel.ViewWithBlink(m.ledBlink)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, bottomLeftView, " ", bottomRightView)
	sb.WriteString(bottomRow)
	sb.WriteString("\n\n\n\n")

	// Status bar
	sb.WriteString(m.statusBar.ViewWithBlink(m.ledBlink))
	content := sb.String()

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
		lipgloss.Top,
		content,
	)
}

// updateLayout recalculates panel sizes based on terminal dimensions
func (m *Model) updateLayout() {
	availableWidth := m.width - 4
	leftWidth := availableWidth / 2
	rightWidth := availableWidth - leftWidth - 2
	panelHeight := m.height / 2

	m.logo.SetWidth(leftWidth)
	m.logo.SetHeight(panelHeight)
	m.source.SetWidth(leftWidth)
	m.source.SetHeight(panelHeight)
	m.leftPanel.SetWidth(rightWidth)
	m.leftPanel.SetHeight(panelHeight)
	m.rightPanel.SetWidth(rightWidth)
	m.rightPanel.SetHeight(panelHeight)
	m.statusBar.SetWidth(m.width)
}

// detectSourceType determines the source type from URL
func detectSourceType(url string) SourceType {
	url = strings.ToLower(url)
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return SourceYouTube
	}
	if strings.Contains(url, "dropbox.com") {
		return SourceDropbox
	}
	return SourceNone
}

// pasteFromClipboard reads from the system clipboard using native tools
func pasteFromClipboard() tea.Cmd {
	return func() tea.Msg {
		var text string
		var err error

		// Try wl-paste first (Wayland)
		if _, lookErr := exec.LookPath("wl-paste"); lookErr == nil {
			cmd := exec.Command("wl-paste", "-n")
			output, cmdErr := cmd.Output()
			if cmdErr == nil {
				text = string(output)
			} else {
				err = cmdErr
			}
		} else if _, lookErr := exec.LookPath("xclip"); lookErr == nil {
			// Try xclip (X11)
			cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
			output, cmdErr := cmd.Output()
			if cmdErr == nil {
				text = string(output)
			} else {
				err = cmdErr
			}
		} else if _, lookErr := exec.LookPath("xsel"); lookErr == nil {
			// Try xsel (X11)
			cmd := exec.Command("xsel", "--clipboard", "--output")
			output, cmdErr := cmd.Output()
			if cmdErr == nil {
				text = string(output)
			} else {
				err = cmdErr
			}
		} else {
			// No clipboard tool available
			return ClipboardPasteMsg{Text: "", Error: "NO CLIPBOARD TOOL FOUND (install wl-paste, xclip, or xsel)"}
		}

		if err != nil {
			return ClipboardPasteMsg{Text: "", Error: "CLIPBOARD READ FAILED"}
		}

		// Clean up the text - trim whitespace but preserve newlines
		text = strings.TrimSpace(text)
		return ClipboardPasteMsg{Text: text}
	}
}

// fetchMetadata fetches metadata for a URL
func fetchMetadata(ctx context.Context, url string) tea.Cmd {
	return func() tea.Msg {
		// Create a temporary downloader just for metadata
		downloader, err := source.NewDownloader("/tmp")
		if err != nil {
			return MetadataResultMsg{URL: url, Error: err}
		}

		meta, err := downloader.GetMetadata(ctx, url)
		if err != nil {
			return MetadataResultMsg{URL: url, Error: err}
		}

		return MetadataResultMsg{URL: url, Metadata: meta}
	}
}

// sanitizeTrackTitle cleans up track title for file naming
// Removes leading special characters like "-" and spaces
func sanitizeTrackTitle(title string) string {
	// Trim leading and trailing whitespace
	title = strings.TrimSpace(title)

	// Remove leading dashes, dots, underscores, and spaces
	title = strings.TrimLeft(title, "- _.")

	// Trim again after removing leading chars
	title = strings.TrimSpace(title)

	return title
}

// parseSetlist parses setlist text and returns track list
// Format: HH:MM:SS TITLE or MM:SS TITLE
func parseSetlist(text string) ([]Track, error) {
	lines := strings.Split(text, "\n")
	var tracks []Track
	trackNum := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // Skip empty lines
		}

		// Find the first space to separate timestamp from title
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			// If no space, skip this line or treat whole line as title with 0:00 timestamp
			continue
		}

		timestampStr := strings.TrimSpace(parts[0])
		title := strings.TrimSpace(parts[1])

		if title == "" {
			continue // Skip if no title
		}

		// Sanitize track title (remove leading -, spaces, etc.)
		title = sanitizeTrackTitle(title)

		if title == "" {
			continue // Skip if title is empty after sanitization
		}

		// Parse timestamp (supports HH:MM:SS or MM:SS or H:MM:SS)
		duration, err := parseTimestamp(timestampStr)
		if err != nil {
			// Skip invalid timestamps
			continue
		}

		tracks = append(tracks, Track{
			Number:    trackNum,
			Title:     title,
			StartTime: duration,
			Status:    TrackPending,
			Selected:  true, // Default to selected
		})
		trackNum++
	}

	// Calculate EndTime for each track (next track's StartTime)
	for i := 0; i < len(tracks)-1; i++ {
		tracks[i].EndTime = tracks[i+1].StartTime
	}
	// Last track has no end time (will be set to recording duration during splitting)

	return tracks, nil
}

// parseTimestamp converts HH:MM:SS or MM:SS to time.Duration
func parseTimestamp(ts string) (time.Duration, error) {
	parts := strings.Split(ts, ":")

	var hours, minutes, seconds int
	var err error

	switch len(parts) {
	case 2: // MM:SS
		minutes, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %w", err)
		}
		seconds, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds: %w", err)
		}
	case 3: // HH:MM:SS
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid hours: %w", err)
		}
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %w", err)
		}
		seconds, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds: %w", err)
		}
	default:
		return 0, fmt.Errorf("invalid timestamp format: %s", ts)
	}

	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second

	return duration, nil
}

func dummyTracks() []Track {
	return []Track{
		{Number: 1, Title: "INTRO", StartTime: 0, Status: TrackPending},
		{Number: 2, Title: "FIRST SONG", StartTime: 15 * time.Minute, Status: TrackPending},
		{Number: 3, Title: "SECOND SONG", StartTime: 19*time.Minute + 7*time.Second, Status: TrackPending},
		{Number: 4, Title: "THIRD SONG", StartTime: 25 * time.Minute, Status: TrackPending},
		{Number: 5, Title: "ENCORE", StartTime: 32*time.Minute + 30*time.Second, Status: TrackPending},
	}
}

// formatProgressBar creates an ASCII progress bar
func formatProgressBar(label string, percent float64) string {
	barWidth := 20
	filled := int(percent * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%s [%s] %3d%%", label, bar, int(percent*100))
}

// ledBlinkTick creates a command that sends periodic LED blink messages
func ledBlinkTick() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return LEDBlinkMsg{}
	})
}
