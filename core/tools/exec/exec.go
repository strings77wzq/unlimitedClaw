// Package exec provides the "exec" tool that lets the AI agent run shell
// commands in a sandboxed working directory. Commands are executed with a
// configurable timeout; output (stdout + stderr combined) is returned as
// the tool result. The working directory is set at construction time via [New]
// and cannot be escaped by the agent.
package exec

import (
	"bytes"
	"context"
	"fmt"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/strings77wzq/unlimitedClaw/core/tools"
)

type ExecTool struct {
	workspace string
	timeout   time.Duration
	denyList  []string
}

type Option func(*ExecTool)

func WithTimeout(d time.Duration) Option {
	return func(t *ExecTool) {
		t.timeout = d
	}
}

func WithDenyList(commands []string) Option {
	return func(t *ExecTool) {
		t.denyList = append(t.denyList, commands...)
	}
}

var defaultDenyList = []string{
	"rm -rf /",
	"rm -rf ~",
	"shutdown",
	"reboot",
	"halt",
	"poweroff",
	"mkfs",
	"dd if=",
	":(){ :|:& };:",
	"chmod -R 777 /",
	"wget",
	"curl",
}

func New(workspace string, opts ...Option) *ExecTool {
	tool := &ExecTool{
		workspace: workspace,
		timeout:   30 * time.Second,
		denyList:  make([]string, len(defaultDenyList)),
	}

	copy(tool.denyList, defaultDenyList)

	for _, opt := range opts {
		opt(tool)
	}

	return tool
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command within the workspace sandbox"
}

func (t *ExecTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "command",
			Type:        "string",
			Description: "The shell command to execute",
			Required:    true,
		},
		{
			Name:        "timeout",
			Type:        "number",
			Description: "Timeout in seconds (default: 30)",
			Required:    false,
		},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	command, ok := args["command"].(string)
	if !ok {
		return &tools.ToolResult{
			ForLLM:  "command parameter is required and must be a string",
			IsError: true,
		}, nil
	}

	timeout := t.timeout
	if timeoutVal, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	if blocked, reason := t.isBlocked(command); blocked {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Command blocked by sandbox: %s", reason),
			IsError: true,
		}, nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := osexec.CommandContext(cmdCtx, "sh", "-c", command)
	cmd.Dir = t.workspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if len(output) > 0 {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded || strings.Contains(err.Error(), "signal: killed") {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("Command timed out after %v", timeout),
				ForUser: fmt.Sprintf("$ %s\nTimeout after %v", command, timeout),
				IsError: true,
			}, nil
		}

		exitMsg := fmt.Sprintf("Command failed: %v", err)
		if len(output) > 0 {
			output = output + "\n" + exitMsg
		} else {
			output = exitMsg
		}

		forUser := fmt.Sprintf("$ %s\n%s", command, output)
		if len(forUser) > 500 {
			forUser = fmt.Sprintf("$ %s\n%s", command, forUser[len("$ "+command+"\n"):500])
		}

		forLLM := output
		if len(forLLM) > 10000 {
			forLLM = forLLM[:10000] + fmt.Sprintf("\n... (truncated, %d more chars)", len(forLLM)-10000)
		}

		return &tools.ToolResult{
			ForLLM:  forLLM,
			ForUser: forUser,
			IsError: true,
		}, nil
	}

	if len(output) == 0 {
		output = "(no output)"
	}

	forUser := fmt.Sprintf("$ %s\n%s", command, output)
	if len(forUser) > 500 {
		forUser = fmt.Sprintf("$ %s\n%s", command, output[:min(500-len("$ "+command+"\n"), len(output))])
	}

	forLLM := output
	if len(forLLM) > 10000 {
		forLLM = forLLM[:10000] + fmt.Sprintf("\n... (truncated, %d more chars)", len(forLLM)-10000)
	}

	return &tools.ToolResult{
		ForLLM:  forLLM,
		ForUser: forUser,
	}, nil
}

func (t *ExecTool) isBlocked(command string) (bool, string) {
	cmdLower := strings.ToLower(command)

	for _, denied := range t.denyList {
		deniedLower := strings.ToLower(denied)

		if strings.Contains(deniedLower, "|") && strings.Contains(cmdLower, "|") {
			if strings.Contains(cmdLower, deniedLower) {
				return true, fmt.Sprintf("contains denied pattern: %s", denied)
			}
		}

		if strings.Contains(cmdLower, deniedLower) {
			return true, fmt.Sprintf("contains denied command: %s", denied)
		}
	}

	return false, ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
