package ws

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Room represents a chat room with its own hub
type Room struct {
	Code string // 12-character unique code
	Name string // User-defined room name
	Hub  *Hub   // Each room has its own hub
}

// RoomManager manages all active rooms
type RoomManager struct {
	mu       sync.RWMutex
	rooms    map[string]*Room // map[code]*Room
	releaser PersonaReleaser
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// SetPersonaReleaser sets the persona releaser for cleanup
func (rm *RoomManager) SetPersonaReleaser(pr PersonaReleaser) {
	rm.releaser = pr
}

// GenerateRoomCode generates a 12-character hex code
func GenerateRoomCode() string {
	bytes := make([]byte, 6) // 6 bytes = 12 hex characters
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CreateRoom creates a new room with the given name
func (rm *RoomManager) CreateRoom(name string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	code := GenerateRoomCode()
	// Ensure unique code
	for _, exists := rm.rooms[code]; exists; {
		code = GenerateRoomCode()
	}

	hub := NewHub()
	hub.SetPersonaReleaser(rm.releaser)
	hub.roomManager = rm
	hub.roomCode = code

	room := &Room{
		Code: code,
		Name: name,
		Hub:  hub,
	}

	rm.rooms[code] = room
	go hub.Run()

	return room
}

// GetRoom returns a room by its code
func (rm *RoomManager) GetRoom(code string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rooms[code]
}

// DeleteRoom removes a room
func (rm *RoomManager) DeleteRoom(code string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.rooms[code]; exists {
		delete(rm.rooms, code)
	}
}

// RoomExists checks if a room exists
func (rm *RoomManager) RoomExists(code string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	_, exists := rm.rooms[code]
	return exists
}

// GetRoomCount returns the number of active rooms
func (rm *RoomManager) GetRoomCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}
