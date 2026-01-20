package ws

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// HandlePartyChange processes party mode change requests from host
func (h *Hub) HandlePartyChange(c *Client, msg domain.Message) {
	// 1. Verify Host Authorization
	h.mu.RLock()
	isHost := c.ID == h.hostID
	h.mu.RUnlock()

	if !isHost {
		return // Ignore non-host requests
	}

	// 2. Parse payload
	var payload domain.PartyModePayload
	payloadBytes, _ := json.Marshal(msg.Payload)
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return
	}

	// 3. Update State
	h.mu.Lock()
	// Validation: Ensure mode is not empty
	if payload.Mode == "" {
		payload.Mode = "normal"
	}
	h.currentPartyMode = payload.Mode
	h.mu.Unlock()

	// 4. Broadcast to ALL clients
	h.broadcastPartyMode()
}

// broadcastPartyMode sends the current party mode state to all clients
func (h *Hub) broadcastPartyMode() {
	h.mu.RLock()
	mode := h.currentPartyMode
	h.mu.RUnlock()

	// Default to normal if empty
	if mode == "" {
		mode = "normal"
	}

	payloadBytes, _ := json.Marshal(domain.PartyModePayload{
		Mode: mode,
	})

	syncMsg := domain.Message{
		ID:      uuid.New().String(),
		Type:    domain.MessageTypePartyChange,
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

// broadcastPartyModeUnlocked sends party mode without acquiring lock (caller must hold lock)
func (h *Hub) broadcastPartyModeUnlocked() {
	mode := h.currentPartyMode

	// Default to normal if empty
	if mode == "" {
		mode = "normal"
	}

	payloadBytes, _ := json.Marshal(domain.PartyModePayload{
		Mode: mode,
	})

	syncMsg := domain.Message{
		ID:      uuid.New().String(),
		Type:    domain.MessageTypePartyChange,
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
