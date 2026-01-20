package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// === NOBAR (WATCH TOGETHER) FEATURE TESTS ===

func TestHub_HandleNobar_Play(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "NobarHost")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Host starts nobar
	payload := domain.NobarPayload{
		Action:  "play",
		VideoID: "dQw4w9WgXcQ",
		Title:   "Epic Video",
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := domain.Message{
		ID:      "nobar1",
		Type:    domain.MessageTypeNobar,
		Payload: payloadBytes,
	}

	hub.HandleNobar(host, msg)

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentNobar == nil {
		t.Fatal("Expected nobar session to be active")
	}
	if hub.currentNobar.VideoID != "dQw4w9WgXcQ" {
		t.Errorf("Expected video ID 'dQw4w9WgXcQ', got %s", hub.currentNobar.VideoID)
	}
	if !hub.currentNobar.IsPlaying {
		t.Error("Expected nobar to be playing")
	}
}

func TestHub_HandleNobar_AutoPausesMusic(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Music is currently playing
	hub.mu.Lock()
	hub.currentMusic = &domain.MusicPayload{
		VideoID:   "mNO345pqr56",
		IsPlaying: true,
	}
	hub.mu.Unlock()

	// Start nobar
	payload := domain.NobarPayload{Action: "play", VideoID: "abc123XYZ_-", Title: "Video"}
	payloadBytes, _ := json.Marshal(payload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: payloadBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Music should be paused
	if hub.currentMusic.IsPlaying {
		t.Error("Expected music to be auto-paused when nobar starts")
	}
	if hub.currentMusic.Action != "pause" {
		t.Errorf("Expected music action to be 'pause', got %s", hub.currentMusic.Action)
	}
	// Nobar should be active
	if hub.currentNobar == nil || !hub.currentNobar.IsPlaying {
		t.Error("Expected nobar to be playing")
	}
}

func TestHub_HandleNobar_ResetsPartyMode(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Party mode is active
	hub.mu.Lock()
	hub.currentPartyMode = "party"
	hub.mu.Unlock()

	// Start nobar
	payload := domain.NobarPayload{Action: "play", VideoID: "abc123XYZ_-", Title: "Video"}
	payloadBytes, _ := json.Marshal(payload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: payloadBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentPartyMode != "normal" {
		t.Errorf("Expected party mode to reset to 'normal', got %s", hub.currentPartyMode)
	}
}

func TestHub_HandleNobar_Pause(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Start nobar first
	playPayload := domain.NobarPayload{Action: "play", VideoID: "abc123XYZ_-", Title: "Video"}
	playBytes, _ := json.Marshal(playPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: playBytes})

	// Wait a bit to simulate time passing
	time.Sleep(100 * time.Millisecond)

	// Pause
	pausePayload := domain.NobarPayload{Action: "pause"}
	pauseBytes, _ := json.Marshal(pausePayload)
	hub.HandleNobar(host, domain.Message{ID: "n2", Type: domain.MessageTypeNobar, Payload: pauseBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentNobar.IsPlaying {
		t.Error("Expected nobar to be paused")
	}
	// CurrentTime should have been calculated (elapsed time added)
	if hub.currentNobar.CurrentTime <= 0 {
		t.Error("Expected CurrentTime to be updated after pause")
	}
}

func TestHub_HandleNobar_Resume(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar is paused
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{
		VideoID:     "abc123XYZ_-",
		IsPlaying:   false,
		CurrentTime: 30.0,
	}
	hub.mu.Unlock()

	// Resume
	resumePayload := domain.NobarPayload{Action: "resume"}
	resumeBytes, _ := json.Marshal(resumePayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: resumeBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if !hub.currentNobar.IsPlaying {
		t.Error("Expected nobar to be playing after resume")
	}
}

func TestHub_HandleNobar_Stop(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar active with queue and viewers
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "abc123XYZ_-", IsPlaying: true}
	hub.nobarQueue = []domain.NobarQueueItem{{ID: "q1", VideoID: "xyz789ABC_-"}}
	hub.nobarRequests = []domain.NobarQueueItem{{ID: "r1", VideoID: "qwe456RTY_-"}}
	hub.nobarViewers = map[string]domain.NobarViewer{
		"viewer1": {ID: "v1", PersonaName: "Viewer"},
	}
	hub.currentPartyMode = "party"
	hub.mu.Unlock()

	// Stop
	stopPayload := domain.NobarPayload{Action: "stop"}
	stopBytes, _ := json.Marshal(stopPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: stopBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentNobar != nil {
		t.Error("Expected nobar to be nil after stop")
	}
	if len(hub.nobarQueue) != 0 {
		t.Error("Expected nobar queue to be cleared")
	}
	if len(hub.nobarRequests) != 0 {
		t.Error("Expected nobar requests to be cleared")
	}
	if len(hub.nobarViewers) != 0 {
		t.Error("Expected nobar viewers to be cleared")
	}
	if hub.currentPartyMode != "normal" {
		t.Error("Expected party mode to reset to normal")
	}
}

func TestHub_HandleNobar_Seek(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar active
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "abc123XYZ_-", IsPlaying: true}
	hub.mu.Unlock()

	// Seek
	seekPayload := domain.NobarPayload{Action: "seek", CurrentTime: 120.0}
	seekBytes, _ := json.Marshal(seekPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: seekBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentNobar.CurrentTime != 120.0 {
		t.Errorf("Expected current time to be 120.0, got %f", hub.currentNobar.CurrentTime)
	}
}

func TestHub_HandleNobar_UserRequest(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// User requests a video
	requestPayload := domain.NobarPayload{Action: "request", VideoID: "aBC123def45", Title: "User Video"}
	requestBytes, _ := json.Marshal(requestPayload)
	hub.HandleNobar(user, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: requestBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if len(hub.nobarRequests) != 1 {
		t.Fatalf("Expected 1 nobar request, got %d", len(hub.nobarRequests))
	}
	if hub.nobarRequests[0].VideoID != "aBC123def45" {
		t.Errorf("Expected video ID 'aBC123def45', got %s", hub.nobarRequests[0].VideoID)
	}
}

func TestHub_HandleNobar_HostRequestAddsToQueue(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar already playing something
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "abc123XYZ_-", IsPlaying: true}
	hub.mu.Unlock()

	// Host adds another video
	requestPayload := domain.NobarPayload{Action: "play", VideoID: "dEF456ghi78", Title: "Host Video"}
	requestBytes, _ := json.Marshal(requestPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: requestBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Should be in queue (not requests)
	if len(hub.nobarQueue) != 1 {
		t.Fatalf("Expected 1 item in nobar queue, got %d", len(hub.nobarQueue))
	}
	if hub.nobarQueue[0].VideoID != "dEF456ghi78" {
		t.Errorf("Expected queue item 'dEF456ghi78', got %s", hub.nobarQueue[0].VideoID)
	}
	// Requests should be empty (host bypasses approval)
	if len(hub.nobarRequests) != 0 {
		t.Error("Expected no pending requests for host")
	}
}

func TestHub_HandleNobar_Approve(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Pending request
	hub.mu.Lock()
	hub.nobarRequests = []domain.NobarQueueItem{
		{ID: "req1", VideoID: "gHI789jkl01", Title: "Pending"},
	}
	hub.mu.Unlock()

	// Approve
	approvePayload := domain.NobarPayload{Action: "approve", VideoID: "req1"}
	approveBytes, _ := json.Marshal(approvePayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: approveBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Request should be moved to queue/played
	if len(hub.nobarRequests) != 0 {
		t.Error("Expected requests to be empty after approval")
	}
	// Since nothing was playing, it should start playing
	if hub.currentNobar == nil {
		t.Fatal("Expected nobar to start playing after approval")
	}
	if hub.currentNobar.VideoID != "gHI789jkl01" {
		t.Errorf("Expected approved video to be playing, got %s", hub.currentNobar.VideoID)
	}
}

func TestHub_HandleNobar_Reject(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Pending request
	hub.mu.Lock()
	hub.nobarRequests = []domain.NobarQueueItem{
		{ID: "req1", VideoID: "jKL012mno34", Title: "Bad Video"},
	}
	hub.mu.Unlock()

	// Reject
	rejectPayload := domain.NobarPayload{Action: "reject", VideoID: "req1"}
	rejectBytes, _ := json.Marshal(rejectPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: rejectBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if len(hub.nobarRequests) != 0 {
		t.Error("Expected requests to be empty after rejection")
	}
	if hub.currentNobar != nil {
		t.Error("Expected no nobar to be playing after rejection")
	}
}

func TestHub_HandleNobar_Ended(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar active with queue
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "abc123XYZ_-", IsPlaying: true}
	hub.nobarQueue = []domain.NobarQueueItem{
		{ID: "q1", VideoID: "xyz789ABC_-", Title: "Video 2"},
	}
	hub.mu.Unlock()

	// Video ended
	endedPayload := domain.NobarPayload{Action: "ended"}
	endedBytes, _ := json.Marshal(endedPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: endedBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentNobar == nil {
		t.Fatal("Expected next video to be playing")
	}
	if hub.currentNobar.VideoID != "xyz789ABC_-" {
		t.Errorf("Expected xyz789ABC_- to auto-play, got %s", hub.currentNobar.VideoID)
	}
	if len(hub.nobarQueue) != 0 {
		t.Error("Expected queue to be empty after auto-play")
	}
}

func TestHub_HandleNobar_EndedStopsWhenQueueEmpty(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar active with empty queue
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "abc123XYZ_-", IsPlaying: true}
	hub.nobarQueue = []domain.NobarQueueItem{} // Empty queue
	hub.mu.Unlock()

	// Video ended
	endedPayload := domain.NobarPayload{Action: "ended"}
	endedBytes, _ := json.Marshal(endedPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: endedBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentNobar != nil {
		t.Error("Expected nobar to stop when queue is empty")
	}
}

func TestHub_HandleNobar_View(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	viewer := newMockClient(hub, "Viewer")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(viewer)
	time.Sleep(30 * time.Millisecond)

	// Viewer marks themselves as viewing
	viewPayload := domain.NobarPayload{Action: "view"}
	viewBytes, _ := json.Marshal(viewPayload)
	hub.HandleNobar(viewer, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: viewBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if len(hub.nobarViewers) != 1 {
		t.Fatalf("Expected 1 viewer, got %d", len(hub.nobarViewers))
	}
	if _, ok := hub.nobarViewers[viewer.ID]; !ok {
		t.Error("Expected viewer to be in nobarViewers map")
	}
}

func TestHub_HandleNobar_Unview(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	viewer := newMockClient(hub, "Viewer")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(viewer)
	time.Sleep(30 * time.Millisecond)

	// Setup: Viewer is viewing
	hub.mu.Lock()
	hub.nobarViewers[viewer.ID] = domain.NobarViewer{ID: viewer.ID, PersonaName: "Viewer"}
	hub.mu.Unlock()

	// Viewer stops viewing
	unviewPayload := domain.NobarPayload{Action: "unview"}
	unviewBytes, _ := json.Marshal(unviewPayload)
	hub.HandleNobar(viewer, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: unviewBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if len(hub.nobarViewers) != 0 {
		t.Error("Expected no viewers after unview")
	}
}

func TestHub_HandleNobar_NonHostPlayTreatedAsRequest(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// Non-host sends "play" action
	playPayload := domain.NobarPayload{Action: "play", VideoID: "aBC123def45", Title: "User Video"}
	playBytes, _ := json.Marshal(playPayload)
	hub.HandleNobar(user, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: playBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Should be treated as request, not play
	if hub.currentNobar != nil {
		t.Error("Non-host play should not start nobar directly")
	}
	if len(hub.nobarRequests) != 1 {
		t.Fatalf("Expected 1 pending request, got %d", len(hub.nobarRequests))
	}
}

func TestHub_HandleNobar_NonHostActionsIgnored(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar playing
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "abc123XYZ_-", IsPlaying: true}
	hub.mu.Unlock()

	// Non-host tries to pause
	pausePayload := domain.NobarPayload{Action: "pause"}
	pauseBytes, _ := json.Marshal(pausePayload)
	hub.HandleNobar(user, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: pauseBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if !hub.currentNobar.IsPlaying {
		t.Error("Non-host should not be able to pause nobar")
	}
}

func TestHub_HandleNobar_StopResumesMusic(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: Nobar active, music in queue
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "abc123XYZ_-", IsPlaying: true}
	hub.musicQueue = []domain.MusicQueueItem{
		{ID: "m1", VideoID: "mNO345pqr56", Title: "Song 1"},
	}
	hub.mu.Unlock()

	// Stop nobar
	stopPayload := domain.NobarPayload{Action: "stop"}
	stopBytes, _ := json.Marshal(stopPayload)
	hub.HandleNobar(host, domain.Message{ID: "n1", Type: domain.MessageTypeNobar, Payload: stopBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Music should resume from queue
	if hub.currentMusic == nil {
		t.Fatal("Expected music to resume after nobar stops")
	}
	if hub.currentMusic.VideoID != "mNO345pqr56" {
		t.Errorf("Expected mNO345pqr56 to play, got %s", hub.currentMusic.VideoID)
	}
}
