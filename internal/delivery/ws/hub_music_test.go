package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mmuslimabdulj/goat-chat/internal/domain"
)

// === MUSIC FEATURE TESTS ===

func TestHub_HandleMusic_HostPlayNothingPlaying(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "MusicHost")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Construct music play message
	payload := domain.MusicPayload{
		Action:  "play",
		VideoID: "dQw4w9WgXcQ",
		Title:   "Never Gonna Give You Up",
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := &domain.Message{
		ID:      "test-music-1",
		Type:    domain.MessageTypeMusic,
		Payload: payloadBytes,
	}

	// Host plays music (nothing currently playing)
	hub.HandleMusic(host, msg)

	// Verify music is now playing
	hub.mu.RLock()
	currentMusic := hub.currentMusic
	hub.mu.RUnlock()

	if currentMusic == nil {
		t.Fatal("Expected music to be playing")
	}
	if currentMusic.VideoID != "dQw4w9WgXcQ" {
		t.Errorf("Expected video ID 'dQw4w9WgXcQ', got %s", currentMusic.VideoID)
	}
	if !currentMusic.IsPlaying {
		t.Error("Expected IsPlaying to be true")
	}
}

func TestHub_HandleMusic_HostPlayAddsToQueue(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "MusicHost")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Play first song
	payload1 := domain.MusicPayload{Action: "play", VideoID: "dQw4w9WgXcQ", Title: "Song 1"}
	payloadBytes1, _ := json.Marshal(payload1)
	msg1 := &domain.Message{ID: "m1", Type: domain.MessageTypeMusic, Payload: payloadBytes1}
	hub.HandleMusic(host, msg1)

	// Play second song while first is playing
	payload2 := domain.MusicPayload{Action: "play", VideoID: "xvFZjo5PgG0", Title: "Song 2"}
	payloadBytes2, _ := json.Marshal(payload2)
	msg2 := &domain.Message{ID: "m2", Type: domain.MessageTypeMusic, Payload: payloadBytes2}
	hub.HandleMusic(host, msg2)

	// Verify first song is playing
	hub.mu.RLock()
	if hub.currentMusic.VideoID != "dQw4w9WgXcQ" {
		t.Errorf("Expected current music to be 'dQw4w9WgXcQ', got %s", hub.currentMusic.VideoID)
	}
	// Verify second song is in queue
	if len(hub.musicQueue) != 1 {
		t.Fatalf("Expected 1 song in queue, got %d", len(hub.musicQueue))
	}
	if hub.musicQueue[0].VideoID != "xvFZjo5PgG0" {
		t.Errorf("Expected queue[0] to be 'xvFZjo5PgG0', got %s", hub.musicQueue[0].VideoID)
	}
	hub.mu.RUnlock()
}

func TestHub_HandleMusic_UserRequestAddsToPending(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "RegularUser")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// User (non-host) requests a song
	payload := domain.MusicPayload{Action: "play", VideoID: "kJQP7kiw5Fk", Title: "User Song"}
	payloadBytes, _ := json.Marshal(payload)
	msg := &domain.Message{ID: "um1", Type: domain.MessageTypeMusic, Payload: payloadBytes}

	hub.HandleMusic(user, msg)

	// Verify song is in pending queue, NOT playing
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentMusic != nil {
		t.Error("Expected no music playing (user request needs approval)")
	}
	if len(hub.pendingQueue) != 1 {
		t.Fatalf("Expected 1 pending request, got %d", len(hub.pendingQueue))
	}
	if hub.pendingQueue[0].VideoID != "kJQP7kiw5Fk" {
		t.Errorf("Expected pending song 'kJQP7kiw5Fk', got %s", hub.pendingQueue[0].VideoID)
	}
}

func TestHub_HandleMusicApprove(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// User requests a song
	payload := domain.MusicPayload{Action: "play", VideoID: "L_jWHffIx5E", Title: "Pending Song"}
	payloadBytes, _ := json.Marshal(payload)
	msg := &domain.Message{ID: "pm1", Type: domain.MessageTypeMusic, Payload: payloadBytes}
	hub.HandleMusic(user, msg)

	// Get the request ID
	hub.mu.RLock()
	if len(hub.pendingQueue) == 0 {
		hub.mu.RUnlock()
		t.Fatal("Expected pending request")
	}
	requestID := hub.pendingQueue[0].ID
	hub.mu.RUnlock()

	// Host approves
	approvePayload := domain.MusicApprovePayload{RequestID: requestID}
	approveBytes, _ := json.Marshal(approvePayload)
	approveMsg := &domain.Message{ID: "approve1", Type: domain.MessageTypeMusicApprove, Payload: approveBytes}

	hub.HandleMusicApprove(host, approveMsg)

	// Verify: pending cleared, song playing (since nothing was playing)
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if len(hub.pendingQueue) != 0 {
		t.Errorf("Expected pending queue to be empty, got %d", len(hub.pendingQueue))
	}
	if hub.currentMusic == nil {
		t.Fatal("Expected music to be playing after approval")
	}
	if hub.currentMusic.VideoID != "L_jWHffIx5E" {
		t.Errorf("Expected approved song to be playing, got %s", hub.currentMusic.VideoID)
	}
}

func TestHub_HandleMusicApprove_NonHostIgnored(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// Add a pending request manually
	hub.mu.Lock()
	hub.pendingQueue = append(hub.pendingQueue, domain.MusicQueueItem{
		ID:      "req1",
		VideoID: "dQw4w9WgXcQ",
		Title:   "Song 1",
	})
	hub.mu.Unlock()

	// Non-host tries to approve
	approvePayload := domain.MusicApprovePayload{RequestID: "req1"}
	approveBytes, _ := json.Marshal(approvePayload)
	approveMsg := &domain.Message{ID: "a1", Type: domain.MessageTypeMusicApprove, Payload: approveBytes}

	hub.HandleMusicApprove(user, approveMsg)

	// Verify: pending NOT cleared (non-host can't approve)
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if len(hub.pendingQueue) != 1 {
		t.Error("Non-host should not be able to approve requests")
	}
}

func TestHub_HandleMusicReject(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Add pending request
	hub.mu.Lock()
	hub.pendingQueue = append(hub.pendingQueue, domain.MusicQueueItem{
		ID:      "reject1",
		VideoID: "hTWKbfoikeg",
		Title:   "Bad Song",
	})
	hub.mu.Unlock()

	// Host rejects
	rejectPayload := domain.MusicRejectPayload{RequestID: "reject1"}
	rejectBytes, _ := json.Marshal(rejectPayload)
	rejectMsg := &domain.Message{ID: "r1", Type: domain.MessageTypeMusicReject, Payload: rejectBytes}

	hub.HandleMusicReject(host, rejectMsg)

	// Verify: pending cleared
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if len(hub.pendingQueue) != 0 {
		t.Error("Expected pending queue to be empty after rejection")
	}
}

func TestHub_HandleMusic_PauseResume(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Start playing
	playPayload := domain.MusicPayload{Action: "play", VideoID: "dQw4w9WgXcQ", Title: "Song 1"}
	playBytes, _ := json.Marshal(playPayload)
	playMsg := &domain.Message{ID: "p1", Type: domain.MessageTypeMusic, Payload: playBytes}
	hub.HandleMusic(host, playMsg)

	// Pause
	pausePayload := domain.MusicPayload{Action: "pause"}
	pauseBytes, _ := json.Marshal(pausePayload)
	pauseMsg := &domain.Message{ID: "pause1", Type: domain.MessageTypeMusic, Payload: pauseBytes}
	hub.HandleMusic(host, pauseMsg)

	hub.mu.RLock()
	if hub.currentMusic.IsPlaying {
		t.Error("Expected music to be paused")
	}
	hub.mu.RUnlock()

	// Resume
	resumePayload := domain.MusicPayload{Action: "resume"}
	resumeBytes, _ := json.Marshal(resumePayload)
	resumeMsg := &domain.Message{ID: "resume1", Type: domain.MessageTypeMusic, Payload: resumeBytes}
	hub.HandleMusic(host, resumeMsg)

	hub.mu.RLock()
	if !hub.currentMusic.IsPlaying {
		t.Error("Expected music to be playing after resume")
	}
	hub.mu.RUnlock()
}

func TestHub_HandleMusic_Stop(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Start playing and add to queue
	playPayload := domain.MusicPayload{Action: "play", VideoID: "dQw4w9WgXcQ", Title: "Song 1"}
	playBytes, _ := json.Marshal(playPayload)
	playMsg := &domain.Message{ID: "p1", Type: domain.MessageTypeMusic, Payload: playBytes}
	hub.HandleMusic(host, playMsg)

	play2Payload := domain.MusicPayload{Action: "play", VideoID: "xvFZjo5PgG0", Title: "Song 2"}
	play2Bytes, _ := json.Marshal(play2Payload)
	play2Msg := &domain.Message{ID: "p2", Type: domain.MessageTypeMusic, Payload: play2Bytes}
	hub.HandleMusic(host, play2Msg)

	// Stop
	stopPayload := domain.MusicPayload{Action: "stop"}
	stopBytes, _ := json.Marshal(stopPayload)
	stopMsg := &domain.Message{ID: "stop1", Type: domain.MessageTypeMusic, Payload: stopBytes}
	hub.HandleMusic(host, stopMsg)

	time.Sleep(20 * time.Millisecond) // Wait for broadcast

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentMusic != nil {
		t.Error("Expected no music after stop")
	}
	if len(hub.musicQueue) != 0 {
		t.Error("Expected queue to be cleared after stop")
	}
}

func TestHub_HandleMusic_Next(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Play two songs
	play1 := domain.MusicPayload{Action: "play", VideoID: "dQw4w9WgXcQ", Title: "Song 1"}
	p1Bytes, _ := json.Marshal(play1)
	hub.HandleMusic(host, &domain.Message{ID: "p1", Type: domain.MessageTypeMusic, Payload: p1Bytes})

	play2 := domain.MusicPayload{Action: "play", VideoID: "xvFZjo5PgG0", Title: "Song 2"}
	p2Bytes, _ := json.Marshal(play2)
	hub.HandleMusic(host, &domain.Message{ID: "p2", Type: domain.MessageTypeMusic, Payload: p2Bytes})

	// Skip to next
	nextPayload := domain.MusicPayload{Action: "next"}
	nextBytes, _ := json.Marshal(nextPayload)
	hub.HandleMusic(host, &domain.Message{ID: "next1", Type: domain.MessageTypeMusic, Payload: nextBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentMusic == nil {
		t.Fatal("Expected music to be playing")
	}
	if hub.currentMusic.VideoID != "xvFZjo5PgG0" {
		t.Errorf("Expected xvFZjo5PgG0 to be playing after next, got %s", hub.currentMusic.VideoID)
	}
	if len(hub.musicQueue) != 0 {
		t.Error("Expected queue to be empty after next")
	}
}

func TestHub_HandleMusic_Seek(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Start playing
	playPayload := domain.MusicPayload{Action: "play", VideoID: "dQw4w9WgXcQ", Title: "Song 1"}
	playBytes, _ := json.Marshal(playPayload)
	hub.HandleMusic(host, &domain.Message{ID: "p1", Type: domain.MessageTypeMusic, Payload: playBytes})

	// Seek to 60 seconds
	seekPayload := domain.MusicPayload{Action: "seek", CurrentTime: 60.0}
	seekBytes, _ := json.Marshal(seekPayload)
	hub.HandleMusic(host, &domain.Message{ID: "seek1", Type: domain.MessageTypeMusic, Payload: seekBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentMusic.CurrentTime != 60.0 {
		t.Errorf("Expected current time to be 60.0, got %f", hub.currentMusic.CurrentTime)
	}
}

func TestHub_HandleMusic_SongEnded(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: current music + queue
	hub.mu.Lock()
	hub.currentMusic = &domain.MusicPayload{
		VideoID:   "dQw4w9WgXcQ",
		IsPlaying: true,
		StartTime: time.Now().Add(-10 * time.Second), // Started 10s ago to pass debounce
	}
	hub.musicQueue = []domain.MusicQueueItem{
		{ID: "q1", VideoID: "xvFZjo5PgG0", Title: "Song 2"},
	}
	hub.mu.Unlock()

	// Song ended
	endedPayload := domain.MusicPayload{Action: "ended"}
	endedBytes, _ := json.Marshal(endedPayload)
	hub.HandleMusic(host, &domain.Message{ID: "e1", Type: domain.MessageTypeMusic, Payload: endedBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.currentMusic == nil {
		t.Fatal("Expected next song to be playing")
	}
	if hub.currentMusic.VideoID != "xvFZjo5PgG0" {
		t.Errorf("Expected xvFZjo5PgG0 to auto-play, got %s", hub.currentMusic.VideoID)
	}
}

func TestHub_HandleMusic_SongEndedDebounce(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Setup: current music started very recently (less than 5s ago)
	hub.mu.Lock()
	hub.currentMusic = &domain.MusicPayload{
		VideoID:   "dQw4w9WgXcQ",
		IsPlaying: true,
		StartTime: time.Now(), // Just started!
	}
	hub.musicQueue = []domain.MusicQueueItem{
		{ID: "q1", VideoID: "xvFZjo5PgG0", Title: "Song 2"},
	}
	hub.mu.Unlock()

	// Song ended (should be ignored due to debounce)
	endedPayload := domain.MusicPayload{Action: "ended"}
	endedBytes, _ := json.Marshal(endedPayload)
	hub.HandleMusic(host, &domain.Message{ID: "e1", Type: domain.MessageTypeMusic, Payload: endedBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Should still be playing dQw4w9WgXcQ (debounce ignored the ended event)
	if hub.currentMusic.VideoID != "dQw4w9WgXcQ" {
		t.Errorf("Expected dQw4w9WgXcQ to still be playing (debounce), got %s", hub.currentMusic.VideoID)
	}
	if len(hub.musicQueue) != 1 {
		t.Error("Expected queue to still have 1 song (debounce)")
	}
}

func TestHub_HandleMusic_NobarBlocksPlay(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	hub.Register(host)
	time.Sleep(30 * time.Millisecond)

	// Set Nobar as active
	hub.mu.Lock()
	hub.currentNobar = &domain.NobarPayload{VideoID: "JGwWNGJdvx8", IsPlaying: true}
	hub.mu.Unlock()

	// Host tries to play music
	playPayload := domain.MusicPayload{Action: "play", VideoID: "dQw4w9WgXcQ", Title: "Song 1"}
	playBytes, _ := json.Marshal(playPayload)
	hub.HandleMusic(host, &domain.Message{ID: "p1", Type: domain.MessageTypeMusic, Payload: playBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Music should NOT be playing (Nobar blocks it)
	if hub.currentMusic != nil {
		t.Error("Expected no music when Nobar is active")
	}
	// Music should be queued instead
	if len(hub.musicQueue) != 1 {
		t.Fatalf("Expected song to be queued, got %d items", len(hub.musicQueue))
	}
	if hub.musicQueue[0].VideoID != "dQw4w9WgXcQ" {
		t.Errorf("Expected queued song to be dQw4w9WgXcQ, got %s", hub.musicQueue[0].VideoID)
	}
}

func TestHub_HandleMusic_NonHostActionsIgnored(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	host := newMockClient(hub, "Host")
	user := newMockClient(hub, "User")

	hub.Register(host)
	time.Sleep(30 * time.Millisecond)
	hub.Register(user)
	time.Sleep(30 * time.Millisecond)

	// Host plays music
	playPayload := domain.MusicPayload{Action: "play", VideoID: "dQw4w9WgXcQ", Title: "Song 1"}
	playBytes, _ := json.Marshal(playPayload)
	hub.HandleMusic(host, &domain.Message{ID: "p1", Type: domain.MessageTypeMusic, Payload: playBytes})

	// Non-host tries to pause
	pausePayload := domain.MusicPayload{Action: "pause"}
	pauseBytes, _ := json.Marshal(pausePayload)
	hub.HandleMusic(user, &domain.Message{ID: "pause1", Type: domain.MessageTypeMusic, Payload: pauseBytes})

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Music should still be playing (non-host can't pause)
	if !hub.currentMusic.IsPlaying {
		t.Error("Non-host should not be able to pause music")
	}
}
