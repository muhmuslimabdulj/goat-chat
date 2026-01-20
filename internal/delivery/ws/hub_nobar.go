package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// HandleNobar handles nobar (watch together) messages
func (h *Hub) HandleNobar(c *Client, msg domain.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var payload domain.NobarPayload
	payloadBytes, _ := json.Marshal(msg.Payload)
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return
	}

	// Determine if this is a request or a command
	isHost := c.ID == h.hostID

	// If payload action is "request", anyone can do it
	if payload.Action == "request" {
		h.handleNobarRequest(c, payload)
		return
	}
	
	// View/Unview actions (anyone can do it)
	if payload.Action == "view" {
		h.handleNobarView(c)
		return
	}
	if payload.Action == "unview" {
		h.handleNobarUnview(c)
		return
	}

	// If play command comes from non-host, treat it as request ONLY IF IT'S NEW
	// But usually client sends 'play' for /nobar command.
	// So we change behaviour: /nobar -> 'play' action from client -> treat as request.
	if payload.Action == "play" && !isHost {
		payload.Action = "request"
		h.handleNobarRequest(c, payload)
		return
	}

	// All other actions require host privileges
	if !isHost {
		return
	}

	switch payload.Action {
	case "play":
		// If Nobar is active, add to queue playlist regardless of sender (Host)
		if h.currentNobar != nil && h.currentNobar.VideoID != "" {
			h.handleNobarRequest(c, payload)
		} else {
			h.handleNobarPlay(payload)
		}
	case "pause":
		h.handleNobarPause()
	case "resume":
		h.handleNobarResume()
	case "stop":
		h.handleNobarStop()
	case "seek":
		h.handleNobarSeek(payload.CurrentTime)
	case "sync_meta":
		h.handleNobarSyncMeta(payload)
	case "approve":
		h.handleNobarApprove(payload)
	case "reject":
		h.handleNobarReject(payload)
	case "ended":
		h.handleNobarEnded()
	case "skip":
		h.handleNobarEnded()
	}
}

// handleNobarPlay starts a new nobar session
func (h *Hub) handleNobarPlay(payload domain.NobarPayload) {
	// Validate YouTube video ID format
	if !IsValidYouTubeVideoID(payload.VideoID) {
		return
	}

	// Auto-pause music if playing (Nobar > YTM priority)
	if h.currentMusic != nil && h.currentMusic.IsPlaying {
		h.currentMusic.IsPlaying = false
		h.currentMusic.Action = "pause"
		h.broadcastMusicSync()
	}

	// Reset party mode to normal when starting new nobar
	// Note: caller (HandleNobar) already holds h.mu lock
	h.currentPartyMode = "normal"
	h.broadcastPartyModeUnlocked()

	h.currentNobar = &domain.NobarPayload{
		VideoID:     payload.VideoID,
		Title:       payload.Title,
		StartTime:   time.Now(),
		CurrentTime: 0,
		Duration:    payload.Duration,
		IsPlaying:   true,
	}
	h.broadcastNobarSync()
}

// handleNobarPause pauses the current nobar session
func (h *Hub) handleNobarPause() {
	if h.currentNobar == nil {
		return
	}
	// Calculate current time before pausing
	elapsed := time.Since(h.currentNobar.StartTime).Seconds()
	h.currentNobar.CurrentTime = h.currentNobar.CurrentTime + elapsed
	h.currentNobar.StartTime = time.Now()
	h.currentNobar.IsPlaying = false
	h.broadcastNobarSync()
}

// handleNobarResume resumes the current nobar session
func (h *Hub) handleNobarResume() {
	if h.currentNobar == nil {
		return
	}
	h.currentNobar.StartTime = time.Now()
	h.currentNobar.IsPlaying = true
	h.broadcastNobarSync()
}

// handleNobarStop stops and clears the nobar session
func (h *Hub) handleNobarStop() {
	// Reset party mode to normal when nobar ends
	// Note: caller (HandleNobar) already holds h.mu lock
	h.currentPartyMode = "normal"
	h.broadcastPartyModeUnlocked()

	h.currentNobar = nil
	h.nobarQueue = []domain.NobarQueueItem{}
	h.nobarRequests = []domain.NobarQueueItem{}
	// Clear viewers list when nobar stops
	h.nobarViewers = make(map[string]domain.NobarViewer)

	h.broadcastNobarSync()
	h.broadcastNobarQueue()
	h.broadcastNobarViewers()

	// Auto-play music from queue if available (Nobar ended, YTM takes over)
	if len(h.musicQueue) > 0 && h.currentMusic == nil {
		h.playNextFromQueue()
	}
}

// handleNobarSeek seeks to a specific time
func (h *Hub) handleNobarSeek(seekTime float64) {
	if h.currentNobar == nil {
		return
	}
	h.currentNobar.CurrentTime = seekTime
	h.currentNobar.StartTime = time.Now()
	h.broadcastNobarSync()
}

// handleNobarSyncMeta updates metadata (like duration) without changing state
func (h *Hub) handleNobarSyncMeta(payload domain.NobarPayload) {
	if h.currentNobar == nil {
		return
	}
	if payload.Duration > 0 {
		h.currentNobar.Duration = payload.Duration
	}
	h.broadcastNobarSync()
}

// broadcastNobarSync sends nobar state to all clients
func (h *Hub) broadcastNobarSync() {
	// Create a snapshot with calculated current position
	var payloadBytes []byte
	if h.currentNobar != nil {
		snapshot := *h.currentNobar
		if snapshot.IsPlaying {
			elapsed := time.Since(h.currentNobar.StartTime).Seconds()
			snapshot.CurrentTime = h.currentNobar.CurrentTime + elapsed
		}
		snapshot.StartTime = time.Now()
		payloadBytes, _ = json.Marshal(snapshot)
	} else {
		payloadBytes, _ = json.Marshal(h.currentNobar)
	}
	
	syncMsg := domain.Message{
		ID:      uuid.New().String(),
		Type:    domain.MessageTypeNobarSync,
		Payload: payloadBytes,
	}

	data, err := json.Marshal(syncMsg)
	if err != nil {
		return
	}

	for _, client := range h.clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

// sendNobarSyncToClient sends current nobar state to a specific client
func (h *Hub) sendNobarSyncToClient(c *Client) {
	if h.currentNobar == nil {
		return
	}

	// Create a snapshot with calculated current position
	snapshot := *h.currentNobar
	if snapshot.IsPlaying {
		// Calculate actual current position
		elapsed := time.Since(h.currentNobar.StartTime).Seconds()
		snapshot.CurrentTime = h.currentNobar.CurrentTime + elapsed
	}
	// Reset start time to now, so clients can calculate from this point
	snapshot.StartTime = time.Now()

	payloadBytes, _ := json.Marshal(snapshot)
	
	syncMsg := domain.Message{
		ID:      uuid.New().String(),
		Type:    domain.MessageTypeNobarSync,
		Payload: payloadBytes,
	}

	data, err := json.Marshal(syncMsg)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
	}
}

// handleNobarRequest adds a video request to the appropriate queue
func (h *Hub) handleNobarRequest(c *Client, payload domain.NobarPayload) {
	reqID := uuid.New().String()
	item := domain.NobarQueueItem{
		ID:              reqID,
		VideoID:         payload.VideoID,
		Title:           payload.Title,
		RequestedBy:     c.User.ID.String(),
		RequestedByName: c.User.PersonaName,
	}

	// If host, add to active Playlist Queue
	if c.ID == h.hostID {
		h.nobarQueue = append(h.nobarQueue, item)


		// If not playing, pop and play immediately
		if h.currentNobar == nil || h.currentNobar.VideoID == "" {
			if len(h.nobarQueue) > 0 {
				next := h.nobarQueue[0]
				h.nobarQueue = h.nobarQueue[1:]
				h.handleNobarPlay(domain.NobarPayload{
					VideoID: next.VideoID,
					Title:   next.Title,
				})
			}
		}
	} else {
		// If user, add to Pending Requests
		h.nobarRequests = append(h.nobarRequests, item)


		// Send success notification
		successPayload, _ := json.Marshal(map[string]string{
			"action": "request_success",
			"title":  payload.Title,
		})

		successMsg := domain.Message{
			ID:      uuid.New().String(),
			Type:    domain.MessageTypeNobar,
			Payload: successPayload,
		}

		data, _ := json.Marshal(successMsg)
		select {
		case c.send <- data:
		default:
		}
	}

	// Sync updated queue to host
	// Sync updated queue to all
	h.broadcastNobarQueue()
}

// handleNobarApprove approves a request and plays it
func (h *Hub) handleNobarApprove(payload domain.NobarPayload) {
	// payload.VideoID is used as Request ID here
	reqID := payload.VideoID
	
	var targetItem *domain.NobarQueueItem
	idx := -1
	
	for i, item := range h.nobarRequests {
		if item.ID == reqID {
			targetItem = &item
			idx = i
			break
		}
	}
	
	if targetItem != nil {
		// Remove from Requests
		h.nobarRequests = append(h.nobarRequests[:idx], h.nobarRequests[idx+1:]...)
		
		// Add to Playlist Queue
		h.nobarQueue = append(h.nobarQueue, *targetItem)
		
		// If Nobar is idle, play it immediately (Pop from queue)
		if h.currentNobar == nil || h.currentNobar.VideoID == "" {
			if len(h.nobarQueue) > 0 {
				next := h.nobarQueue[0]
				h.nobarQueue = h.nobarQueue[1:]
				h.handleNobarPlay(domain.NobarPayload{
					VideoID: next.VideoID,
					Title:   next.Title,
				})
			}
		}
		
		// Sync queue to host
		// Sync queue to all
		h.broadcastNobarQueue()
	}
}

// handleNobarReject rejects (removes) a request or removes from queue
func (h *Hub) handleNobarReject(payload domain.NobarPayload) {
	reqID := payload.VideoID
	
	// Check Pending Requests
	for i, item := range h.nobarRequests {
		if item.ID == reqID {
			h.nobarRequests = append(h.nobarRequests[:i], h.nobarRequests[i+1:]...)
			h.broadcastNobarQueue()
			return
		}
	}
	
	// Check Active Queue
	for i, item := range h.nobarQueue {
		if item.ID == reqID {
			h.nobarQueue = append(h.nobarQueue[:i], h.nobarQueue[i+1:]...)
			h.broadcastNobarQueue()
			return
		}
	}
}

// broadcastNobarQueue sends the current request queue and playlist to ALL clients
func (h *Hub) broadcastNobarQueue() {
	payloadBytes, _ := json.Marshal(domain.NobarQueueSyncPayload{
		Requests: h.nobarRequests,
		Queue:    h.nobarQueue,
	})

	syncMsg := domain.Message{
		ID:      uuid.New().String(),
		Type:    domain.MessageTypeNobarQueueSync,
		Payload: payloadBytes,
	}

	data, err := json.Marshal(syncMsg)
	if err == nil {
		select {
		case h.broadcast <- data:
		default:
		}
	}
}

// handleNobarEnded plays the next video in queue
func (h *Hub) handleNobarEnded() {
	if len(h.nobarQueue) > 0 {
		next := h.nobarQueue[0]
		h.nobarQueue = h.nobarQueue[1:]
		

		
		h.handleNobarPlay(domain.NobarPayload{
			VideoID: next.VideoID,
			Title:   next.Title,
		})
		
		h.broadcastNobarQueue()
	} else {
		// No more videos
		h.handleNobarStop()
	}
}

// handleNobarView adds client to viewer list
func (h *Hub) handleNobarView(c *Client) {
	viewer := domain.NobarViewer{
		ID:           c.User.ID.String(),
		PersonaName:  c.User.PersonaName,
		PersonaColor: c.User.PersonaColor,
	}
	h.nobarViewers[c.ID] = viewer
	h.broadcastNobarViewers()
}

// handleNobarUnview removes client from viewer list
func (h *Hub) handleNobarUnview(c *Client) {
	if _, ok := h.nobarViewers[c.ID]; ok {
		delete(h.nobarViewers, c.ID)
		h.broadcastNobarViewers()
	}
}

// broadcastNobarViewers sends the list of active viewers to all clients
func (h *Hub) broadcastNobarViewers() {
	var viewers []domain.NobarViewer
	for _, v := range h.nobarViewers {
		viewers = append(viewers, v)
	}

	payloadBytes, _ := json.Marshal(domain.NobarViewersSyncPayload{
		Viewers: viewers,
		Count:   len(viewers),
	})

	syncMsg := domain.Message{
		ID:      uuid.New().String(),
		Type:    domain.MessageTypeNobarViewers,
		Payload: payloadBytes,
	}

	data, err := json.Marshal(syncMsg)
	if err == nil {
		select {
		case h.broadcast <- data:
		default:
		}
	}
}
