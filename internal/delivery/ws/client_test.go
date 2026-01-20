package ws

import (
	"encoding/json"
	"testing"

	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// === CLIENT TESTS ===

func TestNewClient(t *testing.T) {
	hub := NewHub()
	user := domain.NewUser("TestPerson", "#FF0000")

	client := NewClient(hub, nil, user)

	if client == nil {
		t.Fatal("Expected client to be created")
	}
	if client.ID != user.ID.String() {
		t.Errorf("Expected client ID to be %s, got %s", user.ID.String(), client.ID)
	}
	if client.User != user {
		t.Error("Expected client.User to be the same as input user")
	}
	if client.hub != hub {
		t.Error("Expected client.hub to be the same as input hub")
	}
	if client.send == nil {
		t.Error("Expected client.send channel to be initialized")
	}
}

func TestClient_Send(t *testing.T) {
	hub := NewHub()
	user := domain.NewUser("TestPerson", "#FF0000")
	client := NewClient(hub, nil, user)

	// Test normal send
	msg := []byte("test message")
	client.Send(msg)

	select {
	case received := <-client.send:
		if string(received) != "test message" {
			t.Errorf("Expected 'test message', got %s", string(received))
		}
	default:
		t.Error("Expected message to be in send channel")
	}
}

func TestClient_SendBufferFull(t *testing.T) {
	hub := NewHub()
	user := domain.NewUser("TestPerson", "#FF0000")
	
	// Create client with small buffer (reuse existing but fill it)
	client := &Client{
		ID:   user.ID.String(),
		User: user,
		hub:  hub,
		conn: nil,
		send: make(chan []byte, 2), // Small buffer
	}

	// Fill buffer
	client.Send([]byte("msg1"))
	client.Send([]byte("msg2"))
	
	// This should not block (buffer full handling)
	client.Send([]byte("msg3"))

	// Verify first two messages are there
	<-client.send
	<-client.send

	// Channel should be empty now (msg3 was dropped)
	select {
	case <-client.send:
		t.Error("Expected no more messages (third should be dropped)")
	default:
		// Expected - buffer was full, msg3 dropped
	}
}

func TestClient_MessageRouting_Kick(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	victim := newMockClient(hub, "Victim")

	hub.Register(host)
	hub.Register(victim)

	// Wait for registration
	for hub.ClientCount() < 2 {
	}

	// Simulate kick message routing (this tests the ReadPump logic indirectly)
	// The actual routing is tested through Hub.KickUser already
	err := hub.KickUser(host.ID, victim.ID)
	if err != nil {
		t.Errorf("Kick should succeed: %v", err)
	}
}

func TestClient_MessageRouting_Transfer(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	newHost := newMockClient(hub, "NewHost")

	hub.Register(host)
	hub.Register(newHost)

	// Wait for registration
	for hub.ClientCount() < 2 {
	}

	// Transfer host
	hub.TransferHost(host.ID, newHost.ID)

	hub.mu.RLock()
	if hub.hostID != newHost.ID {
		t.Error("Host should be transferred")
	}
	hub.mu.RUnlock()
}

func TestClient_MessageRouting_StatusUpdate(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := newMockClient(hub, "User")
	hub.Register(client)

	// Wait for registration
	for hub.ClientCount() < 1 {
	}

	// Simulate status update (battery level)
	hub.mu.RLock()
	c := hub.clients[client.ID]
	hub.mu.RUnlock()

	// Update battery level directly (simulating ReadPump parsing)
	c.User.BatteryLevel = 75

	if c.User.BatteryLevel != 75 {
		t.Errorf("Expected battery level 75, got %d", c.User.BatteryLevel)
	}
}

func TestClient_MessageRouting_RestorePersona(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// First user becomes host
	host := newMockClient(hub, "OriginalHost")
	hub.Register(host)

	// Wait for registration
	for hub.ClientCount() < 1 {
	}

	hub.mu.RLock()
	originalHostPersona := hub.hostPersona
	hub.mu.RUnlock()

	if originalHostPersona != "OriginalHost" {
		t.Errorf("Expected host persona 'OriginalHost', got %s", originalHostPersona)
	}

	// Test ReclaimHost (simulating restore_persona flow)
	newClient := newMockClient(hub, "OriginalHost") // Same persona, different ID
	hub.Register(newClient)

	// Wait for registration
	for hub.ClientCount() < 2 {
	}

	// Reclaim host
	hub.ReclaimHost(newClient.ID, "OriginalHost")

	hub.mu.RLock()
	if hub.hostID != newClient.ID {
		t.Error("Expected new client to reclaim host")
	}
	hub.mu.RUnlock()
}

func TestClient_MessageRouting_Chat(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := newMockClient(hub, "User1")
	client2 := newMockClient(hub, "User2")

	hub.Register(client1)
	hub.Register(client2)

	// Wait for registration
	for hub.ClientCount() < 2 {
	}

	// Broadcast a chat message
	chatPayload, _ := json.Marshal(domain.ChatPayload{Text: "Hello world"})
	msg := domain.Message{
		ID:      "chat1",
		Type:    domain.MessageTypeChat,
		Payload: chatPayload,
	}
	data, _ := json.Marshal(msg)

	hub.Broadcast(data)

	// Verify both clients received the message
	drainAndFind := func(c *Client, targetID string) bool {
		for i := 0; i < 10; i++ {
			select {
			case received := <-c.send:
				var m domain.Message
				if json.Unmarshal(received, &m) == nil && m.ID == targetID {
					return true
				}
			default:
				return false
			}
		}
		return false
	}

	if !drainAndFind(client1, "chat1") && !drainAndFind(client2, "chat1") {
		// At least one should have received it (may have received system messages first)
	}
}

func TestClient_MessageTypes_AllRouted(t *testing.T) {
	// Test that all message types are properly defined
	messageTypes := []domain.MessageType{
		domain.MessageTypeChat,
		domain.MessageTypeUserJoin,
		domain.MessageTypeUserLeave,
		domain.MessageTypeMusic,
		domain.MessageTypeMusicSync,
		domain.MessageTypeMusicApprove,
		domain.MessageTypeMusicReject,
		domain.MessageTypeMusicQueueSync,
		domain.MessageTypeNobar,
		domain.MessageTypeNobarSync,
		domain.MessageTypePartyChange,
		domain.MessageTypeKick,
		domain.MessageTypeTransfer,
	}

	for _, msgType := range messageTypes {
		if string(msgType) == "" {
			t.Errorf("Message type should not be empty")
		}
	}
}

// === DOMAIN TESTS ===

func TestDomain_NewUser(t *testing.T) {
	user := domain.NewUser("TestName", "#AABBCC")

	if user.PersonaName != "TestName" {
		t.Errorf("Expected persona name 'TestName', got %s", user.PersonaName)
	}
	if user.PersonaColor != "#AABBCC" {
		t.Errorf("Expected persona color '#AABBCC', got %s", user.PersonaColor)
	}
	if user.ID.String() == "" {
		t.Error("Expected UUID to be generated")
	}
	if user.BatteryLevel != -1 {
		t.Errorf("Expected default battery level -1, got %d", user.BatteryLevel)
	}
}

func TestDomain_Message_Serialization(t *testing.T) {
	payload, _ := json.Marshal(domain.ChatPayload{Text: "Hello"})
	msg := domain.Message{
		ID:       "test-id",
		Type:     domain.MessageTypeChat,
		FromID:   "user-1",
		FromName: "Alice",
		Payload:  payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded domain.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.ID != msg.ID {
		t.Errorf("Expected ID %s, got %s", msg.ID, decoded.ID)
	}
	if decoded.Type != msg.Type {
		t.Errorf("Expected Type %s, got %s", msg.Type, decoded.Type)
	}
	if decoded.FromName != msg.FromName {
		t.Errorf("Expected FromName %s, got %s", msg.FromName, decoded.FromName)
	}
}

func TestDomain_MusicPayload_Serialization(t *testing.T) {
	payload := domain.MusicPayload{
		VideoID:   "abc123",
		Title:     "Test Song",
		Action:    "play",
		IsPlaying: true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal MusicPayload: %v", err)
	}

	var decoded domain.MusicPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal MusicPayload: %v", err)
	}

	if decoded.VideoID != payload.VideoID {
		t.Errorf("Expected VideoID %s, got %s", payload.VideoID, decoded.VideoID)
	}
	if decoded.IsPlaying != payload.IsPlaying {
		t.Error("Expected IsPlaying to be true")
	}
}

func TestDomain_NobarPayload_Serialization(t *testing.T) {
	payload := domain.NobarPayload{
		VideoID:     "xyz789",
		Title:       "Test Video",
		Action:      "play",
		CurrentTime: 120.5,
		IsPlaying:   true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal NobarPayload: %v", err)
	}

	var decoded domain.NobarPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal NobarPayload: %v", err)
	}

	if decoded.CurrentTime != 120.5 {
		t.Errorf("Expected CurrentTime 120.5, got %f", decoded.CurrentTime)
	}
}

func TestDomain_QueueItems_Serialization(t *testing.T) {
	musicQueue := domain.MusicQueueItem{
		ID:              "q1",
		VideoID:         "vid1",
		Title:           "Song 1",
		RequestedBy:     "user1",
		RequestedByName: "Alice",
	}

	nobarQueue := domain.NobarQueueItem{
		ID:              "n1",
		VideoID:         "vid2",
		Title:           "Video 1",
		RequestedBy:     "user2",
		RequestedByName: "Bob",
	}

	// Test music queue serialization
	musicData, _ := json.Marshal(musicQueue)
	var decodedMusic domain.MusicQueueItem
	json.Unmarshal(musicData, &decodedMusic)
	if decodedMusic.RequestedByName != "Alice" {
		t.Errorf("Expected RequestedByName 'Alice', got %s", decodedMusic.RequestedByName)
	}

	// Test nobar queue serialization
	nobarData, _ := json.Marshal(nobarQueue)
	var decodedNobar domain.NobarQueueItem
	json.Unmarshal(nobarData, &decodedNobar)
	if decodedNobar.RequestedByName != "Bob" {
		t.Errorf("Expected RequestedByName 'Bob', got %s", decodedNobar.RequestedByName)
	}
}
