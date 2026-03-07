package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// Load reads config from file path, expanding ${ENV_VAR} patterns
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand ${ENV_VAR} patterns
	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := json.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// expandEnvVars replaces ${VAR_NAME} with the JSON-escaped environment variable value
func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			// JSON-escape the value to prevent breaking JSON structure
			escaped, err := json.Marshal(val)
			if err != nil {
				return match
			}
			// Remove surrounding quotes from json.Marshal output
			// since the value is already inside a JSON string context
			return string(escaped[1 : len(escaped)-1])
		}
		return match // Keep original if not set
	})
}

// Validate checks the config for required fields
func (c *Config) Validate() error {
	if c.Agents.Defaults.ModelName == "" {
		return fmt.Errorf("agents.defaults.model_name is required")
	}
	if len(c.ModelList) == 0 {
		return fmt.Errorf("model_list must contain at least one entry")
	}
	for i, m := range c.ModelList {
		if m.ModelName == "" {
			return fmt.Errorf("model_list[%d].model_name is required", i)
		}
		if m.Model == "" {
			return fmt.Errorf("model_list[%d].model is required", i)
		}
	}
	return nil
}

// FindModel finds a model entry by model_name
func (c *Config) FindModel(name string) (*ModelEntry, error) {
	for i := range c.ModelList {
		if c.ModelList[i].ModelName == name {
			return &c.ModelList[i], nil
		}
	}
	return nil, fmt.Errorf("model %q not found in model_list", name)
}

// DefaultModel returns the default model entry
func (c *Config) DefaultModel() (*ModelEntry, error) {
	return c.FindModel(c.Agents.Defaults.ModelName)
}
