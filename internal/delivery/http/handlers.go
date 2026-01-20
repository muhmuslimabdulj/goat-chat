package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/mmuslimabdulj/goat-chat/internal/delivery/ws"
	"github.com/mmuslimabdulj/goat-chat/internal/domain"
	"github.com/mmuslimabdulj/goat-chat/internal/usecase"
	"github.com/mmuslimabdulj/goat-chat/internal/config"
	"github.com/mmuslimabdulj/goat-chat/view/pages"
)

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string) bool {
	// Empty origin is allowed (same-origin requests)
	if origin == "" {
		return true
	}
	
	for _, allowed := range config.AppConfig.AllowedOrigins {
		if allowed == "*" || origin == allowed {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return isOriginAllowed(origin)
	},
}

// sanitizeRoomName cleans and validates room name input
func sanitizeRoomName(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)
	
	// Limit length to 50 characters
	if utf8.RuneCountInString(name) > 50 {
		runes := []rune(name)
		name = string(runes[:50])
	}
	
	// Remove HTML tags to prevent XSS
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	name = htmlTagRegex.ReplaceAllString(name, "")
	
	// Remove control characters
	controlCharRegex := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	name = controlCharRegex.ReplaceAllString(name, "")
	
	// Trim again after cleaning
	name = strings.TrimSpace(name)
	
	// Default if empty
	if name == "" {
		name = "Room Chat"
	}
	
	return name
}

type Handler struct {
	roomManager *ws.RoomManager
	generator   *usecase.PersonaGenerator
}

func NewHandler(rm *ws.RoomManager, generator *usecase.PersonaGenerator) *Handler {
	return &Handler{
		roomManager: rm,
		generator:   generator,
	}
}

// HandleLobby serves the lobby page (create/join room)
func (h *Handler) HandleLobby(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	component := pages.Lobby()
	component.Render(r.Context(), w)
}

// HandleCreateRoom creates a new room and returns the code
func (h *Handler) HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Sanitize room name
	req.Name = sanitizeRoomName(req.Name)

	room := h.roomManager.CreateRoom(req.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"code": room.Code,
		"name": room.Name,
	})
}

// HandleJoinRoom validates room code and returns room info
func (h *Handler) HandleJoinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	room := h.roomManager.GetRoom(req.Code)
	if room == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Room tidak ditemukan",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"code": room.Code,
		"name": room.Name,
	})
}

// HandleRoom serves the chat room page
// Room code is stored in sessionStorage client-side (not exposed in URL)
func (h *Handler) HandleRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	// Render empty room page - JavaScript will validate room code from sessionStorage
	component := pages.Index(nil, "", "")
	component.Render(r.Context(), w)
}

// HandleWebSocket upgrades HTTP to WebSocket for a specific room
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("room")
	if code == "" {
		http.Error(w, "Room code required", http.StatusBadRequest)
		return
	}

	room := h.roomManager.GetRoom(code)
	if room == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	var user *domain.User
	
	// Check if client is reconnecting with session token
	token := r.URL.Query().Get("token")
	if token != "" {
		// Validate session token
		session, valid := ws.GlobalSessionStore.ValidateToken(token)
		if valid && session.RoomCode == code {
			// Valid token - restore persona
			user = h.generator.GenerateWithPersona(session.PersonaName, session.PersonaColor)
		} else {
			// Invalid token - generate new persona
			user = h.generator.Generate()
		}
	} else {
		// New user - generate fresh persona
		user = h.generator.Generate()
	}
	user.Conn = conn

	// Create client and register with room's hub
	client := ws.NewClient(room.Hub, conn, user)
	room.Hub.Register(client)

	// Generate session token for this user (for future reconnection)
	sessionToken := ws.GlobalSessionStore.GenerateToken(
		user.ID.String(),
		user.PersonaName,
		user.PersonaColor,
		code,
	)

	// Send token to client via WebSocket message
	go func() {
		tokenMsg := map[string]interface{}{
			"type": "session_token",
			"payload": map[string]string{
				"token": sessionToken,
			},
		}
		data, _ := json.Marshal(tokenMsg)
		client.Send(data)
	}()

	// Start read/write pumps in goroutines
	go client.WritePump()
	go client.ReadPump()
}

// HandleGifSearch proxies GIF search requests to GIPHY API
func (h *Handler) HandleGifSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	apiKey := config.AppConfig.GiphyAPIKey
	if apiKey == "" {
		http.Error(w, "GIF service not configured", http.StatusServiceUnavailable)
		return
	}

	// Call GIPHY API
	giphyURL := "https://api.giphy.com/v1/gifs/search?api_key=" + apiKey + 
		"&q=" + query + "&limit=33&rating=g"

	resp, err := http.Get(giphyURL)
	if err != nil {
		http.Error(w, "Failed to fetch GIFs", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var giphyResp struct {
		Data []struct {
			Images struct {
				Original struct {
					URL string `json:"url"`
				} `json:"original"`
				FixedWidth struct {
					URL string `json:"url"`
				} `json:"fixed_width"`
			} `json:"images"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&giphyResp); err != nil {
		http.Error(w, "Failed to parse GIF response", http.StatusInternalServerError)
		return
	}

	type GifResult struct {
		URL     string `json:"url"`
		Preview string `json:"preview"`
	}

	results := make([]GifResult, 0, len(giphyResp.Data))
	for _, item := range giphyResp.Data {
		results = append(results, GifResult{
			URL:     item.Images.Original.URL,
			Preview: item.Images.FixedWidth.URL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
