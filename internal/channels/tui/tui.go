// Package tui implements the interactive Bubble Tea terminal UI for the AI
// agent. It is the ONLY package in the entire codebase that may import
// github.com/charmbracelet/bubbletea — all other packages must remain free
// of that dependency. Token streaming is driven by a recursive Cmd chain
// (waitNextToken) that consumes a chan string without additional goroutines.
// The TUI auto-activates when stdin is a TTY; use --no-tui to suppress it.
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MessageHandler is the subset of agent.MessageHandler needed by the TUI.
type MessageHandler interface {
	HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error
}

type role int

const (
	roleUser role = iota
	roleAssistant
)

type chatMsg struct {
	role    role
	content string
}

type tokenMsg string
type doneMsg struct{ err error }

var (
	userStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	assistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	promptStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
	agent     MessageHandler
	sessionID string
	ctx       context.Context
	cancel    context.CancelFunc

	messages  []chatMsg
	tokenCh   <-chan string
	input     string
	thinking  bool
	lastError string
}

func New(ctx context.Context, sessionID string, handler MessageHandler) Model {
	childCtx, cancel := context.WithCancel(ctx)
	return Model{
		agent:     handler,
		sessionID: sessionID,
		ctx:       childCtx,
		cancel:    cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tokenMsg:
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].role == roleAssistant {
			m.messages[len(m.messages)-1].content += string(msg)
		} else {
			m.messages = append(m.messages, chatMsg{role: roleAssistant, content: string(msg)})
		}
		return m, waitNextToken(m.tokenCh)

	case doneMsg:
		m.thinking = false
		m.tokenCh = nil
		if msg.err != nil {
			m.lastError = msg.err.Error()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancel()
		return m, tea.Quit

	case tea.KeyRunes:
		if string(msg.Runes) == "q" && !m.thinking && m.input == "" {
			m.cancel()
			return m, tea.Quit
		}
		if !m.thinking {
			m.input += string(msg.Runes)
		}

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	case tea.KeySpace:
		if !m.thinking {
			m.input += " "
		}

	case tea.KeyEnter:
		if m.thinking || strings.TrimSpace(m.input) == "" {
			return m, nil
		}
		text := m.input
		m.input = ""
		m.thinking = true
		m.lastError = ""
		m.messages = append(m.messages, chatMsg{role: roleUser, content: text})
		tokens := make(chan string, 64)
		m.tokenCh = tokens
		return m, tea.Batch(m.startStream(text, tokens), waitNextToken(tokens))
	}

	return m, nil
}

func (m Model) startStream(text string, tokens chan<- string) tea.Cmd {
	return func() tea.Msg {
		err := m.agent.HandleMessageStream(m.ctx, m.sessionID, text, tokens)
		if err != nil {
			return doneMsg{err: err}
		}
		return nil
	}
}

func waitNextToken(tokens <-chan string) tea.Cmd {
	if tokens == nil {
		return nil
	}
	return func() tea.Msg {
		tok, ok := <-tokens
		if !ok {
			return doneMsg{}
		}
		return tokenMsg(tok)
	}
}

func (m Model) View() string {
	var b strings.Builder

	for _, msg := range m.messages {
		switch msg.role {
		case roleUser:
			b.WriteString(userStyle.Render("You: "))
			b.WriteString(msg.content)
		case roleAssistant:
			b.WriteString(assistantStyle.Render("AI:  "))
			b.WriteString(msg.content)
		}
		b.WriteString("\n")
	}

	if m.lastError != "" {
		b.WriteString(errorStyle.Render(fmt.Sprintf("error: %s", m.lastError)))
		b.WriteString("\n")
	}

	if m.thinking {
		b.WriteString(promptStyle.Render("AI:  ") + "…\n")
	}

	b.WriteString(promptStyle.Render("> "))
	b.WriteString(m.input)

	return b.String()
}

// Run starts the TUI program and blocks until the user quits.
func Run(ctx context.Context, sessionID string, handler MessageHandler) error {
	m := New(ctx, sessionID, handler)
	p := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}
