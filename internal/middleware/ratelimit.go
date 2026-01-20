package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPRateLimiter manages rate limiting per IP address
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

// NewIPRateLimiter creates a new IP-based rate limiter
// r: requests per second, b: burst size
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
		cleanup:  5 * time.Minute,
	}
	
	// Cleanup old entries periodically
	go limiter.cleanupLoop()
	
	return limiter
}

// GetLimiter returns the rate limiter for the given IP
func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[ip] = limiter
	}

	return limiter
}

// Allow checks if the request from the given IP is allowed
func (l *IPRateLimiter) Allow(ip string) bool {
	return l.GetLimiter(ip).Allow()
}

// cleanupLoop removes old limiters to prevent memory leaks
func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		// Simple cleanup: remove all and let them be recreated
		// In production, you might want to track last access time
		if len(l.limiters) > 10000 {
			l.limiters = make(map[string]*rate.Limiter)
		}
		l.mu.Unlock()
	}
}

// getIP extracts the client IP from the request
func getIP(r *http.Request) string {
	// Check X-Forwarded-For header (for reverse proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	
	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// RateLimitMiddleware creates a middleware that rate limits requests
func RateLimitMiddleware(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)
			
			if !limiter.Allow(ip) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitFunc wraps a HandlerFunc with rate limiting
func RateLimitFunc(limiter *IPRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)
		
		if !limiter.Allow(ip) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	}
}

// Default rate limiters for different purposes
var (
	// APILimiter: 10 requests per second, burst of 20
	APILimiter = NewIPRateLimiter(10, 20)
	
	// WebSocketLimiter: 5 connections per second, burst of 10
	WebSocketLimiter = NewIPRateLimiter(5, 10)
	
	// StrictLimiter: 2 requests per second, burst of 5 (for sensitive operations)
	StrictLimiter = NewIPRateLimiter(2, 5)
)
