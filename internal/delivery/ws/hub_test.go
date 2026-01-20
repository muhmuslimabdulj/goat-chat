package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// MockClient creates a client without an actual websocket connection suitable for testing
func newMockClient(hub *Hub, name string) *Client {
	if name == "" {
		name = "TestPerson"
	}
	user := domain.NewUser(name, "#000000")
	user.ID = uuid.New()
	
	return &Client{
		ID:   user.ID.String(),
		User: user,
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 256),
	}
}

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub.clients == nil {
		t.Error("Clients map not initialized")
	}
	if hub.broadcast == nil {
		t.Error("Broadcast channel not initialized")
	}
	if hub.register == nil {
		t.Error("Register channel not initialized")
	}
	if hub.unregister == nil {
		t.Error("Unregister channel not initialized")
	}
}

func TestHub_Register(t *testing.T) {
	hub := NewHub()
	go hub.Run() // Start hub loop

	client := newMockClient(hub, "TestClientReg")

	// Register
	hub.Register(client)

	// Wait for async operation (short sleep is acceptable for simple unit test)
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", hub.ClientCount())
	}
	
	// Check if client is actually in the map
	hub.mu.RLock()
	_, exists := hub.clients[client.ID]
	hub.mu.RUnlock()
	
	if !exists {
		t.Error("Client ID not found in hub clients map")
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := newMockClient(hub, "TestClientUnreg")
	
	// Register first
	hub.Register(client)
	time.Sleep(20 * time.Millisecond)
	
	// Unregister
	hub.Unregister(client)
	time.Sleep(20 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := newMockClient(hub, "Client1")
	client2 := newMockClient(hub, "Client2")

	hub.Register(client1)
	hub.Register(client2)
	
	// Wait until both are registered
	for i := 0; i < 10; i++ {
		if hub.ClientCount() == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	if hub.ClientCount() != 2 {
		t.Fatalf("Failed to register clients, count: %d", hub.ClientCount())
	}

	// Construct a test message
	msg := domain.Message{
		ID: "test-msg", 
		Type: domain.MessageTypeChat, 
		Payload: json.RawMessage(`{"text":"hello"}`),
	}
	data, _ := json.Marshal(msg)

	// Broadcast
	hub.Broadcast(data)
	
	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check if clients received it
	// Note: Clients also receive "User Join" messages, so we must drain those first or find our message
	
	checkMessage := func(c *Client, name string) {
		timeout := time.After(100 * time.Millisecond)
		for {
			select {
			case received := <-c.send:
				// Try to parse to see if it's our chat message
				var m domain.Message
				if err := json.Unmarshal(received, &m); err == nil {
					if m.ID == "test-msg" {
						return // Found it!
					}
				}
			case <-timeout:
				t.Errorf("%s did not receive expected chat message", name)
				return
			}
		}
	}

	checkMessage(client1, "Client 1")
	checkMessage(client2, "Client 2")
}

func TestHub_RaceCondition(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	
	// Stress test registering/unregistering concurrently
	for i := 0; i < 50; i++ {
		go func() {
			c := newMockClient(hub, "ChaosClient")
			hub.Register(c)
			time.Sleep(time.Millisecond) // Simulate brief connection
			hub.Unregister(c)
		}()
	}
	
	// Wait sufficient time for chaos to settle
	time.Sleep(500 * time.Millisecond)
	
	// Should not panic and ideally count is 0 (or low if some still running)
	// Main goal is ensuring no concurrent map read/write panics
	count := hub.ClientCount()
	if count < 0 {
		t.Errorf("Client count invalid: %d", count)
	}
}

func TestHub_FirstUserBecomesHost(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := newMockClient(hub, "FirstHost")
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	hostID := hub.hostID
	hub.mu.RUnlock()

	if hostID != client.ID {
		t.Errorf("Expected host ID to be %s, got %s", client.ID, hostID)
	}
}

func TestHub_HostTransfersOnDisconnect(t *testing.T) {
	hub := NewHub()
	hub.leaveDelay = 10 * time.Millisecond // Speed up test
	hub.hostTransferDelay = 10 * time.Millisecond // Speed up test
	go hub.Run()

	client1 := newMockClient(hub, "InitialHost")
	client2 := newMockClient(hub, "NextHost")

	hub.Register(client1)
	time.Sleep(30 * time.Millisecond)
	hub.Register(client2)
	time.Sleep(30 * time.Millisecond)

	// Client1 should be host
	hub.mu.RLock()
	initialHost := hub.hostID
	hub.mu.RUnlock()

	if initialHost != client1.ID {
		t.Fatalf("Expected initial host to be client1 (%s), got %s", client1.ID, initialHost)
	}

	// Unregister client1 (the host)
	hub.Unregister(client1)
	time.Sleep(50 * time.Millisecond)

	// Client2 should now be host
	hub.mu.RLock()
	newHost := hub.hostID
	hub.mu.RUnlock()

	if newHost != client2.ID {
		t.Errorf("Expected host to transfer to client2 (%s), got %s", client2.ID, newHost)
	}
}

func TestHub_HostResetsWhenEmpty(t *testing.T) {
	hub := NewHub()
	hub.leaveDelay = 10 * time.Millisecond // Speed up test
	hub.hostTransferDelay = 10 * time.Millisecond // Speed up test
	go hub.Run()

	client := newMockClient(hub, "LonelyHost")
	hub.Register(client)
	time.Sleep(30 * time.Millisecond)

	// Verify host is set
	hub.mu.RLock()
	hostBefore := hub.hostID
	hub.mu.RUnlock()

	if hostBefore == "" {
		t.Fatal("Expected host to be set after registration")
	}

	// Unregister last client
	hub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	// Host should be reset
	hub.mu.RLock()
	hostAfter := hub.hostID
	hub.mu.RUnlock()

	if hostAfter != "" {
		t.Errorf("Expected host to be empty after last client leaves, got %s", hostAfter)
	}
}

func TestHub_KickUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "TheHost")
	victim := newMockClient(hub, "TheVictim")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(victim)
	time.Sleep(30 * time.Millisecond)

	// Verify host is host
	hub.mu.RLock()
	currentHost := hub.hostID
	hub.mu.RUnlock()
	if currentHost != host.ID {
		t.Fatalf("Setup failed: expected %s to be host", host.ID)
	}

	// Host kicks victim
	err := hub.KickUser(host.ID, victim.ID)
	if err != nil {
		t.Errorf("Kick failed: %v", err)
	}

	// Wait for async processing (kick has 500ms delay)
	time.Sleep(600 * time.Millisecond)

	// Victim should be gone
	hub.mu.RLock()
	_, exists := hub.clients[victim.ID]
	hub.mu.RUnlock()

	if exists {
		t.Error("Victim should have been removed from clients map")
	}
}

func TestHub_KickUserNotHost(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "TheHost")
	intruder := newMockClient(hub, "TheIntruder")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(intruder)

	// Intruder tries to kick host
	hub.KickUser(intruder.ID, host.ID)
	time.Sleep(50 * time.Millisecond)

	// Host should still exist
	hub.mu.RLock()
	_, exists := hub.clients[host.ID]
	hub.mu.RUnlock()

	if !exists {
		t.Error("Host should not have been kicked by non-host")
	}
}

func TestHub_TransferHost(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "TheHost")
	successor := newMockClient(hub, "TheSuccessor")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(successor)
	time.Sleep(30 * time.Millisecond) // Wait for registration

	// Transfer host role
	hub.TransferHost(host.ID, successor.ID)
	
	hub.mu.RLock()
	newHost := hub.hostID
	hub.mu.RUnlock()

	if newHost != successor.ID {
		t.Errorf("Expected host to be %s, got %s", successor.ID, newHost)
	}
}

func TestHub_TransferHostNotHost(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "TheHost")
	imposter := newMockClient(hub, "TheImposter")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(imposter)

	// Imposter tries to steal host role
	hub.TransferHost(imposter.ID, imposter.ID)

	hub.mu.RLock()
	currentHost := hub.hostID
	hub.mu.RUnlock()

	if currentHost != host.ID {
		t.Error("Host role should not change when requested by non-host")
	}
}

func TestHub_HostReclaimOnReconnect(t *testing.T) {
	hub := NewHub()
	hub.leaveDelay = 10 * time.Millisecond 
	hub.hostTransferDelay = 200 * time.Millisecond // Long enough to survive "refresh"
	go hub.Run()

	// 1. Original host connects
	originalHost := newMockClient(hub, "MyPersona")
	hub.Register(originalHost)
	time.Sleep(30 * time.Millisecond)

	// Verify initial host
	hub.mu.RLock()
	if hub.hostPersona != "MyPersona" || hub.hostID != originalHost.ID {
		t.Fatalf("Setup failed: expected MyPersona to be host")
	}
	hub.mu.RUnlock()

	// 2. Another user joins
	tempHost := newMockClient(hub, "OtherUser")
	hub.Register(tempHost)
	time.Sleep(30 * time.Millisecond)

	// 3. Original host disconnects (simulation of page refresh)
	hub.Unregister(originalHost)
	time.Sleep(50 * time.Millisecond) // Wait for leaveDelay (10ms) but LESS than hostTransferDelay (200ms)

	// Verify ID and Perks remain with the Disconnected Host (Sticky Host)
	hub.mu.RLock()
	// NOTE: Sticky Host logic means hostID does NOT change immediately
	if hub.hostID == tempHost.ID {
		t.Errorf("Expected hostID to REMAIN with originalHost (sticky), but it transferred too early")
	}
	if hub.hostPersona != "MyPersona" {
		t.Errorf("Expected hostPersona to remain 'MyPersona', got %s", hub.hostPersona)
	}
	hub.mu.RUnlock()

	// 4. Original host reconnects (new ID, same Persona)
	reconnectingHost := newMockClient(hub, "MyPersona") 
	hub.Register(reconnectingHost)
	time.Sleep(50 * time.Millisecond)

	// Verify reclaim
	hub.mu.RLock()
	finalHostID := hub.hostID
	finalHostPersona := hub.hostPersona
	hub.mu.RUnlock()

	if finalHostID != reconnectingHost.ID {
		t.Errorf("Expected returning host to reclaim hostID. Got %s, want %s", finalHostID, reconnectingHost.ID)
	}
	if finalHostPersona != "MyPersona" {
		t.Errorf("Expected hostPersona to remain MyPersona")
	}
}


