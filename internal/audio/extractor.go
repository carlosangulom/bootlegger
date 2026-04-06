package audio

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ExtractionResult contains the result of an audio extraction
type ExtractionResult struct {
	OutputPath string
	Duration   time.Duration
	Error      error
}

// ProgressCallback is called with progress updates during extraction
type ProgressCallback func(percent float64, status string)

// Extractor handles audio extraction using ffmpeg
type Extractor struct {
	ffmpegPath  string
	ffprobePath string
}

// NewExtractor creates a new audio extractor
func NewExtractor() (*Extractor, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	ffprobePath, _ := exec.LookPath("ffprobe") // Optional, used for duration

	return &Extractor{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}, nil
}

// GetDuration returns the duration of a media file
func (e *Extractor) GetDuration(ctx context.Context, inputPath string) (time.Duration, error) {
	if e.ffprobePath == "" {
		return 0, fmt.Errorf("ffprobe not available")
	}

	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, e.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return time.Duration(durationSec * float64(time.Second)), nil
}

// ExtractToWAV extracts audio from input file to PCM WAV format
// This is the canonical intermediate format for all processing
func (e *Extractor) ExtractToWAV(ctx context.Context, inputPath string, outputDir string, onProgress ProgressCallback) (*ExtractionResult, error) {
	result := &ExtractionResult{}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate output filename
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(outputDir, baseName+".wav")
	result.OutputPath = outputPath

	// Get duration for progress calculation
	var totalDuration time.Duration
	if e.ffprobePath != "" {
		dur, err := e.GetDuration(ctx, inputPath)
		if err == nil {
			totalDuration = dur
			result.Duration = dur
		}
	}

	// Build ffmpeg command
	// -y: overwrite output
	// -i: input file
	// -vn: no video
	// -acodec pcm_s16le: 16-bit PCM
	// -ar 48000: 48kHz sample rate
	// -ac 2: stereo
	args := []string{
		"-y",
		"-i", inputPath,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "48000",
		"-ac", "2",
		"-progress", "pipe:1", // Output progress to stdout
		outputPath,
	}

	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)

	// Capture stdout for progress
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr for errors
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Progress parsing regex for -progress output
	// Format: out_time_ms=123456789
	outTimeMsRe := regexp.MustCompile(`out_time_ms=(\d+)`)
	// Alternative: out_time=00:01:23.456789
	outTimeRe := regexp.MustCompile(`out_time=(\d+):(\d+):(\d+)\.(\d+)`)

	// Read progress from stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// Try out_time_ms first
			if matches := outTimeMsRe.FindStringSubmatch(line); len(matches) > 1 {
				if ms, err := strconv.ParseInt(matches[1], 10, 64); err == nil && totalDuration > 0 {
					currentTime := time.Duration(ms) * time.Microsecond
					percent := float64(currentTime) / float64(totalDuration)
					if percent > 1.0 {
						percent = 1.0
					}
					if onProgress != nil {
						onProgress(percent, "PROCESSING AUDIO")
					}
				}
			}

			// Try out_time format
			if matches := outTimeRe.FindStringSubmatch(line); len(matches) > 4 {
				hours, _ := strconv.Atoi(matches[1])
				mins, _ := strconv.Atoi(matches[2])
				secs, _ := strconv.Atoi(matches[3])
				// matches[4] is microseconds

				currentTime := time.Duration(hours)*time.Hour +
					time.Duration(mins)*time.Minute +
					time.Duration(secs)*time.Second

				if totalDuration > 0 {
					percent := float64(currentTime) / float64(totalDuration)
					if percent > 1.0 {
						percent = 1.0
					}
					if onProgress != nil {
						onProgress(percent, "PROCESSING AUDIO")
					}
				}
			}
		}
	}()

	// Collect stderr for errors
	var stderrOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrOutput.WriteString(scanner.Text() + "\n")
		}
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		errMsg := stderrOutput.String()
		// Extract last few lines of error
		lines := strings.Split(errMsg, "\n")
		if len(lines) > 5 {
			lines = lines[len(lines)-5:]
		}
		return nil, fmt.Errorf("ffmpeg failed: %s", strings.Join(lines, "\n"))
	}

	if onProgress != nil {
		onProgress(1.0, "EXTRACTION COMPLETE")
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("output file was not created")
	}

	return result, nil
}

// IsAudioFile checks if the input is likely an audio-only file
func IsAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	audioExts := map[string]bool{
		".mp3":  true,
		".wav":  true,
		".flac": true,
		".aac":  true,
		".ogg":  true,
		".opus": true,
		".m4a":  true,
		".wma":  true,
	}
	return audioExts[ext]
}

// IsVideoFile checks if the input is likely a video file
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	videoExts := map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".webm": true,
		".avi":  true,
		".mov":  true,
		".flv":  true,
		".wmv":  true,
		".m4v":  true,
	}
	return videoExts[ext]
}

// TrackInfo contains information about a track to split
type TrackInfo struct {
	Number    int
	Title     string
	StartTime time.Duration
	EndTime   time.Duration
}

// SplitResult contains the result of splitting one track
type SplitResult struct {
	TrackNumber int
	OutputPath  string
	Error       error
}

// SplitTrack splits a single track from the WAV file
// Uses -ss and -to for sample-accurate splitting with -c copy
func (e *Extractor) SplitTrack(ctx context.Context, inputWAV string, track TrackInfo, outputDir string, onProgress ProgressCallback) (*SplitResult, error) {
	result := &SplitResult{TrackNumber: track.Number}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %w", err)
		return result, result.Error
	}

	// Sanitize filename
	safeTitle := sanitizeFilename(track.Title)
	filename := fmt.Sprintf("%02d - %s.flac", track.Number, safeTitle)
	outputPath := filepath.Join(outputDir, filename)
	result.OutputPath = outputPath

	// Build ffmpeg command for splitting
	// -ss: start time
	// -to: end time (if not last track)
	// -i: input file
	// -c:a flac: encode to FLAC
	// -compression_level 8: FLAC compression level
	args := []string{
		"-y",
		"-ss", formatDuration(track.StartTime),
	}

	// Add end time if not zero (last track has no end time)
	if track.EndTime > 0 {
		args = append(args, "-to", formatDuration(track.EndTime))
	}

	args = append(args,
		"-i", inputWAV,
		"-c:a", "flac",
		"-compression_level", "8",
		outputPath,
	)

	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)

	// Capture stderr for errors
	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create stderr pipe: %w", err)
		return result, result.Error
	}

	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start ffmpeg: %w", err)
		return result, result.Error
	}

	// Collect stderr
	var stderrOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrOutput.WriteString(scanner.Text() + "\n")
		}
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		errMsg := stderrOutput.String()
		lines := strings.Split(errMsg, "\n")
		if len(lines) > 5 {
			lines = lines[len(lines)-5:]
		}
		result.Error = fmt.Errorf("ffmpeg split failed: %s", strings.Join(lines, "\n"))
		return result, result.Error
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		result.Error = fmt.Errorf("output file was not created")
		return result, result.Error
	}

	if onProgress != nil {
		onProgress(1.0, fmt.Sprintf("TRACK %02d COMPLETE", track.Number))
	}

	return result, nil
}

// formatDuration formats a time.Duration as HH:MM:SS.mmm for ffmpeg
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}

// sanitizeFilename removes characters that are unsafe for filenames
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Replace multiple spaces with single space
	result = strings.Join(strings.Fields(result), " ")

	// Trim spaces
	result = strings.TrimSpace(result)

	return result
}
