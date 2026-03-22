package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type mockHandler struct {
	tokens []string
	err    error
}

func (m *mockHandler) HandleMessageStream(_ context.Context, _ string, _ string, out chan<- string) error {
	defer close(out)
	for _, t := range m.tokens {
		out <- t
	}
	return m.err
}

func TestModelInitialState(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	if m.thinking {
		t.Error("expected thinking=false on init")
	}
	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(m.messages))
	}
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
}

func TestModelKeyInput(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = raw.(Model)
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = raw.(Model)

	if m.input != "hi" {
		t.Errorf("input = %q, want %q", m.input, "hi")
	}
}

func TestModelBackspace(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	for _, ch := range "abc" {
		raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = raw.(Model)
	}
	raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = raw.(Model)

	if m.input != "ab" {
		t.Errorf("input after backspace = %q, want %q", m.input, "ab")
	}
}

func TestModelEnterSendsMessage(t *testing.T) {
	handler := &mockHandler{tokens: []string{"hello"}}
	m := New(context.Background(), "sess", handler)

	for _, ch := range "hi" {
		raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = raw.(Model)
	}

	raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = raw.(Model)

	if !m.thinking {
		t.Error("expected thinking=true after Enter")
	}
	if len(m.messages) != 1 || m.messages[0].role != roleUser || m.messages[0].content != "hi" {
		t.Errorf("unexpected messages: %+v", m.messages)
	}
	if m.input != "" {
		t.Errorf("expected input cleared, got %q", m.input)
	}
}

func TestModelTokenReceived(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	raw, _ := m.Update(tokenMsg("hello "))
	m = raw.(Model)
	raw, _ = m.Update(tokenMsg("world"))
	m = raw.(Model)

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].content != "hello world" {
		t.Errorf("content = %q, want %q", m.messages[0].content, "hello world")
	}
}

func TestModelDoneClears(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	raw, _ := m.Update(doneMsg{})
	m = raw.(Model)

	if m.thinking {
		t.Error("expected thinking=false after doneMsg")
	}
}

func TestModelDoneWithError(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	raw, _ := m.Update(doneMsg{err: context.DeadlineExceeded})
	m = raw.(Model)

	if m.thinking {
		t.Error("expected thinking=false after doneMsg with error")
	}
	if m.lastError == "" {
		t.Error("expected lastError to be set")
	}
}

func TestModelQuitOnQ(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit command on 'q' with empty input")
	}
}

func TestModelNoQuitOnQWithInput(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.input = "query"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Error("should not quit when input is non-empty")
	}
}

func TestModelViewContainsInput(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.input = "test input"

	view := m.View()
	if !strings.Contains(view, "test input") {
		t.Errorf("view does not contain input: %q", view)
	}
}

func TestModelViewThinking(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	view := m.View()
	if !strings.Contains(view, "…") {
		t.Errorf("view does not contain thinking indicator: %q", view)
	}
}

func TestViewportInitialization(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	if m.ready {
		t.Error("expected ready=false before WindowSizeMsg")
	}

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = raw.(Model)

	if !m.ready {
		t.Error("expected ready=true after WindowSizeMsg")
	}
	if m.width != 80 {
		t.Errorf("width = %d, want 80", m.width)
	}
	if m.height != 24 {
		t.Errorf("height = %d, want 24", m.height)
	}
	if m.viewport.Width != 80 {
		t.Errorf("viewport width = %d, want 80", m.viewport.Width)
	}
	if m.viewport.Height != 22 {
		t.Errorf("viewport height = %d, want 22 (24-2 for input)", m.viewport.Height)
	}
}

func TestViewportResize(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = raw.(Model)

	raw, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = raw.(Model)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 30 {
		t.Errorf("height = %d, want 30", m.height)
	}
	if m.viewport.Width != 100 {
		t.Errorf("viewport width = %d, want 100", m.viewport.Width)
	}
	if m.viewport.Height != 28 {
		t.Errorf("viewport height = %d, want 28 (30-2 for input)", m.viewport.Height)
	}
}

func TestViewportMinimumHeight(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 2})
	m = raw.(Model)

	if m.viewport.Height != 1 {
		t.Errorf("viewport height = %d, want minimum 1", m.viewport.Height)
	}
}

func TestScrollUp(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 20 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoBottom()

	initialYOffset := m.viewport.YOffset
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = raw.(Model)

	if m.viewport.YOffset >= initialYOffset {
		t.Errorf("expected YOffset to decrease after scroll up, got %d -> %d", initialYOffset, m.viewport.YOffset)
	}
}

func TestScrollDown(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 20 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoTop()

	initialYOffset := m.viewport.YOffset
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = raw.(Model)

	if m.viewport.YOffset <= initialYOffset {
		t.Errorf("expected YOffset to increase after scroll down, got %d -> %d", initialYOffset, m.viewport.YOffset)
	}
}

func TestScrollPageUp(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 50 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoBottom()

	initialYOffset := m.viewport.YOffset
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = raw.(Model)

	if m.viewport.YOffset >= initialYOffset {
		t.Errorf("expected YOffset to decrease after page up, got %d -> %d", initialYOffset, m.viewport.YOffset)
	}
}

func TestScrollPageDown(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 50 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoTop()

	initialYOffset := m.viewport.YOffset
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = raw.(Model)

	if m.viewport.YOffset <= initialYOffset {
		t.Errorf("expected YOffset to increase after page down, got %d -> %d", initialYOffset, m.viewport.YOffset)
	}
}

func TestScrollCtrlU(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 50 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoBottom()

	initialYOffset := m.viewport.YOffset
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = raw.(Model)

	if m.viewport.YOffset >= initialYOffset {
		t.Errorf("expected YOffset to decrease after Ctrl+U, got %d -> %d", initialYOffset, m.viewport.YOffset)
	}
}

func TestScrollCtrlD(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 50 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoTop()

	initialYOffset := m.viewport.YOffset
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = raw.(Model)

	if m.viewport.YOffset <= initialYOffset {
		t.Errorf("expected YOffset to increase after Ctrl+D, got %d -> %d", initialYOffset, m.viewport.YOffset)
	}
}

func TestScrollHome(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 50 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoBottom()

	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m = raw.(Model)

	if !m.viewport.AtTop() {
		t.Errorf("expected viewport at top after Home key, YOffset=%d", m.viewport.YOffset)
	}
	if m.atBottom {
		t.Error("expected atBottom=false after Home key")
	}
}

func TestScrollEnd(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 50 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoTop()

	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = raw.(Model)

	if !m.viewport.AtBottom() {
		t.Errorf("expected viewport at bottom after End key, YOffset=%d", m.viewport.YOffset)
	}
	if !m.atBottom {
		t.Error("expected atBottom=true after End key")
	}
}

func TestAutoScrollWhenAtBottom(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 20 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoBottom()
	m.atBottom = true

	raw, _ = m.Update(tokenMsg("new content"))
	m = raw.(Model)

	if !m.viewport.AtBottom() {
		t.Error("expected viewport to stay at bottom when new content arrives while at bottom")
	}
}

func TestPreserveScrollPositionWhenNotAtBottom(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 30 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoTop()
	m.atBottom = false

	initialYOffset := m.viewport.YOffset

	raw, _ = m.Update(tokenMsg("new content"))
	m = raw.(Model)

	if m.viewport.YOffset != initialYOffset {
		t.Errorf("expected YOffset to stay at %d, got %d (should preserve scroll position)", initialYOffset, m.viewport.YOffset)
	}
}

func TestScrollKeysDoNotAffectInput(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	m.input = "test input"

	scrollKeys := []tea.KeyMsg{
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyPgUp},
		{Type: tea.KeyPgDown},
		{Type: tea.KeyHome},
		{Type: tea.KeyEnd},
		{Type: tea.KeyCtrlU},
		{Type: tea.KeyCtrlD},
	}

	for _, key := range scrollKeys {
		raw, _ = m.Update(key)
		m = raw.(Model)
		if m.input != "test input" {
			t.Errorf("scroll key %v modified input: got %q, want %q", key.Type, m.input, "test input")
		}
	}
}

func TestAtBottomTracking(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = raw.(Model)

	for range 30 {
		m.messages = append(m.messages, chatMsg{role: roleUser, content: "line"})
	}
	m.viewport.SetContent(m.buildTranscript())
	m.viewport.GotoBottom()
	m.atBottom = true

	if !m.atBottom {
		t.Error("expected atBottom=true after GotoBottom")
	}

	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = raw.(Model)

	if m.atBottom {
		t.Error("expected atBottom=false after scrolling up from bottom")
	}

	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = raw.(Model)

	if !m.atBottom {
		t.Error("expected atBottom=true after pressing End key")
	}
}

func TestViewBeforeReady(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.input = "hello"

	view := m.View()

	if !strings.Contains(view, "hello") {
		t.Errorf("view should contain input before ready: %q", view)
	}
}

func TestViewAfterReady(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.input = "hello"

	raw, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = raw.(Model)

	view := m.View()

	if !strings.Contains(view, "hello") {
		t.Errorf("view should contain input after ready: %q", view)
	}
}
