package middleware

import (
	"net/http"
)

// SecurityHeaders adds security headers to HTTP responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		
		// XSS Protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy
		// Allow self, inline scripts (Alpine.js needs this), YouTube embeds
		w.Header().Set("Content-Security-Policy", 
			"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://www.youtube.com https://s.ytimg.com; "+
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
			"font-src 'self' https://fonts.gstatic.com; "+
			"img-src 'self' data: https: blob:; "+
			"media-src 'self' https: blob:; "+
			"frame-src https://www.youtube.com https://www.youtube-nocookie.com; "+
			"connect-src 'self' ws: wss: https://www.youtube.com")
		
		// Permissions Policy (formerly Feature-Policy)
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersFunc is a convenience wrapper for http.HandlerFunc
func SecurityHeadersFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SecurityHeaders(http.HandlerFunc(next)).ServeHTTP(w, r)
	}
}
