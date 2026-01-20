package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port string

	// Security
	AllowedOrigins []string
	SessionTTL     time.Duration

	// Rate Limiting
	RateLimitAPI    rate.Limit
	RateLimitWS     rate.Limit
	RateLimitStrict rate.Limit

	// Logging
	LogLevel string

	// WebSocket
	MaxMessageSize int
	MaxHistorySize int

	// External APIs
	GiphyAPIKey string
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Port:            "8080",
		AllowedOrigins:  []string{"http://localhost:8080", "http://localhost:3000"},
		SessionTTL:      24 * time.Hour,
		RateLimitAPI:    10,
		RateLimitWS:     5,
		RateLimitStrict: 2,
		LogLevel:        "info", // Options: debug, info, warn, error, silent
		MaxMessageSize:  4096,
		MaxHistorySize:  200,
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	// Server
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	// Security
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = parseOrigins(origins)
	}

	if ttl := os.Getenv("SESSION_TTL_HOURS"); ttl != "" {
		if hours, err := strconv.Atoi(ttl); err == nil && hours > 0 {
			cfg.SessionTTL = time.Duration(hours) * time.Hour
		}
	}

	// Rate Limiting
	if rl := os.Getenv("RATE_LIMIT_API"); rl != "" {
		if val, err := strconv.Atoi(rl); err == nil && val > 0 {
			cfg.RateLimitAPI = rate.Limit(val)
		}
	}

	if rl := os.Getenv("RATE_LIMIT_WS"); rl != "" {
		if val, err := strconv.Atoi(rl); err == nil && val > 0 {
			cfg.RateLimitWS = rate.Limit(val)
		}
	}

	if rl := os.Getenv("RATE_LIMIT_STRICT"); rl != "" {
		if val, err := strconv.Atoi(rl); err == nil && val > 0 {
			cfg.RateLimitStrict = rate.Limit(val)
		}
	}

	// Logging
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	// WebSocket
	if size := os.Getenv("MAX_MESSAGE_SIZE"); size != "" {
		if val, err := strconv.Atoi(size); err == nil && val > 0 {
			cfg.MaxMessageSize = val
		}
	}

	if size := os.Getenv("MAX_HISTORY_SIZE"); size != "" {
		if val, err := strconv.Atoi(size); err == nil && val > 0 {
			cfg.MaxHistorySize = val
		}
	}

	// External APIs
	if key := os.Getenv("GIPHY_API_KEY"); key != "" {
		cfg.GiphyAPIKey = key
	}

	return cfg
}

// parseOrigins parses comma-separated origins
func parseOrigins(origins string) []string {
	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// Global configuration instance
var AppConfig = LoadFromEnv()
