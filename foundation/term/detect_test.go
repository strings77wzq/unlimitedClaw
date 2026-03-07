package term_test

import (
	"testing"

	"github.com/strings77wzq/unlimitedClaw/foundation/term"
)

func TestIsPiped(t *testing.T) {
	// In test environment, stdin/stdout are typically not TTYs
	// so IsPiped() should return true
	result := term.IsPiped()
	if !result {
		t.Log("IsPiped returned false; test is likely running in a TTY")
	}
}

func TestIsInputTTY(t *testing.T) {
	// Just verify it doesn't panic and returns a bool
	_ = term.IsInputTTY()
}

func TestIsOutputTTY(t *testing.T) {
	_ = term.IsOutputTTY()
}

func TestReadStdin_WhenTTY(t *testing.T) {
	if !term.IsInputTTY() {
		t.Skip("stdin is not a TTY in this test environment")
	}
	s, err := term.ReadStdin()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "" {
		t.Fatalf("expected empty string from TTY stdin, got %q", s)
	}
}
