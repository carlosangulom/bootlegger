package source

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SourceType identifies the input source
type SourceType string

const (
	SourceNone    SourceType = "NONE"
	SourceYouTube SourceType = "YOUTUBE"
	SourceDropbox SourceType = "DROPBOX"
)

// DownloadResult contains the result of a download operation
type DownloadResult struct {
	FilePath    string
	Title       string
	Duration    float64
	Description string
	Error       error
}

// ProgressCallback is called with progress updates during download
type ProgressCallback func(percent float64, status string)

// Downloader handles downloading from various sources
type Downloader struct {
	OutputDir string
	ytdlpPath string
}

// NewDownloader creates a new downloader
func NewDownloader(outputDir string) (*Downloader, error) {
	// Find yt-dlp binary
	ytdlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp not found in PATH: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &Downloader{
		OutputDir: outputDir,
		ytdlpPath: ytdlpPath,
	}, nil
}

// DetectSourceType determines the source type from a URL
func DetectSourceType(url string) SourceType {
	url = strings.ToLower(url)
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return SourceYouTube
	}
	if strings.Contains(url, "dropbox.com") {
		return SourceDropbox
	}
	return SourceNone
}

// Download downloads media from the given URL with optional format selection
func (d *Downloader) Download(ctx context.Context, url string, formatID string, onProgress ProgressCallback) (*DownloadResult, error) {
	sourceType := DetectSourceType(url)

	switch sourceType {
	case SourceYouTube:
		return d.downloadYouTube(ctx, url, formatID, onProgress)
	case SourceDropbox:
		return d.downloadDropbox(ctx, url, formatID, onProgress)
	default:
		// Try yt-dlp anyway, it supports many sites
		return d.downloadYouTube(ctx, url, formatID, onProgress)
	}
}

// downloadYouTube downloads from YouTube using yt-dlp
func (d *Downloader) downloadYouTube(ctx context.Context, url string, formatID string, onProgress ProgressCallback) (*DownloadResult, error) {
	result := &DownloadResult{}

	// Output template
	outputTemplate := filepath.Join(d.OutputDir, "%(title)s.%(ext)s")

	// Determine format selector
	formatSelector := "bestaudio/best"
	if formatID != "" {
		formatSelector = formatID
	}

	// Build yt-dlp command
	// -f: format selector
	// --no-playlist: don't download playlists
	// --newline: progress on new lines for parsing
	args := []string{
		"-f", formatSelector,
		"--no-playlist",
		"--newline",
		"--progress",
		"-o", outputTemplate,
		"--print", "after_move:filepath",
		url,
	}

	cmd := exec.CommandContext(ctx, d.ytdlpPath, args...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	// Progress regex: [download]  45.2% of 12.34MiB at 1.23MiB/s ETA 00:05
	progressRe := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%`)
	// Destination regex: [download] Destination: filename.ext
	destRe := regexp.MustCompile(`\[download\] Destination: (.+)`)
	// Already downloaded regex
	alreadyRe := regexp.MustCompile(`\[download\] (.+) has already been downloaded`)

	// Read stdout for progress
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// Check for progress
			if matches := progressRe.FindStringSubmatch(line); len(matches) > 1 {
				if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
					if onProgress != nil {
						onProgress(percent/100.0, "DOWNLOADING")
					}
				}
			}

			// Check for destination file
			if matches := destRe.FindStringSubmatch(line); len(matches) > 1 {
				result.FilePath = matches[1]
			}

			// Check for already downloaded
			if matches := alreadyRe.FindStringSubmatch(line); len(matches) > 1 {
				result.FilePath = matches[1]
				if onProgress != nil {
					onProgress(1.0, "ALREADY DOWNLOADED")
				}
			}

			// Check if line is the final filepath (from --print)
			if strings.HasPrefix(line, d.OutputDir) && !strings.HasPrefix(line, "[") {
				result.FilePath = line
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
		if errMsg != "" {
			return nil, fmt.Errorf("yt-dlp failed: %s", errMsg)
		}
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	if onProgress != nil {
		onProgress(1.0, "DOWNLOAD COMPLETE")
	}

	// Get metadata
	if result.FilePath != "" {
		result.Title = extractTitleFromPath(result.FilePath)
	}

	return result, nil
}

// downloadDropbox handles Dropbox URLs
func (d *Downloader) downloadDropbox(ctx context.Context, url string, formatID string, onProgress ProgressCallback) (*DownloadResult, error) {
	// Convert Dropbox share URL to direct download URL
	// Replace ?dl=0 with ?dl=1 or add ?dl=1
	directURL := url
	if strings.Contains(url, "?dl=0") {
		directURL = strings.Replace(url, "?dl=0", "?dl=1", 1)
	} else if !strings.Contains(url, "dl=1") {
		if strings.Contains(url, "?") {
			directURL = url + "&dl=1"
		} else {
			directURL = url + "?dl=1"
		}
	}

	// Use yt-dlp for Dropbox too (it handles many sites)
	return d.downloadYouTube(ctx, directURL, formatID, onProgress)
}

// GetMetadata retrieves detailed metadata for a URL without downloading
func (d *Downloader) GetMetadata(ctx context.Context, url string) (*VideoMetadata, error) {
	// Use JSON output for comprehensive metadata
	args := []string{
		"--no-download",
		"--dump-json",
		"--no-playlist",
		url,
	}

	cmd := exec.CommandContext(ctx, d.ytdlpPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	meta := &VideoMetadata{}
	if err := parseJSONMetadata(output, meta); err != nil {
		return nil, err
	}

	return meta, nil
}

// parseJSONMetadata parses yt-dlp JSON output into VideoMetadata
func parseJSONMetadata(data []byte, meta *VideoMetadata) error {
	// Parse JSON using standard library
	var rawMeta map[string]interface{}
	if err := json.Unmarshal(data, &rawMeta); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract string fields
	if title, ok := rawMeta["title"].(string); ok {
		meta.Title = title
	}
	if uploader, ok := rawMeta["uploader"].(string); ok {
		meta.Uploader = uploader
	}
	if channel, ok := rawMeta["channel"].(string); ok {
		meta.Channel = channel
	}
	if uploadDate, ok := rawMeta["upload_date"].(string); ok {
		meta.UploadDate = uploadDate
	}
	if description, ok := rawMeta["description"].(string); ok {
		meta.Description = description
	}
	if thumbnail, ok := rawMeta["thumbnail"].(string); ok {
		meta.Thumbnail = thumbnail
	}
	if webpageURL, ok := rawMeta["webpage_url"].(string); ok {
		meta.WebpageURL = webpageURL
	}

	// Extract numeric fields
	if duration, ok := rawMeta["duration"].(float64); ok {
		meta.Duration = duration
	}
	if viewCount, ok := rawMeta["view_count"].(float64); ok {
		meta.ViewCount = int64(viewCount)
	}

	// Parse formats
	if formatsRaw, ok := rawMeta["formats"].([]interface{}); ok {
		meta.Formats = parseFormatsFromJSON(formatsRaw)
	}

	return nil
}


// parseFormatsFromJSON extracts available formats from parsed JSON
func parseFormatsFromJSON(formatsRaw []interface{}) []AudioFormat {
	var formats []AudioFormat
	seen := make(map[string]bool)

	for _, formatRaw := range formatsRaw {
		formatMap, ok := formatRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract format ID
		formatID, ok := formatMap["format_id"].(string)
		if !ok || seen[formatID] {
			continue
		}
		seen[formatID] = true

		// Extract extension
		ext, _ := formatMap["ext"].(string)

		// Extract audio codec
		acodec, _ := formatMap["acodec"].(string)
		if acodec == "none" || acodec == "" {
			continue // Skip formats without audio
		}

		// Extract video codec to determine if it has video
		vcodec, _ := formatMap["vcodec"].(string)
		hasVideo := vcodec != "" && vcodec != "none"

		format := AudioFormat{
			FormatID: formatID,
			Ext:      ext,
			Acodec:   acodec,
			HasVideo: hasVideo,
		}

		// Extract audio bitrate
		if abr, ok := formatMap["abr"].(float64); ok {
			format.Abr = abr
		}

		// Extract audio sample rate
		if asr, ok := formatMap["asr"].(float64); ok {
			format.Asr = int(asr)
		}

		// Extract filesize
		if filesize, ok := formatMap["filesize"].(float64); ok {
			format.Filesize = int64(filesize)
		}

		// Extract format note
		if formatNote, ok := formatMap["format_note"].(string); ok {
			format.FormatNote = formatNote
		}

		formats = append(formats, format)
	}

	// Sort by quality (higher bitrate first), audio-only preferred
	sortFormats(formats)

	return formats
}

// sortFormats sorts formats by quality
func sortFormats(formats []AudioFormat) {
	// Simple bubble sort (small list)
	for i := 0; i < len(formats); i++ {
		for j := i + 1; j < len(formats); j++ {
			// Prefer audio-only, then higher bitrate
			swapNeeded := false
			if !formats[i].HasVideo && formats[j].HasVideo {
				// i is audio-only, j has video - i should come first, no swap
			} else if formats[i].HasVideo && !formats[j].HasVideo {
				// j is audio-only - swap
				swapNeeded = true
			} else if formats[j].Abr > formats[i].Abr {
				// Same type, higher bitrate should come first
				swapNeeded = true
			}
			if swapNeeded {
				formats[i], formats[j] = formats[j], formats[i]
			}
		}
	}
}

// VideoMetadata contains metadata about a video
type VideoMetadata struct {
	Title       string
	Uploader    string
	Channel     string
	Duration    float64
	UploadDate  string
	Description string
	Thumbnail   string
	WebpageURL  string
	ViewCount   int64
	Formats     []AudioFormat
}

// AudioFormat represents an available audio format
type AudioFormat struct {
	FormatID   string
	Ext        string
	Acodec     string
	Abr        float64 // Audio bitrate in kbps
	Asr        int     // Audio sample rate
	Filesize   int64
	FormatNote string
	HasVideo   bool
}

// FormatDescription returns a human-readable description of the format
func (f AudioFormat) FormatDescription() string {
	var parts []string

	// Codec
	codec := strings.ToUpper(f.Acodec)
	if codec == "OPUS" || codec == "VORBIS" {
		parts = append(parts, f.Ext)
	} else {
		parts = append(parts, codec)
	}

	// Bitrate
	if f.Abr > 0 {
		parts = append(parts, fmt.Sprintf("%.0fkbps", f.Abr))
	}

	// Sample rate
	if f.Asr > 0 {
		parts = append(parts, fmt.Sprintf("%dHz", f.Asr))
	}

	// File size
	if f.Filesize > 0 {
		parts = append(parts, formatFilesize(f.Filesize))
	}

	// Video indicator
	if f.HasVideo {
		parts = append(parts, "+VIDEO")
	}

	return strings.Join(parts, " | ")
}

// formatFilesize formats bytes as human-readable string
func formatFilesize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// extractTitleFromPath extracts the title from a file path
func extractTitleFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
