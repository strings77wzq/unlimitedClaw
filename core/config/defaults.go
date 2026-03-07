package config

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Agents: AgentConfig{
			Defaults: AgentDefaults{
				ModelName:    "mock",
				MaxTokens:    4096,
				SystemPrompt: "You are Golem, a helpful AI assistant.",
			},
		},
		ModelList: []ModelEntry{
			{
				ModelName: "mock",
				Model:     "mock/echo",
				APIKey:    "",
			},
		},
	}
}
