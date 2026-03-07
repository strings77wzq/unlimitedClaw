package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Loader discovers and loads skills from filesystem.
type Loader struct{}

// NewLoader creates a new skill loader.
func NewLoader() *Loader {
	return &Loader{}
}

// LoadFromDirectory scans directory for skill subdirectories.
// Each subdirectory containing a skill.json file is treated as a skill.
// Returns all discovered skills, or partial results with aggregated errors.
func (l *Loader) LoadFromDirectory(dir string) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %w", dir, err)
	}

	var skills []*Skill
	var errors []error

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(dir, entry.Name())
		skillJSONPath := filepath.Join(skillDir, "skill.json")

		if _, err := os.Stat(skillJSONPath); os.IsNotExist(err) {
			continue
		}

		skill, err := l.LoadSkill(skillDir)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		skills = append(skills, skill)
	}

	if len(errors) > 0 {
		return skills, fmt.Errorf("encountered %d errors loading skills: %v", len(errors), errors)
	}

	return skills, nil
}

// LoadSkill loads a single skill from a directory.
// Reads skill.json, loads prompt files, and validates required fields.
func (l *Loader) LoadSkill(skillDir string) (*Skill, error) {
	skillJSONPath := filepath.Join(skillDir, "skill.json")

	data, err := os.ReadFile(skillJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill.json in %q: %w", skillDir, err)
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return nil, fmt.Errorf("invalid JSON in %q: %w", skillJSONPath, err)
	}

	if err := l.validateSkill(&skill); err != nil {
		return nil, fmt.Errorf("validation failed for skill in %q: %w", skillDir, err)
	}

	absPath, err := filepath.Abs(skillDir)
	if err != nil {
		absPath = skillDir
	}
	skill.Path = absPath

	for i := range skill.Prompts {
		prompt := &skill.Prompts[i]
		if prompt.File != "" {
			promptPath := filepath.Join(skillDir, prompt.File)
			content, err := os.ReadFile(promptPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read prompt file %q: %w", promptPath, err)
			}
			prompt.Content = string(content)
		}
	}

	return &skill, nil
}

func (l *Loader) validateSkill(skill *Skill) error {
	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if skill.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	if skill.Version == "" {
		return fmt.Errorf("skill version is required")
	}
	return nil
}
