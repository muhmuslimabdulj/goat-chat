package ws

import "testing"

func TestIsValidYouTubeVideoID(t *testing.T) {
	tests := []struct {
		name     string
		videoID  string
		expected bool
	}{
		{"Valid ID", "dQw4w9WgXcQ", true},
		{"Valid ID with underscore", "_abc123XYZ-", true},
		{"Valid ID with hyphen", "abc-123-XYZ", true},
		{"Empty", "", false},
		{"Too short", "abc123", false},
		{"Too long", "abc123456789012", false},
		{"Invalid chars", "abc!@#$%^&*(", false},
		{"Contains space", "abc 123 XYZ", false},
		{"Exactly 11 chars", "12345678901", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidYouTubeVideoID(tc.videoID)
			if result != tc.expected {
				t.Errorf("IsValidYouTubeVideoID(%q) = %v, expected %v", tc.videoID, result, tc.expected)
			}
		})
	}
}
