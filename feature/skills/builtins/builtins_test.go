package builtins

import (
	"testing"

	"github.com/strings77wzq/unlimitedClaw/feature/skills"
)

func TestSummarizeSkill(t *testing.T) {
	s := SummarizeSkill()
	if s.Name != "summarize" {
		t.Errorf("expected name 'summarize', got %q", s.Name)
	}
	if s.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", s.Version)
	}
	if len(s.Prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(s.Prompts))
	}
	if s.Prompts[0].Content == "" {
		t.Error("expected non-empty system prompt")
	}
}

func TestCodeReviewSkill(t *testing.T) {
	s := CodeReviewSkill()
	if s.Name != "code-review" {
		t.Errorf("expected name 'code-review', got %q", s.Name)
	}
	if len(s.Tools) != 1 || s.Tools[0] != "file_read" {
		t.Errorf("expected tools [file_read], got %v", s.Tools)
	}
}

func TestRegisterAll(t *testing.T) {
	registry := skills.NewRegistry()
	if err := RegisterAll(registry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if registry.Count() != 2 {
		t.Errorf("expected 2 skills registered, got %d", registry.Count())
	}

	s, ok := registry.Get("summarize")
	if !ok {
		t.Fatal("summarize skill not found")
	}
	if s.Description == "" {
		t.Error("expected non-empty description")
	}

	s, ok = registry.Get("code-review")
	if !ok {
		t.Fatal("code-review skill not found")
	}
	if s.Author != "unlimitedClaw" {
		t.Errorf("expected author 'unlimitedClaw', got %q", s.Author)
	}
}

func TestRegisterAllDuplicatePreventsDouble(t *testing.T) {
	registry := skills.NewRegistry()
	if err := RegisterAll(registry); err != nil {
		t.Fatalf("first RegisterAll failed: %v", err)
	}
	err := RegisterAll(registry)
	if err == nil {
		t.Fatal("expected error on duplicate registration")
	}
}
