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
	"github.com/strings77wzq/unlimitedClaw/pkg/session"
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

			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			b := bus.New()
			registry := tools.NewRegistry()
			factory := providers.NewFactory()
			factory.Register("mock", providers.NewMockProvider("mock"))
			store := session.NewMemoryStore()
			history := session.NewHistoryManager(cfg.Agents.Defaults.MaxTokens)
			log := logger.New(logger.DefaultOptions())

			ag := agent.New(b, registry, factory, store, history, log, cfg)

			if message != "" {
				return runAgentOneShot(ag, b, message)
			}

			return runAgentInteractive(ag, b)
		},
	}
	cmd.Flags().StringP("message", "m", "", "Send a single message and exit")
	return cmd
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
			factory := providers.NewFactory()
			factory.Register("mock", providers.NewMockProvider("mock"))
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
