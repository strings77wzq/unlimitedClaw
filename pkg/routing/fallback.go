package routing

import (
	"sync"
	"time"
)

// FallbackChain manages provider fallback with cooldown tracking.
type FallbackChain struct {
	mu               sync.Mutex
	models           []string
	cooldownDuration time.Duration
	failureTimes     map[string]time.Time
}

// NewFallbackChain creates a new fallback chain with the given models and cooldown duration.
func NewFallbackChain(models []string, cooldownDuration time.Duration) *FallbackChain {
	return &FallbackChain{
		models:           models,
		cooldownDuration: cooldownDuration,
		failureTimes:     make(map[string]time.Time),
	}
}

// Next returns the next available model (skipping cooled-down ones).
// Returns (modelName, true) if available, ("", false) if all models are in cooldown.
func (f *FallbackChain) Next() (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	for _, model := range f.models {
		if failTime, failed := f.failureTimes[model]; failed {
			if now.Sub(failTime) < f.cooldownDuration {
				continue
			}
		}
		return model, true
	}
	return "", false
}

// MarkFailed puts the model in cooldown.
func (f *FallbackChain) MarkFailed(model string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failureTimes[model] = time.Now()
}

// MarkSuccess resets the cooldown for the model.
func (f *FallbackChain) MarkSuccess(model string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.failureTimes, model)
}

// Reset resets all cooldowns.
func (f *FallbackChain) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failureTimes = make(map[string]time.Time)
}
