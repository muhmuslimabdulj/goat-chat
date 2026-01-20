package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(10, 20)

	if limiter == nil {
		t.Fatal("Expected limiter to be created")
	}
	if limiter.rate != 10 {
		t.Errorf("Expected rate 10, got %v", limiter.rate)
	}
	if limiter.burst != 20 {
		t.Errorf("Expected burst 20, got %d", limiter.burst)
	}
}

func TestIPRateLimiter_GetLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(10, 20)

	// Get limiter for first IP
	l1 := limiter.GetLimiter("192.168.1.1")
	if l1 == nil {
		t.Fatal("Expected limiter for IP")
	}

	// Get limiter for same IP - should be the same instance
	l2 := limiter.GetLimiter("192.168.1.1")
	if l1 != l2 {
		t.Error("Expected same limiter instance for same IP")
	}

	// Get limiter for different IP - should be different
	l3 := limiter.GetLimiter("192.168.1.2")
	if l1 == l3 {
		t.Error("Expected different limiter instance for different IP")
	}
}

func TestIPRateLimiter_Allow(t *testing.T) {
	// Very low rate for testing
	limiter := NewIPRateLimiter(1, 2) // 1 per second, burst of 2

	ip := "192.168.1.1"

	// First two should be allowed (burst)
	if !limiter.Allow(ip) {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Error("Second request should be allowed (within burst)")
	}

	// Third should be denied (burst exhausted)
	if limiter.Allow(ip) {
		t.Error("Third request should be denied (burst exhausted)")
	}
}

func TestIPRateLimiter_AllowAfterWait(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Limit(10), 1) // 10 per second, burst of 1

	ip := "192.168.1.1"

	// First should be allowed
	if !limiter.Allow(ip) {
		t.Error("First request should be allowed")
	}

	// Immediate second should be denied
	if limiter.Allow(ip) {
		t.Error("Immediate second request should be denied")
	}

	// Wait for token refill
	time.Sleep(150 * time.Millisecond)

	// Now should be allowed again
	if !limiter.Allow(ip) {
		t.Error("Request after wait should be allowed")
	}
}

func TestIPRateLimiter_Concurrency(t *testing.T) {
	limiter := NewIPRateLimiter(100, 100)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ip := "192.168.1.1"
			limiter.Allow(ip)
		}(i)
	}

	wg.Wait()
	close(errors)

	// Should not have any errors (no race conditions)
	for err := range errors {
		t.Errorf("Concurrent error: %v", err)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Very strict limiter for testing
	limiter := NewIPRateLimiter(1, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rateLimited := RateLimitMiddleware(limiter)(handler)

	// First request - should pass
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	rateLimited.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("First request should be OK, got %d", w.Result().StatusCode)
	}

	// Second request - should be rate limited
	w = httptest.NewRecorder()
	rateLimited.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got %d", w.Result().StatusCode)
	}
}

func TestRateLimitFunc(t *testing.T) {
	limiter := NewIPRateLimiter(1, 1)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	rateLimited := RateLimitFunc(limiter, handler)

	// First request - should pass
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	rateLimited(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("First request should be OK, got %d", w.Result().StatusCode)
	}
}

func TestGetIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getIP(req)
	if ip != "1.2.3.4" {
		t.Errorf("Expected IP from X-Forwarded-For, got %s", ip)
	}
}

func TestGetIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "5.6.7.8")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getIP(req)
	if ip != "5.6.7.8" {
		t.Errorf("Expected IP from X-Real-IP, got %s", ip)
	}
}

func TestGetIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getIP(req)
	if ip != "192.168.1.1:12345" {
		t.Errorf("Expected IP from RemoteAddr, got %s", ip)
	}
}

func TestDefaultLimiters(t *testing.T) {
	// Check that default limiters are initialized
	if APILimiter == nil {
		t.Error("APILimiter should be initialized")
	}
	if WebSocketLimiter == nil {
		t.Error("WebSocketLimiter should be initialized")
	}
	if StrictLimiter == nil {
		t.Error("StrictLimiter should be initialized")
	}
}
