package tools

import (
	"context"
	"sync"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	tool := &MockTool{
		ToolName:        "echo",
		ToolDescription: "echoes input",
		ToolParameters:  []ToolParameter{},
	}

	err := r.Register(tool)
	if err != nil {
		t.Fatalf("unexpected error registering tool: %v", err)
	}

	got, ok := r.Get("echo")
	if !ok {
		t.Fatal("expected to find registered tool")
	}
	if got.Name() != "echo" {
		t.Errorf("expected name 'echo', got %q", got.Name())
	}
}

func TestDuplicateRegistration(t *testing.T) {
	r := NewRegistry()

	tool1 := &MockTool{
		ToolName:        "duplicate",
		ToolDescription: "first",
		ToolParameters:  []ToolParameter{},
	}

	tool2 := &MockTool{
		ToolName:        "duplicate",
		ToolDescription: "second",
		ToolParameters:  []ToolParameter{},
	}

	err := r.Register(tool1)
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	err = r.Register(tool2)
	if err == nil {
		t.Error("expected error on duplicate registration")
	}

	if r.Count() != 1 {
		t.Errorf("expected count 1 after duplicate registration, got %d", r.Count())
	}

	tool, _ := r.Get("duplicate")
	if tool.Description() != "first" {
		t.Errorf("expected first tool to remain, got description %q", tool.Description())
	}
}

func TestListToolsAlphabetical(t *testing.T) {
	r := NewRegistry()

	toolZ := &MockTool{ToolName: "zebra", ToolDescription: "z", ToolParameters: []ToolParameter{}}
	toolA := &MockTool{ToolName: "alpha", ToolDescription: "a", ToolParameters: []ToolParameter{}}
	toolM := &MockTool{ToolName: "middle", ToolDescription: "m", ToolParameters: []ToolParameter{}}

	r.Register(toolZ)
	r.Register(toolA)
	r.Register(toolM)

	tools := r.ListTools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	if tools[0].Name() != "alpha" {
		t.Errorf("expected first tool to be 'alpha', got %q", tools[0].Name())
	}
	if tools[1].Name() != "middle" {
		t.Errorf("expected second tool to be 'middle', got %q", tools[1].Name())
	}
	if tools[2].Name() != "zebra" {
		t.Errorf("expected third tool to be 'zebra', got %q", tools[2].Name())
	}
}

func TestListDefinitions(t *testing.T) {
	r := NewRegistry()

	r.Register(&MockTool{
		ToolName:        "beta",
		ToolDescription: "beta tool",
		ToolParameters: []ToolParameter{
			{Name: "x", Type: "string", Description: "param x", Required: true},
		},
	})
	r.Register(&MockTool{
		ToolName:        "alpha",
		ToolDescription: "alpha tool",
		ToolParameters:  []ToolParameter{},
	})

	defs := r.ListDefinitions()
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}

	if defs[0].Name != "alpha" {
		t.Errorf("expected first definition to be 'alpha', got %q", defs[0].Name)
	}
	if defs[1].Name != "beta" {
		t.Errorf("expected second definition to be 'beta', got %q", defs[1].Name)
	}

	if defs[1].Description != "beta tool" {
		t.Errorf("expected description 'beta tool', got %q", defs[1].Description)
	}
	if len(defs[1].Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(defs[1].Parameters))
	}
	if defs[1].Parameters[0].Name != "x" {
		t.Errorf("expected parameter name 'x', got %q", defs[1].Parameters[0].Name)
	}
}

func TestGetNotFound(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for non-existent tool")
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			toolName := string(rune('A' + n%26))
			tool := &MockTool{
				ToolName:        toolName,
				ToolDescription: "concurrent tool",
				ToolParameters:  []ToolParameter{},
			}
			r.Register(tool)
		}(i)
	}

	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			toolName := string(rune('A' + n%26))
			r.Get(toolName)
			r.Count()
			r.ListTools()
			r.ListDefinitions()
		}(i)
	}

	wg.Wait()

	if r.Count() == 0 {
		t.Error("expected tools to be registered after concurrent access")
	}

	if r.Count() > 26 {
		t.Errorf("expected at most 26 unique tools, got %d", r.Count())
	}
}

func TestMockTool(t *testing.T) {
	executeCalled := false
	expectedArgs := map[string]interface{}{"key": "value"}

	mock := &MockTool{
		ToolName:        "test",
		ToolDescription: "test tool",
		ToolParameters: []ToolParameter{
			{Name: "key", Type: "string", Description: "test param", Required: true},
		},
		ExecuteFn: func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
			executeCalled = true
			if args["key"] != expectedArgs["key"] {
				t.Errorf("expected args %v, got %v", expectedArgs, args)
			}
			return &ToolResult{
				ForLLM: "execution successful",
			}, nil
		},
	}

	result, err := mock.Execute(context.Background(), expectedArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executeCalled {
		t.Error("ExecuteFn was not called")
	}
	if result.ForLLM != "execution successful" {
		t.Errorf("expected ForLLM 'execution successful', got %q", result.ForLLM)
	}
}

func TestCount(t *testing.T) {
	r := NewRegistry()

	if r.Count() != 0 {
		t.Errorf("expected initial count 0, got %d", r.Count())
	}

	r.Register(&MockTool{ToolName: "tool1", ToolDescription: "t1", ToolParameters: []ToolParameter{}})
	if r.Count() != 1 {
		t.Errorf("expected count 1 after first registration, got %d", r.Count())
	}

	r.Register(&MockTool{ToolName: "tool2", ToolDescription: "t2", ToolParameters: []ToolParameter{}})
	if r.Count() != 2 {
		t.Errorf("expected count 2 after second registration, got %d", r.Count())
	}

	r.Register(&MockTool{ToolName: "tool1", ToolDescription: "duplicate", ToolParameters: []ToolParameter{}})
	if r.Count() != 2 {
		t.Errorf("expected count 2 after duplicate registration attempt, got %d", r.Count())
	}
}
