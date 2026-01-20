package domain

import (
	"encoding/json"
	"time"
)

// MessageType defines the type of message being sent
type MessageType string

const (
	MessageTypeChat         MessageType = "chat"
	MessageTypeVibrate      MessageType = "vibrate"
	MessageTypeChaos        MessageType = "chaos"
	MessageTypeReaction     MessageType = "reaction"
	MessageTypeStatusUpdate MessageType = "status_update"
	MessageTypeUserJoin     MessageType = "user_join"
	MessageTypeUserLeave    MessageType = "user_leave"
	MessageTypeUserSync     MessageType = "user_sync" // Silent sync
	MessageTypeSystem       MessageType = "system"
	MessageTypeIdentity     MessageType = "identity"
	
	// Fun features
	MessageTypeDice     MessageType = "dice"      // Dice roll result
	MessageTypeFlip     MessageType = "flip"      // Flipped text
	MessageTypeSpin     MessageType = "spin"      // Spinning text
	MessageTypeWhisper  MessageType = "whisper"   // Private message
	MessageTypeGif      MessageType = "gif"       // GIF message
	MessageTypeSuit     MessageType = "suit"      // Rock Paper Scissors
	MessageTypeTod      MessageType = "tod"       // Truth or Dare
	MessageTypePoll     MessageType = "poll"      // Quick poll
	MessageTypeVote     MessageType = "vote"      // Poll vote
	MessageTypeTyping   MessageType = "typing"    // Typing indicator
	MessageTypeConfetti MessageType = "confetti"  // Confetti bomb
	MessageTypeTheme    MessageType = "theme"     // Theme change
	MessageTypeHostChange MessageType = "host_change" // Host changed
	MessageTypeKick       MessageType = "kick"        // Kick user
	MessageTypeTransfer   MessageType = "transfer_host" // Transfer host role
	MessageTypeYoutube    MessageType = "youtube"       // YouTube embed
	MessageTypeMusic      MessageType = "music"         // Music control (play/pause/stop)
	MessageTypeMusicSync  MessageType = "music_sync"    // Sync state to clients
	MessageTypeMusicApprove  MessageType = "music_approve"   // Host approves request
	MessageTypeMusicReject   MessageType = "music_reject"    // Host rejects request
	MessageTypeMusicQueueSync MessageType = "music_queue_sync" // Sync full queue state
	MessageTypeNobar         MessageType = "nobar"           // Nobar (watch together) control
	MessageTypeNobarSync     MessageType = "nobar_sync"      // Sync nobar state to clients
	MessageTypeNobarQueueSync MessageType = "nobar_queue_sync" // Sync nobar requests (for host)
	MessageTypeNobarViewers  MessageType = "nobar_viewers_sync" // Sync active viewers
	MessageTypePartyChange   MessageType = "party_change"       // Party mode change
	MessageTypeTts           MessageType = "tts"                // Text to speech
)

// NobarViewer represents an active viewer
type NobarViewer struct {
	ID          string `json:"id"`
	PersonaName string `json:"persona_name"`
	PersonaColor string `json:"persona_color"`
}

// NobarViewersSyncPayload represents the list of active viewers
type NobarViewersSyncPayload struct {
	Viewers []NobarViewer `json:"viewers"`
	Count   int           `json:"count"`
}

// MusicPayload represents the payload for music control
type MusicPayload struct {
	VideoID     string    `json:"video_id,omitempty"`
	Title       string    `json:"title,omitempty"`
	Action      string    `json:"action,omitempty"` // play, pause, stop, next, ended, seek
	StartTime   time.Time `json:"start_time,omitempty"`
	CurrentTime float64   `json:"current_time,omitempty"` // in seconds
	Duration    int       `json:"duration,omitempty"`
	IsPlaying   bool      `json:"is_playing,omitempty"`
}

// MusicQueueItem represents a song in the queue
type MusicQueueItem struct {
	ID              string `json:"id"`
	VideoID         string `json:"video_id"`
	Title           string `json:"title,omitempty"`
	RequestedBy     string `json:"requested_by"`
	RequestedByName string `json:"requested_by_name"`
}

// MusicApprovePayload represents host approval of a request
type MusicApprovePayload struct {
	RequestID string `json:"request_id"`
}

// MusicRejectPayload represents host rejection of a request
type MusicRejectPayload struct {
	RequestID string `json:"request_id"`
}

// MusicQueueSyncPayload represents the full queue state sent to clients
type MusicQueueSyncPayload struct {
	Queue        []MusicQueueItem `json:"queue"`
	PendingQueue []MusicQueueItem `json:"pending_queue"`
	CurrentMusic *MusicPayload    `json:"current_music,omitempty"`
}

// NobarPayload represents the payload for watch together (nobar) feature
type NobarPayload struct {
	VideoID     string    `json:"video_id,omitempty"`
	Title       string    `json:"title,omitempty"`
	Action      string    `json:"action,omitempty"` // play, pause, stop, seek
	StartTime   time.Time `json:"start_time,omitempty"`
	CurrentTime float64   `json:"current_time,omitempty"` // in seconds
	Duration    int       `json:"duration,omitempty"`
	IsPlaying   bool      `json:"is_playing,omitempty"`
}

// NobarQueueItem represents a video request for nobar
type NobarQueueItem struct {
	ID              string `json:"id"`
	VideoID         string `json:"video_id"`
	Title           string `json:"title,omitempty"`
	RequestedBy     string `json:"requested_by"`
	RequestedByName string `json:"requested_by_name"`
}

// NobarQueueSyncPayload represents the list of pending nobar requests and active queue
type NobarQueueSyncPayload struct {
	Requests []NobarQueueItem `json:"requests"` // Pending approval
	Queue    []NobarQueueItem `json:"queue"`    // Approved/Playing queue
}

// Message represents a chat message or command
type Message struct {
	ID        string          `json:"id"`
	Type      MessageType     `json:"type"`
	FromID    string          `json:"from_id"`
	FromName  string          `json:"from_name"`
	FromColor string          `json:"from_color,omitempty"`
	ToID      string          `json:"to_id,omitempty"` // For whisper
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// ChatPayload is the payload for chat messages
type ChatPayload struct {
	Text string `json:"text"`
}

// YoutubePayload is the payload for YouTube messages
type YoutubePayload struct {
	VideoID string `json:"video_id"`
	URL     string `json:"url"`
}

// VibratePayload is the payload for vibrate/nudge messages
type VibratePayload struct {
	Pattern []int `json:"pattern"` // e.g., [200, 100, 200]
}

// ChaosPayload is the payload for chaos mode messages
type ChaosPayload struct {
	DurationMs int `json:"duration_ms"` // default 5000
}

// ReactionPayload is the payload for reaction bomb messages
type ReactionPayload struct {
	Emoji string  `json:"emoji"`
	X     float64 `json:"x"` // 0-1 relative position
	Y     float64 `json:"y"` // 0-1 relative position
}

// StatusUpdatePayload is the payload for battery/location updates
type StatusUpdatePayload struct {
	Battery   int     `json:"battery,omitempty"` // 0-100
}

// DicePayload is the payload for dice roll
type DicePayload struct {
	Max    int `json:"max"`    // Max value (e.g., 6 for d6, 20 for d20)
	Result int `json:"result"` // The rolled value
}

// FlipPayload is the payload for flipped text
type FlipPayload struct {
	Original string `json:"original"`
	Flipped  string `json:"flipped"`
}

// SpinPayload is the payload for spinning text
type SpinPayload struct {
	Text string `json:"text"`
}

// WhisperPayload is the payload for private messages
type WhisperPayload struct {
	ToID   string `json:"to_id"`
	ToName string `json:"to_name"`
	Text   string `json:"text"`
}

// GifPayload is the payload for GIF messages
type GifPayload struct {
	URL     string `json:"url"`
	Preview string `json:"preview,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
}

// SuitPayload is the payload for Rock Paper Scissors
type SuitPayload struct {
	ChallengerID   string `json:"challenger_id"`
	ChallengerName string `json:"challenger_name"`
	OpponentID     string `json:"opponent_id,omitempty"`
	OpponentName   string `json:"opponent_name,omitempty"`
	ChallengerMove string `json:"challenger_move,omitempty"` // rock, paper, scissors
	OpponentMove   string `json:"opponent_move,omitempty"`
	Winner         string `json:"winner,omitempty"` // challenger_id, opponent_id, or "draw"
	Status         string `json:"status"`           // pending, accepted, completed
}

// TodPayload is the payload for Truth or Dare
type TodPayload struct {
	Type     string `json:"type"` // truth or dare
	Question string `json:"question"`
}

// PollPayload is the payload for polls
type PollPayload struct {
	Question string         `json:"question"`
	Options  []string       `json:"options"`
	Votes    map[string]int `json:"votes,omitempty"`  // option -> count
	Voters   []string       `json:"voters,omitempty"` // who voted
	PollID   string         `json:"poll_id"`
}

// VotePayload is the payload for poll votes
type VotePayload struct {
	PollID      string `json:"poll_id"`
	OptionIndex int    `json:"option_index"`
}

// TypingPayload is the payload for typing indicator
type TypingPayload struct {
	IsTyping bool `json:"is_typing"`
}

// ConfettiPayload is the payload for confetti
type ConfettiPayload struct {
	Duration int    `json:"duration"` // ms
	Color    string `json:"color,omitempty"`
}

// PartyModePayload represents the payload for changing party mode
type PartyModePayload struct {
	Mode string `json:"mode"`
}

// TtsPayload is the payload for text-to-speech
type TtsPayload struct {
    Text string `json:"text"`
}
