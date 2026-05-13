package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session holds all paths and metadata for a single bootlegger run
type Session struct {
	ID        string
	BaseDir   string
	Dir       string
	SourceDir string
	AudioDir  string
	SplitsDir string
	CreatedAt time.Time
}

// NewSession creates a Session rooted at baseDir (e.g. ~/BOOTLEGGER).
// Directories are not created until EnsureDirs is called.
func NewSession(baseDir string) *Session {
	id := time.Now().Format("20060102-150405")
	dir := filepath.Join(baseDir, "sessions", id)
	return &Session{
		ID:        id,
		BaseDir:   baseDir,
		Dir:       dir,
		SourceDir: filepath.Join(dir, "source"),
		AudioDir:  filepath.Join(dir, "audio"),
		SplitsDir: filepath.Join(dir, "splits"),
		CreatedAt: time.Now(),
	}
}

// DefaultBaseDir returns ~/BOOTLEGGER
func DefaultBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, "BOOTLEGGER"), nil
}

// EnsureDirs creates all session subdirectories
func (s *Session) EnsureDirs() error {
	for _, dir := range []string{s.SourceDir, s.AudioDir, s.SplitsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	return nil
}
