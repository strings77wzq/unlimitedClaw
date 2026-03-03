package config

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

type OnReloadFunc func(old, new *Config)

type Reloader struct {
	path     string
	current  atomic.Pointer[Config]
	mu       sync.Mutex
	onChange []OnReloadFunc
}

func NewReloader(path string) (*Reloader, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, fmt.Errorf("initial config load: %w", err)
	}

	r := &Reloader{path: path}
	r.current.Store(cfg)
	return r, nil
}

func (r *Reloader) Config() *Config {
	return r.current.Load()
}

func (r *Reloader) OnReload(fn OnReloadFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onChange = append(r.onChange, fn)
}

func (r *Reloader) Reload() error {
	newCfg, err := Load(r.path)
	if err != nil {
		return fmt.Errorf("reloading config: %w", err)
	}

	old := r.current.Swap(newCfg)

	r.mu.Lock()
	callbacks := make([]OnReloadFunc, len(r.onChange))
	copy(callbacks, r.onChange)
	r.mu.Unlock()

	for _, fn := range callbacks {
		fn(old, newCfg)
	}

	return nil
}

func (r *Reloader) WatchSIGHUP(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			if err := r.Reload(); err != nil {
				fmt.Fprintf(os.Stderr, "config reload failed: %v\n", err)
			}
		}
	}
}
