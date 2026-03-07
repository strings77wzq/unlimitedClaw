// Package telegram adapts the Telegram Bot API to the agent's [agent.MessageHandler]
// interface. Incoming Telegram messages are forwarded to the agent; responses
// are sent back as Telegram messages. This adapter is wired in at the
// composition root (cmd/golem) when a bot token is configured.
package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/foundation/logger"
)

// AdapterConfig configures the Telegram adapter
type AdapterConfig struct {
	Token       string
	PollTimeout int
}

// Adapter connects Telegram to the message bus
type Adapter struct {
	cfg     AdapterConfig
	client  *Client
	msgBus  bus.Bus
	log     logger.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	offset  int
	outChan <-chan interface{}
}

// NewAdapter creates a new Telegram adapter
func NewAdapter(cfg AdapterConfig, msgBus bus.Bus, log logger.Logger, opts ...ClientOption) *Adapter {
	if cfg.PollTimeout == 0 {
		cfg.PollTimeout = 30
	}

	return &Adapter{
		cfg:    cfg,
		client: NewClient(cfg.Token, opts...),
		msgBus: msgBus,
		log:    log,
	}
}

// Start begins the polling loop and message handling
func (a *Adapter) Start(ctx context.Context) error {
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.outChan = a.msgBus.Subscribe("outbound")

	a.wg.Add(2)
	go a.pollLoop()
	go a.outboundLoop()

	return nil
}

// Stop signals the adapter to shut down
func (a *Adapter) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
}

func (a *Adapter) pollLoop() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			updates, err := a.client.GetUpdates(a.ctx, a.offset, a.cfg.PollTimeout)
			if err != nil {
				if a.ctx.Err() != nil {
					return
				}
				a.log.Error("failed to get updates", "error", err)
				continue
			}

			for _, update := range updates {
				if update.Message != nil && update.Message.Text != "" {
					a.log.Debug("received message",
						"chat_id", update.Message.Chat.ID,
						"text", update.Message.Text)

					msg := map[string]interface{}{
						"chat_id": update.Message.Chat.ID,
						"text":    update.Message.Text,
						"user_id": int64(0),
					}
					if update.Message.From != nil {
						msg["user_id"] = update.Message.From.ID
					}

					a.msgBus.Publish("inbound", msg)
				}

				if update.UpdateID >= a.offset {
					a.offset = update.UpdateID + 1
				}
			}
		}
	}
}

func (a *Adapter) outboundLoop() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		case msg, ok := <-a.outChan:
			if !ok {
				return
			}

			outMsg, ok := msg.(map[string]interface{})
			if !ok {
				a.log.Error("invalid outbound message type", "type", fmt.Sprintf("%T", msg))
				continue
			}

			chatID, ok := outMsg["chat_id"].(int64)
			if !ok {
				a.log.Error("missing or invalid chat_id in outbound message")
				continue
			}

			text, ok := outMsg["text"].(string)
			if !ok {
				a.log.Error("missing or invalid text in outbound message")
				continue
			}

			if err := a.client.SendMessage(a.ctx, chatID, text); err != nil {
				a.log.Error("failed to send message",
					"chat_id", chatID,
					"error", err)
			} else {
				a.log.Debug("sent message", "chat_id", chatID)
			}
		}
	}
}
