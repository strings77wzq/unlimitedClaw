package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SandboxConfig holds configuration for path and command validation
type SandboxConfig struct {
	AllowedPaths   []string
	DeniedPaths    []string
	DeniedCommands []string
}

// Sandbox validates paths and commands against configured rules
type Sandbox struct {
	allowedPaths   []string
	deniedPaths    []string
	deniedCommands map[string]bool
}

// NewSandbox creates a new sandbox validator with the given configuration
func NewSandbox(cfg SandboxConfig) *Sandbox {
	deniedCmds := make(map[string]bool)
	for _, cmd := range cfg.DeniedCommands {
		deniedCmds[strings.ToLower(cmd)] = true
	}

	return &Sandbox{
		allowedPaths:   cfg.AllowedPaths,
		deniedPaths:    cfg.DeniedPaths,
		deniedCommands: deniedCmds,
	}
}

// ValidatePath checks if the given path is allowed by sandbox rules
func (s *Sandbox) ValidatePath(path string) error {
	cleanPath := filepath.Clean(path)

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	for _, denied := range s.deniedPaths {
		deniedAbs, err := filepath.Abs(denied)
		if err != nil {
			continue
		}
		if absPath == deniedAbs || strings.HasPrefix(absPath, deniedAbs+string(filepath.Separator)) {
			return fmt.Errorf("path denied: %s", absPath)
		}
	}

	if len(s.allowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range s.allowedPaths {
			allowedAbs, err := filepath.Abs(allowedPath)
			if err != nil {
				continue
			}
			if absPath == allowedAbs || strings.HasPrefix(absPath, allowedAbs+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed list: %s", absPath)
		}
	}

	return nil
}

// ValidateCommand checks if the given command is allowed by sandbox rules
func (s *Sandbox) ValidateCommand(cmd string) error {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCmd := filepath.Base(fields[0])
	baseCmdLower := strings.ToLower(baseCmd)

	if s.deniedCommands[baseCmdLower] {
		return fmt.Errorf("command denied: %s", baseCmd)
	}

	return nil
}
