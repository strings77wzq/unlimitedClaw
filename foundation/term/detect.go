// Package term provides TTY detection helpers used to decide whether to
// activate the interactive TUI or fall back to plain I/O. Detections are
// computed once and cached via [sync.OnceValue] so repeated calls are cheap
// and thread-safe.
package term

import (
	"io"
	"os"
	"sync"

	"github.com/mattn/go-isatty"
)

var (
	IsInputTTY = sync.OnceValue(func() bool {
		return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
	})

	IsOutputTTY = sync.OnceValue(func() bool {
		return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	})
)

// IsPiped returns true when either stdin or stdout is not a TTY.
func IsPiped() bool {
	return !IsInputTTY() || !IsOutputTTY()
}

// ReadStdin reads all of stdin when it is piped (not a TTY).
// Returns empty string if stdin is a TTY.
func ReadStdin() (string, error) {
	if IsInputTTY() {
		return "", nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
