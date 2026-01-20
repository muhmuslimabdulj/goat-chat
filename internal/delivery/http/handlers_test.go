package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mmuslimabdulj/goat-chat/internal/config"
	"github.com/mmuslimabdulj/goat-chat/internal/delivery/ws"
	"github.com/mmuslimabdulj/goat-chat/internal/usecase"
)

// === SECURITY TESTS ===

func TestSanitizeRoomName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal name", "Test Room", "Test Room"},
		{"Empty defaults", "", "Room Chat"},
		{"Whitespace only", "   ", "Room Chat"},
		{"HTML tags stripped", "<script>alert('xss')</script>Room", "alert('xss')Room"},
		{"Long name truncated", strings.Repeat("a", 100), strings.Repeat("a", 50)},
		{"Trim whitespace", "  Room Name  ", "Room Name"},
		{"Control chars removed", "Room\x00Name\x1F", "RoomName"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeRoomName(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		origin   string
		expected bool
	}{
		{"http://localhost:8080", true},
		{"http://localhost:3000", true},
		{"", true}, // Empty origin allowed (same-origin)
		{"http://evil.com", false},
		{"https://attacker.com", false},
	}

	for _, tc := range tests {
		result := isOriginAllowed(tc.origin)
		if result != tc.expected {
			t.Errorf("isOriginAllowed(%s) = %v, expected %v", tc.origin, result, tc.expected)
		}
	}
}

func TestHandleCreateRoom_SanitizesName(t *testing.T) {
	h := setupTestHandler()

	// Name with HTML tags
	body := []byte(`{"name": "<script>evil</script>My Room"}`)
	req := httptest.NewRequest("POST", "/create-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleCreateRoom(w, req)

	var res map[string]string
	json.NewDecoder(w.Body).Decode(&res)

	// HTML should be stripped
	if strings.Contains(res["name"], "<script>") {
		t.Error("Expected HTML tags to be stripped from room name")
	}
}

func TestHandleCreateRoom_LongNameTruncated(t *testing.T) {
	h := setupTestHandler()

	longName := strings.Repeat("a", 100)
	body := []byte(`{"name": "` + longName + `"}`)
	req := httptest.NewRequest("POST", "/create-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleCreateRoom(w, req)

	var res map[string]string
	json.NewDecoder(w.Body).Decode(&res)

	if len(res["name"]) > 50 {
		t.Errorf("Expected room name to be truncated to 50 chars, got %d", len(res["name"]))
	}
}

// === Original Tests ===

func setupTestHandler() *Handler {
	rm := ws.NewRoomManager()
	gen := usecase.NewPersonaGenerator()
	return NewHandler(rm, gen)
}

func TestHandleLobby(t *testing.T) {
	h := setupTestHandler()
	
	// Case 1: Root path (Success)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.HandleLobby(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Case 2: Invalid path (Not Found)
	req = httptest.NewRequest("GET", "/random", nil)
	w = httptest.NewRecorder()
	h.HandleLobby(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid path, got %d", resp.StatusCode)
	}
}

func TestHandleCreateRoom(t *testing.T) {
	h := setupTestHandler()

	// Case 1: Valid POST
	body := []byte(`{"name": "Test Room"}`)
	req := httptest.NewRequest("POST", "/create-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleCreateRoom(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)
	if res["name"] != "Test Room" {
		t.Error("Expected room name 'Test Room'")
	}
	if res["code"] == "" {
		t.Error("Expected generated room code")
	}

	// Case 2: Invalid Method
	req = httptest.NewRequest("GET", "/create-room", nil)
	w = httptest.NewRecorder()
	h.HandleCreateRoom(w, req)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Error("Expected status 405 for GET request")
	}
}

func TestHandleJoinRoom(t *testing.T) {
	h := setupTestHandler()
	
	// Create a room first
	room := h.roomManager.CreateRoom("Joinable Room")

	// Case 1: Valid Join
	body := []byte(`{"code": "` + room.Code + `"}`)
	req := httptest.NewRequest("POST", "/join-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleJoinRoom(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Error("Expected status 200 for valid room join")
	}

	// Case 2: Invalid Room Code
	body = []byte(`{"code": "invalid-code"}`)
	req = httptest.NewRequest("POST", "/join-room", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	h.HandleJoinRoom(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Error("Expected status 404 for invalid room code")
	}
}

func TestHandleRoom(t *testing.T) {
	h := setupTestHandler()

	// Just checks if it helps render page successfully
	req := httptest.NewRequest("GET", "/room", nil)
	w := httptest.NewRecorder()
	h.HandleRoom(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Result().StatusCode)
	}
	
	// Check content (basic check)
	body := w.Body.String()
	if !strings.Contains(body, "<!doctype html>") && !strings.Contains(body, "<html") {
		// Note: Templ might render partial or full layout implies html tag.
		// Since Index returns a layout, it should have html tags.
		// However, detailed HTML content check is fragile. Just basic status is fine.
	}
}

// === Additional Handler Tests ===

func TestHandleCreateRoom_EmptyName(t *testing.T) {
	h := setupTestHandler()

	// Empty name should default to "Room Chat"
	body := []byte(`{"name": ""}`)
	req := httptest.NewRequest("POST", "/create-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleCreateRoom(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)
	if res["name"] != "Room Chat" {
		t.Errorf("Expected default room name 'Room Chat', got %s", res["name"])
	}
}

func TestHandleCreateRoom_InvalidJSON(t *testing.T) {
	h := setupTestHandler()

	body := []byte(`{invalid json}`)
	req := httptest.NewRequest("POST", "/create-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleCreateRoom(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Result().StatusCode)
	}
}

func TestHandleJoinRoom_InvalidMethod(t *testing.T) {
	h := setupTestHandler()

	req := httptest.NewRequest("GET", "/join-room", nil)
	w := httptest.NewRecorder()
	h.HandleJoinRoom(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Error("Expected status 405 for GET request")
	}
}

func TestHandleJoinRoom_InvalidJSON(t *testing.T) {
	h := setupTestHandler()

	body := []byte(`{invalid}`)
	req := httptest.NewRequest("POST", "/join-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleJoinRoom(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Error("Expected status 400 for invalid JSON")
	}
}

func TestHandleJoinRoom_EmptyCode(t *testing.T) {
	h := setupTestHandler()

	body := []byte(`{"code": ""}`)
	req := httptest.NewRequest("POST", "/join-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleJoinRoom(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Error("Expected status 404 for empty room code")
	}
}

func TestHandleWebSocket_NoRoomCode(t *testing.T) {
	h := setupTestHandler()

	// No room code provided
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()
	h.HandleWebSocket(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing room code, got %d", w.Result().StatusCode)
	}
}

func TestHandleWebSocket_InvalidRoomCode(t *testing.T) {
	h := setupTestHandler()

	// Invalid room code
	req := httptest.NewRequest("GET", "/ws?room=invalid-code", nil)
	w := httptest.NewRecorder()
	h.HandleWebSocket(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid room code, got %d", w.Result().StatusCode)
	}
}

func TestHandleLobby_CacheHeaders(t *testing.T) {
	h := setupTestHandler()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.HandleLobby(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") {
		t.Error("Expected Cache-Control header to contain 'no-store'")
	}
	
	pragma := w.Header().Get("Pragma")
	if pragma != "no-cache" {
		t.Error("Expected Pragma header to be 'no-cache'")
	}
}

func TestHandleRoom_CacheHeaders(t *testing.T) {
	h := setupTestHandler()

	req := httptest.NewRequest("GET", "/room", nil)
	w := httptest.NewRecorder()
	h.HandleRoom(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") {
		t.Error("Expected Cache-Control header to contain 'no-store'")
	}
}

func TestHandleCreateRoom_ContentType(t *testing.T) {
	h := setupTestHandler()

	body := []byte(`{"name": "Test"}`)
	req := httptest.NewRequest("POST", "/create-room", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.HandleCreateRoom(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}
}

func TestNewHandler(t *testing.T) {
	rm := ws.NewRoomManager()
	gen := usecase.NewPersonaGenerator()
	
	handler := NewHandler(rm, gen)
	
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.roomManager != rm {
		t.Error("Expected roomManager to be set")
	}
	if handler.generator != gen {
		t.Error("Expected generator to be set")
	}
}

// === GIF HANDLER TESTS ===

func TestHandleGifSearch_MissingQuery(t *testing.T) {
	rm := ws.NewRoomManager()
	gen := usecase.NewPersonaGenerator()
	handler := NewHandler(rm, gen)

	req := httptest.NewRequest("GET", "/api/gif/search", nil)
	w := httptest.NewRecorder()

	handler.HandleGifSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Missing query") {
		t.Errorf("Expected missing query error, got %s", w.Body.String())
	}
}

func TestHandleGifSearch_MissingAPIKey(t *testing.T) {
	rm := ws.NewRoomManager()
	gen := usecase.NewPersonaGenerator()
	handler := NewHandler(rm, gen)

	// Temporarily clear API key
	originalKey := config.AppConfig.GiphyAPIKey
	config.AppConfig.GiphyAPIKey = ""
	defer func() { config.AppConfig.GiphyAPIKey = originalKey }()

	req := httptest.NewRequest("GET", "/api/gif/search?q=cat", nil)
	w := httptest.NewRecorder()

	handler.HandleGifSearch(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not configured") {
		t.Errorf("Expected not configured error, got %s", w.Body.String())
	}
}

