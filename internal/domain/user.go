package domain

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// User represents a chat participant with their persona and connection info
type User struct {
	ID           uuid.UUID       `json:"id"`
	PersonaName  string          `json:"persona_name"`
	PersonaColor string          `json:"persona_color"` // hex neon color
	BatteryLevel int             `json:"battery_level"` // 0-100, optional
	Conn         *websocket.Conn `json:"-"` // not serialized
}

// NewUser creates a new User with generated ID
func NewUser(personaName, personaColor string) *User {
	return &User{
		ID:           uuid.New(),
		PersonaName:  personaName,
		PersonaColor: personaColor,
		BatteryLevel: -1, // -1 means unknown/not shared
	}
}
