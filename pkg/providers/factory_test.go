package providers

import (
	"sync"
	"testing"
)

func TestFactoryRegisterAndGet(t *testing.T) {
	factory := NewFactory()
	mockProvider := NewMockProvider("openai")

	factory.Register("openai", mockProvider)

	provider, err := factory.GetProvider("openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "openai" {
		t.Errorf("Provider name mismatch: got %s, want openai", provider.Name())
	}

	if provider != mockProvider {
		t.Error("Expected to get the same provider instance")
	}
}

func TestFactoryUnknownVendor(t *testing.T) {
	factory := NewFactory()

	_, err := factory.GetProvider("unknown-vendor")
	if err == nil {
		t.Fatal("expected error for unknown vendor")
	}

	expected := "no provider registered for vendor: unknown-vendor"
	if err.Error() != expected {
		t.Errorf("Error message mismatch: got %q, want %q", err.Error(), expected)
	}
}

func TestFactoryGetProviderForModel(t *testing.T) {
	factory := NewFactory()
	mockProvider := NewMockProvider("openai")

	factory.Register("openai", mockProvider)

	provider, modelName, err := factory.GetProviderForModel("openai/gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "openai" {
		t.Errorf("Provider name mismatch: got %s, want openai", provider.Name())
	}

	if modelName != "gpt-4" {
		t.Errorf("Model name mismatch: got %s, want gpt-4", modelName)
	}
}

func TestFactoryGetProviderForModelNoSlash(t *testing.T) {
	factory := NewFactory()

	_, _, err := factory.GetProviderForModel("gpt-4")
	if err == nil {
		t.Fatal("expected error for model without vendor prefix")
	}

	if err.Error() != "model name must include vendor prefix (format: vendor/model): gpt-4" {
		t.Errorf("Error message mismatch: got %q", err.Error())
	}
}

func TestFactoryGetProviderForModelUnknownVendor(t *testing.T) {
	factory := NewFactory()

	_, _, err := factory.GetProviderForModel("unknown/model-123")
	if err == nil {
		t.Fatal("expected error for unknown vendor")
	}

	if err.Error() != "no provider registered for vendor: unknown" {
		t.Errorf("Error message mismatch: got %q", err.Error())
	}
}

func TestFactoryMultipleVendors(t *testing.T) {
	factory := NewFactory()

	openaiProvider := NewMockProvider("openai")
	anthropicProvider := NewMockProvider("anthropic")
	googleProvider := NewMockProvider("google")

	factory.Register("openai", openaiProvider)
	factory.Register("anthropic", anthropicProvider)
	factory.Register("google", googleProvider)

	tests := []struct {
		model          string
		expectedVendor string
		expectedModel  string
	}{
		{"openai/gpt-4", "openai", "gpt-4"},
		{"anthropic/claude-3", "anthropic", "claude-3"},
		{"google/gemini-pro", "google", "gemini-pro"},
		{"openai/gpt-3.5-turbo", "openai", "gpt-3.5-turbo"},
	}

	for _, tt := range tests {
		provider, modelName, err := factory.GetProviderForModel(tt.model)
		if err != nil {
			t.Errorf("model %s: unexpected error: %v", tt.model, err)
			continue
		}

		if provider.Name() != tt.expectedVendor {
			t.Errorf("model %s: vendor mismatch: got %s, want %s", tt.model, provider.Name(), tt.expectedVendor)
		}

		if modelName != tt.expectedModel {
			t.Errorf("model %s: model name mismatch: got %s, want %s", tt.model, modelName, tt.expectedModel)
		}
	}
}

func TestFactoryConcurrentAccess(t *testing.T) {
	factory := NewFactory()

	const numGoroutines = 50
	const numVendors = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			vendorID := id % numVendors
			vendorName := string(rune('a' + vendorID))
			provider := NewMockProvider(vendorName)

			factory.Register(vendorName, provider)

			retrieved, err := factory.GetProvider(vendorName)
			if err != nil {
				t.Errorf("goroutine %d: GetProvider failed: %v", id, err)
				return
			}

			if retrieved.Name() != vendorName {
				t.Errorf("goroutine %d: provider name mismatch: got %s, want %s", id, retrieved.Name(), vendorName)
			}

			modelName := vendorName + "/model-x"
			p, m, err := factory.GetProviderForModel(modelName)
			if err != nil {
				t.Errorf("goroutine %d: GetProviderForModel failed: %v", id, err)
				return
			}

			if p.Name() != vendorName {
				t.Errorf("goroutine %d: provider name from model mismatch: got %s, want %s", id, p.Name(), vendorName)
			}

			if m != "model-x" {
				t.Errorf("goroutine %d: model name mismatch: got %s, want model-x", id, m)
			}
		}(i)
	}

	wg.Wait()
}

func TestFactoryModelWithMultipleSlashes(t *testing.T) {
	factory := NewFactory()
	mockProvider := NewMockProvider("openrouter")

	factory.Register("openrouter", mockProvider)

	provider, modelName, err := factory.GetProviderForModel("openrouter/anthropic/claude-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "openrouter" {
		t.Errorf("Provider name mismatch: got %s, want openrouter", provider.Name())
	}

	if modelName != "anthropic/claude-3" {
		t.Errorf("Model name mismatch: got %s, want anthropic/claude-3", modelName)
	}
}
