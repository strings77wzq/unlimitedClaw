package agent

import (
	"context"
	"fmt"

	"github.com/strings77wzq/unlimitedClaw/core/bus"
	"github.com/strings77wzq/unlimitedClaw/core/providers"
	"github.com/strings77wzq/unlimitedClaw/core/session"
)

func (a *Agent) handleMessage(ctx context.Context, msg bus.InboundMessage) {
	sess, found := a.sessionStore.Get(msg.SessionID)
	if !found {
		sess = session.NewSession(msg.SessionID)
		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save new session", err)
			a.publishError(msg.SessionID, "failed to create session")
			return
		}
	}

	if len(sess.GetMessages()) == 0 && a.systemPrompt != "" {
		sess.AddMessage(providers.Message{
			Role:    providers.RoleSystem,
			Content: a.systemPrompt,
		})
	}

	sess.AddMessage(providers.Message{
		Role:    providers.RoleUser,
		Content: msg.Content,
	})

	model := a.config.Agents.Defaults.ModelName
	toolDefs := a.toolRegistry.ListDefinitions()

	for i := 0; i < a.maxToolIterations; i++ {
		contextMsgs := a.historyManager.GetContextWindow(sess.GetMessages())

		provider, modelName, err := a.providerFactory.GetProviderForModel(model)
		if err != nil {
			a.logger.Error("failed to get provider for model", err)
			a.publishError(msg.SessionID, fmt.Sprintf("failed to get provider: %v", err))
			return
		}

		resp, err := provider.Chat(ctx, contextMsgs, toolDefs, modelName, nil)
		if err != nil {
			a.logger.Error("LLM chat failed", err)
			a.publishError(msg.SessionID, fmt.Sprintf("LLM error: %v", err))
			return
		}

		if len(resp.ToolCalls) == 0 {
			sess.AddMessage(providers.Message{
				Role:    providers.RoleAssistant,
				Content: resp.Content,
			})
			if err := a.sessionStore.Save(sess); err != nil {
				a.logger.Error("failed to save session", err)
			}

			a.bus.Publish(TopicOutbound, bus.OutboundMessage{
				SessionID: msg.SessionID,
				Content:   resp.Content,
				Role:      bus.RoleAssistant,
				Done:      true,
				Usage: &bus.TokenUsage{
					PromptTokens:     resp.Usage.PromptTokens,
					CompletionTokens: resp.Usage.CompletionTokens,
					TotalTokens:      resp.Usage.TotalTokens,
				},
			})
			return
		}

		sess.AddMessage(providers.Message{
			Role:      providers.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			tool, found := a.toolRegistry.Get(tc.Name)
			if !found {
				sess.AddMessage(providers.Message{
					Role:       providers.RoleTool,
					Content:    fmt.Sprintf("tool %q not found", tc.Name),
					ToolCallID: tc.ID,
				})
				continue
			}

			result, err := tool.Execute(ctx, tc.Arguments)
			if err != nil {
				sess.AddMessage(providers.Message{
					Role:       providers.RoleTool,
					Content:    fmt.Sprintf("tool execution error: %v", err),
					ToolCallID: tc.ID,
				})
				continue
			}

			sess.AddMessage(providers.Message{
				Role:       providers.RoleTool,
				Content:    result.ForLLM,
				ToolCallID: tc.ID,
			})

			if result.ForUser != "" && !result.Silent {
				a.bus.Publish(TopicOutbound, bus.OutboundMessage{
					SessionID: msg.SessionID,
					Content:   result.ForUser,
					Role:      bus.RoleTool,
					Done:      false,
				})
			}
		}

		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save session", err)
		}
	}

	a.bus.Publish(TopicOutbound, bus.OutboundMessage{
		SessionID: msg.SessionID,
		Content:   "max tool iterations reached",
		Role:      bus.RoleAssistant,
		Done:      true,
	})
}

func (a *Agent) publishError(sessionID, errMsg string) {
	a.bus.Publish(TopicOutbound, bus.OutboundMessage{
		SessionID: sessionID,
		Content:   errMsg,
		Role:      bus.RoleAssistant,
		Done:      true,
	})
}
