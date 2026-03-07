// Package routing provides a fallback-capable [Router] that wraps multiple
// [providers.LLMProvider] instances. If the primary provider returns an error,
// the router automatically retries with the next registered provider.
// This is a reference implementation in the feature/ layer and is NOT wired
// into main.go by default.
package routing

import (
	"context"
	"sync"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

// Router routes model requests to providers with fallback support.
type Router struct {
	mu      sync.RWMutex
	routes  map[string][]string
	factory *providers.Factory
}

// NewRouter creates a new router with the given provider factory.
func NewRouter(factory *providers.Factory) *Router {
	return &Router{
		routes:  make(map[string][]string),
		factory: factory,
	}
}

// AddRoute maps a model alias to one or more provider/model pairs (in fallback order).
// Example: router.AddRoute("default", "openai/gpt-4o", "anthropic/claude-sonnet-4-20250514")
func (r *Router) AddRoute(modelName string, providerModels ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[modelName] = providerModels
}

// Chat routes the request to the appropriate provider with fallback support.
// It tries each provider in the route's fallback chain until one succeeds.
// If no route is found, it tries the factory's GetProviderForModel directly.
func (r *Router) Chat(ctx context.Context, modelName string, messages []providers.Message, toolDefs []tools.ToolDefinition, opts *providers.ChatOptions) (*providers.LLMResponse, error) {
	r.mu.RLock()
	route, hasRoute := r.routes[modelName]
	r.mu.RUnlock()

	if hasRoute {
		var lastErr error
		for _, providerModel := range route {
			provider, model, err := r.factory.GetProviderForModel(providerModel)
			if err != nil {
				lastErr = err
				continue
			}

			resp, err := provider.Chat(ctx, messages, toolDefs, model, opts)
			if err != nil {
				lastErr = err
				if IsRetryable(err) {
					continue
				}
				return nil, err
			}
			return resp, nil
		}
		return nil, lastErr
	}

	provider, model, err := r.factory.GetProviderForModel(modelName)
	if err != nil {
		return nil, err
	}

	return provider.Chat(ctx, messages, toolDefs, model, opts)
}
