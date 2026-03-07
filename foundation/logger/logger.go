// Package logger wraps the standard library's [log/slog] package to provide
// structured, levelled logging with a consistent interface across all layers.
// Use [New] with [DefaultOptions] to obtain a logger; pass it as a dependency
// rather than using a global singleton.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Logger defines the logging interface for Golem
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// Format specifies the log output format
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Options configures the logger
type Options struct {
	Level  slog.Level
	Format Format
	Output io.Writer
}

// DefaultOptions returns sensible default options
func DefaultOptions() Options {
	return Options{
		Level:  slog.LevelInfo,
		Format: FormatText,
		Output: os.Stderr,
	}
}

// slogLogger wraps slog.Logger to implement our Logger interface
type slogLogger struct {
	inner *slog.Logger
}

// New creates a new Logger with the given options
func New(opts Options) Logger {
	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{Level: opts.Level}

	output := opts.Output
	if output == nil {
		output = os.Stderr
	}

	switch opts.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(output, handlerOpts)
	default:
		handler = slog.NewTextHandler(output, handlerOpts)
	}

	return &slogLogger{inner: slog.New(handler)}
}

func (l *slogLogger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *slogLogger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *slogLogger) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *slogLogger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{inner: l.inner.With(args...)}
}

// NopLogger returns a no-op logger (for testing)
func NopLogger() Logger {
	return New(Options{
		Level:  slog.LevelError + 1, // Above all levels
		Format: FormatText,
		Output: io.Discard,
	})
}
