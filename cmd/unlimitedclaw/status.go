package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		Long:  "Display version, configuration, and service health information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("unlimitedClaw Status\n")
			fmt.Printf("====================\n\n")

			fmt.Printf("Version:    %s\n", version)
			fmt.Printf("Commit:     %s\n", commit)
			fmt.Printf("Build Date: %s\n\n", date)

			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			fmt.Printf("Config:     %s\n", configPath)

			if _, err := os.Stat(configPath); err == nil {
				fmt.Printf("            (exists)\n")
				if err := showConfigModels(configPath); err != nil {
					fmt.Printf("            error reading config: %v\n", err)
				}
			} else {
				fmt.Printf("            (not found)\n")
			}

			fmt.Println()
			checkGatewayHealth()

			return nil
		},
	}
}

func showConfigModels(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var cfg struct {
		Agents struct {
			Defaults struct {
				ModelName string `json:"model_name"`
			} `json:"defaults"`
		} `json:"agents"`
		ModelList []struct {
			ModelName string `json:"model_name"`
			Model     string `json:"model"`
		} `json:"model_list"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if cfg.Agents.Defaults.ModelName != "" {
		fmt.Printf("            default model: %s\n", cfg.Agents.Defaults.ModelName)
	}

	if len(cfg.ModelList) > 0 {
		fmt.Printf("            configured models: %d\n", len(cfg.ModelList))
		for _, m := range cfg.ModelList {
			fmt.Printf("              - %s (%s)\n", m.ModelName, m.Model)
		}
	}

	return nil
}

func checkGatewayHealth() {
	fmt.Printf("Gateway:    ")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:18790/api/health")
	if err != nil {
		fmt.Printf("stopped (not reachable)\n")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("running (http://localhost:18790)\n")
	} else {
		fmt.Printf("unhealthy (status: %d)\n", resp.StatusCode)
	}
}
