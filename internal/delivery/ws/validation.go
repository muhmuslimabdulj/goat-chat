package ws

import "regexp"

// youtubeVideoIDRegex matches valid YouTube video IDs (11 characters, alphanumeric + - and _)
var youtubeVideoIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

// IsValidYouTubeVideoID validates a YouTube video ID format
// YouTube video IDs are exactly 11 characters containing alphanumeric, - and _
func IsValidYouTubeVideoID(videoID string) bool {
	if videoID == "" {
		return false
	}
	return youtubeVideoIDRegex.MatchString(videoID)
}
