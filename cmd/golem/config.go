package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Manage Golem configuration file",
	}

	cmd.AddCommand(
		newConfigSetCommand(),
		newConfigGetCommand(),
		newConfigListCommand(),
	)

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			if err := ensureConfigDir(configPath); err != nil {
				return err
			}

			cfg := make(map[string]interface{})
			if data, err := os.ReadFile(configPath); err == nil {
				if err := json.Unmarshal(data, &cfg); err != nil {
					return fmt.Errorf("parsing existing config: %w", err)
				}
			}

			cfg[key] = value

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}

			if err := os.WriteFile(configPath, data, 0644); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}

			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			data, err := os.ReadFile(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("config file does not exist: %s", configPath)
				}
				return fmt.Errorf("reading config: %w", err)
			}

			cfg := make(map[string]interface{})
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parsing config: %w", err)
			}

			value, ok := cfg[key]
			if !ok {
				return fmt.Errorf("key %q not found in config", key)
			}

			fmt.Println(value)
			return nil
		},
	}
}

func newConfigListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			data, err := os.ReadFile(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("{}")
					return nil
				}
				return fmt.Errorf("reading config: %w", err)
			}

			var cfg interface{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parsing config: %w", err)
			}

			formatted, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("formatting config: %w", err)
			}

			fmt.Println(string(formatted))
			return nil
		},
	}
}

func getConfigPath(cmd *cobra.Command) (string, error) {
	configPath, _ := cmd.Root().Flags().GetString("config")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		configPath = filepath.Join(home, ".golem", "config.json")
	}
	return configPath, nil
}

func ensureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return nil
}
