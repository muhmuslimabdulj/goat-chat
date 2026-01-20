package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// PersonaReleaser is used to release persona names when clients disconnect
type PersonaReleaser interface {
	Release(name string)
}


// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	mu              sync.RWMutex
	leaveDelay      time.Duration
	hostTransferDelay time.Duration

	clients         map[string]*Client
	broadcast       chan []byte
	register        chan *Client
	unregister      chan *Client
	personaReleaser PersonaReleaser
	messageHistory  *RingBuffer
	roomManager     *RoomManager
	roomCode        string
	roomName        string
	hostID          string
	hostPersona     string // Persistent host identity
	shutdownTimer   *time.Timer
	currentMusic    *domain.MusicPayload
	musicQueue      []domain.MusicQueueItem
	pendingQueue    []domain.MusicQueueItem
	currentNobar    *domain.NobarPayload
	delayedLeavers  map[string]*time.Timer // Spam prevention
	nobarRequests   []domain.NobarQueueItem
	nobarQueue      []domain.NobarQueueItem
	nobarViewers    map[string]domain.NobarViewer
	currentPartyMode string
}

// MusicState tracks the current playing song
type MusicState struct {
	VideoID   string    `json:"video_id"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	Duration  int       `json:"duration"`
	IsPlaying bool      `json:"is_playing"`
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[string]*Client),
		broadcast:      make(chan []byte, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		leaveDelay:     domain.LeaveDelay,
		hostTransferDelay: domain.HostTransferDelay,
		messageHistory: NewRingBuffer(domain.MaxHistorySize),
		hostID:         "",
		musicQueue:     make([]domain.MusicQueueItem, 0),
		pendingQueue:   make([]domain.MusicQueueItem, 0),
		delayedLeavers: make(map[string]*time.Timer),
		nobarRequests:  make([]domain.NobarQueueItem, 0),
		nobarQueue:     make([]domain.NobarQueueItem, 0),
		nobarViewers:   make(map[string]domain.NobarViewer),
		currentPartyMode: "normal",
	}
}

// SetPersonaReleaser sets the persona releaser for cleanup
func (h *Hub) SetPersonaReleaser(pr PersonaReleaser) {
	h.personaReleaser = pr
}

// cancelShutdown stops pending destroy timer
func (h *Hub) cancelShutdown() {
	if h.shutdownTimer != nil {
		h.shutdownTimer.Stop()
		h.shutdownTimer = nil
	}
}

// scheduleShutdown starts the grace period timer
func (h *Hub) scheduleShutdown() {
	if h.roomManager != nil && h.roomCode != "" {
		// Wait 60 seconds before destroying empty room to allow reconnects
		h.shutdownTimer = time.AfterFunc(60*time.Second, func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if len(h.clients) == 0 {
				if h.roomManager != nil {
					h.roomManager.DeleteRoom(h.roomCode)
				}
			}
		})
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.cancelShutdown()

			h.clients[client.ID] = client
			
			// Host assignment logic:
			// 1. If hostPersona is empty (first user), assign host
			// 2. If user's persona matches hostPersona (reconnecting host), reclaim host
			// 3. Otherwise, keep existing host
			if h.hostPersona == "" {
				h.hostID = client.ID
				h.hostPersona = client.User.PersonaName
			} else if client.User.PersonaName == h.hostPersona {
				// Reconnecting host reclaims their role
				h.hostID = client.ID
			}

			// Check if this is a silent rejoin (user reconnected quickly)
			silentRejoin := false
			if timer, ok := h.delayedLeavers[client.User.PersonaName]; ok {
				timer.Stop()
				delete(h.delayedLeavers, client.User.PersonaName)
				silentRejoin = true
			}

			count := len(h.clients) // Get count AFTER adding
			
			// Send IDENTITY message first (Critical for reconnects)
			identityMsg := h.buildUserEventMessage(client, domain.MessageTypeIdentity, count)	
			select {
			case client.send <- identityMsg:
			default:
			}

			// Build user join message
			joinMsg := h.buildUserEventMessage(client, domain.MessageTypeUserJoin, count)
			
			// Send message history to new client FIRST
			for _, histMsg := range h.messageHistory.GetAll() {
				select {
				case client.send <- histMsg:
				default:
				}
			}
			
			// If NOT silent rejoin, send join event to self and others
			if !silentRejoin {
				// Send join event to the new client
				select {
				case client.send <- joinMsg:
				default:
				}
				
				h.mu.Unlock()

				// Broadcast user join to ALL OTHER clients
				h.mu.RLock()
				for _, c := range h.clients {
					if c.ID != client.ID {
						select {
						case c.send <- joinMsg:
						default:
						}
					}
				}
			} else {
				// Silent rejoin: Just send UserSync to SELF so their list updates
				// But do NOT broadcast to others (they think user never left)
				// Note: We hold Lock here, so buildUserEventMessage is safe.
				syncMsg := h.buildUserEventMessage(client, domain.MessageTypeUserSync, len(h.clients))
				
				select {
				case client.send <- syncMsg:
				default:
				}
				
				h.mu.Unlock()
				h.mu.RLock()
			}

			// Send current music state to new client if playing
			if h.currentMusic != nil && h.currentMusic.IsPlaying {
				payloadBytes, _ := json.Marshal(h.currentMusic)
				syncMsg := domain.Message{
					ID:        uuid.New().String(),
					Type:      domain.MessageTypeMusicSync,
					Payload:   payloadBytes,
					CreatedAt: time.Now(),
				}
				data, _ := json.Marshal(syncMsg)
				select {
				case client.send <- data:
				default:
				}
			}

			// Send queue state to new client
			if h.currentMusic != nil || len(h.musicQueue) > 0 || len(h.pendingQueue) > 0 {
				h.sendQueueSyncToClient(client)
			}
			
			// Send nobar state to new client
			if h.currentNobar != nil {
				h.sendNobarSyncToClient(client)
			}

			// Send party mode to new client
			if h.currentPartyMode != "" && h.currentPartyMode != "normal" {
				payloadBytes, _ := json.Marshal(domain.PartyModePayload{
					Mode: h.currentPartyMode,
				})
				syncMsg := domain.Message{
					ID:        uuid.New().String(),
					Type:      domain.MessageTypePartyChange,
					Payload:   payloadBytes,
					CreatedAt: time.Now(),
				}
				data, _ := json.Marshal(syncMsg)
				select {
				case client.send <- data:
				default:
				}
			}

			h.mu.RUnlock()
			

			
			// Send a delayed sync to ensure client has accurate user list after any race conditions settle
			go func(c *Client, clientID string) {
				defer func() {
					if r := recover(); r != nil {
						// Client disconnected before sync, ignore
					}
				}()
				
				time.Sleep(500 * time.Millisecond)
				
				h.mu.RLock()
				// Check if client is still registered
				if _, ok := h.clients[clientID]; !ok {
					h.mu.RUnlock()
					return
				}
				syncMsg := h.buildUserEventMessage(c, domain.MessageTypeUserSync, len(h.clients))
				h.mu.RUnlock()
				
				select {
				case c.send <- syncMsg:
				default:
				}
			}(client, client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			// Check if client exists - prevent double unregister
			if _, ok := h.clients[client.ID]; !ok {
				h.mu.Unlock()
				continue // Client already unregistered, skip
			}

			delete(h.clients, client.ID)
			
			// Clean up from nobar viewers if present
			if _, ok := h.nobarViewers[client.ID]; ok {
				delete(h.nobarViewers, client.ID)
				h.broadcastNobarViewers()
			}

			close(client.send)
			
			// Delay leave broadcast to prevent spam on refresh
			personaName := client.User.PersonaName
			clientToCheck := client
			
			leaveTimer := time.AfterFunc(h.leaveDelay, func() {
				h.mu.Lock()
				defer h.mu.Unlock()
				
				// Make sure we are still tracking this leave
				if _, ok := h.delayedLeavers[personaName]; !ok {
					return
				}
				delete(h.delayedLeavers, personaName)

				// Release persona name
				if h.personaReleaser != nil {
					h.personaReleaser.Release(personaName)
				}

				count := len(h.clients)
				
				// Check if room is now empty
				if count == 0 {
					h.hostID = ""
					h.hostPersona = "" // Reset host persona
					h.handleStop() // Stop music and clear queue
					h.scheduleShutdown()
				} else if personaName == h.hostPersona {
					// Host left.
					// Note: Host transfer is handled by a separate goroutine (line 348)
					// which waits for hostTransferDelay (15s) to allow for reconnects.
				}

				// Broadcast user leave with accurate count
				// Fix Deadlock: Do NOT call broadcastUserEventWithCount because it acquires RLock while we hold Lock
				data := h.buildUserEventMessage(clientToCheck, domain.MessageTypeUserLeave, count)
				h.Broadcast(data)
				
				// Warn last user that room will be destroyed if they leave
				if count == 1 && h.roomCode != "" {
					h.sendLastUserWarning()
				}
			})
			
			h.delayedLeavers[personaName] = leaveTimer
			
			// Host transfer check (separate 15s timer for ROLE persistence)
			// This runs immediately upon disconnect, parallel to the detailed leave timer
			if count := len(h.clients); count > 0 && client.User.PersonaName == h.hostPersona {
				previousHostPersona := h.hostPersona
				
				go func(personaToCheck string) {
					time.Sleep(h.hostTransferDelay)
					
					h.mu.Lock()
					defer h.mu.Unlock()
					
					// If host persona changed in the meantime, abort
					if h.hostPersona != personaToCheck {
						return
					}

					// Check if original host is back
					isBack := false
					for _, c := range h.clients {
						if c.User.PersonaName == personaToCheck {
							isBack = true
							h.hostID = c.ID // Ensure ID is correct
							break
						}
					}
					
					if !isBack {
						// Host really left. Pick new host.
						// Now we UPDATE hostPersona because the old host is gone for good
						for id, c := range h.clients {
							h.hostID = id
							h.hostPersona = c.User.PersonaName
							break
						}
						
						// Only broadcast if we actually found a new host
						if h.hostID != "" {
							h.broadcastHostChange()
						}
					}
				}(previousHostPersona)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			// Store message in history using ring buffer (O(1)) - Filter ephemeral types
			var msgH domain.Message
			if err := json.Unmarshal(message, &msgH); err == nil {
				switch msgH.Type {
				case domain.MessageTypeChat, domain.MessageTypeSystem, 
					domain.MessageTypeUserJoin, domain.MessageTypeUserLeave,
					domain.MessageTypeDice, domain.MessageTypeFlip, domain.MessageTypeSpin,
					domain.MessageTypeWhisper, domain.MessageTypeGif, domain.MessageTypeSuit,
					domain.MessageTypeTod, domain.MessageTypePoll, domain.MessageTypeVote,
					domain.MessageTypeYoutube, domain.MessageTypeHostChange,
					domain.MessageTypeVibrate, domain.MessageTypeChaos, domain.MessageTypeConfetti,
					domain.MessageTypeTts:
					h.messageHistory.Add(message)
				}
			}
			
			// Broadcast to all clients
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, close connection and remove client
					close(client.send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.Unlock()
		}
	}
}
