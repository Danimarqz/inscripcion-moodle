package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	requests int
	window   time.Duration
	mu       sync.RWMutex
	limiters map[string]*limiterEntry
}

func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	if requests <= 0 {
		requests = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	rl := &RateLimiter{
		requests: requests,
		window:   window,
		limiters: make(map[string]*limiterEntry),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		threshold := time.Now().Add(-3 * rl.window)
		rl.mu.Lock()
		for key, entry := range rl.limiters {
			if entry.lastSeen.Before(threshold) {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	entry, ok := rl.limiters[key]
	rl.mu.RUnlock()
	if ok {
		rl.mu.Lock()
		entry.lastSeen = time.Now()
		rl.mu.Unlock()
		return entry.limiter
	}

	interval := rl.window / time.Duration(rl.requests)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	limiter := rate.NewLimiter(rate.Every(interval), rl.requests)

	rl.mu.Lock()
	// Double-checked locking: re-check after acquiring write lock.
	if e, exists := rl.limiters[key]; exists {
		e.lastSeen = time.Now()
		rl.mu.Unlock()
		return e.limiter
	}
	rl.limiters[key] = &limiterEntry{limiter: limiter, lastSeen: time.Now()}
	rl.mu.Unlock()
	return limiter
}

func clientIP(r *http.Request) string {
	if header := r.Header.Get("X-Forwarded-For"); header != "" {
		parts := strings.Split(header, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
