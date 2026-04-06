package ui

import (
	"context"
	"path/filepath"
	"time"

	"bootlegger/internal/audio"

	tea "github.com/charmbracelet/bubbletea"
)

// StartSplittingAsync starts track splitting process
func StartSplittingAsync(ctx context.Context, wavPath string, tracks []Track, sessionDir string, totalDuration time.Duration) tea.Cmd {
	return func() tea.Msg {
		// Create splits output directory
		splitsDir := filepath.Join(sessionDir, "splits")

		extractor, err := audio.NewExtractor()
		if err != nil {
			return TrackUpdateMsg{TrackNumber: 0, Status: TrackError}
		}

		// Start splitting in goroutine
		go func() {
			for i, track := range tracks {
				// Skip unselected tracks
				if !track.Selected {
					continue
				}

				// Send processing message
				trackUpdateChan <- TrackUpdateMsg{
					TrackNumber: track.Number,
					Status:      TrackProcessing,
				}

				// Determine end time - last track uses total duration
				endTime := track.EndTime
				if i == len(tracks)-1 || track.EndTime == 0 {
					endTime = totalDuration
				}

				// Split this track
				trackInfo := audio.TrackInfo{
					Number:    track.Number,
					Title:     track.Title,
					StartTime: track.StartTime,
					EndTime:   endTime,
				}

				_, err := extractor.SplitTrack(ctx, wavPath, trackInfo, splitsDir, nil)

				if err != nil {
					trackUpdateChan <- TrackUpdateMsg{
						TrackNumber: track.Number,
						Status:      TrackError,
					}
					return
				}

				// Send completion message
				trackUpdateChan <- TrackUpdateMsg{
					TrackNumber: track.Number,
					Status:      TrackComplete,
				}
			}

			// All tracks complete
			trackUpdateChan <- SplittingCompleteMsg{}
		}()

		// Return initial message
		return TrackUpdateMsg{TrackNumber: 0, Status: TrackPending}
	}
}

// Global channel for track updates
var trackUpdateChan = make(chan tea.Msg, 100)

// WaitForTrackUpdate waits for next track update
func WaitForTrackUpdate() tea.Cmd {
	return func() tea.Msg {
		return <-trackUpdateChan
	}
}

// SplittingCompleteMsg signals all tracks are done
type SplittingCompleteMsg struct{}
