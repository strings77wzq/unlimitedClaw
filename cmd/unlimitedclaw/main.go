// unlimitedClaw - Progressive Go AI Assistant
// Learning by building, inspired by PicoClaw
// License: MIT

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/strings77wzq/unlimitedClaw/pkg/agent"
	"github.com/strings77wzq/unlimitedClaw/pkg/bus"
	"github.com/strings77wzq/unlimitedClaw/pkg/config"
	"github.com/strings77wzq/unlimitedClaw/pkg/gateway"
	"github.com/strings77wzq/unlimitedClaw/pkg/logger"
	"github.com/strings77wzq/unlimitedClaw/pkg/providers"
	"github.com/strings77wzq/unlimitedClaw/pkg/providers/anthropic"
	"github.com/strings77wzq/unlimitedClaw/pkg/providers/openai"
	"github.com/strings77wzq/unlimitedClaw/pkg/session"
	"github.com/strings77wzq/unlimitedClaw/pkg/term"
	"github.com/strings77wzq/unlimitedClaw/pkg/tools"
)

// Version info injected at build time via ldflags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlimitedclaw",
		Short: "unlimitedClaw - Progressive AI Assistant",
		Long:  "A progressive Go AI assistant — learning by building, inspired by PicoClaw",
		Example: `  unlimitedclaw agent
  unlimitedclaw gateway
  unlimitedclaw version`,
	}

	cmd.PersistentFlags().StringP("config", "c", "", "config file path (default: ~/.unlimitedclaw/config.json)")

	cmd.AddCommand(
		newVersionCommand(),
		newAgentCommand(),
		newGatewayCommand(),
		newConfigCommand(),
		newStatusCommand(),
		newSessionCommand(),
	)

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("unlimitedclaw version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("date: %s\n", date)
			return nil
		},
	}
}

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Start the AI agent",
		Long:  "Start the unlimitedClaw AI agent process",
		RunE: func(cmd *cobra.Command, args []string) error {
			message, _ := cmd.Flags().GetString("message")
			modelFlag, _ := cmd.Flags().GetString("model")

			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			// Override default model if -M flag is set
			if modelFlag != "" {
				if _, findErr := cfg.FindModel(modelFlag); findErr != nil {
					return fmt.Errorf("model %q not found in config; available: %s", modelFlag, listModelNames(cfg))
				}
				cfg.Agents.Defaults.ModelName = modelFlag
			}

			b := bus.New()
			registry := tools.NewRegistry()
			factory := registerProviders(cfg)
			log := logger.New(logger.DefaultOptions())

			// Use SQLite-backed session store for persistence
			store, err := openAgentSessionStore(cmd)
			if err != nil {
				// Fall back to in-memory store if SQLite fails (e.g., read-only fs)
				log.Warn("SQLite session store unavailable, using in-memory", "err", err)
				store = nil
			}
			var sessionStore session.SessionStore
			if store != nil {
				defer store.Close()
				sessionStore = store
			} else {
				sessionStore = session.NewMemoryStore()
			}

			history := session.NewHistoryManager(cfg.Agents.Defaults.MaxTokens)
			ag := agent.New(b, registry, factory, sessionStore, history, log, cfg)

			// Handle piped stdin: combine with -m flag message
			if !term.IsInputTTY() {
				stdinContent, readErr := term.ReadStdin()
				if readErr != nil {
					return fmt.Errorf("reading stdin: %w", readErr)
				}
				stdinContent = strings.TrimSpace(stdinContent)
				if stdinContent != "" {
					if message != "" {
						message = message + "\n\n" + stdinContent
					} else {
						message = stdinContent
					}
				}
			}

			if message != "" {
				return runAgentOneShot(ag, b, message)
			}

			// If stdin is piped but no message came through, nothing to do
			if !term.IsInputTTY() {
				return fmt.Errorf("no input: use -m flag or pipe content via stdin")
			}

			return runAgentInteractive(ag, b)
		},
	}
	cmd.Flags().StringP("message", "m", "", "Send a single message and exit")
	cmd.Flags().StringP("model", "M", "", "Model to use (overrides config default)")
	return cmd
}

// registerProviders creates a Factory and registers providers from config ModelList.
func registerProviders(cfg *config.Config) *providers.Factory {
	factory := providers.NewFactory()

	// Track registered vendors to avoid re-registering with different keys
	registered := make(map[string]bool)

	for _, entry := range cfg.ModelList {
		vendor := entry.Vendor()
		if registered[vendor] {
			continue
		}
		registered[vendor] = true

		switch vendor {
		case "openai":
			var opts []openai.Option
			if entry.APIBase != "" {
				opts = append(opts, openai.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, openai.New(entry.APIKey, opts...))
		case "anthropic":
			var opts []anthropic.Option
			if entry.APIBase != "" {
				opts = append(opts, anthropic.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, anthropic.New(entry.APIKey, opts...))
		case "mock":
			factory.Register(vendor, providers.NewMockProvider("mock"))
		}
	}

	// Ensure mock is always available for testing/default config
	if !registered["mock"] {
		factory.Register("mock", providers.NewMockProvider("mock"))
	}

	return factory
}

// listModelNames returns a comma-separated list of configured model names.
func listModelNames(cfg *config.Config) string {
	names := make([]string, 0, len(cfg.ModelList))
	for _, entry := range cfg.ModelList {
		names = append(names, entry.ModelName)
	}
	return strings.Join(names, ", ")
}

// openAgentSessionStore opens the SQLite session store in the config directory.
func openAgentSessionStore(cmd *cobra.Command) (*session.SQLiteAdapter, error) {
	configPath, err := getConfigPath(cmd)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(configPath)
	if err := ensureConfigDir(configPath); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, "sessions.db")
	return session.NewSQLiteAdapter(dbPath)
}

func runAgentOneShot(ag *agent.Agent, b bus.Bus, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go ag.Start(ctx)

	sessionID := uuid.New().String()
	outCh := b.Subscribe(agent.TopicOutbound)
	defer b.Unsubscribe(agent.TopicOutbound, outCh)

	b.Publish(agent.TopicInbound, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	})

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for response")
		case raw := <-outCh:
			outMsg, ok := raw.(bus.OutboundMessage)
			if !ok {
				continue
			}
			if outMsg.SessionID != sessionID {
				continue
			}
			fmt.Print(outMsg.Content)
			if outMsg.Done {
				fmt.Println()
				return nil
			}
		}
	}
}

func runAgentInteractive(ag *agent.Agent, b bus.Bus) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go ag.Start(ctx)

	sessionID := uuid.New().String()
	outCh := b.Subscribe(agent.TopicOutbound)
	defer b.Unsubscribe(agent.TopicOutbound, outCh)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case raw := <-outCh:
				outMsg, ok := raw.(bus.OutboundMessage)
				if !ok {
					continue
				}
				if outMsg.SessionID != sessionID {
					continue
				}
				fmt.Print(outMsg.Content)
				if outMsg.Done {
					fmt.Printf("\n> ")
				}
			}
		}
	}()

	fmt.Println("Interactive mode: type messages, Ctrl+C to quit")
	fmt.Printf("> ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			fmt.Printf("> ")
			continue
		}

		b.Publish(agent.TopicInbound, bus.InboundMessage{
			SessionID: sessionID,
			Content:   line,
			Role:      bus.RoleUser,
		})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func newGatewayCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gateway",
		Short: "Start the HTTP gateway server",
		Long:  "Start the HTTP gateway server for agent communication",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			b := bus.New()
			registry := tools.NewRegistry()
			factory := registerProviders(cfg)
			store := session.NewMemoryStore()
			history := session.NewHistoryManager(cfg.Agents.Defaults.MaxTokens)
			log := logger.New(logger.DefaultOptions())

			ag := agent.New(b, registry, factory, store, history, log, cfg)

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			go ag.Start(ctx)

			handler := &agentBridge{bus: b, agent: ag}
			serverCfg := gateway.DefaultServerConfig()
			server := gateway.NewServer(serverCfg, handler, log)

			log.Info("starting gateway server", "addr", serverCfg.Addr)
			return server.Start()
		},
	}
}

type agentBridge struct {
	bus   bus.Bus
	agent *agent.Agent
}

func (a *agentBridge) HandleMessage(ctx context.Context, sessionID string, message string) (string, error) {
	outCh := a.bus.Subscribe(agent.TopicOutbound)
	defer a.bus.Unsubscribe(agent.TopicOutbound, outCh)

	a.bus.Publish(agent.TopicInbound, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	})

	var response string
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case raw := <-outCh:
			outMsg, ok := raw.(bus.OutboundMessage)
			if !ok {
				continue
			}
			if outMsg.SessionID != sessionID {
				continue
			}
			response += outMsg.Content
			if outMsg.Done {
				return response, nil
			}
		}
	}
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Root().Flags().GetString("config")
	if configPath != "" {
		return config.Load(configPath)
	}
	return config.DefaultConfig(), nil
}

func main() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
