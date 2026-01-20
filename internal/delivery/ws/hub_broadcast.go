package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// buildUserEventMessage creates a user event message as JSON bytes
// NOTE: Caller must hold at least RLock when calling this
func (h *Hub) buildUserEventMessage(client *Client, eventType domain.MessageType, count int) []byte {
	// Build list of all online users
	onlineUsers := make([]map[string]interface{}, 0, len(h.clients))
	for _, c := range h.clients {
		onlineUsers = append(onlineUsers, map[string]interface{}{
			"id":      c.User.ID.String(),
			"persona": c.User.PersonaName,
			"color":   c.User.PersonaColor,
			"battery": c.User.BatteryLevel,
		})
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"user_id":      client.User.ID.String(),
		"persona":      client.User.PersonaName,
		"color":        client.User.PersonaColor,
		"user_count":   count,
		"host_id":      h.hostID,
		"online_users": onlineUsers,
	})

	msg := domain.Message{
		ID:        uuid.New().String(),
		Type:      eventType,
		FromID:    client.User.ID.String(),
		FromName:  client.User.PersonaName,
		FromColor: client.User.PersonaColor,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	data, _ := json.Marshal(msg)
	return data
}

// broadcastHostChange sends a host change event to all clients
// NOTE: Caller must hold at least RLock when calling this
func (h *Hub) broadcastHostChange() {
	var hostName string
	if hostClient, ok := h.clients[h.hostID]; ok {
		hostName = hostClient.User.PersonaName
	}

	payload, _ := json.Marshal(map[string]string{
		"host_id":   h.hostID,
		"host_name": hostName,
	})

	msg := domain.Message{
		ID:        uuid.New().String(),
		Type:      domain.MessageTypeHostChange,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	data, _ := json.Marshal(msg)
	h.Broadcast(data)
}



// sendLastUserWarning sends a warning to the last remaining user
func (h *Hub) sendLastUserWarning() {
	warningMsg := domain.Message{
		ID:        uuid.New().String(),
		Type:      domain.MessageTypeSystem,
		FromName:  "⚠️ Kamu adalah user terakhir! Room akan dihapus jika kamu keluar.",
		CreatedAt: time.Now(),
	}

	data, _ := json.Marshal(warningMsg)

	// Send directly to clients (should be only 1)
	// Do NOT use h.Broadcast() because that stores in history
	// Note: Caller (timer callback) already holds h.mu Lock
	for _, client := range h.clients {
		select {
		case client.send <- data:
		default:
		}
	}
}
