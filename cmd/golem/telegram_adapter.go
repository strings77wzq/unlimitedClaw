package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/foundation/logger"
	"github.com/strings77wzq/golem/internal/channels/telegram"
)

func ParseTelegramConfig(jsonStr string) (config.TelegramConfig, error) {
	cfg := config.TelegramConfig{}

	if jsonStr == "" {
		return cfg, nil
	}

	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		cfg.Token = jsonStr
	}

	return cfg, nil
}

func StartTelegramAdapter(ctx context.Context, cfg config.TelegramConfig, b bus.Bus, log logger.Logger) (*telegram.Adapter, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}

	adapterCfg := telegram.AdapterConfig{
		Token:       cfg.Token,
		PollTimeout: cfg.PollTimeout,
	}

	adapter := telegram.NewAdapter(adapterCfg, b, log)

	if cfg.Mode == "webhook" && cfg.WebhookURL != "" {
		if err := adapter.Client().SetWebhook(ctx, cfg.WebhookURL, cfg.WebhookSecret); err != nil {
			return nil, fmt.Errorf("setting webhook: %w", err)
		}
		log.Info("telegram webhook configured", "url", cfg.WebhookURL)
	} else {
		if err := adapter.Client().DeleteWebhook(ctx, true); err != nil {
			return nil, fmt.Errorf("deleting webhook: %w", err)
		}
		log.Info("telegram polling mode enabled")
	}

	return adapter, nil
}
