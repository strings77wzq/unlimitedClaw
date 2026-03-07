package providers

import (
	"fmt"
	"strings"
	"sync"
)

// Factory creates and caches LLM providers by vendor name.
type Factory struct {
	mu        sync.RWMutex
	providers map[string]LLMProvider
}

// NewFactory creates a new provider factory.
func NewFactory() *Factory {
	return &Factory{
		providers: make(map[string]LLMProvider),
	}
}

// Register adds a provider for a vendor name.
func (f *Factory) Register(vendor string, provider LLMProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[vendor] = provider
}

// GetProvider returns the provider for a vendor name.
// Returns error if no provider registered for that vendor.
func (f *Factory) GetProvider(vendor string) (LLMProvider, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	provider, ok := f.providers[vendor]
	if !ok {
		return nil, fmt.Errorf("no provider registered for vendor: %s", vendor)
	}
	return provider, nil
}

// GetProviderForModel extracts vendor from model name (e.g., "openai/gpt-4" → "openai")
// and returns the corresponding provider.
// Returns: provider, modelName (without vendor prefix), error
func (f *Factory) GetProviderForModel(model string) (LLMProvider, string, error) {
	parts := strings.SplitN(model, "/", 2)
	if len(parts) < 2 {
		return nil, "", fmt.Errorf("model name must include vendor prefix (format: vendor/model): %s", model)
	}

	vendor := parts[0]
	modelName := parts[1]

	provider, err := f.GetProvider(vendor)
	if err != nil {
		return nil, "", err
	}

	return provider, modelName, nil
}
