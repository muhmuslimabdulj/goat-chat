package ws

import (
	"strings"
	"testing"
	"time"
)

func TestNewSessionStore(t *testing.T) {
	store := NewSessionStore()
	if store == nil {
		t.Fatal("Expected session store to be created")
	}
	if store.Count() != 0 {
		t.Errorf("Expected empty store, got %d", store.Count())
	}
}

func TestSessionStore_GenerateToken(t *testing.T) {
	store := NewSessionStore()

	token := store.GenerateToken("user1", "CoolGoat", "#FF0000", "ABC123")

	if token == "" {
		t.Fatal("Expected token to be generated")
	}
	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("Expected 64 char token, got %d", len(token))
	}
	if store.Count() != 1 {
		t.Errorf("Expected 1 session, got %d", store.Count())
	}
}

func TestSessionStore_ValidateToken(t *testing.T) {
	store := NewSessionStore()

	token := store.GenerateToken("user1", "CoolGoat", "#FF0000", "ROOM1")

	// Valid token
	session, valid := store.ValidateToken(token)
	if !valid {
		t.Fatal("Expected token to be valid")
	}
	if session.PersonaName != "CoolGoat" {
		t.Errorf("Expected persona 'CoolGoat', got %s", session.PersonaName)
	}
	if session.RoomCode != "ROOM1" {
		t.Errorf("Expected room 'ROOM1', got %s", session.RoomCode)
	}

	// Invalid token
	_, valid = store.ValidateToken("invalid-token")
	if valid {
		t.Error("Expected invalid token to fail validation")
	}
}

func TestSessionStore_TokenReplacesOld(t *testing.T) {
	store := NewSessionStore()

	token1 := store.GenerateToken("user1", "OldGoat", "#FF0000", "ROOM1")
	token2 := store.GenerateToken("user1", "NewGoat", "#00FF00", "ROOM2")

	// Old token should be invalid
	_, valid := store.ValidateToken(token1)
	if valid {
		t.Error("Old token should be invalid after generating new one")
	}

	// New token should be valid
	session, valid := store.ValidateToken(token2)
	if !valid {
		t.Fatal("New token should be valid")
	}
	if session.PersonaName != "NewGoat" {
		t.Errorf("Expected persona 'NewGoat', got %s", session.PersonaName)
	}

	// Should only have 1 session
	if store.Count() != 1 {
		t.Errorf("Expected 1 session, got %d", store.Count())
	}
}

func TestSessionStore_RemoveToken(t *testing.T) {
	store := NewSessionStore()

	token := store.GenerateToken("user1", "CoolGoat", "#FF0000", "ROOM1")
	store.RemoveToken(token)

	_, valid := store.ValidateToken(token)
	if valid {
		t.Error("Removed token should be invalid")
	}
	if store.Count() != 0 {
		t.Errorf("Expected 0 sessions, got %d", store.Count())
	}
}

func TestSessionStore_RemoveByUserID(t *testing.T) {
	store := NewSessionStore()

	token := store.GenerateToken("user1", "CoolGoat", "#FF0000", "ROOM1")
	store.RemoveByUserID("user1")

	_, valid := store.ValidateToken(token)
	if valid {
		t.Error("Token for removed user should be invalid")
	}
}

func TestSessionStore_GetTokenByUserID(t *testing.T) {
	store := NewSessionStore()

	token := store.GenerateToken("user1", "CoolGoat", "#FF0000", "ROOM1")

	gotToken, exists := store.GetTokenByUserID("user1")
	if !exists {
		t.Fatal("Expected to find token for user")
	}
	if gotToken != token {
		t.Errorf("Expected token %s, got %s", token, gotToken)
	}

	// Non-existent user
	_, exists = store.GetTokenByUserID("user-notexist")
	if exists {
		t.Error("Should not find token for non-existent user")
	}
}

func TestSessionStore_TokenExpiry(t *testing.T) {
	store := &SessionStore{
		tokens:  make(map[string]*SessionToken),
		userIDs: make(map[string]string),
		ttl:     100 * time.Millisecond, // Very short TTL for testing
	}

	token := store.GenerateToken("user1", "CoolGoat", "#FF0000", "ROOM1")

	// Valid immediately
	_, valid := store.ValidateToken(token)
	if !valid {
		t.Fatal("Token should be valid immediately")
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, valid = store.ValidateToken(token)
	if valid {
		t.Error("Token should be expired")
	}
}

func TestSessionStore_Concurrency(t *testing.T) {
	store := NewSessionStore()

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(i int) {
			userID := "user" + string(rune(i))
			token := store.GenerateToken(userID, "Persona", "#000000", "ROOM")
			store.ValidateToken(token)
			store.GetTokenByUserID(userID)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	// Should not panic
}

func TestSessionToken_SecureRandomness(t *testing.T) {
	store := NewSessionStore()

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := store.GenerateToken("user"+string(rune(i)), "P", "#000", "R")
		if tokens[token] {
			t.Error("Duplicate token generated - not cryptographically random")
		}
		tokens[token] = true

		// Token should be hex
		for _, c := range token {
			if !strings.ContainsRune("0123456789abcdef", c) {
				t.Errorf("Token contains non-hex character: %c", c)
			}
		}
	}
}

func TestGlobalSessionStore(t *testing.T) {
	// Verify global instance exists
	if GlobalSessionStore == nil {
		t.Fatal("Global session store should be initialized")
	}
}
