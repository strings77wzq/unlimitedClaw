package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Allow() should succeed for burst %d", i)
		}
	}

	if limiter.Allow() {
		t.Error("Allow() should fail after burst exhausted")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	if limiter.Allow() {
		t.Error("Allow() should fail immediately after burst")
	}

	time.Sleep(150 * time.Millisecond)

	if !limiter.Allow() {
		t.Error("Allow() should succeed after tokens refilled")
	}
}

func TestRateLimiterAllowN(t *testing.T) {
	limiter := NewRateLimiter(10, 10)

	if !limiter.AllowN(5) {
		t.Error("AllowN(5) should succeed with 10 tokens")
	}

	if !limiter.AllowN(5) {
		t.Error("AllowN(5) should succeed with 5 tokens remaining")
	}

	if limiter.AllowN(1) {
		t.Error("AllowN(1) should fail with 0 tokens")
	}
}

func TestRateLimiterWait(t *testing.T) {
	limiter := NewRateLimiter(10, 1)
	limiter.Allow()

	start := time.Now()
	ctx := context.Background()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait() failed: %v", err)
	}

	if elapsed < 50*time.Millisecond {
		t.Errorf("Wait() returned too quickly: %v", elapsed)
	}
}

func TestRateLimiterWaitCtxCancel(t *testing.T) {
	limiter := NewRateLimiter(1, 1)
	limiter.Allow()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("Wait() should fail with cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestRateLimiterTokens(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	tokens := limiter.Tokens()
	if tokens != 5 {
		t.Errorf("Expected 5 tokens, got %f", tokens)
	}

	limiter.Allow()
	limiter.Allow()

	tokens = limiter.Tokens()
	if tokens < 2.9 || tokens > 3.1 {
		t.Errorf("Expected ~3 tokens after 2 allows, got %f", tokens)
	}

	time.Sleep(200 * time.Millisecond)

	tokens = limiter.Tokens()
	if tokens < 4.5 {
		t.Errorf("Expected tokens to refill to ~5, got %f", tokens)
	}
}

func TestRateLimiterConcurrent(t *testing.T) {
	limiter := NewRateLimiter(100, 50)
	var allowed int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow() {
				atomic.AddInt32(&allowed, 1)
			}
		}()
	}

	wg.Wait()

	allowedCount := atomic.LoadInt32(&allowed)
	if allowedCount > 50 {
		t.Errorf("Too many allowed requests: %d > 50", allowedCount)
	}
	if allowedCount < 40 {
		t.Errorf("Too few allowed requests: %d < 40", allowedCount)
	}
}
