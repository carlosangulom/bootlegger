package ui

import (
	"context"
	"path/filepath"

	"bootlegger/internal/audio"

	tea "github.com/charmbracelet/bubbletea"
)

// Global channels for extraction progress
var extractionProgressChan chan ExtractionProgressMsg
var extractionResultChan chan ExtractionCompleteMsg

// InitExtractionChannels initializes the extraction communication channels
func InitExtractionChannels() {
	extractionProgressChan = make(chan ExtractionProgressMsg, 100)
	extractionResultChan = make(chan ExtractionCompleteMsg, 1)
}

// StartExtractionAsync starts audio extraction in a goroutine
func StartExtractionAsync(ctx context.Context, inputPath string, sessionDir string) tea.Cmd {
	return func() tea.Msg {
		// Initialize channels if needed
		if extractionProgressChan == nil {
			InitExtractionChannels()
		}

		// Create audio output directory
		audioDir := filepath.Join(sessionDir, "audio")

		extractor, err := audio.NewExtractor()
		if err != nil {
			return ExtractionCompleteMsg{Error: err}
		}

		// Start extraction in goroutine
		go func() {
			result, err := extractor.ExtractToWAV(ctx, inputPath, audioDir, func(percent float64, status string) {
				select {
				case extractionProgressChan <- ExtractionProgressMsg{Percent: percent, Status: status}:
				default:
					// Channel full, skip this update
				}
			})

			// Send completion message
			if err != nil {
				extractionResultChan <- ExtractionCompleteMsg{Error: err}
			} else {
				extractionResultChan <- ExtractionCompleteMsg{
					OutputPath: result.OutputPath,
					Duration:   result.Duration,
					Error:      nil,
				}
			}
		}()

		// Return initial message
		return ExtractionProgressMsg{Percent: 0, Status: "STARTING EXTRACTION"}
	}
}

// WaitForExtractionProgress creates a command that waits for extraction progress
func WaitForExtractionProgress() tea.Cmd {
	return func() tea.Msg {
		if extractionProgressChan == nil || extractionResultChan == nil {
			return ExtractionCompleteMsg{Error: nil}
		}

		// Check for completion first
		select {
		case result := <-extractionResultChan:
			return result
		default:
		}

		// Wait for progress (blocking)
		select {
		case progress := <-extractionProgressChan:
			return progress
		case result := <-extractionResultChan:
			return result
		}
	}
}

// ExtractionTickMsg triggers another poll for extraction progress
type ExtractionTickMsg struct{}
