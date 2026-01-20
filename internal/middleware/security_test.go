package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with security middleware
	secured := SecurityHeaders(handler)

	// Make request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	secured.ServeHTTP(w, req)

	// Check headers
	tests := []struct {
		header   string
		expected string
		contains bool
	}{
		{"X-Frame-Options", "DENY", false},
		{"X-Content-Type-Options", "nosniff", false},
		{"X-XSS-Protection", "1; mode=block", false},
		{"Referrer-Policy", "strict-origin-when-cross-origin", false},
		{"Content-Security-Policy", "default-src", true},
		{"Permissions-Policy", "camera=()", true},
	}

	for _, tc := range tests {
		got := w.Header().Get(tc.header)
		if tc.contains {
			if got == "" || !contains(got, tc.expected) {
				t.Errorf("Expected %s header to contain '%s', got '%s'", tc.header, tc.expected, got)
			}
		} else {
			if got != tc.expected {
				t.Errorf("Expected %s header to be '%s', got '%s'", tc.header, tc.expected, got)
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSecurityHeadersFunc(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	secured := SecurityHeadersFunc(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	secured(w, req)

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("Expected X-Frame-Options to be set")
	}
}

func TestSecurityHeaders_PassesThrough(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("Hello"))
	})

	secured := SecurityHeaders(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	secured.ServeHTTP(w, req)

	if !called {
		t.Error("Expected handler to be called")
	}

	if w.Body.String() != "Hello" {
		t.Errorf("Expected body 'Hello', got '%s'", w.Body.String())
	}
}
