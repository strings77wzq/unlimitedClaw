package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func createTestSkill(t *testing.T, dir, name string, skillData map[string]interface{}, prompts map[string]string) {
	t.Helper()

	skillDir := filepath.Join(dir, name)
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	skillJSON, err := json.MarshalIndent(skillData, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal skill.json: %v", err)
	}

	skillJSONPath := filepath.Join(skillDir, "skill.json")
	if err := os.WriteFile(skillJSONPath, skillJSON, 0644); err != nil {
		t.Fatalf("failed to write skill.json: %v", err)
	}

	for filename, content := range prompts {
		promptPath := filepath.Join(skillDir, filename)
		if err := os.WriteFile(promptPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write prompt file %s: %v", filename, err)
		}
	}
}

func TestSkillRegistration(t *testing.T) {
	registry := NewRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Version:     "1.0.0",
	}

	if err := registry.Register(skill); err != nil {
		t.Fatalf("failed to register skill: %v", err)
	}

	retrieved, ok := registry.Get("test-skill")
	if !ok {
		t.Fatal("skill not found after registration")
	}

	if retrieved.Name != skill.Name {
		t.Errorf("expected name %q, got %q", skill.Name, retrieved.Name)
	}
}

func TestDuplicateRegistration(t *testing.T) {
	registry := NewRegistry()

	skill := &Skill{
		Name:        "duplicate-skill",
		Description: "A skill",
		Version:     "1.0.0",
	}

	if err := registry.Register(skill); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err := registry.Register(skill)
	if err == nil {
		t.Fatal("expected error when registering duplicate skill, got nil")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	skills := []*Skill{
		{Name: "zebra", Description: "Last", Version: "1.0.0"},
		{Name: "alpha", Description: "First", Version: "1.0.0"},
		{Name: "beta", Description: "Second", Version: "1.0.0"},
	}

	for _, skill := range skills {
		if err := registry.Register(skill); err != nil {
			t.Fatalf("failed to register skill %s: %v", skill.Name, err)
		}
	}

	list := registry.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(list))
	}

	if list[0].Name != "alpha" {
		t.Errorf("expected first skill to be 'alpha', got %q", list[0].Name)
	}
	if list[1].Name != "beta" {
		t.Errorf("expected second skill to be 'beta', got %q", list[1].Name)
	}
	if list[2].Name != "zebra" {
		t.Errorf("expected third skill to be 'zebra', got %q", list[2].Name)
	}
}

func TestRegistryRemove(t *testing.T) {
	registry := NewRegistry()

	skill := &Skill{
		Name:        "removable",
		Description: "Will be removed",
		Version:     "1.0.0",
	}

	if err := registry.Register(skill); err != nil {
		t.Fatalf("failed to register skill: %v", err)
	}

	if !registry.Remove("removable") {
		t.Fatal("expected Remove to return true for existing skill")
	}

	if _, ok := registry.Get("removable"); ok {
		t.Fatal("skill still exists after removal")
	}

	if registry.Remove("removable") {
		t.Fatal("expected Remove to return false for non-existent skill")
	}
}

func TestLoadSkill(t *testing.T) {
	dir := t.TempDir()

	skillData := map[string]interface{}{
		"name":        "code-review",
		"description": "Reviews code for common issues",
		"version":     "1.0.0",
		"author":      "golem",
		"prompts": []map[string]string{
			{"name": "system", "file": "system.md"},
		},
		"tools": []string{"file_read", "exec"},
	}

	prompts := map[string]string{
		"system.md": "You are a code reviewer.",
	}

	createTestSkill(t, dir, "code-review", skillData, prompts)

	loader := NewLoader()
	skill, err := loader.LoadSkill(filepath.Join(dir, "code-review"))
	if err != nil {
		t.Fatalf("failed to load skill: %v", err)
	}

	if skill.Name != "code-review" {
		t.Errorf("expected name 'code-review', got %q", skill.Name)
	}
	if skill.Description != "Reviews code for common issues" {
		t.Errorf("unexpected description: %q", skill.Description)
	}
	if skill.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", skill.Version)
	}
	if skill.Author != "golem" {
		t.Errorf("expected author 'golem', got %q", skill.Author)
	}
	if len(skill.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(skill.Tools))
	}
	if skill.Path == "" {
		t.Error("expected Path to be set")
	}
}

func TestLoadSkillWithPromptFiles(t *testing.T) {
	dir := t.TempDir()

	skillData := map[string]interface{}{
		"name":        "multi-prompt",
		"description": "Skill with multiple prompts",
		"version":     "1.0.0",
		"prompts": []map[string]string{
			{"name": "system", "file": "system.md"},
			{"name": "review", "file": "review.md"},
		},
	}

	prompts := map[string]string{
		"system.md": "System prompt content",
		"review.md": "Review prompt content",
	}

	createTestSkill(t, dir, "multi-prompt", skillData, prompts)

	loader := NewLoader()
	skill, err := loader.LoadSkill(filepath.Join(dir, "multi-prompt"))
	if err != nil {
		t.Fatalf("failed to load skill: %v", err)
	}

	if len(skill.Prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(skill.Prompts))
	}

	if skill.Prompts[0].Name != "system" {
		t.Errorf("expected first prompt name 'system', got %q", skill.Prompts[0].Name)
	}
	if skill.Prompts[0].Content != "System prompt content" {
		t.Errorf("unexpected first prompt content: %q", skill.Prompts[0].Content)
	}

	if skill.Prompts[1].Name != "review" {
		t.Errorf("expected second prompt name 'review', got %q", skill.Prompts[1].Name)
	}
	if skill.Prompts[1].Content != "Review prompt content" {
		t.Errorf("unexpected second prompt content: %q", skill.Prompts[1].Content)
	}
}

func TestSkillDiscovery(t *testing.T) {
	dir := t.TempDir()

	skills := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "skill-one",
			data: map[string]interface{}{
				"name":        "skill-one",
				"description": "First skill",
				"version":     "1.0.0",
			},
		},
		{
			name: "skill-two",
			data: map[string]interface{}{
				"name":        "skill-two",
				"description": "Second skill",
				"version":     "2.0.0",
			},
		},
		{
			name: "skill-three",
			data: map[string]interface{}{
				"name":        "skill-three",
				"description": "Third skill",
				"version":     "3.0.0",
			},
		},
	}

	for _, s := range skills {
		createTestSkill(t, dir, s.name, s.data, nil)
	}

	loader := NewLoader()
	loaded, err := loader.LoadFromDirectory(dir)
	if err != nil {
		t.Fatalf("failed to load from directory: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(loaded))
	}
}

func TestSkillDiscoverySkipsNonSkillDirs(t *testing.T) {
	dir := t.TempDir()

	createTestSkill(t, dir, "valid-skill", map[string]interface{}{
		"name":        "valid-skill",
		"description": "Valid skill",
		"version":     "1.0.0",
	}, nil)

	nonSkillDir := filepath.Join(dir, "not-a-skill")
	if err := os.Mkdir(nonSkillDir, 0755); err != nil {
		t.Fatalf("failed to create non-skill dir: %v", err)
	}

	loader := NewLoader()
	loaded, err := loader.LoadFromDirectory(dir)
	if err != nil {
		t.Fatalf("failed to load from directory: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(loaded))
	}
	if loaded[0].Name != "valid-skill" {
		t.Errorf("expected skill name 'valid-skill', got %q", loaded[0].Name)
	}
}

func TestLoadSkillMissingJSON(t *testing.T) {
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "no-json")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	loader := NewLoader()
	_, err := loader.LoadSkill(skillDir)
	if err == nil {
		t.Fatal("expected error when skill.json is missing, got nil")
	}
}

func TestLoadSkillInvalidJSON(t *testing.T) {
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "bad-json")
	if err := os.Mkdir(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	skillJSONPath := filepath.Join(skillDir, "skill.json")
	if err := os.WriteFile(skillJSONPath, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	loader := NewLoader()
	_, err := loader.LoadSkill(skillDir)
	if err == nil {
		t.Fatal("expected error when JSON is invalid, got nil")
	}
}

func TestLoadSkillMissingPromptFile(t *testing.T) {
	dir := t.TempDir()

	skillData := map[string]interface{}{
		"name":        "missing-prompt",
		"description": "Skill with missing prompt file",
		"version":     "1.0.0",
		"prompts": []map[string]string{
			{"name": "system", "file": "nonexistent.md"},
		},
	}

	createTestSkill(t, dir, "missing-prompt", skillData, nil)

	loader := NewLoader()
	_, err := loader.LoadSkill(filepath.Join(dir, "missing-prompt"))
	if err == nil {
		t.Fatal("expected error when prompt file is missing, got nil")
	}
}

func TestLoadSkillValidation(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name      string
		skillData map[string]interface{}
	}{
		{
			name: "missing-name",
			skillData: map[string]interface{}{
				"description": "Missing name field",
				"version":     "1.0.0",
			},
		},
		{
			name: "missing-description",
			skillData: map[string]interface{}{
				"name":    "test",
				"version": "1.0.0",
			},
		},
		{
			name: "missing-version",
			skillData: map[string]interface{}{
				"name":        "test",
				"description": "Missing version",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createTestSkill(t, dir, tt.name, tt.skillData, nil)

			loader := NewLoader()
			_, err := loader.LoadSkill(filepath.Join(dir, tt.name))
			if err == nil {
				t.Fatalf("expected validation error for %s, got nil", tt.name)
			}
		})
	}
}
