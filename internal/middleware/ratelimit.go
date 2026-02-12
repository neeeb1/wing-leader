package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const cleanUpInterval = 1 * time.Hour

type IPRateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    sync.RWMutex
	rate  rate.Limit
	burst int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips:   make(map[string]*rate.Limiter),
		rate:  r,
		burst: b,
	}
	go limiter.cleanup()
	return limiter

}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.ips[ip] = limiter
	}
	return limiter
}

func (i *IPRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)
		limiter := i.GetLimiter(ip)

		if !limiter.Allow() {
			log.Info().Msgf("Client exceeded rate limit (%s)", ip)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getIP(r *http.Request) string {
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		return strings.Split(xForwardedFor, ",")[0]
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func (i *IPRateLimiter) cleanup() {
	i.mu.RLock()
	defer i.mu.RUnlock()

	log.Info().Msgf("Started ip cleanup routine for ratelimiter %v", i)
	ticker := time.NewTicker(cleanUpInterval)

	go func() {
		for range ticker.C {
			i.mu.Lock()
			i.ips = make(map[string]*rate.Limiter)
			i.mu.Unlock()
		}
	}()
}
