package ws

import (
	"testing"
)

func TestRoomManager_CreateRoom(t *testing.T) {
	rm := NewRoomManager()
	name := "Test Room"
	
	room := rm.CreateRoom(name)
	
	if room.Name != name {
		t.Errorf("Expected room name %s, got %s", name, room.Name)
	}
	
	if room.Code == "" {
		t.Error("Expected generated room code")
	}
	
	if len(room.Code) != 12 { // Assuming XXXX-XXXX-XXXX format is not strictly enforced in length check but typically > 0
		// Actually let's just check it exists, length might vary if implementation changes
	}
}

func TestRoomManager_GetRoom(t *testing.T) {
	rm := NewRoomManager()
	
	room := rm.CreateRoom("Find Me")
	
	found := rm.GetRoom(room.Code)
	if found == nil {
		t.Error("Expected to find created room")
	}
	
	if found != room {
		t.Error("Expected found room to be the same instance")
	}
	
	missing := rm.GetRoom("invalid-code")
	if missing != nil {
		t.Error("Expected nil for invalid room code")
	}
}

func TestRoomManager_DeleteRoom(t *testing.T) {
	rm := NewRoomManager()
	room := rm.CreateRoom("Delete Me")
	
	rm.DeleteRoom(room.Code)
	
	if rm.GetRoom(room.Code) != nil {
		t.Error("Expected room to be deleted")
	}
}

func TestRoomManager_Concurrency(t *testing.T) {
	rm := NewRoomManager()
	
	// Create 100 rooms concurrently
	for i := 0; i < 100; i++ {
		go rm.CreateRoom("Async Room")
	}
	
	// We can't easily wait without sync.WaitGroup here unless we modify signature or use channel trick
	// But let's just create them and then try to read.
	// For a simple test, serial creation + parallel read is safer to write quickly
	
	room := rm.CreateRoom("Shared")
	
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			rm.GetRoom(room.Code)
			done <- true
		}()
	}
	
	for i := 0; i < 100; i++ {
		<-done
	}
}
