package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

func seedSessionDatabase(t *testing.T, configPath string, sessions ...*session.Session) {
	t.Helper()
	adapter, err := session.NewSQLiteAdapter(filepath.Join(filepath.Dir(configPath), "sessions.db"))
	if err != nil {
		t.Fatalf("open sqlite adapter: %v", err)
	}
	defer adapter.Close()
	for _, sess := range sessions {
		if err := adapter.Save(sess); err != nil {
			t.Fatalf("save session %s: %v", sess.ID, err)
		}
	}
}

func TestSessionListShowDeleteFlow(t *testing.T) {
	configPath := newTempConfigPath(t)
	sessA := session.NewSession("sess-a")
	sessA.CreatedAt = time.Now().Add(-2 * time.Hour)
	sessA.UpdatedAt = time.Now().Add(-time.Hour)
	sessA.AddMessage(providers.Message{Role: providers.RoleUser, Content: "hello"})
	sessA.UpdatedAt = time.Now().Add(-time.Hour)

	sessB := session.NewSession("sess-b")
	sessB.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "world"})
	seedSessionDatabase(t, configPath, sessA, sessB)

	stdout, stderr, err := executeRootCommand(t, "--config", configPath, "session", "list")
	if err != nil {
		t.Fatalf("session list failed: %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, "sess-a") || !strings.Contains(stdout, "sess-b") {
		t.Fatalf("session list output missing sessions: %q", stdout)
	}

	stdout, stderr, err = executeRootCommand(t, "--config", configPath, "session", "show", "sess-a")
	if err != nil {
		t.Fatalf("session show failed: %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, "Session: sess-a") || !strings.Contains(stdout, "hello") {
		t.Fatalf("session show output unexpected: %q", stdout)
	}

	stdout, stderr, err = executeRootCommand(t, "--config", configPath, "session", "delete", "sess-a")
	if err != nil {
		t.Fatalf("session delete failed: %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, "Deleted session sess-a") {
		t.Fatalf("session delete output unexpected: %q", stdout)
	}

	_, _, err = executeRootCommand(t, "--config", configPath, "session", "show", "sess-a")
	if err == nil {
		t.Fatal("expected deleted session lookup to fail")
	}
}

func TestSessionListEmptyStore(t *testing.T) {
	configPath := newTempConfigPath(t)
	stdout, stderr, err := executeRootCommand(t, "--config", configPath, "session", "list")
	if err != nil {
		t.Fatalf("session list on empty store failed: %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, "No sessions found.") {
		t.Fatalf("unexpected empty session list output: %q", stdout)
	}
}
