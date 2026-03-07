// Package mcp implements a Model Context Protocol (MCP) client that
// communicates with an external MCP server over stdin/stdout JSON-RPC.
// This is a reference implementation in the feature/ layer — it is complete
// and tested but NOT wired into the running binary by default. Wire it in
// via cmd/golem/main.go if you need external MCP tool integration.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

const (
	protocolVersion = "2024-11-05"
	clientName      = "golem"
	clientVersion   = "0.1.0"
)

type Transport interface {
	Send(request *JSONRPCRequest) error
	SendNotification(notification *JSONRPCNotification) error
	Receive() (*JSONRPCResponse, error)
	Close() error
}

type Client struct {
	transport Transport

	mu      sync.Mutex
	nextID  int
	pending map[int]chan *JSONRPCResponse

	initialized bool
	initResult  *InitializeResult

	receiveOnce sync.Once
	done        chan struct{}
}

func NewClient(transport Transport) *Client {
	return &Client{
		transport: transport,
		nextID:    1,
		pending:   make(map[int]chan *JSONRPCResponse),
		done:      make(chan struct{}),
	}
}

func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	c.mu.Lock()
	if c.initialized {
		result := c.initResult
		c.mu.Unlock()
		return result, nil
	}
	c.mu.Unlock()

	params := InitializeParams{
		ProtocolVersion: protocolVersion,
		ClientInfo: ClientInfo{
			Name:    clientName,
			Version: clientVersion,
		},
		Capabilities: Capabilities{},
	}

	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	notification := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	if err := c.transport.SendNotification(&notification); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.mu.Lock()
	c.initialized = true
	c.initResult = &result
	c.mu.Unlock()

	return &result, nil
}

func (c *Client) ListTools(ctx context.Context) ([]MCPTool, error) {
	if !c.isInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	var result ListToolsResult
	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}

	return result.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	if !c.isInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	params := CallToolParams{
		Name:      name,
		Arguments: args,
	}

	var result CallToolResult
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return nil, fmt.Errorf("tools/call failed: %w", err)
	}

	return &result, nil
}

func (c *Client) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	responseChan := make(chan *JSONRPCResponse, 1)
	c.pending[id] = responseChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.transport.Send(request); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	c.receiveOnce.Do(func() {
		go c.receiveLoop()
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case response := <-responseChan:
		if response.Error != nil {
			return fmt.Errorf("JSON-RPC error %d: %s", response.Error.Code, response.Error.Message)
		}

		if result != nil && len(response.Result) > 0 {
			if err := json.Unmarshal(response.Result, result); err != nil {
				return fmt.Errorf("failed to unmarshal result: %w", err)
			}
		}

		return nil
	}
}

func (c *Client) receiveLoop() {
	defer close(c.done)
	for {
		response, err := c.transport.Receive()
		if err != nil {
			c.mu.Lock()
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = make(map[int]chan *JSONRPCResponse)
			c.mu.Unlock()
			return
		}

		c.mu.Lock()
		responseChan, ok := c.pending[response.ID]
		c.mu.Unlock()

		if ok {
			select {
			case responseChan <- response:
			default:
			}
		}
	}
}

func (c *Client) isInitialized() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initialized
}

func (c *Client) Close() error {
	return c.transport.Close()
}
