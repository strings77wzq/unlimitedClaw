// Memory Adapter - wires feature/memory into the main agent
// Provides long-term memory storage and recall capabilities

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/feature/memory"
)

type MemoryConfig struct {
	FilePath    string `json:"file_path"`
	RecallLimit int    `json:"recall_limit"`
}

// LoadMemoryTools creates memory store and recall tools and registers them
func LoadMemoryTools(ctx context.Context, cfg MemoryConfig) (*tools.Registry, memory.Memory, error) {
	registry := tools.NewRegistry()

	filePath := cfg.FilePath
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, fmt.Errorf("getting home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, ".golem", "memory.json")
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, nil, fmt.Errorf("creating memory directory: %w", err)
	}

	mem, err := memory.NewFileMemory(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing memory: %w", err)
	}

	recallLimit := cfg.RecallLimit
	if recallLimit <= 0 {
		recallLimit = 5
	}

	// Create and register memory_store tool
	storeTool := &memoryStoreTool{memory: mem}
	if err := registry.Register(storeTool); err != nil {
		return nil, nil, fmt.Errorf("registering memory_store: %w", err)
	}

	// Create and register memory_recall tool
	recallTool := &memoryRecallTool{memory: mem, limit: recallLimit}
	if err := registry.Register(recallTool); err != nil {
		return nil, nil, fmt.Errorf("registering memory_recall: %w", err)
	}

	return registry, mem, nil
}

// ParseMemoryConfig parses memory configuration from JSON string
func ParseMemoryConfig(jsonStr string) (MemoryConfig, error) {
	cfg := MemoryConfig{
		RecallLimit: 5,
	}

	if jsonStr == "" {
		return cfg, nil
	}

	// Try to parse as JSON
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		// If not JSON, treat as file path
		cfg.FilePath = jsonStr
	}

	return cfg, nil
}

// memoryStoreTool implements tools.Tool for storing memories
type memoryStoreTool struct {
	memory memory.Memory
}

func (m *memoryStoreTool) Name() string {
	return "memory_store"
}

func (m *memoryStoreTool) Description() string {
	return "Store information in long-term memory. Use this to remember important facts, user preferences, or key information that should persist across conversations."
}

func (m *memoryStoreTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "content",
			Type:        "string",
			Description: "The information to remember",
			Required:    true,
		},
		{
			Name:        "importance",
			Type:        "number",
			Description: "Importance score (0.0 to 1.0, default 0.8)",
			Required:    false,
		},
		{
			Name:        "tags",
			Type:        "string",
			Description: "Comma-separated tags for categorization (e.g. 'user-preference,work')",
			Required:    false,
		},
	}
}

func (m *memoryStoreTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &tools.ToolResult{
			ForLLM:  "Error: 'content' parameter is required",
			ForUser: "Please provide content to store",
			IsError: true,
		}, nil
	}

	// Parse optional importance
	importance := 0.8
	if imp, ok := args["importance"]; ok {
		switch v := imp.(type) {
		case float64:
			importance = v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				importance = f
			}
		}
	}
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}

	// Parse optional tags
	var tags []string
	if tagStr, ok := args["tags"].(string); ok && tagStr != "" {
		tags = strings.Split(tagStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	entry := memory.NewEntry(content, tags...)
	entry.Importance = importance

	if err := m.memory.Store(ctx, entry); err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Error storing memory: %v", err),
			ForUser: fmt.Sprintf("Error: %v", err),
			IsError: true,
		}, nil
	}

	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("Memory stored successfully (ID: %s)", entry.ID),
		ForUser: "Stored in memory",
		IsError: false,
	}, nil
}

// memoryRecallTool implements tools.Tool for recalling memories
type memoryRecallTool struct {
	memory memory.Memory
	limit  int
}

func (m *memoryRecallTool) Name() string {
	return "memory_recall"
}

func (m *memoryRecallTool) Description() string {
	return "Recall information from long-term memory. Use this to retrieve previously stored facts, user preferences, or key information."
}

func (m *memoryRecallTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query to find relevant memories",
			Required:    true,
		},
		{
			Name:        "limit",
			Type:        "number",
			Description: "Maximum number of memories to return (default 5)",
			Required:    false,
		},
	}
}

func (m *memoryRecallTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &tools.ToolResult{
			ForLLM:  "Error: 'query' parameter is required",
			ForUser: "Please provide a search query",
			IsError: true,
		}, nil
	}

	limit := m.limit
	if l, ok := args["limit"]; ok {
		switch v := l.(type) {
		case float64:
			limit = int(v)
		case string:
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
	}
	if limit <= 0 {
		limit = 5
	}

	entries, err := m.memory.Recall(ctx, query, limit)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Error recalling memories: %v", err),
			ForUser: fmt.Sprintf("Error: %v", err),
			IsError: true,
		}, nil
	}

	if len(entries) == 0 {
		return &tools.ToolResult{
			ForLLM:  "No memories found matching the query",
			ForUser: "No memories found",
			IsError: false,
		}, nil
	}

	// Format results for the LLM
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant memories:\n\n", len(entries)))

	for i, entry := range entries {
		sb.WriteString(fmt.Sprintf("--- Memory %d (importance: %.2f) ---\n", i+1, entry.Importance))
		sb.WriteString(fmt.Sprintf("Content: %s\n", entry.Content))
		if len(entry.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(entry.Tags, ", ")))
		}
		sb.WriteString("\n")
	}

	return &tools.ToolResult{
		ForLLM:  sb.String(),
		ForUser: fmt.Sprintf("Recalled %d memories", len(entries)),
		IsError: false,
	}, nil
}
