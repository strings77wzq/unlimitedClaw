package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/strings77wzq/unlimitedClaw/pkg/tools"
)

type ServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type serverConnection struct {
	config    ServerConfig
	transport *StdioTransport
	client    *Client
	tools     []MCPTool
}

type Manager struct {
	mu      sync.RWMutex
	servers map[string]*serverConnection
}

func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*serverConnection),
	}
}

func (m *Manager) AddServer(cfg ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if cfg.Command == "" {
		return fmt.Errorf("server command is required")
	}

	if _, exists := m.servers[cfg.Name]; exists {
		return fmt.Errorf("server %s already exists", cfg.Name)
	}

	m.servers[cfg.Name] = &serverConnection{
		config: cfg,
	}

	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	serverList := make([]*serverConnection, 0, len(m.servers))
	for _, conn := range m.servers {
		serverList = append(serverList, conn)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(serverList))

	for _, conn := range serverList {
		wg.Add(1)
		go func(conn *serverConnection) {
			defer wg.Done()

			env := make([]string, 0, len(conn.config.Env))
			for k, v := range conn.config.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}

			transport := NewStdioTransport(conn.config.Command, conn.config.Args, env)
			if err := transport.Start(ctx); err != nil {
				errChan <- fmt.Errorf("failed to start server %s: %w", conn.config.Name, err)
				return
			}

			client := NewClient(transport)
			if _, err := client.Initialize(ctx); err != nil {
				transport.Close()
				errChan <- fmt.Errorf("failed to initialize server %s: %w", conn.config.Name, err)
				return
			}

			toolsList, err := client.ListTools(ctx)
			if err != nil {
				transport.Close()
				errChan <- fmt.Errorf("failed to list tools for server %s: %w", conn.config.Name, err)
				return
			}

			m.mu.Lock()
			conn.transport = transport
			conn.client = client
			conn.tools = toolsList
			m.mu.Unlock()
		}(conn)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start %d server(s): %v", len(errors), errors)
	}

	return nil
}

func (m *Manager) DiscoverTools(ctx context.Context) ([]MCPToolProxy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var proxies []MCPToolProxy
	for serverName, conn := range m.servers {
		if conn.client == nil {
			continue
		}

		for _, mcpTool := range conn.tools {
			proxy := MCPToolProxy{
				serverName: serverName,
				mcpTool:    mcpTool,
				manager:    m,
			}
			proxies = append(proxies, proxy)
		}
	}

	return proxies, nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error
	for _, conn := range m.servers {
		if conn.client != nil {
			if err := conn.client.Close(); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to close %d connection(s): %v", len(errors), errors)
	}

	return nil
}

func (m *Manager) callTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (*CallToolResult, error) {
	m.mu.RLock()
	conn, exists := m.servers[serverName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	if conn.client == nil {
		return nil, fmt.Errorf("server %s not initialized", serverName)
	}

	return conn.client.CallTool(ctx, toolName, args)
}

type MCPToolProxy struct {
	serverName string
	mcpTool    MCPTool
	manager    *Manager
}

func (p MCPToolProxy) Name() string {
	return fmt.Sprintf("mcp_%s_%s", p.serverName, p.mcpTool.Name)
}

func (p MCPToolProxy) Description() string {
	return p.mcpTool.Description
}

func (p MCPToolProxy) Parameters() []tools.ToolParameter {
	var schema struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}

	if err := json.Unmarshal(p.mcpTool.InputSchema, &schema); err != nil {
		return nil
	}

	params := make([]tools.ToolParameter, 0, len(schema.Properties))
	requiredMap := make(map[string]bool)
	for _, req := range schema.Required {
		requiredMap[req] = true
	}

	for name, propData := range schema.Properties {
		var prop struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(propData, &prop); err != nil {
			continue
		}

		params = append(params, tools.ToolParameter{
			Name:        name,
			Type:        prop.Type,
			Description: prop.Description,
			Required:    requiredMap[name],
		})
	}

	return params
}

func (p MCPToolProxy) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	result, err := p.manager.callTool(ctx, p.serverName, p.mcpTool.Name, args)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Error calling MCP tool %s: %v", p.mcpTool.Name, err),
			ForUser: "",
			IsError: true,
		}, err
	}

	var output string
	for _, block := range result.Content {
		if block.Type == "text" {
			output += block.Text
		}
	}

	return &tools.ToolResult{
		ForLLM:  output,
		ForUser: output,
		IsError: result.IsError,
	}, nil
}
