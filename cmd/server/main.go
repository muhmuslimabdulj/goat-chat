package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	httpHandler "github.com/mmuslimabdulj/goat-chat/internal/delivery/http"
	"github.com/mmuslimabdulj/goat-chat/internal/delivery/ws"
	"github.com/mmuslimabdulj/goat-chat/internal/middleware"
	"github.com/mmuslimabdulj/goat-chat/internal/usecase"
	"github.com/mmuslimabdulj/goat-chat/internal/config"
)

func main() {
	// Load .env file (ignore error if not exists, e.g. in production)
	_ = godotenv.Load()
	
	// Reload config after loading .env
	config.AppConfig = config.LoadFromEnv()

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configuring Logging
	if os.Getenv("LOG_LEVEL") == "silent" || os.Getenv("LOG_LEVEL") == "off" {
		log.SetOutput(io.Discard)
	}

	// Initialize dependencies
	roomManager := ws.NewRoomManager()
	generator := usecase.NewPersonaGenerator()
	roomManager.SetPersonaReleaser(generator)
	handler := httpHandler.NewHandler(roomManager, generator)

	// Setup routes
	mux := http.NewServeMux()
	
	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// PWA Service Worker (served from root for scope)
	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, "./static/sw.js")
	})

	// Page routes
	mux.HandleFunc("/", handler.HandleLobby)
	mux.HandleFunc("/room", handler.HandleRoom)
	
	// WebSocket route with rate limiting
	mux.HandleFunc("/ws", middleware.RateLimitFunc(middleware.WebSocketLimiter, handler.HandleWebSocket))
	
	// API routes with rate limiting
	mux.HandleFunc("/api/room/create", middleware.RateLimitFunc(middleware.APILimiter, handler.HandleCreateRoom))
	mux.HandleFunc("/api/room/join", middleware.RateLimitFunc(middleware.APILimiter, handler.HandleJoinRoom))
	mux.HandleFunc("/api/gif/search", middleware.RateLimitFunc(middleware.APILimiter, handler.HandleGifSearch))

	// Apply security headers middleware to all requests
	securedHandler := middleware.SecurityHeaders(mux)
	
	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      securedHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("GOAT chat running at http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
