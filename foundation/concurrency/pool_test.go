package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolSubmit(t *testing.T) {
	pool := NewPool(PoolConfig{MaxWorkers: 3})
	var counter int32

	for i := 0; i < 10; i++ {
		pool.Submit(func() {
			atomic.AddInt32(&counter, 1)
		})
	}

	pool.Wait()

	if counter != 10 {
		t.Errorf("Expected counter to be 10, got %d", counter)
	}
}

func TestPoolLimit(t *testing.T) {
	maxWorkers := 5
	pool := NewPool(PoolConfig{MaxWorkers: maxWorkers})
	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			current := atomic.AddInt32(&active, 1)

			if current > int32(maxWorkers) {
				t.Errorf("Too many concurrent workers: %d > %d", current, maxWorkers)
			}

			for {
				max := atomic.LoadInt32(&maxActive)
				if current <= max {
					break
				}
				if atomic.CompareAndSwapInt32(&maxActive, max, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&active, -1)
		})
	}

	pool.Wait()
	wg.Wait()

	if maxActive > int32(maxWorkers) {
		t.Errorf("Max active workers %d exceeded limit %d", maxActive, maxWorkers)
	}
}

func TestPoolWait(t *testing.T) {
	pool := NewPool(PoolConfig{MaxWorkers: 2})
	var completed int32

	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
		})
	}

	pool.Wait()

	if completed != 5 {
		t.Errorf("Expected 5 completed tasks, got %d", completed)
	}
}

func TestPoolSubmitCtx(t *testing.T) {
	pool := NewPool(PoolConfig{MaxWorkers: 1})
	pool.Submit(func() {
		time.Sleep(100 * time.Millisecond)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.SubmitCtx(ctx, func() {})
	if err == nil {
		t.Error("SubmitCtx should fail with cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	pool.Wait()
}

func TestPoolConcurrentSubmit(t *testing.T) {
	pool := NewPool(PoolConfig{MaxWorkers: 10})
	var counter int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.Submit(func() {
				atomic.AddInt32(&counter, 1)
			})
		}()
	}

	wg.Wait()
	pool.Wait()

	if counter != 100 {
		t.Errorf("Expected counter to be 100, got %d", counter)
	}
}

func TestPoolRunning(t *testing.T) {
	pool := NewPool(PoolConfig{MaxWorkers: 3})

	done := make(chan struct{})

	for i := 0; i < 3; i++ {
		pool.Submit(func() {
			<-done
		})
	}

	time.Sleep(50 * time.Millisecond)

	running := pool.Running()
	if running != 3 {
		t.Errorf("Expected 3 running tasks, got %d", running)
	}

	close(done)
	pool.Wait()

	running = pool.Running()
	if running != 0 {
		t.Errorf("Expected 0 running tasks after wait, got %d", running)
	}
}
