// Package skills provides a composable skill registry. Each [Skill] bundles a
// system prompt with metadata; the agent selects the appropriate skill and
// injects its prompt before calling the LLM. Built-in skills (summarize,
// code-review) live in the builtins sub-package. This package is a reference
// implementation and is NOT wired into main.go by default.
package skills

// Skill represents a skill with metadata and prompts.
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author,omitempty"`
	Prompts     []Prompt `json:"prompts,omitempty"`
	Tools       []string `json:"tools,omitempty"` // tool names this skill requires
	Path        string   `json:"-"`               // filesystem path (not serialized)
}

// Prompt represents a prompt within a skill.
type Prompt struct {
	Name    string `json:"name"`
	Content string `json:"content"`        // loaded from file or inline
	File    string `json:"file,omitempty"` // relative path to prompt file
}
