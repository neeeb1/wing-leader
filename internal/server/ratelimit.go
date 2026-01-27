package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/neeeb1/rate_birds/internal/api"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    sync.RWMutex
	rate  rate.Limit
	burst int
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
			log.Info().Msg(fmt.Sprintf("Client exceeded rate limit (%s)", ip))
			api.RespondWithError(w, 429, "Rate limit exceeded")
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
