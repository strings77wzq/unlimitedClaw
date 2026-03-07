package mcp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/strings77wzq/unlimitedClaw/core/tools"
)

func TestJSONRPCSerialization(t *testing.T) {
	t.Run("request serialization", func(t *testing.T) {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "test/method",
			Params:  map[string]string{"key": "value"},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		var decoded JSONRPCRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		if decoded.JSONRPC != "2.0" {
			t.Errorf("expected jsonrpc 2.0, got %s", decoded.JSONRPC)
		}
		if decoded.ID != 1 {
			t.Errorf("expected id 1, got %d", decoded.ID)
		}
		if decoded.Method != "test/method" {
			t.Errorf("expected method test/method, got %s", decoded.Method)
		}
	})

	t.Run("response serialization", func(t *testing.T) {
		result := map[string]string{"result": "success"}
		resultJSON, _ := json.Marshal(result)

		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  resultJSON,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal response: %v", err)
		}

		var decoded JSONRPCResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if decoded.JSONRPC != "2.0" {
			t.Errorf("expected jsonrpc 2.0, got %s", decoded.JSONRPC)
		}
		if decoded.Error != nil {
			t.Errorf("expected no error, got %+v", decoded.Error)
		}
	})

	t.Run("error response serialization", func(t *testing.T) {
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error: &JSONRPCError{
				Code:    -32600,
				Message: "Invalid Request",
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal error response: %v", err)
		}

		var decoded JSONRPCResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal error response: %v", err)
		}

		if decoded.Error == nil {
			t.Fatal("expected error, got nil")
		}
		if decoded.Error.Code != -32600 {
			t.Errorf("expected error code -32600, got %d", decoded.Error.Code)
		}
		if decoded.Error.Message != "Invalid Request" {
			t.Errorf("expected error message 'Invalid Request', got %s", decoded.Error.Message)
		}
	})
}

func TestInitializeRequest(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: ClientInfo{
			Name:    "unlimitedclaw",
			Version: "0.1.0",
		},
		Capabilities: Capabilities{},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal initialize params: %v", err)
	}

	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal initialize params: %v", err)
	}

	if decoded.ProtocolVersion != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %s", decoded.ProtocolVersion)
	}
	if decoded.ClientInfo.Name != "unlimitedclaw" {
		t.Errorf("expected client name unlimitedclaw, got %s", decoded.ClientInfo.Name)
	}
	if decoded.ClientInfo.Version != "0.1.0" {
		t.Errorf("expected client version 0.1.0, got %s", decoded.ClientInfo.Version)
	}
}

func TestListToolsParsing(t *testing.T) {
	jsonData := `{
		"tools": [
			{
				"name": "test_tool",
				"description": "A test tool",
				"inputSchema": {
					"type": "object",
					"properties": {
						"param1": {"type": "string", "description": "First parameter"}
					},
					"required": ["param1"]
				}
			}
		]
	}`

	var result ListToolsResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal list tools result: %v", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}

	tool := result.Tools[0]
	if tool.Name != "test_tool" {
		t.Errorf("expected tool name test_tool, got %s", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %s", tool.Description)
	}
	if len(tool.InputSchema) == 0 {
		t.Error("expected input schema, got empty")
	}
}

func TestCallToolParsing(t *testing.T) {
	jsonData := `{
		"content": [
			{"type": "text", "text": "Tool execution result"}
		],
		"isError": false
	}`

	var result CallToolResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal call tool result: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	block := result.Content[0]
	if block.Type != "text" {
		t.Errorf("expected type text, got %s", block.Type)
	}
	if block.Text != "Tool execution result" {
		t.Errorf("expected text 'Tool execution result', got %s", block.Text)
	}
	if result.IsError {
		t.Error("expected isError false, got true")
	}
}

type mockTransport struct {
	mu            sync.Mutex
	requests      []*JSONRPCRequest
	notifications []*JSONRPCNotification
	responseQueue chan *JSONRPCResponse
	started       bool
	closed        bool

	responseFunc func(req *JSONRPCRequest) *JSONRPCResponse
}

func newMockTransport(responseFunc func(req *JSONRPCRequest) *JSONRPCResponse) *mockTransport {
	return &mockTransport{
		responseQueue: make(chan *JSONRPCResponse, 10),
		responseFunc:  responseFunc,
	}
}

func (m *mockTransport) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	return nil
}

func (m *mockTransport) Send(request *JSONRPCRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return io.EOF
	}
	m.requests = append(m.requests, request)

	if m.responseFunc != nil {
		resp := m.responseFunc(request)
		if resp != nil {
			m.responseQueue <- resp
		}
	}

	return nil
}

func (m *mockTransport) SendNotification(notification *JSONRPCNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return io.EOF
	}
	m.notifications = append(m.notifications, notification)
	return nil
}

func (m *mockTransport) Receive() (*JSONRPCResponse, error) {
	select {
	case resp := <-m.responseQueue:
		return resp, nil
	case <-time.After(100 * time.Millisecond):
		m.mu.Lock()
		closed := m.closed
		m.mu.Unlock()
		if closed {
			return nil, io.EOF
		}
		return m.Receive()
	}
}

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	close(m.responseQueue)
	return nil
}

func TestRequestIDIncrement(t *testing.T) {
	initResultJSON, _ := json.Marshal(InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
		Capabilities: ServerCapabilities{},
	})

	toolsResultJSON, _ := json.Marshal(ListToolsResult{
		Tools: []MCPTool{},
	})

	mock := newMockTransport(func(req *JSONRPCRequest) *JSONRPCResponse {
		switch req.Method {
		case "initialize":
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: initResultJSON}
		case "tools/list":
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: toolsResultJSON}
		default:
			return nil
		}
	})

	mock.Start(context.Background())
	client := NewClient(mock)

	ctx := context.Background()
	if _, err := client.Initialize(ctx); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	if _, err := client.ListTools(ctx); err != nil {
		t.Fatalf("list tools failed: %v", err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if len(mock.requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(mock.requests))
	}

	if mock.requests[0].ID != 1 {
		t.Errorf("expected first request ID 1, got %d", mock.requests[0].ID)
	}
	if mock.requests[1].ID != 2 {
		t.Errorf("expected second request ID 2, got %d", mock.requests[1].ID)
	}
}

func TestMCPToolProxy(t *testing.T) {
	inputSchemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "Search query"},
			"limit": {"type": "number", "description": "Result limit"}
		},
		"required": ["query"]
	}`)

	mcpTool := MCPTool{
		Name:        "search",
		Description: "Search for items",
		InputSchema: inputSchemaJSON,
	}

	manager := NewManager()
	proxy := MCPToolProxy{
		serverName: "test-server",
		mcpTool:    mcpTool,
		manager:    manager,
	}

	t.Run("name formatting", func(t *testing.T) {
		name := proxy.Name()
		expected := "mcp_test-server_search"
		if name != expected {
			t.Errorf("expected name %s, got %s", expected, name)
		}
	})

	t.Run("description", func(t *testing.T) {
		desc := proxy.Description()
		if desc != "Search for items" {
			t.Errorf("expected description 'Search for items', got %s", desc)
		}
	})

	t.Run("parameters", func(t *testing.T) {
		params := proxy.Parameters()
		if len(params) != 2 {
			t.Fatalf("expected 2 parameters, got %d", len(params))
		}

		foundQuery := false
		foundLimit := false
		for _, param := range params {
			if param.Name == "query" {
				foundQuery = true
				if param.Type != "string" {
					t.Errorf("expected query type string, got %s", param.Type)
				}
				if !param.Required {
					t.Error("expected query to be required")
				}
			}
			if param.Name == "limit" {
				foundLimit = true
				if param.Type != "number" {
					t.Errorf("expected limit type number, got %s", param.Type)
				}
				if param.Required {
					t.Error("expected limit to not be required")
				}
			}
		}

		if !foundQuery {
			t.Error("query parameter not found")
		}
		if !foundLimit {
			t.Error("limit parameter not found")
		}
	})

	t.Run("implements Tool interface", func(t *testing.T) {
		var _ tools.Tool = proxy
	})
}

func TestManagerAddServer(t *testing.T) {
	manager := NewManager()

	cfg := ServerConfig{
		Name:    "test-server",
		Command: "echo",
		Args:    []string{"hello"},
	}

	if err := manager.AddServer(cfg); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}

	t.Run("duplicate server name", func(t *testing.T) {
		err := manager.AddServer(cfg)
		if err == nil {
			t.Fatal("expected error for duplicate server, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got %v", err)
		}
	})

	t.Run("empty server name", func(t *testing.T) {
		err := manager.AddServer(ServerConfig{Command: "test"})
		if err == nil {
			t.Fatal("expected error for empty server name, got nil")
		}
		if !strings.Contains(err.Error(), "name is required") {
			t.Errorf("expected 'name is required' error, got %v", err)
		}
	})

	t.Run("empty command", func(t *testing.T) {
		err := manager.AddServer(ServerConfig{Name: "test2"})
		if err == nil {
			t.Fatal("expected error for empty command, got nil")
		}
		if !strings.Contains(err.Error(), "command is required") {
			t.Errorf("expected 'command is required' error, got %v", err)
		}
	})
}

func TestClientWithMockServer(t *testing.T) {
	initResultJSON, _ := json.Marshal(InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "mock-server",
			Version: "1.0.0",
		},
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
	})

	toolsResultJSON, _ := json.Marshal(ListToolsResult{
		Tools: []MCPTool{
			{
				Name:        "echo",
				Description: "Echo back the input",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`),
			},
		},
	})

	callResultJSON, _ := json.Marshal(CallToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Hello from tool"},
		},
		IsError: false,
	})

	mock := newMockTransport(func(req *JSONRPCRequest) *JSONRPCResponse {
		switch req.Method {
		case "initialize":
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: initResultJSON}
		case "tools/list":
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: toolsResultJSON}
		case "tools/call":
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: callResultJSON}
		default:
			return nil
		}
	})

	mock.Start(context.Background())
	client := NewClient(mock)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("initialize", func(t *testing.T) {
		result, err := client.Initialize(ctx)
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}
		if result.ServerInfo.Name != "mock-server" {
			t.Errorf("expected server name mock-server, got %s", result.ServerInfo.Name)
		}

		mock.mu.Lock()
		if len(mock.notifications) != 1 {
			t.Errorf("expected 1 notification, got %d", len(mock.notifications))
		} else if mock.notifications[0].Method != "notifications/initialized" {
			t.Errorf("expected notifications/initialized, got %s", mock.notifications[0].Method)
		}
		mock.mu.Unlock()
	})

	t.Run("list tools", func(t *testing.T) {
		tools, err := client.ListTools(ctx)
		if err != nil {
			t.Fatalf("list tools failed: %v", err)
		}
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
		if tools[0].Name != "echo" {
			t.Errorf("expected tool name echo, got %s", tools[0].Name)
		}
	})

	t.Run("call tool", func(t *testing.T) {
		result, err := client.CallTool(ctx, "echo", map[string]interface{}{"message": "test"})
		if err != nil {
			t.Fatalf("call tool failed: %v", err)
		}
		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(result.Content))
		}
		if result.Content[0].Text != "Hello from tool" {
			t.Errorf("expected 'Hello from tool', got %s", result.Content[0].Text)
		}
	})
}
