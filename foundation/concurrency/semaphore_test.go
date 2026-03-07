package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSemaphoreAcquireRelease(t *testing.T) {
	sem := NewSemaphore(2)

	if err := sem.Acquire(); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if err := sem.Acquire(); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if sem.Available() != 0 {
		t.Errorf("Expected 0 available permits, got %d", sem.Available())
	}

	sem.Release()
	if sem.Available() != 1 {
		t.Errorf("Expected 1 available permit, got %d", sem.Available())
	}

	sem.Release()
	if sem.Available() != 2 {
		t.Errorf("Expected 2 available permits, got %d", sem.Available())
	}
}

func TestSemaphoreTryAcquire(t *testing.T) {
	sem := NewSemaphore(1)

	if !sem.TryAcquire() {
		t.Error("TryAcquire should succeed when permit available")
	}

	if sem.TryAcquire() {
		t.Error("TryAcquire should fail when no permit available")
	}

	sem.Release()
	if !sem.TryAcquire() {
		t.Error("TryAcquire should succeed after release")
	}
}

func TestSemaphoreAcquireCtx(t *testing.T) {
	sem := NewSemaphore(1)
	sem.Acquire()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sem.AcquireCtx(ctx)
	if err == nil {
		t.Error("AcquireCtx should fail with cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestSemaphoreAvailable(t *testing.T) {
	sem := NewSemaphore(5)

	if sem.Available() != 5 {
		t.Errorf("Expected 5 available permits, got %d", sem.Available())
	}

	sem.Acquire()
	sem.Acquire()
	if sem.Available() != 3 {
		t.Errorf("Expected 3 available permits, got %d", sem.Available())
	}

	sem.Release()
	if sem.Available() != 4 {
		t.Errorf("Expected 4 available permits, got %d", sem.Available())
	}
}

func TestSemaphoreConcurrent(t *testing.T) {
	sem := NewSemaphore(10)
	var counter int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem.Acquire()
			atomic.AddInt32(&counter, 1)
			time.Sleep(time.Millisecond)
			atomic.AddInt32(&counter, -1)
			sem.Release()
		}()
	}

	wg.Wait()

	if counter != 0 {
		t.Errorf("Expected counter to be 0, got %d", counter)
	}
}
