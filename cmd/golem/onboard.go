package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/strings77wzq/golem/core/config"
)

type providerPreset struct {
	label   string
	vendor  string
	apiBase string
	model   string
}

var providerPresets = []providerPreset{
	{"OpenAI (GPT-4o)", "openai", "", "openai/gpt-4o"},
	{"Anthropic (Claude 3.5 Sonnet)", "anthropic", "", "anthropic/claude-3-5-sonnet-20241022"},
	{"DeepSeek", "deepseek", "https://api.deepseek.com", "deepseek/deepseek-chat"},
	{"Moonshot (Kimi)", "moonshot", "https://api.moonshot.cn", "moonshot/moonshot-v1-8k"},
	{"Zhipu (GLM)", "zhipu", "https://open.bigmodel.cn/api/paas", "zhipu/glm-4"},
	{"MiniMax", "minimax", "https://api.minimax.chat", "minimax/abab6.5s-chat"},
	{"DashScope (Qwen)", "dashscope", "https://dashscope.aliyuncs.com/compatible-mode", "dashscope/qwen-turbo"},
}

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Configure Golem for first use",
		Long:  "Interactive setup wizard: choose a provider, set your API key, and write config",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}
			return runOnboardWizard(configPath)
		},
	}
}

func runOnboardWizard(configPath string) error {
	r := bufio.NewReader(os.Stdin)

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Reconfigure? [y/N]: ")
		ans, _ := r.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) != "y" {
			fmt.Println("Keeping existing config.")
			return nil
		}
	}

	fmt.Println("=== Golem setup ===")
	fmt.Println("Choose a provider:")
	for i, p := range providerPresets {
		fmt.Printf("  %d. %s\n", i+1, p.label)
	}

	preset, err := choosePreset(r)
	if err != nil {
		return err
	}

	fmt.Printf("API key for %s: ", preset.label)
	apiKey, err := readLine(r)
	if err != nil {
		return err
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	apiBase := preset.apiBase
	if apiBase != "" {
		fmt.Printf("API base URL [%s]: ", apiBase)
		custom, _ := readLine(r)
		custom = strings.TrimSpace(custom)
		if custom != "" {
			apiBase = custom
		}
	}

	modelName := preset.model
	fmt.Printf("Default model name [%s]: ", modelName)
	customModel, _ := readLine(r)
	customModel = strings.TrimSpace(customModel)
	if customModel != "" {
		modelName = customModel
	}

	cfg := &config.Config{
		Agents: config.AgentConfig{
			Defaults: config.AgentDefaults{
				ModelName: modelName,
				MaxTokens: 4096,
			},
		},
		ModelList: []config.ModelEntry{
			{
				ModelName: modelName,
				Model:     modelName,
				APIKey:    apiKey,
				APIBase:   apiBase,
			},
		},
	}

	if err := ensureConfigDir(configPath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\nConfig written to %s\n", configPath)
	fmt.Printf("Run: golem agent\n")
	return nil
}

func choosePreset(r *bufio.Reader) (providerPreset, error) {
	for {
		fmt.Printf("Enter number (1-%d): ", len(providerPresets))
		line, err := readLine(r)
		if err != nil {
			return providerPreset{}, err
		}
		line = strings.TrimSpace(line)
		var n int
		if _, err := fmt.Sscanf(line, "%d", &n); err == nil {
			if n >= 1 && n <= len(providerPresets) {
				return providerPresets[n-1], nil
			}
		}
		fmt.Println("Invalid choice, try again.")
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
