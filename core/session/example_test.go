package session_test

import (
	"fmt"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

func ExampleNewSession() {
	sess := session.NewSession("test-session")

	msg := providers.Message{
		Role:    providers.RoleUser,
		Content: "hello",
	}
	sess.AddMessage(msg)

	fmt.Println(sess.MessageCount())
	// Output: 1
}

func ExampleNewMemoryStore() {
	store := session.NewMemoryStore()

	sess := session.NewSession("session-123")
	store.Save(sess)

	retrieved, ok := store.Get("session-123")
	if ok {
		fmt.Println(retrieved.ID)
	}
	// Output: session-123
}
