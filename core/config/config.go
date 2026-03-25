// Package config defines the configuration schema for Golem and
// provides helpers to load, validate, and access it. Configuration is stored
// as JSON at ~/.golem/config.json and supports hot reload via SIGHUP.
package config

import (
	"strings"
)

// Config is the root configuration structure
type Config struct {
	Agents    AgentConfig    `json:"agents"`
	Gateway   GatewayConfig  `json:"gateway"`
	Telegram  TelegramConfig `json:"telegram,omitempty"`
	ModelList []ModelEntry   `json:"model_list"`
}

type TelegramConfig struct {
	Token         string `json:"token,omitempty"`
	Mode          string `json:"mode,omitempty"`
	WebhookURL    string `json:"webhook_url,omitempty"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	PollTimeout   int    `json:"poll_timeout,omitempty"`
}

// AgentConfig holds agent-related defaults
type AgentConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

// AgentDefaults holds default values for agents
type AgentDefaults struct {
	ModelName    string `json:"model_name"`
	MaxTokens    int    `json:"max_tokens"`
	SystemPrompt string `json:"system_prompt"`
}

// GatewayConfig holds HTTP gateway configuration
type GatewayConfig struct {
	Addr            string   `json:"addr"`
	EnableAuth      bool     `json:"enable_auth"`
	AuthToken       string   `json:"auth_token"`
	EnableRateLimit bool     `json:"enable_rate_limit"`
	RateLimitRPS    int      `json:"rate_limit_rps"`
	RateLimitBurst  int      `json:"rate_limit_burst"`
	AllowedOrigins  []string `json:"allowed_origins"`
}

// ModelEntry represents a single model configuration in model_list format
type ModelEntry struct {
	ModelName string `json:"model_name"`
	Model     string `json:"model"` // format: "vendor/model-id"
	APIKey    string `json:"api_key"`
	APIBase   string `json:"api_base,omitempty"`
}

// Vendor returns the vendor prefix from the Model field (e.g., "openai" from "openai/gpt-4o")
func (m ModelEntry) Vendor() string {
	parts := strings.SplitN(m.Model, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// ModelID returns the model identifier without vendor prefix
func (m ModelEntry) ModelID() string {
	parts := strings.SplitN(m.Model, "/", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return m.Model
}
