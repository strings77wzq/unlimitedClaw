package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	Rate    float64
	Burst   int
	Enabled bool
}

type limiter struct {
	tokens     float64
	lastRefill time.Time
}

type rateLimitStore struct {
	limiters map[string]*limiter
	mu       sync.Mutex
	rate     float64
	burst    int
	done     chan struct{}
}

func newRateLimitStore(rate float64, burst int) *rateLimitStore {
	store := &rateLimitStore{
		limiters: make(map[string]*limiter),
		rate:     rate,
		burst:    burst,
		done:     make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (rl *rateLimitStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, lim := range rl.limiters {
				if now.Sub(lim.lastRefill) > 10*time.Minute {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

func (rl *rateLimitStore) Close() {
	close(rl.done)
}

func (rl *rateLimitStore) allow(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	lim, exists := rl.limiters[ip]

	if !exists {
		lim = &limiter{
			tokens:     float64(rl.burst),
			lastRefill: now,
		}
		rl.limiters[ip] = lim
	}

	elapsed := now.Sub(lim.lastRefill).Seconds()
	lim.tokens += elapsed * rl.rate
	if lim.tokens > float64(rl.burst) {
		lim.tokens = float64(rl.burst)
	}
	lim.lastRefill = now

	if lim.tokens >= 1 {
		lim.tokens -= 1
		return true, 0
	}

	retryAfter := time.Duration((1 - lim.tokens) / rl.rate * float64(time.Second))
	return false, retryAfter
}

// RateLimitMiddleware returns HTTP middleware that rate-limits by client IP
func RateLimitMiddleware(cfg RateLimitConfig) func(http.Handler) http.Handler {
	var store *rateLimitStore
	if cfg.Enabled {
		store = newRateLimitStore(cfg.Rate, cfg.Burst)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := getClientIP(r)
			allowed, retryAfter := store.allow(clientIP)

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
