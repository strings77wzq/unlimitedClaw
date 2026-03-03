package concurrency

import (
	"context"
)

// Semaphore is a counting semaphore implementation using buffered channels
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore creates a new semaphore with n permits
func NewSemaphore(n int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, n),
	}
}

// Acquire blocks until a permit is available
func (s *Semaphore) Acquire() error {
	s.ch <- struct{}{}
	return nil
}

// AcquireCtx blocks until a permit is available or context is cancelled
func (s *Semaphore) AcquireCtx(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire a permit without blocking
// Returns true if successful, false if no permits available
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases one permit back to the semaphore
func (s *Semaphore) Release() {
	<-s.ch
}

// Available returns the number of available permits
func (s *Semaphore) Available() int {
	return cap(s.ch) - len(s.ch)
}
