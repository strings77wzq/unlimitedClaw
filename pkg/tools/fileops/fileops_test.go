package fileops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileRead(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileReadTool(workspace)

	testContent := "Hello, World!"
	testFile := filepath.Join(workspace, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "test.txt",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.IsError {
		t.Fatalf("result marked as error: %s", result.ForLLM)
	}

	if result.ForLLM != testContent {
		t.Errorf("expected content %q, got %q", testContent, result.ForLLM)
	}
}

func TestFileWrite(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileWriteTool(workspace)

	testContent := "Test content"
	testPath := "output.txt"

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":    testPath,
		"content": testContent,
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.IsError {
		t.Fatalf("result marked as error: %s", result.ForLLM)
	}

	actualPath := filepath.Join(workspace, testPath)
	content, err := os.ReadFile(actualPath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected content %q, got %q", testContent, string(content))
	}

	if !strings.Contains(result.ForUser, testPath) {
		t.Errorf("ForUser should contain path %q, got %q", testPath, result.ForUser)
	}
}

func TestFileWriteCreatesDirectories(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileWriteTool(workspace)

	testContent := "Nested content"
	testPath := "nested/dir/file.txt"

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":    testPath,
		"content": testContent,
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.IsError {
		t.Fatalf("result marked as error: %s", result.ForLLM)
	}

	actualPath := filepath.Join(workspace, testPath)
	content, err := os.ReadFile(actualPath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected content %q, got %q", testContent, string(content))
	}
}

func TestFileList(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileListTool(workspace)

	if err := os.WriteFile(filepath.Join(workspace, "file1.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "file2.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(workspace, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": ".",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.IsError {
		t.Fatalf("result marked as error: %s", result.ForLLM)
	}

	output := result.ForLLM
	if !strings.Contains(output, "file1.txt") {
		t.Errorf("output should contain file1.txt, got: %s", output)
	}
	if !strings.Contains(output, "file2.txt") {
		t.Errorf("output should contain file2.txt, got: %s", output)
	}
	if !strings.Contains(output, "subdir") {
		t.Errorf("output should contain subdir, got: %s", output)
	}
	if !strings.Contains(output, "DIR:") {
		t.Errorf("output should mark directories with DIR:, got: %s", output)
	}
	if !strings.Contains(output, "FILE:") {
		t.Errorf("output should mark files with FILE:, got: %s", output)
	}
}

func TestPathTraversalBlocked(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileReadTool(workspace)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "../../../etc/passwd",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error for path traversal attempt")
	}

	if !strings.Contains(result.ForLLM, "invalid path") && !strings.Contains(result.ForLLM, "outside workspace") {
		t.Errorf("expected path traversal error message, got: %s", result.ForLLM)
	}
}

func TestAbsolutePathBlocked(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileReadTool(workspace)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "/etc/passwd",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error for absolute path")
	}

	if !strings.Contains(result.ForLLM, "absolute paths are not allowed") && !strings.Contains(result.ForLLM, "invalid path") {
		t.Errorf("expected absolute path error message, got: %s", result.ForLLM)
	}
}

func TestSymlinkEscapeBlocked(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileReadTool(workspace)

	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}

	symlinkPath := filepath.Join(workspace, "escape")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skip("symlink creation not supported on this platform")
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "escape/secret.txt",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error for symlink escape attempt")
	}

	if !strings.Contains(result.ForLLM, "symlink") && !strings.Contains(result.ForLLM, "outside workspace") {
		t.Errorf("expected symlink escape error message, got: %s", result.ForLLM)
	}
}

func TestFileReadMissingPath(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileReadTool(workspace)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error for missing path parameter")
	}
}

func TestFileWriteMissingContent(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileWriteTool(workspace)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "test.txt",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error for missing content parameter")
	}
}

func TestFileListDefaultPath(t *testing.T) {
	workspace := t.TempDir()
	tool := NewFileListTool(workspace)

	if err := os.WriteFile(filepath.Join(workspace, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.IsError {
		t.Fatalf("result marked as error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForLLM, "file.txt") {
		t.Errorf("output should contain file.txt when using default path, got: %s", result.ForLLM)
	}
}
