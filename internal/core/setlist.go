package core

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// setlistReFirst matches timestamp-first lines:
//
//	"0:00 Track Name", "[1:23:45] - Title", "(0:30) Title"
var setlistReFirst = regexp.MustCompile(
	`^[\[(•]?\s*(\d+):(\d{2})(?::(\d{2}))?[\])]?\s*[-–—|]?\s*(.+?)\s*$`,
)

// setlistReLast matches lines where the timestamp comes at the end:
//
//	"1 - Crumbling Castle 0:19", "01. Intro 0:00", "Track Name 3:45"
var setlistReLast = regexp.MustCompile(
	`^(?:\d+\s*[.\-–—]\s*)?(.+?)\s+(\d+):(\d{2})(?::(\d{2}))?\s*$`,
)

// ParseSetlist parses raw setlist text into tracks.
// Accepts both timestamp-first ("0:00 Title") and title-first ("1 - Title 0:00") formats.
// Returns nil if no tracks are found (not an error — caller decides).
func ParseSetlist(text string) []Track {
	var tracks []Track
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var a, b int
		var thirdGroup string
		var title string

		if m := setlistReFirst.FindStringSubmatch(line); m != nil {
			a, _ = strconv.Atoi(m[1])
			b, _ = strconv.Atoi(m[2])
			thirdGroup = m[3]
			title = sanitizeTitle(m[4])
		} else if m := setlistReLast.FindStringSubmatch(line); m != nil {
			title = sanitizeTitle(m[1])
			a, _ = strconv.Atoi(m[2])
			b, _ = strconv.Atoi(m[3])
			thirdGroup = m[4]
		} else {
			continue
		}

		var secs int
		if thirdGroup != "" {
			c, _ := strconv.Atoi(thirdGroup)
			secs = a*3600 + b*60 + c
		} else {
			secs = a*60 + b
		}

		tracks = append(tracks, Track{
			Number:    len(tracks) + 1,
			Title:     title,
			StartTime: time.Duration(secs) * time.Second,
			EndTime:   0,
			Selected:  true,
			Status:    TrackPending,
		})
	}
	for i := 0; i < len(tracks)-1; i++ {
		tracks[i].EndTime = tracks[i+1].StartTime
	}
	return tracks
}

// sanitizeTitle trims whitespace and leading noise characters from a track title.
func sanitizeTitle(s string) string {
	s = strings.TrimSpace(s)
	return strings.TrimLeft(s, "-._\t ")
}
