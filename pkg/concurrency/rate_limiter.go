package concurrency

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiting algorithm
type RateLimiter struct {
	rate       float64
	burst      int
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the given rate and burst
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.rate
	if r.tokens > float64(r.burst) {
		r.tokens = float64(r.burst)
	}
	r.lastRefill = now
}

// Allow attempts to consume 1 token without blocking
func (r *RateLimiter) Allow() bool {
	return r.AllowN(1)
}

// AllowN attempts to consume n tokens without blocking
func (r *RateLimiter) AllowN(n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= float64(n) {
		r.tokens -= float64(n)
		return true
	}
	return false
}

// Wait blocks until 1 token is available or context is cancelled
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		r.refill()
		if r.tokens >= 1 {
			r.tokens -= 1
			r.mu.Unlock()
			return nil
		}

		deficit := 1 - r.tokens
		waitTime := time.Duration(deficit/r.rate*1e9) * time.Nanosecond
		r.mu.Unlock()

		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Tokens returns the current number of available tokens
func (r *RateLimiter) Tokens() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	return r.tokens
}
