package ws

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// SessionToken represents a reconnection token with associated persona info
type SessionToken struct {
	Token        string
	PersonaName  string
	PersonaColor string
	RoomCode     string
	UserID       string
	CreatedAt    time.Time
	LastUsed     time.Time
}

// SessionStore manages session tokens for secure reconnection
type SessionStore struct {
	tokens  map[string]*SessionToken // token -> session
	userIDs map[string]string        // userID -> token (for cleanup)
	mu      sync.RWMutex
	ttl     time.Duration
}

// NewSessionStore creates a new session store
func NewSessionStore() *SessionStore {
	store := &SessionStore{
		tokens:  make(map[string]*SessionToken),
		userIDs: make(map[string]string),
		ttl:     24 * time.Hour, // Tokens valid for 24 hours
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store
}

// GenerateToken creates a new session token for a user
func (s *SessionStore) GenerateToken(userID, personaName, personaColor, roomCode string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old token if exists
	if oldToken, exists := s.userIDs[userID]; exists {
		delete(s.tokens, oldToken)
	}

	// Generate secure random token
	tokenBytes := make([]byte, 32) // 256 bits
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	// Store session
	session := &SessionToken{
		Token:        token,
		PersonaName:  personaName,
		PersonaColor: personaColor,
		RoomCode:     roomCode,
		UserID:       userID,
		CreatedAt:    time.Now(),
		LastUsed:     time.Now(),
	}

	s.tokens[token] = session
	s.userIDs[userID] = token

	return token
}

// ValidateToken checks if a token is valid and returns the session
func (s *SessionStore) ValidateToken(token string) (*SessionToken, bool) {
	s.mu.RLock()
	session, exists := s.tokens[token]
	s.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(session.CreatedAt) > s.ttl {
		s.RemoveToken(token)
		return nil, false
	}

	// Update last used
	s.mu.Lock()
	session.LastUsed = time.Now()
	s.mu.Unlock()

	return session, true
}

// RemoveToken removes a token from the store
func (s *SessionStore) RemoveToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.tokens[token]; exists {
		delete(s.userIDs, session.UserID)
		delete(s.tokens, token)
	}
}

// RemoveByUserID removes a token by user ID
func (s *SessionStore) RemoveByUserID(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token, exists := s.userIDs[userID]; exists {
		delete(s.tokens, token)
		delete(s.userIDs, userID)
	}
}

// GetTokenByUserID returns the token for a user ID
func (s *SessionStore) GetTokenByUserID(userID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.userIDs[userID]
	return token, exists
}

// cleanupLoop periodically removes expired tokens
func (s *SessionStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes expired tokens
func (s *SessionStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for token, session := range s.tokens {
		if now.Sub(session.CreatedAt) > s.ttl {
			delete(s.userIDs, session.UserID)
			delete(s.tokens, token)
		}
	}
}

// Count returns the number of active sessions
func (s *SessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens)
}

// Global session store
var GlobalSessionStore = NewSessionStore()
