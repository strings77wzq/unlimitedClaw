package config

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, path, modelName string) {
	t.Helper()
	data := []byte(`{
		"agents": {"defaults": {"model_name": "` + modelName + `", "max_tokens": 4096, "system_prompt": "test"}},
		"model_list": [{"model_name": "` + modelName + `", "model": "mock/echo"}]
	}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReloaderInitialLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, cfgPath, "gpt4")

	r, err := NewReloader(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := r.Config()
	if cfg.Agents.Defaults.ModelName != "gpt4" {
		t.Errorf("expected model_name 'gpt4', got %q", cfg.Agents.Defaults.ModelName)
	}
}

func TestReloaderReload(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, cfgPath, "gpt4")

	r, err := NewReloader(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	writeTestConfig(t, cfgPath, "claude")

	if err := r.Reload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	cfg := r.Config()
	if cfg.Agents.Defaults.ModelName != "claude" {
		t.Errorf("expected model_name 'claude' after reload, got %q", cfg.Agents.Defaults.ModelName)
	}
}

func TestReloaderOnReloadCallback(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, cfgPath, "gpt4")

	r, err := NewReloader(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var callbackCalled atomic.Bool
	var oldName, newName string
	r.OnReload(func(old, new *Config) {
		oldName = old.Agents.Defaults.ModelName
		newName = new.Agents.Defaults.ModelName
		callbackCalled.Store(true)
	})

	writeTestConfig(t, cfgPath, "claude")
	if err := r.Reload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if !callbackCalled.Load() {
		t.Fatal("expected callback to be called")
	}
	if oldName != "gpt4" {
		t.Errorf("expected old name 'gpt4', got %q", oldName)
	}
	if newName != "claude" {
		t.Errorf("expected new name 'claude', got %q", newName)
	}
}

func TestReloaderInvalidConfigDoesNotReplace(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, cfgPath, "gpt4")

	r, err := NewReloader(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	os.WriteFile(cfgPath, []byte("{invalid json"), 0644)

	err = r.Reload()
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	cfg := r.Config()
	if cfg.Agents.Defaults.ModelName != "gpt4" {
		t.Errorf("config should not change on invalid reload, got %q", cfg.Agents.Defaults.ModelName)
	}
}

func TestReloaderWatchSIGHUPCancellation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, cfgPath, "gpt4")

	r, err := NewReloader(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.WatchSIGHUP(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WatchSIGHUP did not return after context cancellation")
	}
}
