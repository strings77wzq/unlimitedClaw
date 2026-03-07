package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/strings77wzq/unlimitedClaw/core/session"
)

func newSessionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage conversation sessions",
	}

	cmd.AddCommand(
		newSessionListCommand(),
		newSessionShowCommand(),
		newSessionDeleteCommand(),
	)

	return cmd
}

func newSessionListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all saved sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			adapter, err := openSessionStore(cmd)
			if err != nil {
				return err
			}
			defer adapter.Close()

			sessions := adapter.List()
			if len(sessions) == 0 {
				fmt.Println("No sessions found.")
				return nil
			}

			fmt.Printf("%-36s  %-8s  %-20s\n", "SESSION ID", "MESSAGES", "UPDATED")
			for _, s := range sessions {
				fmt.Printf("%-36s  %-8d  %s\n",
					s.ID, s.MessageCount(), s.UpdatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
}

func newSessionShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <session-id>",
		Short: "Show messages in a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adapter, err := openSessionStore(cmd)
			if err != nil {
				return err
			}
			defer adapter.Close()

			sess, ok := adapter.Get(args[0])
			if !ok {
				return fmt.Errorf("session %q not found", args[0])
			}

			msgs := sess.GetMessages()
			fmt.Printf("Session: %s (%d messages)\n\n", sess.ID, len(msgs))
			for i, msg := range msgs {
				fmt.Printf("[%d] %s:\n%s\n\n", i+1, msg.Role, msg.Content)
			}
			return nil
		},
	}
}

func newSessionDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <session-id>",
		Short: "Delete a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adapter, err := openSessionStore(cmd)
			if err != nil {
				return err
			}
			defer adapter.Close()

			if err := adapter.Delete(args[0]); err != nil {
				return fmt.Errorf("deleting session: %w", err)
			}
			fmt.Printf("Deleted session %s\n", args[0])
			return nil
		},
	}
}

func openSessionStore(cmd *cobra.Command) (*session.SQLiteAdapter, error) {
	configPath, err := getConfigPath(cmd)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(configPath)
	dbPath := filepath.Join(dir, "sessions.db")

	if err := ensureConfigDir(configPath); err != nil {
		return nil, err
	}

	return session.NewSQLiteAdapter(dbPath)
}
