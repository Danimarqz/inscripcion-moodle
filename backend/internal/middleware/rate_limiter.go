package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	requests int
	window   time.Duration
	limiters sync.Map
}

func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	if requests <= 0 {
		requests = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		requests: requests,
		window:   window,
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
	value, ok := rl.limiters.Load(key)
	if ok {
		return value.(*rate.Limiter)
	}

	interval := rl.window / time.Duration(rl.requests)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	limit := rate.Every(interval)

	limiter := rate.NewLimiter(limit, rl.requests)
	rl.limiters.Store(key, limiter)
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
