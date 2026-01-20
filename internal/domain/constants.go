package domain

import "time"

// ==== WebSocket Constants ====

// MaxMessageSize is the maximum allowed WebSocket message size in bytes
const MaxMessageSize = 4096

// MaxHistorySize is the maximum number of messages to store for new clients
const MaxHistorySize = 200

// ==== Session Constants ====

// SessionTTL is the default session token time-to-live
const SessionTTL = 24 * time.Hour

// ==== Rate Limit Constants ====

const (
	// DefaultRateLimitAPI is the default rate limit for API endpoints (requests/sec)
	DefaultRateLimitAPI = 10

	// DefaultRateLimitWS is the default rate limit for WebSocket connections (req/sec)
	DefaultRateLimitWS = 5

	// DefaultRateLimitStrict is the stricter rate limit for sensitive endpoints
	DefaultRateLimitStrict = 2
)

// ==== Timing Constants ====

const (
	// LeaveDelay is the delay before broadcasting user leave event
	LeaveDelay = 5 * time.Second

	// HostTransferDelay is the delay before transferring host on disconnect
	HostTransferDelay = 15 * time.Second

	// ShutdownGracePeriod is the time to wait before destroying empty room
	ShutdownGracePeriod = 60 * time.Second

	// SongEndedDebounce prevents rapid song-ended events
	SongEndedDebounce = 5 * time.Second
)
