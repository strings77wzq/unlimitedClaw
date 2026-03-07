package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type StdioTransport struct {
	command string
	args    []string
	env     []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	scanner *bufio.Scanner
	mu      sync.Mutex
	started bool
	closed  bool
}

func NewStdioTransport(command string, args []string, env []string) *StdioTransport {
	return &StdioTransport{
		command: command,
		args:    args,
		env:     env,
	}
}

func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("transport already started")
	}
	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	t.cmd = exec.CommandContext(ctx, t.command, t.args...)

	if len(t.env) > 0 {
		t.cmd.Env = t.env
	}

	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	t.scanner = bufio.NewScanner(t.stdout)
	t.started = true

	return nil
}

func (t *StdioTransport) Send(request *JSONRPCRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return fmt.Errorf("transport not started")
	}
	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	data = append(data, '\n')
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

func (t *StdioTransport) SendNotification(notification *JSONRPCNotification) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return fmt.Errorf("transport not started")
	}
	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	data = append(data, '\n')
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}

func (t *StdioTransport) Receive() (*JSONRPCResponse, error) {
	if !t.started {
		return nil, fmt.Errorf("transport not started")
	}
	if t.closed {
		return nil, fmt.Errorf("transport is closed")
	}

	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanner error: %w", err)
		}
		return nil, io.EOF
	}

	line := t.scanner.Bytes()
	var response JSONRPCResponse
	if err := json.Unmarshal(line, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	var errs []error

	if t.stdin != nil {
		if err := t.stdin.Close(); err != nil {
			errs = append(errs, fmt.Errorf("stdin close: %w", err))
		}
	}

	if t.cmd != nil && t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			errs = append(errs, fmt.Errorf("process kill: %w", err))
		}
	}

	if t.stdout != nil {
		if err := t.stdout.Close(); err != nil {
			errs = append(errs, fmt.Errorf("stdout close: %w", err))
		}
	}

	if t.stderr != nil {
		if err := t.stderr.Close(); err != nil {
			errs = append(errs, fmt.Errorf("stderr close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}
