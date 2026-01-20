package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// KickUser removes a user from the room, only if requester is host
func (h *Hub) KickUser(requesterID, targetID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Verify requester is host
	if h.hostID != requesterID {
		return nil // Ignore unauthorized kick attempts
	}

	// Cannot kick yourself
	if requesterID == targetID {
		return nil
	}

	targetClient, exists := h.clients[targetID]
	if !exists {
		return nil // User already gone
	}

	// Send kick notification to target
	payload, _ := json.Marshal(map[string]string{
		"reason": "You have been kicked by the host.",
	})

	kickMsg := domain.Message{
		ID:        uuid.New().String(),
		Type:      domain.MessageTypeKick,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	data, _ := json.Marshal(kickMsg)
	targetClient.send <- data

	// Give more time for message to send, then unregister
	go func(c *Client) {
		time.Sleep(500 * time.Millisecond)
		h.unregister <- c
	}(targetClient)

	return nil
}

// TransferHost transfers host role to another user
func (h *Hub) TransferHost(requesterID, newHostID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Verify requester is host
	if h.hostID != requesterID {
		return
	}

	// Verify new host exists
	newHostClient, exists := h.clients[newHostID]
	if !exists {
		return
	}

	h.hostID = newHostID
	h.hostPersona = newHostClient.User.PersonaName
	h.broadcastHostChange()

	// Force full sync to ensure clients have correct host ID
	// We build message directly since we already hold the lock (to avoid deadlock)
	syncData := h.buildUserEventMessage(newHostClient, domain.MessageTypeUserSync, len(h.clients))
	h.Broadcast(syncData)
}

// ReclaimHost allows a reconnecting user to reclaim host if their persona matches
func (h *Hub) ReclaimHost(clientID, personaName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if personaName == h.hostPersona && h.hostID != clientID {
		h.hostID = clientID
		h.broadcastHostChange()
	}
}
