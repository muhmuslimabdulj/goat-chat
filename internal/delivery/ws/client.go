package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second // Relaxed to 60s for mobile stability

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096
)

// Client represents a single websocket connection
type Client struct {
	ID   string
	User *domain.User
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// NewClient creates a new Client
func NewClient(hub *Hub, conn *websocket.Conn, user *domain.User) *Client {
	return &Client{
		ID:   user.ID.String(),
		User: user,
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 1024),
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Silent close - no logging for privacy
			}
			break
		}

		// Parse incoming message
		var incoming struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.Unmarshal(message, &incoming); err != nil {
			// Silent continue - no logging for privacy
			continue
		}

		// Create domain message with user info
		msg := domain.Message{
			ID:        uuid.New().String(),
			Type:      domain.MessageType(incoming.Type),
			FromID:    c.User.ID.String(),
			FromName:  c.User.PersonaName,
			FromColor: c.User.PersonaColor,
			Payload:   incoming.Payload,
			CreatedAt: time.Now(),
		}


		// Handle specific message types
		switch msg.Type {
		case domain.MessageTypeKick:
			var payload map[string]string
			if err := json.Unmarshal(msg.Payload, &payload); err == nil {
				if targetID, ok := payload["target_id"]; ok {
					c.hub.KickUser(c.ID, targetID)
				}
			}
			continue

		case domain.MessageTypeTransfer:
			var payload map[string]string
			if err := json.Unmarshal(msg.Payload, &payload); err == nil {
				if newHostID, ok := payload["new_host_id"]; ok {
					c.hub.TransferHost(c.ID, newHostID)
				}
			}
			continue

		case domain.MessageTypeStatusUpdate:
			var status domain.StatusUpdatePayload
			if err := json.Unmarshal(incoming.Payload, &status); err == nil {
				if status.Battery > 0 {
					c.User.BatteryLevel = status.Battery
				}
			}
		}

		// Handle music control
		if msg.Type == domain.MessageTypeMusic {
			c.hub.HandleMusic(c, &msg)
			continue
		}

		if msg.Type == domain.MessageTypeMusicApprove {
			c.hub.HandleMusicApprove(c, &msg)
			continue
		}
		
		if msg.Type == domain.MessageTypePartyChange {
			c.hub.HandlePartyChange(c, msg)
			continue
		}

		if msg.Type == domain.MessageTypeMusicReject {
			c.hub.HandleMusicReject(c, &msg)
			continue
		}

		if msg.Type == domain.MessageTypeNobar {
			c.hub.HandleNobar(c, msg)
			continue
		}

		// Broadcast message to all clients
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		c.hub.Broadcast(data)
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send adds a message to the client's send queue
func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		// Buffer full
	}
}
