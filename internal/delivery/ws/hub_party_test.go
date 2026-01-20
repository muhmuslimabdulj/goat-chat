package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// === PARTY MODE FEATURE TESTS ===

func TestHub_HandlePartyChange_HostOnly(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// Non-host tries to change party mode
	payload := domain.PartyModePayload{Mode: "party"}
	payloadBytes, _ := json.Marshal(payload)

	msg := domain.Message{
		ID:      "party1",
		Type:    domain.MessageTypePartyChange,
		Payload: payloadBytes,
	}

	hub.HandlePartyChange(user, msg)

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentPartyMode != "normal" {
		t.Errorf("Non-host should not be able to change party mode. Expected 'normal', got %s", hub.currentPartyMode)
	}
}

func TestHub_HandlePartyChange_ValidModes(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Change to party mode
	partyPayload := domain.PartyModePayload{Mode: "party"}
	partyBytes, _ := json.Marshal(partyPayload)
	hub.HandlePartyChange(host, domain.Message{
		ID:      "p1",
		Type:    domain.MessageTypePartyChange,
		Payload: partyBytes,
	})

	hub.mu.RLock()
	if hub.currentPartyMode != "party" {
		t.Errorf("Expected party mode 'party', got %s", hub.currentPartyMode)
	}
	hub.mu.RUnlock()

	// Change back to normal
	normalPayload := domain.PartyModePayload{Mode: "normal"}
	normalBytes, _ := json.Marshal(normalPayload)
	hub.HandlePartyChange(host, domain.Message{
		ID:      "p2",
		Type:    domain.MessageTypePartyChange,
		Payload: normalBytes,
	})

	hub.mu.RLock()
	if hub.currentPartyMode != "normal" {
		t.Errorf("Expected party mode 'normal', got %s", hub.currentPartyMode)
	}
	hub.mu.RUnlock()
}

func TestHub_HandlePartyChange_EmptyDefaultsToNormal(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Set to party first
	hub.mu.Lock()
	hub.currentPartyMode = "party"
	hub.mu.Unlock()

	// Send empty mode
	emptyPayload := domain.PartyModePayload{Mode: ""}
	emptyBytes, _ := json.Marshal(emptyPayload)
	hub.HandlePartyChange(host, domain.Message{
		ID:      "p1",
		Type:    domain.MessageTypePartyChange,
		Payload: emptyBytes,
	})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentPartyMode != "normal" {
		t.Errorf("Expected empty mode to default to 'normal', got %s", hub.currentPartyMode)
	}
}

func TestHub_HandlePartyChange_BroadcastsToAll(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// Drain any initial messages
	drainChannel := func(ch chan []byte) {
		for {
			select {
			case <-ch:
			case <-time.After(50 * time.Millisecond):
				return
			}
		}
	}
	drainChannel(host.send)
	drainChannel(user.send)

	// Host changes party mode
	partyPayload := domain.PartyModePayload{Mode: "party"}
	partyBytes, _ := json.Marshal(partyPayload)
	hub.HandlePartyChange(host, domain.Message{
		ID:      "p1",
		Type:    domain.MessageTypePartyChange,
		Payload: partyBytes,
	})

	// Wait for broadcast
	time.Sleep(50 * time.Millisecond)

	// Check that user received the party mode change
	select {
	case msg := <-user.send:
		var received domain.Message
		if err := json.Unmarshal(msg, &received); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		if received.Type != domain.MessageTypePartyChange {
			t.Errorf("Expected MessageTypePartyChange, got %s", received.Type)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("User did not receive party mode broadcast")
	}
}

func TestHub_PartyMode_InitialState(t *testing.T) {
	hub := NewHub()

	if hub.currentPartyMode != "normal" {
		t.Errorf("Expected initial party mode to be 'normal', got %s", hub.currentPartyMode)
	}
}

func TestHub_HandlePartyChange_ConcurrentSafety(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Concurrent party mode changes
	done := make(chan bool)
	for i := 0; i < 50; i++ {
		go func(i int) {
			mode := "normal"
			if i%2 == 0 {
				mode = "party"
			}
			payload := domain.PartyModePayload{Mode: mode}
			payloadBytes, _ := json.Marshal(payload)
			hub.HandlePartyChange(host, domain.Message{
				ID:      "concurrent",
				Type:    domain.MessageTypePartyChange,
				Payload: payloadBytes,
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Should not panic and mode should be either "party" or "normal"
	hub.mu.RLock()
	mode := hub.currentPartyMode
	hub.mu.RUnlock()

	if mode != "party" && mode != "normal" {
		t.Errorf("Unexpected party mode after concurrent changes: %s", mode)
	}
}
