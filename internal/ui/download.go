package ui

import (
	"context"
	"path/filepath"

	"bootlegger/internal/source"

	tea "github.com/charmbracelet/bubbletea"
)

// DownloadState holds the state of an active download
type DownloadState struct {
	URL        string
	SessionDir string
	Progress   float64
	Status     string
	Done       bool
	Result     *source.DownloadResult
	Error      error
}

// Global channel for download progress (simple approach)
var downloadProgressChan chan DownloadProgressMsg
var downloadResultChan chan DownloadCompleteMsg

// InitDownloadChannels initializes the download communication channels
func InitDownloadChannels() {
	downloadProgressChan = make(chan DownloadProgressMsg, 100)
	downloadResultChan = make(chan DownloadCompleteMsg, 1)
}

// StartDownloadAsync starts a download in a goroutine and sends progress to channels
func StartDownloadAsync(ctx context.Context, url string, sessionDir string, formatID string) tea.Cmd {
	return func() tea.Msg {
		// Initialize channels if needed
		if downloadProgressChan == nil {
			InitDownloadChannels()
		}

		// Create source directory
		sourceDir := filepath.Join(sessionDir, "source")
		downloader, err := source.NewDownloader(sourceDir)
		if err != nil {
			return DownloadCompleteMsg{Error: err}
		}

		// Start download in goroutine
		go func() {
			result, err := downloader.Download(ctx, url, formatID, func(percent float64, status string) {
				select {
				case downloadProgressChan <- DownloadProgressMsg{Percent: percent, Status: status}:
				default:
					// Channel full, skip this update
				}
			})

			// Send completion message
			if err != nil {
				downloadResultChan <- DownloadCompleteMsg{Error: err}
			} else {
				downloadResultChan <- DownloadCompleteMsg{
					FilePath: result.FilePath,
					Title:    result.Title,
					Error:    nil,
				}
			}
		}()

		// Return initial message
		return DownloadProgressMsg{Percent: 0, Status: "STARTING DOWNLOAD"}
	}
}

// PollDownloadProgress returns a command that checks for download progress
func PollDownloadProgress() tea.Cmd {
	return func() tea.Msg {
		// Check for completion first
		select {
		case result := <-downloadResultChan:
			return result
		default:
		}

		// Check for progress
		select {
		case progress := <-downloadProgressChan:
			return progress
		default:
			// No progress available, return a tick to check again
			return DownloadTickMsg{}
		}
	}
}

// DownloadTickMsg triggers another poll
type DownloadTickMsg struct{}

// WaitForDownloadProgress creates a command that waits for progress
func WaitForDownloadProgress() tea.Cmd {
	return func() tea.Msg {
		if downloadProgressChan == nil || downloadResultChan == nil {
			return DownloadCompleteMsg{Error: nil}
		}

		// Check for completion first
		select {
		case result := <-downloadResultChan:
			return result
		default:
		}

		// Wait for progress (blocking)
		select {
		case progress := <-downloadProgressChan:
			return progress
		case result := <-downloadResultChan:
			return result
		}
	}
}
