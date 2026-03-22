package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type scriptedTransport struct {
	sendFn             func(*JSONRPCRequest) error
	notificationFn     func(*JSONRPCNotification) error
	receiveFn          func() (*JSONRPCResponse, error)
	closeFn            func() error
	closeCalled        bool
	notificationCalled bool
	requestCount       int
}

func (s *scriptedTransport) Send(req *JSONRPCRequest) error {
	s.requestCount++
	if s.sendFn != nil {
		return s.sendFn(req)
	}
	return nil
}

func (s *scriptedTransport) SendNotification(notification *JSONRPCNotification) error {
	s.notificationCalled = true
	if s.notificationFn != nil {
		return s.notificationFn(notification)
	}
	return nil
}

func (s *scriptedTransport) Receive() (*JSONRPCResponse, error) {
	if s.receiveFn != nil {
		return s.receiveFn()
	}
	return nil, io.EOF
}

func (s *scriptedTransport) Close() error {
	s.closeCalled = true
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}

func initializedResultJSON() json.RawMessage {
	data, _ := json.Marshal(InitializeResult{
		ProtocolVersion: protocolVersion,
		ServerInfo:      ServerInfo{Name: "test-server", Version: "1.0.0"},
		Capabilities:    ServerCapabilities{},
	})
	return data
}

func TestClientGuardsAndCaching(t *testing.T) {
	transport := newMockTransport(func(req *JSONRPCRequest) *JSONRPCResponse {
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: initializedResultJSON()}
	})
	if err := transport.Start(context.Background()); err != nil {
		t.Fatalf("start mock transport: %v", err)
	}
	client := NewClient(transport)

	if _, err := client.ListTools(context.Background()); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected list tools initialization error, got %v", err)
	}
	if _, err := client.CallTool(context.Background(), "echo", nil); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected call tool initialization error, got %v", err)
	}

	ctx := context.Background()
	if _, err := client.Initialize(ctx); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	if len(transport.notifications) != 1 {
		t.Fatal("expected initialized notification to be sent")
	}
	requestCount := len(transport.requests)
	if _, err := client.Initialize(ctx); err != nil {
		t.Fatalf("second initialize failed: %v", err)
	}
	if len(transport.requests) != requestCount {
		t.Fatalf("expected second initialize to reuse cached result, requestCount=%d want %d", len(transport.requests), requestCount)
	}
}

func TestClientErrorPaths(t *testing.T) {
	t.Run("notification failure", func(t *testing.T) {
		requestIDs := make(chan int, 1)
		transport := &scriptedTransport{}
		client := NewClient(transport)
		transport.sendFn = func(req *JSONRPCRequest) error {
			requestIDs <- req.ID
			return nil
		}
		transport.receiveFn = func() (*JSONRPCResponse, error) {
			id := <-requestIDs
			return &JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: initializedResultJSON()}, nil
		}
		transport.notificationFn = func(*JSONRPCNotification) error { return errors.New("notification failed") }
		if _, err := client.Initialize(context.Background()); err == nil || !strings.Contains(err.Error(), "initialized notification") {
			t.Fatalf("expected notification error, got %v", err)
		}
	})

	t.Run("send failure", func(t *testing.T) {
		client := NewClient(&scriptedTransport{sendFn: func(*JSONRPCRequest) error { return errors.New("send failed") }})
		var result InitializeResult
		if err := client.call(context.Background(), "initialize", nil, &result); err == nil || !strings.Contains(err.Error(), "failed to send request") {
			t.Fatalf("expected send failure, got %v", err)
		}
	})

	t.Run("json rpc error", func(t *testing.T) {
		transport := newMockTransport(func(req *JSONRPCRequest) *JSONRPCResponse {
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &JSONRPCError{Code: -32000, Message: "boom"}}
		})
		if err := transport.Start(context.Background()); err != nil {
			t.Fatalf("start mock transport: %v", err)
		}
		client := NewClient(transport)
		var result InitializeResult
		if err := client.call(context.Background(), "initialize", nil, &result); err == nil || !strings.Contains(err.Error(), "JSON-RPC error -32000: boom") {
			t.Fatalf("expected JSON-RPC error, got %v", err)
		}
	})

	t.Run("unmarshal error", func(t *testing.T) {
		transport := newMockTransport(func(req *JSONRPCRequest) *JSONRPCResponse {
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(`{"bad"`)}
		})
		if err := transport.Start(context.Background()); err != nil {
			t.Fatalf("start mock transport: %v", err)
		}
		client := NewClient(transport)
		var result InitializeResult
		if err := client.call(context.Background(), "initialize", nil, &result); err == nil || !strings.Contains(err.Error(), "failed to unmarshal result") {
			t.Fatalf("expected unmarshal error, got %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		transport := &scriptedTransport{
			sendFn: func(*JSONRPCRequest) error { return nil },
			receiveFn: func() (*JSONRPCResponse, error) {
				time.Sleep(50 * time.Millisecond)
				return nil, io.EOF
			},
		}
		client := NewClient(transport)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		var result InitializeResult
		if err := client.call(ctx, "initialize", nil, &result); err == nil || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context deadline exceeded, got %v", err)
		}
	})

	t.Run("close delegates to transport", func(t *testing.T) {
		transport := &scriptedTransport{}
		client := NewClient(transport)
		if err := client.Close(); err != nil {
			t.Fatalf("close failed: %v", err)
		}
		if !transport.closeCalled {
			t.Fatal("expected transport close to be called")
		}
	})
}

func TestClientReceiveLoopCleanup(t *testing.T) {
	client := NewClient(&scriptedTransport{receiveFn: func() (*JSONRPCResponse, error) { return nil, io.EOF }})
	ch1 := make(chan *JSONRPCResponse)
	ch2 := make(chan *JSONRPCResponse)
	client.pending[1] = ch1
	client.pending[2] = ch2

	client.receiveLoop()

	if len(client.pending) != 0 {
		t.Fatalf("expected pending map to be cleared, got %d", len(client.pending))
	}
	select {
	case _, ok := <-ch1:
		if ok {
			t.Fatal("expected channel 1 to be closed")
		}
	default:
		t.Fatal("expected channel 1 close to be observable")
	}
	select {
	case <-client.done:
	default:
		t.Fatal("expected done channel to be closed")
	}
}

func TestManagerLifecycleAndProxyExecution(t *testing.T) {
	manager := NewManager()
	if err := manager.AddServer(ServerConfig{Name: "helper", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess", "--"}, Env: map[string]string{"GO_WANT_MCP_HELPER_PROCESS": "1", "GO_MCP_HELPER_MODE": "mcp"}}); err != nil {
		t.Fatalf("add server failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("manager start failed: %v", err)
	}
	defer manager.Close()

	proxies, err := manager.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("discover tools failed: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(proxies))
	}
	if proxies[0].Name() != "mcp_helper_echo" {
		t.Fatalf("proxy name = %q, want mcp_helper_echo", proxies[0].Name())
	}

	params := proxies[0].Parameters()
	if len(params) != 1 || params[0].Name != "message" || !params[0].Required {
		t.Fatalf("unexpected proxy params: %#v", params)
	}

	toolResult, err := proxies[0].Execute(ctx, map[string]interface{}{"message": "hi"})
	if err != nil {
		t.Fatalf("proxy execute failed: %v", err)
	}
	if toolResult.ForUser != "hello from helper" || toolResult.ForLLM != "hello from helper" || toolResult.IsError {
		t.Fatalf("unexpected tool result: %#v", toolResult)
	}

	callResult, err := manager.callTool(ctx, "helper", "echo", map[string]interface{}{"message": "hi"})
	if err != nil {
		t.Fatalf("manager callTool failed: %v", err)
	}
	if len(callResult.Content) != 1 || callResult.Content[0].Text != "hello from helper" {
		t.Fatalf("unexpected callTool result: %#v", callResult)
	}
}

func TestManagerFailures(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	proxies, err := manager.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("discover tools on empty manager failed: %v", err)
	}
	if len(proxies) != 0 {
		t.Fatalf("expected no proxies, got %d", len(proxies))
	}

	if _, err := manager.callTool(ctx, "missing", "echo", nil); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing server error, got %v", err)
	}

	if err := manager.AddServer(ServerConfig{Name: "helper", Command: "definitely-not-a-real-command"}); err != nil {
		t.Fatalf("add server failed: %v", err)
	}
	if err := manager.Start(context.Background()); err == nil {
		t.Fatal("expected manager start to fail for invalid command")
	}

	uninitialized := NewManager()
	if err := uninitialized.AddServer(ServerConfig{Name: "idle", Command: "noop"}); err != nil {
		t.Fatalf("add uninitialized server: %v", err)
	}
	if _, err := uninitialized.callTool(ctx, "idle", "echo", nil); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected uninitialized server error, got %v", err)
	}

	proxy := MCPToolProxy{
		serverName: "missing",
		mcpTool:    MCPTool{Name: "echo", InputSchema: json.RawMessage(`not-json`)},
		manager:    NewManager(),
	}
	if params := proxy.Parameters(); params != nil {
		t.Fatalf("expected nil params for invalid schema, got %#v", params)
	}
	toolResult, err := proxy.Execute(ctx, nil)
	if err == nil || toolResult == nil || !toolResult.IsError {
		t.Fatalf("expected proxy execution error result, got result=%#v err=%v", toolResult, err)
	}
}
