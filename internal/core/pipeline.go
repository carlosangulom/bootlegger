package core

import (
	"context"
	"fmt"
	"time"

	"bootlegger/internal/audio"
	"bootlegger/internal/source"
)

// RunConfig carries all inputs needed to execute the full pipeline
type RunConfig struct {
	URL      string
	FormatID string
	Tracks   []Track // nil or empty = no splitting
}

// Pipeline orchestrates download → extract → split for a Session.
// It emits Events on a channel so callers (TUI, API, CLI) can react
// without knowing about each other.
type Pipeline struct {
	Session *Session
}

// NewPipeline creates a Pipeline bound to a Session
func NewPipeline(session *Session) *Pipeline {
	return &Pipeline{Session: session}
}

// Run executes the full pipeline and streams Events to the provided channel.
// The channel must be buffered or consumed promptly; Run never blocks on sends.
// Returns the paths of the downloaded file and extracted WAV on success.
func (p *Pipeline) Run(ctx context.Context, cfg RunConfig, events chan<- Event) (downloadedFile, wavFile string, err error) {
	send := func(e Event) {
		select {
		case events <- e:
		default:
		}
	}

	if err := p.Session.EnsureDirs(); err != nil {
		return "", "", fmt.Errorf("session dirs: %w", err)
	}

	// ── Stage 1: Download ────────────────────────────────────────────────────
	send(Event{Kind: KindLog, Stage: StageDownload, Message: "STARTING DOWNLOAD"})

	downloader, err := source.NewDownloader(p.Session.SourceDir)
	if err != nil {
		return "", "", fmt.Errorf("downloader init: %w", err)
	}

	result, err := downloader.Download(ctx, cfg.URL, cfg.FormatID, func(pct float64, status string) {
		send(Event{Kind: KindProgress, Stage: StageDownload, Percent: pct, Message: status})
	})
	if err != nil {
		return "", "", fmt.Errorf("download: %w", err)
	}
	downloadedFile = result.FilePath
	send(Event{Kind: KindProgress, Stage: StageDownload, Percent: 1.0, Message: "DOWNLOAD COMPLETE"})
	send(Event{Kind: KindLog, Stage: StageDownload, Message: fmt.Sprintf("DOWNLOADED: %s", result.Title)})

	// ── Stage 2: Extract ─────────────────────────────────────────────────────
	send(Event{Kind: KindLog, Stage: StageExtract, Message: "STARTING EXTRACTION"})

	extractor, err := audio.NewExtractor()
	if err != nil {
		return downloadedFile, "", fmt.Errorf("extractor init: %w", err)
	}

	extraction, err := extractor.ExtractToWAV(ctx, downloadedFile, p.Session.AudioDir, func(pct float64, status string) {
		send(Event{Kind: KindProgress, Stage: StageExtract, Percent: pct, Message: status})
	})
	if err != nil {
		return downloadedFile, "", fmt.Errorf("extract: %w", err)
	}
	wavFile = extraction.OutputPath
	totalDuration := extraction.Duration
	send(Event{Kind: KindProgress, Stage: StageExtract, Percent: 1.0, Message: "EXTRACTION COMPLETE"})

	// ── Stage 3: Split (optional) ─────────────────────────────────────────────
	if len(cfg.Tracks) == 0 {
		send(Event{Kind: KindDone, Message: "NO TRACKS TO SPLIT – COMPLETE"})
		return downloadedFile, wavFile, nil
	}

	send(Event{Kind: KindLog, Stage: StageSplit, Message: fmt.Sprintf("SPLITTING %d TRACKS", len(cfg.Tracks))})

	for i, track := range cfg.Tracks {
		if !track.Selected {
			continue
		}

		send(Event{Kind: KindTrack, Stage: StageSplit, Track: track.Number, Message: "PROCESSING"})

		endTime := track.EndTime
		if i == len(cfg.Tracks)-1 || track.EndTime == 0 {
			endTime = totalDuration
		}
		// Clamp: endTime must not exceed total duration
		if endTime > totalDuration && totalDuration > 0 {
			endTime = totalDuration
		}

		trackInfo := audio.TrackInfo{
			Number:    track.Number,
			Title:     track.Title,
			StartTime: track.StartTime,
			EndTime:   endTime,
		}

		if _, err := extractor.SplitTrack(ctx, wavFile, trackInfo, p.Session.SplitsDir, nil); err != nil {
			send(Event{Kind: KindTrack, Stage: StageSplit, Track: track.Number, Message: "ERROR", Error: err})
			return downloadedFile, wavFile, fmt.Errorf("split track %d: %w", track.Number, err)
		}

		send(Event{Kind: KindTrack, Stage: StageSplit, Track: track.Number, Message: "COMPLETE"})

		// Approximate progress across tracks
		pct := float64(i+1) / float64(len(cfg.Tracks))
		send(Event{Kind: KindProgress, Stage: StageSplit, Percent: pct})
	}

	send(Event{Kind: KindDone, Message: "ALL TRACKS COMPLETE"})
	return downloadedFile, wavFile, nil
}

// FetchMetadata retrieves video metadata without downloading
func FetchMetadata(ctx context.Context, url string) (*source.VideoMetadata, error) {
	// Temporary downloader pointed at /tmp just for metadata queries
	d, err := source.NewDownloader("/tmp/bootlegger-meta")
	if err != nil {
		return nil, err
	}
	return d.GetMetadata(ctx, url)
}

// DetectSource returns the SourceType for a URL
func DetectSource(url string) SourceType {
	switch source.DetectSourceType(url) {
	case source.SourceYouTube:
		return SourceYouTube
	case source.SourceDropbox:
		return SourceDropbox
	default:
		return SourceNone
	}
}

// FormatDuration formats a time.Duration as HH:MM:SS
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
