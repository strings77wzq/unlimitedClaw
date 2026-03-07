package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	configJSON := `{
		"agents": {
			"defaults": {
				"model_name": "gpt4",
				"max_tokens": 8192,
				"system_prompt": "You are a test assistant."
			}
		},
		"model_list": [
			{
				"model_name": "gpt4",
				"model": "openai/gpt-4o",
				"api_key": "sk-test-123",
				"api_base": "https://api.openai.com/v1"
			}
		]
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Agents.Defaults.ModelName != "gpt4" {
		t.Errorf("expected model_name gpt4, got %s", cfg.Agents.Defaults.ModelName)
	}
	if cfg.Agents.Defaults.MaxTokens != 8192 {
		t.Errorf("expected max_tokens 8192, got %d", cfg.Agents.Defaults.MaxTokens)
	}
	if len(cfg.ModelList) != 1 {
		t.Fatalf("expected 1 model, got %d", len(cfg.ModelList))
	}
	if cfg.ModelList[0].Vendor() != "openai" {
		t.Errorf("expected vendor openai, got %s", cfg.ModelList[0].Vendor())
	}
	if cfg.ModelList[0].ModelID() != "gpt-4o" {
		t.Errorf("expected model_id gpt-4o, got %s", cfg.ModelList[0].ModelID())
	}
}

func TestEnvVarExpansion(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	configJSON := `{
		"agents": {"defaults": {"model_name": "gpt4", "max_tokens": 8192, "system_prompt": "test"}},
		"model_list": [{"model_name": "gpt4", "model": "openai/gpt-4o", "api_key": "${TEST_API_KEY}"}]
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_API_KEY", "sk-expanded-key-456")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ModelList[0].APIKey != "sk-expanded-key-456" {
		t.Errorf("expected expanded key, got %s", cfg.ModelList[0].APIKey)
	}
}

func TestInvalidConfig(t *testing.T) {
	dir := t.TempDir()

	// Test missing model_name
	configPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(configPath, []byte(`{"agents":{"defaults":{"model_name":"","max_tokens":1}},"model_list":[{"model_name":"x","model":"y"}]}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for empty model_name")
	}

	// Test malformed JSON
	badPath := filepath.Join(dir, "malformed.json")
	if err := os.WriteFile(badPath, []byte(`{bad json`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Load(badPath)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestFindModel(t *testing.T) {
	cfg := DefaultConfig()

	model, err := cfg.FindModel("mock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Model != "mock/echo" {
		t.Errorf("expected mock/echo, got %s", model.Model)
	}

	_, err = cfg.FindModel("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}

func TestDefaultModel(t *testing.T) {
	cfg := DefaultConfig()

	model, err := cfg.DefaultModel()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.ModelName != "mock" {
		t.Errorf("expected mock, got %s", model.ModelName)
	}
}

func TestEnvVarExpansionWithSpecialChars(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	configJSON := `{
		"agents": {"defaults": {"model_name": "gpt4", "max_tokens": 8192, "system_prompt": "test"}},
		"model_list": [{"model_name": "gpt4", "model": "openai/gpt-4o", "api_key": "${TEST_SPECIAL_KEY}"}]
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Value with JSON special characters: quotes, backslashes, newlines
	t.Setenv("TEST_SPECIAL_KEY", `sk-test"with\special"chars\nand\tnewlines`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error loading config with special chars: %v", err)
	}

	expected := `sk-test"with\special"chars\nand\tnewlines`
	if cfg.ModelList[0].APIKey != expected {
		t.Errorf("expected %q, got %q", expected, cfg.ModelList[0].APIKey)
	}
}
