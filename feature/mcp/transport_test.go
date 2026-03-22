package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func helperTransportEnv(mode string) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, "GO_WANT_MCP_HELPER_PROCESS=1", fmt.Sprintf("GO_MCP_HELPER_MODE=%s", mode))
	return env
}

func newHelperTransport(mode string) *StdioTransport {
	return NewStdioTransport(os.Args[0], []string{"-test.run=TestMCPHelperProcess", "--"}, helperTransportEnv(mode))
}

func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER_PROCESS") != "1" {
		return
	}

	mode := os.Getenv("GO_MCP_HELPER_MODE")
	switch mode {
	case "echo":
		helperServeEcho()
	case "invalid-json":
		_, _ = os.Stdout.WriteString("not-json\n")
	case "exit":
		return
	case "mcp":
		helperServeMCP()
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode: %s", mode)
	}
	os.Exit(0)
}

func helperServeEcho() {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var raw map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}
		idValue, ok := raw["id"]
		if !ok {
			continue
		}
		id := int(idValue.(float64))
		_ = encoder.Encode(JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: json.RawMessage(`{"status":"ok"}`)})
	}
}

func helperServeMCP() {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{
				ProtocolVersion: protocolVersion,
				ServerInfo:      ServerInfo{Name: "helper", Version: "1.0.0"},
				Capabilities:    ServerCapabilities{Tools: &ToolsCapability{}},
			})
			_ = encoder.Encode(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
		case "tools/list":
			result, _ := json.Marshal(ListToolsResult{Tools: []MCPTool{{
				Name:        "echo",
				Description: "Echo helper tool",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"message"}},"required":["message"]}`),
			}}})
			_ = encoder.Encode(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
		case "tools/call":
			result, _ := json.Marshal(CallToolResult{Content: []ContentBlock{{Type: "text", Text: "hello from helper"}}})
			_ = encoder.Encode(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
		}
	}
}

func TestNewStdioTransport(t *testing.T) {
	transport := NewStdioTransport("cmd", []string{"arg"}, []string{"A=B"})
	if transport.command != "cmd" {
		t.Fatalf("command = %q, want cmd", transport.command)
	}
	if len(transport.args) != 1 || transport.args[0] != "arg" {
		t.Fatalf("args = %#v", transport.args)
	}
	if len(transport.env) != 1 || transport.env[0] != "A=B" {
		t.Fatalf("env = %#v", transport.env)
	}
}

func TestStdioTransportLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport := newHelperTransport("echo")
	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer transport.Close()

	if err := transport.Send(&JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "ping"}); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if err := transport.SendNotification(&JSONRPCNotification{JSONRPC: "2.0", Method: "notify"}); err != nil {
		t.Fatalf("SendNotification failed: %v", err)
	}

	resp, err := transport.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if resp.ID != 1 {
		t.Fatalf("response ID = %d, want 1", resp.ID)
	}
	if !strings.Contains(string(resp.Result), `"status":"ok"`) {
		t.Fatalf("unexpected response result: %s", string(resp.Result))
	}

	if err := transport.Start(ctx); err == nil {
		t.Fatal("expected second Start to fail")
	}
}

func TestStdioTransportMisuseAndErrors(t *testing.T) {
	transport := NewStdioTransport("cmd", nil, nil)
	if err := transport.Send(&JSONRPCRequest{}); err == nil {
		t.Fatal("expected send before start to fail")
	}
	if err := transport.SendNotification(&JSONRPCNotification{}); err == nil {
		t.Fatal("expected send notification before start to fail")
	}
	if _, err := transport.Receive(); err == nil {
		t.Fatal("expected receive before start to fail")
	}

	ctx := context.Background()
	transport = NewStdioTransport("cmd", nil, nil)
	_ = transport.Close()
	if err := transport.Start(ctx); err == nil {
		t.Fatal("expected start after close to fail")
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("second close should be nil: %v", err)
	}

	invalidTransport := newHelperTransport("invalid-json")
	if err := invalidTransport.Start(ctx); err != nil {
		t.Fatalf("start invalid helper: %v", err)
	}
	defer invalidTransport.Close()
	if _, err := invalidTransport.Receive(); err == nil || !strings.Contains(err.Error(), "failed to unmarshal response") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}

	eofTransport := NewStdioTransport("unused", nil, nil)
	eofTransport.started = true
	eofTransport.scanner = bufio.NewScanner(strings.NewReader(""))
	if _, err := eofTransport.Receive(); err == nil || err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	eofTransport.closed = true
	if err := eofTransport.Send(&JSONRPCRequest{}); err == nil {
		t.Fatal("expected send after close to fail")
	}
}
