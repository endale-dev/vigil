package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	loginRateLimit      = 10
	loginRateWindow     = time.Minute
	maxRateLimitClients = 10000
)

type rateWindow struct {
	started time.Time
	count   int
}

type RateLimiter struct {
	mu              sync.Mutex
	clients         map[string]rateWindow
	trustProxy      bool
	maxClients      int
	limit           int
	window          time.Duration
	cleanupInterval time.Duration
}

func NewRateLimiter(trustProxyHeaders bool) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]rateWindow), trustProxy: trustProxyHeaders,
		maxClients: maxRateLimitClients, limit: loginRateLimit,
		window: loginRateWindow, cleanupInterval: loginRateWindow,
	}
}

func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if !rateLimitedRequest(request) {
				next.ServeHTTP(writer, request)
				return
			}
			if limiter == nil {
				writeServerError(writer, http.StatusInternalServerError, "internal server error")
				return
			}
			if !limiter.allow(clientIP(request, limiter.trustProxy), time.Now()) {
				writeServerError(writer, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func (limiter *RateLimiter) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(limiter.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			limiter.sweep(now)
		}
	}
}

func (limiter *RateLimiter) allow(ip string, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if entry, exists := limiter.clients[ip]; exists {
		if now.Before(entry.started.Add(limiter.window)) {
			if entry.count >= limiter.limit {
				return false
			}
			entry.count++
			limiter.clients[ip] = entry
			return true
		}
		limiter.clients[ip] = rateWindow{started: now, count: 1}
		return true
	}
	if len(limiter.clients) >= limiter.maxClients {
		limiter.sweepLocked(now)
		if len(limiter.clients) >= limiter.maxClients {
			return false
		}
	}
	limiter.clients[ip] = rateWindow{started: now, count: 1}
	return true
}

func (limiter *RateLimiter) sweep(now time.Time) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.sweepLocked(now)
}

func (limiter *RateLimiter) sweepLocked(now time.Time) {
	for ip, entry := range limiter.clients {
		if !now.Before(entry.started.Add(limiter.window)) {
			delete(limiter.clients, ip)
		}
	}
}

func clientIP(request *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0])
		if ip := net.ParseIP(forwarded); ip != nil {
			return ip.String()
		}
	}
	address := request.RemoteAddr
	if host, _, err := net.SplitHostPort(address); err == nil {
		address = host
	}
	if ip := net.ParseIP(address); ip != nil {
		return ip.String()
	}
	return address
}

func rateLimitedRequest(request *http.Request) bool {
	if request.Method != http.MethodPost {
		return false
	}
	switch request.URL.Path {
	case "/api/auth/login", "/api/auth/register", "/api/auth/password/forgot", "/api/auth/password/reset":
		return true
	default:
		return false
	}
}
