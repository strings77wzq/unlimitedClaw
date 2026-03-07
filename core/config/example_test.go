package config_test

import (
	"fmt"

	"github.com/strings77wzq/unlimitedClaw/core/config"
)

func ExampleDefaultConfig() {
	cfg := config.DefaultConfig()

	fmt.Println(cfg.Agents.Defaults.ModelName)
	fmt.Println(len(cfg.ModelList))
	// Output:
	// mock
	// 1
}

func ExampleConfig_FindModel() {
	cfg := config.DefaultConfig()

	for _, model := range cfg.ModelList {
		if model.ModelName == "mock" {
			fmt.Println(model.Vendor())
		}
	}
	// Output: mock
}
