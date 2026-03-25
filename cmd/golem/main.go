// Golem - Progressive Go AI Assistant
// Learning by building, inspired by PicoClaw
// License: MIT

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/strings77wzq/golem/core/agent"
	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/providers/anthropic"
	"github.com/strings77wzq/golem/core/providers/openai"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	toolexec "github.com/strings77wzq/golem/core/tools/exec"
	"github.com/strings77wzq/golem/core/tools/fileops"
	"github.com/strings77wzq/golem/core/tools/websearch"
	"github.com/strings77wzq/golem/feature/mcp"
	"github.com/strings77wzq/golem/feature/skills"
	"github.com/strings77wzq/golem/feature/skills/builtins"
	"github.com/strings77wzq/golem/foundation/logger"
	"github.com/strings77wzq/golem/foundation/term"
	"github.com/strings77wzq/golem/internal/channels/telegram"
	"github.com/strings77wzq/golem/internal/channels/tui"
	"github.com/strings77wzq/golem/internal/gateway"
)

// Version info injected at build time via ldflags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "golem",
		Short: "Golem - Progressive AI Assistant",
		Long:  "A progressive Go AI assistant — learning by building, inspired by PicoClaw",
		Example: `  golem agent
  golem gateway
  golem version`,
	}

	cmd.PersistentFlags().StringP("config", "c", "", "config file path (default: ~/.golem/config.json)")

	cmd.AddCommand(
		newVersionCommand(),
		newAgentCommand(),
		newGatewayCommand(),
		newConfigCommand(),
		newStatusCommand(),
		newSessionCommand(),
		newInitCommand(),
	)

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("golem version %s\n", version)
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
		Long:  "Start the Golem AI agent process",
		RunE: func(cmd *cobra.Command, args []string) error {
			message, _ := cmd.Flags().GetString("message")
			modelFlag, _ := cmd.Flags().GetString("model")
			continueFlag, _ := cmd.Flags().GetString("continue")
			noTUI, _ := cmd.Flags().GetBool("no-tui")
			skillsDir, _ := cmd.Flags().GetString("skills-dir")
			skillsFlag, _ := cmd.Flags().GetString("skills")

			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			// Initialize logger early for skills loading
			log := logger.New(logger.DefaultOptions())

			if modelFlag != "" {
				if _, findErr := cfg.FindModel(modelFlag); findErr != nil {
					return fmt.Errorf("model %q not found in config; available: %s", modelFlag, listModelNames(cfg))
				}
				cfg.Agents.Defaults.ModelName = modelFlag
			}

			skillRegistry := skills.NewRegistry()

			builtins.RegisterAll(skillRegistry)

			if skillsDir != "" {
				loader := skills.NewLoader()
				loaded, loadErr := loader.LoadFromDirectory(skillsDir)
				if loadErr != nil {
					log.Warn("failed to load skills from directory", "dir", skillsDir, "err", loadErr)
				} else {
					for _, s := range loaded {
						if regErr := skillRegistry.Register(s); regErr != nil {
							log.Warn("failed to register skill", "name", s.Name, "err", regErr)
						} else {
							log.Info("loaded skill", "name", s.Name)
						}
					}
				}
			}

			if skillsFlag != "" {
				names := strings.Split(skillsFlag, ",")
				requested := make(map[string]bool)
				for _, name := range names {
					name = strings.TrimSpace(name)
					if name != "" {
						requested[name] = true
					}
				}

				if len(requested) > 0 {
					filtered := skillRegistry.List()[:0]
					for _, s := range skillRegistry.List() {
						if requested[s.Name] {
							filtered = append(filtered, s)
						}
					}

					skillRegistry = skills.NewRegistry()
					for _, s := range filtered {
						skillRegistry.Register(s)
					}
					log.Info("filtered skills", "count", skillRegistry.Count(), "names", skillsFlag)
				}
			}

			b := bus.New()
			workspace, _ := os.Getwd()
			registry := buildToolRegistry(workspace)

			// Load RAG tools if configured
			ragFlag, _ := cmd.Flags().GetString("rag")
			if ragFlag != "" {
				ragCfg, err := ParseRagConfig(ragFlag)
				if err != nil {
					return fmt.Errorf("parsing RAG config: %w", err)
				}
				if ragCfg.APIKey == "" {
					if defaultModel, _ := cfg.FindModel(cfg.Agents.Defaults.ModelName); defaultModel != nil {
						ragCfg.APIKey = defaultModel.APIKey
					}
				}
				ragRegistry, err := LoadRAGTools(context.Background(), ragCfg)
				if err != nil {
					return fmt.Errorf("loading RAG tools: %w", err)
				}
				for _, t := range ragRegistry.ListTools() {
					registry.Register(t)
				}
			}

			mcpFlag, _ := cmd.Flags().GetString("mcp")
			var mcpManager *mcp.Manager
			if mcpFlag != "" {
				mcpCfg, err := ParseMCPConfig(mcpFlag)
				if err != nil {
					return fmt.Errorf("parsing MCP config: %w", err)
				}
				mcpManager, err = LoadMCPTools(context.Background(), mcpCfg)
				if err != nil {
					return fmt.Errorf("loading MCP tools: %w", err)
				}
				mcpProxies, err := MCPToolsToRegistry(mcpManager)
				if err != nil {
					return fmt.Errorf("converting MCP tools: %w", err)
				}
				for _, proxy := range mcpProxies {
					registry.Register(proxy)
				}
				log.Info("loaded MCP tools", "count", len(mcpProxies))
			}

			memoryFlag, _ := cmd.Flags().GetString("memory")
			if memoryFlag != "" {
				memCfg, err := ParseMemoryConfig(memoryFlag)
				if err != nil {
					return fmt.Errorf("parsing memory config: %w", err)
				}
				memRegistry, _, err := LoadMemoryTools(context.Background(), memCfg)
				if err != nil {
					return fmt.Errorf("loading memory tools: %w", err)
				}
				for _, t := range memRegistry.ListTools() {
					registry.Register(t)
				}
				log.Info("loaded memory tools")
			}

			factory := registerProviders(cfg)
			log = logger.New(logger.DefaultOptions())

			store, err := openAgentSessionStore(cmd)
			if err != nil {
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

			// Inject skill prompts into system prompt
			systemPrompt := cfg.Agents.Defaults.SystemPrompt
			if skillRegistry.Count() > 0 {
				var sb strings.Builder
				sb.WriteString("Available skills:\n\n")
				for _, s := range skillRegistry.List() {
					sb.WriteString(fmt.Sprintf("## Skill: %s\n%s\n\n", s.Name, s.Description))
					for _, p := range s.Prompts {
						sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", p.Name, p.Content))
					}
				}
				sb.WriteString("---\n\n")
				sb.WriteString(systemPrompt)
				systemPrompt = sb.String()
				log.Info("injected skill prompts into system prompt", "count", skillRegistry.Count())
			}

			history := session.NewHistoryManager(cfg.Agents.Defaults.MaxTokens)
			ag := agent.New(b, registry, factory, sessionStore, history, log, cfg, agent.WithSystemPrompt(systemPrompt))

			telegramFlag, _ := cmd.Flags().GetString("telegram")
			var telegramAdapter *telegram.Adapter
			if telegramFlag != "" {
				tgCfg, err := ParseTelegramConfig(telegramFlag)
				if err != nil {
					return fmt.Errorf("parsing telegram config: %w", err)
				}
				tgCtx, tgCancel := context.WithCancel(context.Background())
				defer tgCancel()
				telegramAdapter, err = StartTelegramAdapter(tgCtx, tgCfg, b, log)
				if err != nil {
					return fmt.Errorf("starting telegram adapter: %w", err)
				}
				telegramAdapter.Start(tgCtx)
				defer telegramAdapter.Stop()
			}

			var sessionID string
			if continueFlag != "" {
				sessionID, err = resolveSessionID(sessionStore, continueFlag)
				if err != nil {
					return err
				}
				log.Info("resuming session", "id", sessionID)
			}

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
				return runAgentOneShot(ag, b, message, sessionID)
			}

			if !term.IsInputTTY() {
				return fmt.Errorf("no input: use -m flag or pipe content via stdin")
			}

			if !noTUI {
				ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
				defer cancel()
				go ag.Start(ctx)
				if sessionID == "" {
					sessionID = uuid.New().String()
				}
				return tui.Run(ctx, sessionID, ag)
			}

			return runAgentInteractive(ag, b, sessionID)
		},
	}
	cmd.Flags().StringP("message", "m", "", "Send a single message and exit")
	cmd.Flags().StringP("model", "M", "", "Model to use (overrides config default)")
	cmd.Flags().StringP("continue", "C", "", `Resume a session ("last" or session-id)`)
	cmd.Flags().Bool("no-tui", false, "Use plain interactive mode instead of TUI")
	cmd.Flags().String("skills-dir", "", "Directory containing skills (with skill.json files)")
	cmd.Flags().String("skills", "", "Comma-separated skill names to enable (e.g., 'summarize,codereview')")
	cmd.Flags().String("rag", "", "RAG configuration: directory path or JSON config for document index")
	cmd.Flags().String("mcp", "", "MCP servers configuration: JSON array of server configs")
	cmd.Flags().String("memory", "", "Memory file path or JSON config for long-term memory")
	cmd.Flags().String("telegram", "", "Telegram bot token or JSON config for Telegram channel")
	return cmd
}

func buildToolRegistry(workspace string) *tools.Registry {
	registry := tools.NewRegistry()
	registry.Register(toolexec.New(workspace))
	registry.Register(fileops.NewFileReadTool(workspace))
	registry.Register(fileops.NewFileWriteTool(workspace))
	registry.Register(fileops.NewFileListTool(workspace))
	registry.Register(websearch.New())
	return registry
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
		// Chinese LLM providers — all OpenAI-compatible, reuse openai.New with custom API base
		case "deepseek":
			base := entry.APIBase
			if base == "" {
				base = "https://api.deepseek.com"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "moonshot": // Kimi
			base := entry.APIBase
			if base == "" {
				base = "https://api.moonshot.cn"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "zhipu": // GLM / ChatGLM
			base := entry.APIBase
			if base == "" {
				base = "https://open.bigmodel.cn/api/paas"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "minimax":
			base := entry.APIBase
			if base == "" {
				base = "https://api.minimax.chat"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "dashscope": // Qwen / Tongyi Qianwen
			base := entry.APIBase
			if base == "" {
				base = "https://dashscope.aliyuncs.com/compatible-mode"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
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

func resolveSessionID(store session.SessionStore, flag string) (string, error) {
	if flag != "last" {
		if _, ok := store.Get(flag); !ok {
			return "", fmt.Errorf("session %q not found", flag)
		}
		return flag, nil
	}
	sessions := store.List()
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions to resume")
	}
	latest := sessions[0]
	for _, s := range sessions[1:] {
		if s.UpdatedAt.After(latest.UpdatedAt) {
			latest = s
		}
	}
	return latest.ID, nil
}

func printUsage(u *bus.TokenUsage) {
	if u == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "\n[tokens: %d prompt + %d completion = %d total]",
		u.PromptTokens, u.CompletionTokens, u.TotalTokens)
}

func runAgentOneShot(ag *agent.Agent, b bus.Bus, message string, existingSessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go ag.Start(ctx)

	sessionID := existingSessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
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
				printUsage(outMsg.Usage)
				fmt.Println()
				return nil
			}
		}
	}
}

func runAgentInteractive(ag *agent.Agent, b bus.Bus, existingSessionID string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go ag.Start(ctx)

	sessionID := existingSessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
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
					printUsage(outMsg.Usage)
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
	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Start the HTTP gateway server",
		Long:  "Start the HTTP gateway server for agent communication",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			b := bus.New()
			workspace, _ := os.Getwd()
			registry := buildToolRegistry(workspace)
			factory := registerProviders(cfg)
			store := session.NewMemoryStore()
			history := session.NewHistoryManager(cfg.Agents.Defaults.MaxTokens)
			log := logger.New(logger.DefaultOptions())

			ag := agent.New(b, registry, factory, store, history, log, cfg)

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			go ag.Start(ctx)

			serverCfg := gateway.DefaultServerConfig()
			secCfg := gateway.DefaultSecurityConfig()

			// Apply gateway config from file
			if cfg.Gateway.Addr != "" {
				serverCfg.Addr = cfg.Gateway.Addr
			}

			// Check for auth token from environment variable (takes precedence)
			authToken := os.Getenv("GOLEM_AUTH_TOKEN")
			if authToken != "" {
				secCfg.EnableAuth = true
				secCfg.AuthToken = authToken
			} else if cfg.Gateway.AuthToken != "" {
				secCfg.EnableAuth = true
				secCfg.AuthToken = cfg.Gateway.AuthToken
			}

			// Apply rate limit config
			if cfg.Gateway.RateLimitRPS > 0 {
				secCfg.EnableRateLimit = true
				secCfg.RateLimitRPS = float64(cfg.Gateway.RateLimitRPS)
			}
			if cfg.Gateway.RateLimitBurst > 0 {
				secCfg.RateLimitBurst = cfg.Gateway.RateLimitBurst
			}

			// Apply CORS config
			if len(cfg.Gateway.AllowedOrigins) > 0 {
				secCfg.CORS = gateway.CORSConfig{
					AllowedOrigins: cfg.Gateway.AllowedOrigins,
					AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization", "X-Request-ID"},
				}
			}

			server := gateway.NewServerWithSecurity(serverCfg, secCfg, ag, log)

			if cfg.Telegram.Token != "" && cfg.Telegram.Mode == "webhook" {
				tgCfg := cfg.Telegram
				tgCtx, tgCancel := context.WithCancel(context.Background())
				defer tgCancel()
				tgAdapter, err := StartTelegramAdapter(tgCtx, tgCfg, b, log)
				if err != nil {
					return fmt.Errorf("starting telegram adapter: %w", err)
				}
				tgAdapter.Start(tgCtx)
				defer tgAdapter.Stop()
				server.MountHandler("/telegram/webhook", tgAdapter.WebhookHandler(tgCfg.WebhookSecret))
				log.Info("telegram webhook mounted", "path", "/telegram/webhook")
			}

			log.Info("starting gateway server",
				"addr", serverCfg.Addr,
				"auth_enabled", secCfg.EnableAuth,
				"rate_limit_enabled", secCfg.EnableRateLimit,
			)
			return server.Start()
		},
	}

	cmd.Flags().String("auth-token", "", "API token for authentication (can also set GOLEM_AUTH_TOKEN env)")
	cmd.Flags().Int("rate-limit", 100, "Rate limit requests per second")

	return cmd
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, err := getConfigPath(cmd)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config.DefaultConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

func main() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
