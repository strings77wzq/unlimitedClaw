package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  slog.LevelDebug,
		Format: FormatJSON,
		Output: &buf,
	})

	log.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("expected JSON msg field, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("expected key-value pair, got: %s", output)
	}
}

func TestTextOutput(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  slog.LevelDebug,
		Format: FormatText,
		Output: &buf,
	})

	log.Info("hello world")

	output := buf.String()
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected text output, got: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  slog.LevelWarn,
		Format: FormatText,
		Output: &buf,
	})

	log.Debug("should not appear")
	log.Info("should not appear")
	log.Warn("should appear")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Errorf("debug/info should be filtered at warn level, got: %s", output)
	}
	if !strings.Contains(output, "should appear") {
		t.Errorf("warn should appear, got: %s", output)
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  slog.LevelDebug,
		Format: FormatJSON,
		Output: &buf,
	})

	childLog := log.With("component", "agent")
	childLog.Info("processing")

	output := buf.String()
	if !strings.Contains(output, `"component":"agent"`) {
		t.Errorf("expected component field, got: %s", output)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
		wantErr  bool
	}{
		{"debug", slog.LevelDebug, false},
		{"INFO", slog.LevelInfo, false},
		{"Warn", slog.LevelWarn, false},
		{"warning", slog.LevelWarn, false},
		{"ERROR", slog.LevelError, false},
		{"invalid", slog.LevelInfo, true},
	}

	for _, tt := range tests {
		level, err := ParseLevel(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if !tt.wantErr && level != tt.expected {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, level, tt.expected)
		}
	}
}

func TestNopLogger(t *testing.T) {
	log := NopLogger()
	log.Debug("test")
	log.Info("test")
	log.Warn("test")
	log.Error("test")
	log.With("key", "value").Info("test")
}
