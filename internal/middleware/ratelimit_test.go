package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_AllowUnderLimit(t *testing.T) {
	testLimiter := NewIPRateLimiter(10, 20)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		// Set a fixed IP for testing
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler := testLimiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d on iteration %d", http.StatusOK, rr.Code, i)
		}
	}
}

func TestRateLimiter_BlockOverLimit(t *testing.T) {
	testLimiter := NewIPRateLimiter(1, 5)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		// Set a fixed IP for testing
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler := testLimiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		handler.ServeHTTP(rr, req)

		if i < 5 {
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d on iteration %d", http.StatusOK, rr.Code, i)
			}
		} else {
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("Expected status %d, got %d on iteration %d", http.StatusTooManyRequests, rr.Code, i)
			}
		}
	}
}

func TestRateLimiter_SpearateIPs(t *testing.T) {
	testLimiter := NewIPRateLimiter(1, 5)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	for i := 0; i < 5; i++ {
		testLimiter.GetLimiter(ip1).Allow()
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.RemoteAddr = ip2 + ":12345"
	rr := httptest.NewRecorder()

	handler := testLimiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d for separate IP", http.StatusOK, rr.Code)
	}
}
