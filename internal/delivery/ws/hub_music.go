package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// HandleMusic processes music control commands
func (h *Hub) HandleMusic(client *Client, msg *domain.Message) {
	var payload domain.MusicPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	isHost := client.ID == h.hostID

	switch payload.Action {
	case "play":
		if isHost {
			h.handleHostPlay(client, &payload)
		} else {
			h.handleUserRequest(client, &payload)
		}
	case "pause", "resume":
		if isHost {
			h.handlePauseResume(&payload)
		}
	case "stop":
		if isHost {
			h.handleStop()
		}
	case "next":
		if isHost {
			h.handleNext()
		}
	case "seek":
		if isHost {
			h.handleMusicSeek(payload.CurrentTime)
		}
	case "ended":
		// Song ended, play next or stop
		h.handleSongEnded()
	default:
		// Backward compatibility: treat as play if video_id provided
		if payload.VideoID != "" {
			if isHost {
				h.handleHostPlay(client, &payload)
			} else {
				h.handleUserRequest(client, &payload)
			}
		}
	}
}

// handleHostPlay - Host adds song directly
func (h *Hub) handleHostPlay(client *Client, payload *domain.MusicPayload) {
	// Validate YouTube video ID format
	if !IsValidYouTubeVideoID(payload.VideoID) {
		return
	}

	item := domain.MusicQueueItem{
		ID:              uuid.New().String(),
		VideoID:         payload.VideoID,
		Title:           payload.Title,
		RequestedBy:     client.ID,
		RequestedByName: client.User.PersonaName,
	}

	// If Nobar active, always queue (YTM disabled during Nobar)
	// If nothing playing, play immediately
	if h.currentNobar != nil {
		// Nobar active - queue instead of play
		h.musicQueue = append(h.musicQueue, item)
		h.broadcastQueueSync()
	} else if h.currentMusic == nil || !h.currentMusic.IsPlaying {
		h.playNow(item)
	} else {
		// Add to queue
		h.musicQueue = append(h.musicQueue, item)
		h.broadcastQueueSync()
	}
}

// handleUserRequest - Non-host requests song (needs approval)
func (h *Hub) handleUserRequest(client *Client, payload *domain.MusicPayload) {
	// Validate YouTube video ID format
	if !IsValidYouTubeVideoID(payload.VideoID) {
		return
	}

	item := domain.MusicQueueItem{
		ID:              uuid.New().String(),
		VideoID:         payload.VideoID,
		Title:           payload.Title,
		RequestedBy:     client.ID,
		RequestedByName: client.User.PersonaName,
	}

	h.pendingQueue = append(h.pendingQueue, item)
	h.broadcastQueueSync()
}

// HandleMusicApprove - Host approves a pending request
func (h *Hub) HandleMusicApprove(client *Client, msg *domain.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.ID != h.hostID {
		return
	}

	var payload domain.MusicApprovePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	// Find and move from pending to queue
	for i, item := range h.pendingQueue {
		if item.ID == payload.RequestID {
			// Remove from pending
			h.pendingQueue = append(h.pendingQueue[:i], h.pendingQueue[i+1:]...)

			// If Nobar is active, always queue (don't auto-play)
			// If nothing playing, play now
			if h.currentNobar != nil {
				// Nobar active - always queue
				h.musicQueue = append(h.musicQueue, item)
				h.broadcastQueueSync()
			} else if h.currentMusic == nil || !h.currentMusic.IsPlaying {
				h.playNow(item)
			} else {
				// Add to queue
				h.musicQueue = append(h.musicQueue, item)
				h.broadcastQueueSync()
			}
			return
		}
	}
}

// HandleMusicReject - Host rejects a pending request
func (h *Hub) HandleMusicReject(client *Client, msg *domain.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.ID != h.hostID {
		return
	}

	var payload domain.MusicRejectPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	// Remove from pending queue
	for i, item := range h.pendingQueue {
		if item.ID == payload.RequestID {
			h.pendingQueue = append(h.pendingQueue[:i], h.pendingQueue[i+1:]...)
			h.broadcastQueueSync()
			return
		}
	}
}

// playNow starts playing a song immediately
func (h *Hub) playNow(item domain.MusicQueueItem) {
	h.currentMusic = &domain.MusicPayload{
		VideoID:   item.VideoID,
		Title:     item.Title,
		Action:    "play",
		StartTime: time.Now(),
		IsPlaying: true,
	}
	h.broadcastMusicSync()
	h.broadcastQueueSync()
}

// handleNext skips to next song
func (h *Hub) handleNext() {
	if len(h.musicQueue) > 0 {
		// Pop first song from queue and play
		nextSong := h.musicQueue[0]
		h.musicQueue = h.musicQueue[1:]
		h.playNow(nextSong)
	} else {
		// No more songs, stop and hide player
		h.handleStop()
	}
}

// handleMusicSeek seeks to a specific time
func (h *Hub) handleMusicSeek(seekTime float64) {
	if h.currentMusic == nil {
		return
	}
	h.currentMusic.CurrentTime = seekTime
	h.currentMusic.StartTime = time.Now()
	h.broadcastMusicSync()
}

// handleSongEnded - Called when a song finishes
func (h *Hub) handleSongEnded() {
	// Debounce: If song started less than 5 seconds ago, ignore "ended" event
	// This prevents race conditions where the client sends multiple "ended" events,
	// causing the next song to be skipped immediately.
	if h.currentMusic != nil && time.Since(h.currentMusic.StartTime) < 5*time.Second {
		return
	}

	if len(h.musicQueue) > 0 {
		// Auto-play next
		nextSong := h.musicQueue[0]
		h.musicQueue = h.musicQueue[1:]
		h.playNow(nextSong)
	} else {
		// Queue empty, hide player
		h.currentMusic = nil
		h.broadcastMusicSync()
		h.broadcastQueueSync()
	}
}

// playNextFromQueue - Plays the next song from queue (called externally, e.g., after Nobar ends)
func (h *Hub) playNextFromQueue() {
	if len(h.musicQueue) > 0 {
		nextSong := h.musicQueue[0]
		h.musicQueue = h.musicQueue[1:]
		h.playNow(nextSong)
	}
}

// handleStop stops music and clears current
func (h *Hub) handleStop() {
	stopPayload := domain.MusicPayload{Action: "stop", IsPlaying: false}
	payloadBytes, _ := json.Marshal(stopPayload)
	syncMsg := domain.Message{
		ID:        uuid.New().String(),
		Type:      domain.MessageTypeMusicSync,
		Payload:   payloadBytes,
		CreatedAt: time.Now(),
	}
	data, _ := json.Marshal(syncMsg)
	h.Broadcast(data)

	// Reset all music state after broadcast
	h.currentMusic = nil
	h.musicQueue = []domain.MusicQueueItem{}
	h.pendingQueue = []domain.MusicQueueItem{}
	h.broadcastQueueSync()
}

// handlePauseResume handles pause/resume
func (h *Hub) handlePauseResume(payload *domain.MusicPayload) {
	if h.currentMusic == nil {
		return
	}

	h.currentMusic.Action = payload.Action
	h.currentMusic.IsPlaying = payload.Action == "resume"
	h.broadcastMusicSync()
}

// broadcastMusicSync sends current music state to all clients
func (h *Hub) broadcastMusicSync() {
	if h.currentMusic == nil {
		// Send stop message
		stopPayload := domain.MusicPayload{Action: "stop", IsPlaying: false}
		payloadBytes, _ := json.Marshal(stopPayload)
		syncMsg := domain.Message{
			ID:        uuid.New().String(),
			Type:      domain.MessageTypeMusicSync,
			Payload:   payloadBytes,
			CreatedAt: time.Now(),
		}
		data, _ := json.Marshal(syncMsg)
		h.Broadcast(data)
		return
	}

	payloadBytes, _ := json.Marshal(h.currentMusic)
	syncMsg := domain.Message{
		ID:        uuid.New().String(),
		Type:      domain.MessageTypeMusicSync,
		Payload:   payloadBytes,
		CreatedAt: time.Now(),
	}
	data, _ := json.Marshal(syncMsg)
	h.Broadcast(data)
}

// broadcastQueueSync sends queue state to all clients
func (h *Hub) broadcastQueueSync() {
	// Send to each client individually (host sees pending, others don't)
	for _, c := range h.clients {
		h.sendQueueSyncToClient(c)
	}
}

// sendQueueSyncToClient sends queue sync to a specific client
func (h *Hub) sendQueueSyncToClient(client *Client) {
	payload := domain.MusicQueueSyncPayload{
		Queue:        h.musicQueue,
		CurrentMusic: h.currentMusic,
	}

	// Send pending queue to everyone so they can see "Request Masuk" status
	payload.PendingQueue = h.pendingQueue

	// Ensure queue is never nil
	if payload.Queue == nil {
		payload.Queue = []domain.MusicQueueItem{}
	}

	payloadBytes, _ := json.Marshal(payload)
	syncMsg := domain.Message{
		ID:        uuid.New().String(),
		Type:      domain.MessageTypeMusicQueueSync,
		Payload:   payloadBytes,
		CreatedAt: time.Now(),
	}
	data, _ := json.Marshal(syncMsg)

	select {
	case client.send <- data:
	default:
	}
}
