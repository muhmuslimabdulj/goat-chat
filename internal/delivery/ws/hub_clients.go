package ws

// Register adds a client to the hub
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SetRoomInfo sets the room information for this hub
func (h *Hub) SetRoomInfo(code, name string, rm *RoomManager) {
	h.roomCode = code
	h.roomName = name
	h.roomManager = rm
}

// GetRoomInfo returns the room code and name
func (h *Hub) GetRoomInfo() (code, name string) {
	return h.roomCode, h.roomName
}
