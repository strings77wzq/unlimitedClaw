package tools

import (
	"context"
	"testing"
)

func TestToolDefinition(t *testing.T) {
	mock := &MockTool{
		ToolName:        "test_tool",
		ToolDescription: "A test tool",
		ToolParameters: []ToolParameter{
			{Name: "arg1", Type: "string", Description: "First argument", Required: true},
			{Name: "arg2", Type: "number", Description: "Second argument", Required: false},
		},
	}

	def := ToDefinition(mock)

	if def.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", def.Name)
	}
	if def.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", def.Description)
	}
	if len(def.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(def.Parameters))
	}
	if def.Parameters[0].Name != "arg1" {
		t.Errorf("expected first param name 'arg1', got %q", def.Parameters[0].Name)
	}
	if def.Parameters[0].Type != "string" {
		t.Errorf("expected first param type 'string', got %q", def.Parameters[0].Type)
	}
	if !def.Parameters[0].Required {
		t.Error("expected first param to be required")
	}
	if def.Parameters[1].Required {
		t.Error("expected second param to not be required")
	}
}

func TestToolResultFields(t *testing.T) {
	tests := []struct {
		name   string
		result ToolResult
		checks func(t *testing.T, r ToolResult)
	}{
		{
			name: "basic result",
			result: ToolResult{
				ForLLM:  "result for LLM",
				ForUser: "result for user",
				IsError: false,
				Silent:  false,
			},
			checks: func(t *testing.T, r ToolResult) {
				if r.ForLLM != "result for LLM" {
					t.Errorf("expected ForLLM 'result for LLM', got %q", r.ForLLM)
				}
				if r.ForUser != "result for user" {
					t.Errorf("expected ForUser 'result for user', got %q", r.ForUser)
				}
				if r.IsError {
					t.Error("expected IsError to be false")
				}
				if r.Silent {
					t.Error("expected Silent to be false")
				}
			},
		},
		{
			name: "error result",
			result: ToolResult{
				ForLLM:  "error occurred",
				IsError: true,
			},
			checks: func(t *testing.T, r ToolResult) {
				if !r.IsError {
					t.Error("expected IsError to be true")
				}
				if r.ForLLM != "error occurred" {
					t.Errorf("expected ForLLM 'error occurred', got %q", r.ForLLM)
				}
			},
		},
		{
			name: "silent result",
			result: ToolResult{
				ForLLM:  "silent operation",
				ForUser: "this should be ignored",
				Silent:  true,
			},
			checks: func(t *testing.T, r ToolResult) {
				if !r.Silent {
					t.Error("expected Silent to be true")
				}
				if r.ForLLM != "silent operation" {
					t.Errorf("expected ForLLM 'silent operation', got %q", r.ForLLM)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checks(t, tt.result)
		})
	}
}

func TestMockToolExecution(t *testing.T) {
	called := false
	mock := &MockTool{
		ToolName:        "mock",
		ToolDescription: "mock tool",
		ToolParameters:  []ToolParameter{},
		ExecuteFn: func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
			called = true
			return &ToolResult{
				ForLLM: "custom result",
			}, nil
		},
	}

	result, err := mock.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("ExecuteFn was not called")
	}
	if result.ForLLM != "custom result" {
		t.Errorf("expected ForLLM 'custom result', got %q", result.ForLLM)
	}
}

func TestMockToolDefaultExecution(t *testing.T) {
	mock := &MockTool{
		ToolName:        "default_mock",
		ToolDescription: "mock with default execution",
		ToolParameters:  []ToolParameter{},
	}

	result, err := mock.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ForLLM != "mock result" {
		t.Errorf("expected default ForLLM 'mock result', got %q", result.ForLLM)
	}
	if !result.Silent {
		t.Error("expected default result to be silent")
	}
	if result.IsError {
		t.Error("expected default result to not be error")
	}
}
