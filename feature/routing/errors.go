package routing

import "fmt"

// ProviderError represents an error from a provider.
type ProviderError struct {
	Provider   string // provider name
	StatusCode int    // HTTP status code (0 if not HTTP)
	Message    string
	Retryable  bool
	Err        error // wrapped error
}

func (e *ProviderError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("provider %s (status %d): %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("provider %s: %s", e.Provider, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// ToolError represents an error during tool execution.
type ToolError struct {
	ToolName string
	Message  string
	Err      error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool %s: %s", e.ToolName, e.Message)
}

func (e *ToolError) Unwrap() error {
	return e.Err
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error [%s]: %s", e.Field, e.Message)
}

// IsRetryable checks if an error is a retryable ProviderError.
func IsRetryable(err error) bool {
	if provErr, ok := err.(*ProviderError); ok {
		return provErr.Retryable
	}
	return false
}
