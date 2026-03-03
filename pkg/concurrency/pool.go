package concurrency

import (
	"context"
	"sync"
)

// Pool manages a pool of goroutines with limited concurrency
type Pool struct {
	sem *Semaphore
	wg  sync.WaitGroup
}

// PoolConfig configures a goroutine pool
type PoolConfig struct {
	MaxWorkers int
}

// NewPool creates a new goroutine pool with the given configuration
func NewPool(cfg PoolConfig) *Pool {
	return &Pool{
		sem: NewSemaphore(cfg.MaxWorkers),
	}
}

// Submit submits a task to the pool, blocking if the pool is full
func (p *Pool) Submit(task func()) error {
	if err := p.sem.Acquire(); err != nil {
		return err
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer p.sem.Release()
		task()
	}()
	return nil
}

// SubmitCtx submits a task with context support, blocking if pool is full
func (p *Pool) SubmitCtx(ctx context.Context, task func()) error {
	if err := p.sem.AcquireCtx(ctx); err != nil {
		return err
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer p.sem.Release()
		task()
	}()
	return nil
}

// Wait blocks until all submitted tasks have completed
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Running returns the number of currently running tasks
func (p *Pool) Running() int {
	return cap(p.sem.ch) - p.sem.Available()
}
