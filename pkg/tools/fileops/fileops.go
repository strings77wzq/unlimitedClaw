package fileops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/strin/unlimitedclaw/pkg/tools"
)

// safePath validates and returns an absolute path within the workspace.
// It prevents path traversal attacks and ensures the requested path is within workspace bounds.
func safePath(workspace, requestedPath string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace is not defined")
	}

	// Get absolute workspace path
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	// Clean and resolve the requested path
	cleanPath := filepath.Clean(requestedPath)

	// Reject absolute paths
	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	// Join with workspace and get absolute path
	absPath, err := filepath.Abs(filepath.Join(absWorkspace, cleanPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Verify the resolved path is within workspace
	relPath, err := filepath.Rel(absWorkspace, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Check for path traversal (any .. component means escape attempt)
	if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("path traversal detected: path is outside workspace")
	}

	// Check for symlinks that escape workspace (if file exists)
	if evalPath, err := filepath.EvalSymlinks(absPath); err == nil {
		evalRel, err := filepath.Rel(absWorkspace, evalPath)
		if err != nil || strings.HasPrefix(evalRel, "..") {
			return "", fmt.Errorf("symlink resolves outside workspace")
		}
	} else if !os.IsNotExist(err) {
		// Error other than "not exist" means we should check parent
		// This handles case where the file doesn't exist yet but parent might be a symlink
		parent := filepath.Dir(absPath)
		if evalParent, err := filepath.EvalSymlinks(parent); err == nil {
			evalRel, err := filepath.Rel(absWorkspace, evalParent)
			if err != nil || strings.HasPrefix(evalRel, "..") {
				return "", fmt.Errorf("parent symlink resolves outside workspace")
			}
		}
	}

	return absPath, nil
}

// FileReadTool reads file contents within the workspace
type FileReadTool struct {
	workspace string
}

// NewFileReadTool creates a new FileReadTool with the given workspace
func NewFileReadTool(workspace string) *FileReadTool {
	return &FileReadTool{workspace: workspace}
}

func (t *FileReadTool) Name() string {
	return "file_read"
}

func (t *FileReadTool) Description() string {
	return "Read the contents of a file within the workspace"
}

func (t *FileReadTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "path",
			Type:        "string",
			Description: "Path to the file to read (relative to workspace)",
			Required:    true,
		},
	}
}

func (t *FileReadTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &tools.ToolResult{
			ForLLM:  "path parameter is required and must be a string",
			IsError: true,
		}, nil
	}

	safePath, err := safePath(t.workspace, path)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("invalid path: %v", err),
			IsError: true,
		}, nil
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("file not found: %s", path),
				IsError: true,
			}, nil
		}
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("failed to read file: %v", err),
			IsError: true,
		}, nil
	}

	return &tools.ToolResult{
		ForLLM: string(content),
	}, nil
}

// FileWriteTool writes content to a file within the workspace
type FileWriteTool struct {
	workspace string
}

// NewFileWriteTool creates a new FileWriteTool with the given workspace
func NewFileWriteTool(workspace string) *FileWriteTool {
	return &FileWriteTool{workspace: workspace}
}

func (t *FileWriteTool) Name() string {
	return "file_write"
}

func (t *FileWriteTool) Description() string {
	return "Write content to a file within the workspace"
}

func (t *FileWriteTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "path",
			Type:        "string",
			Description: "Path to the file to write (relative to workspace)",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "Content to write to the file",
			Required:    true,
		},
	}
}

func (t *FileWriteTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &tools.ToolResult{
			ForLLM:  "path parameter is required and must be a string",
			IsError: true,
		}, nil
	}

	content, ok := args["content"].(string)
	if !ok {
		return &tools.ToolResult{
			ForLLM:  "content parameter is required and must be a string",
			IsError: true,
		}, nil
	}

	safePath, err := safePath(t.workspace, path)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("invalid path: %v", err),
			IsError: true,
		}, nil
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("failed to create parent directories: %v", err),
			IsError: true,
		}, nil
	}

	// Write the file
	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("failed to write file: %v", err),
			IsError: true,
		}, nil
	}

	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("File written successfully: %s", path),
		ForUser: fmt.Sprintf("Written to: %s", path),
	}, nil
}

// FileListTool lists files and directories in a workspace path
type FileListTool struct {
	workspace string
}

// NewFileListTool creates a new FileListTool with the given workspace
func NewFileListTool(workspace string) *FileListTool {
	return &FileListTool{workspace: workspace}
}

func (t *FileListTool) Name() string {
	return "file_list"
}

func (t *FileListTool) Description() string {
	return "List files and directories in a workspace path"
}

func (t *FileListTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "path",
			Type:        "string",
			Description: "Path to list (relative to workspace, defaults to \".\")",
			Required:    false,
		},
	}
}

func (t *FileListTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		path = "."
	}

	safePath, err := safePath(t.workspace, path)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("invalid path: %v", err),
			IsError: true,
		}, nil
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("directory not found: %s", path),
				IsError: true,
			}, nil
		}
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("failed to list directory: %v", err),
			IsError: true,
		}, nil
	}

	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("DIR:  %s\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("FILE: %s\n", entry.Name()))
		}
	}

	if result.Len() == 0 {
		return &tools.ToolResult{
			ForLLM: "(empty directory)",
		}, nil
	}

	return &tools.ToolResult{
		ForLLM: result.String(),
	}, nil
}
