package main

import (
	"os"
	"strings"
	"testing"
)

func TestConfigSetGetListFlow(t *testing.T) {
	configPath := newTempConfigPath(t)

	stdout, stderr, err := executeRootCommand(t, "--config", configPath, "config", "set", "theme", "solarized")
	if err != nil {
		t.Fatalf("config set failed: %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, "Set theme = solarized") {
		t.Fatalf("unexpected config set output: %q", stdout)
	}

	stdout, stderr, err = executeRootCommand(t, "--config", configPath, "config", "get", "theme")
	if err != nil {
		t.Fatalf("config get failed: %v stderr=%q", err, stderr)
	}
	if strings.TrimSpace(stdout) != "solarized" {
		t.Fatalf("config get output = %q, want solarized", stdout)
	}

	stdout, stderr, err = executeRootCommand(t, "--config", configPath, "config", "list")
	if err != nil {
		t.Fatalf("config list failed: %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, `"theme": "solarized"`) {
		t.Fatalf("config list output missing key: %q", stdout)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestConfigGetMissingKey(t *testing.T) {
	configPath := newTempConfigPath(t)
	writeConfigFile(t, configPath, newTestConfig())

	_, stderr, err := executeRootCommand(t, "--config", configPath, "config", "get", "missing")
	if err == nil {
		t.Fatal("expected missing key error")
	}
	if !strings.Contains(err.Error(), `key "missing" not found in config`) && !strings.Contains(stderr, `key "missing" not found in config`) {
		t.Fatalf("unexpected missing key failure: err=%v stderr=%q", err, stderr)
	}
}

func TestConfigListMissingFileReturnsEmptyObject(t *testing.T) {
	configPath := newTempConfigPath(t)
	stdout, stderr, err := executeRootCommand(t, "--config", configPath, "config", "list")
	if err != nil {
		t.Fatalf("config list missing file failed: %v stderr=%q", err, stderr)
	}
	if strings.TrimSpace(stdout) != "{}" {
		t.Fatalf("config list missing file output = %q, want {}", stdout)
	}
}
