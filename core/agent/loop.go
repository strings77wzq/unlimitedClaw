package agent

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
)

func (a *Agent) handleMessage(ctx context.Context, msg bus.InboundMessage) {
	_, _, _ = a.processMessage(ctx, msg, false, nil, func(out bus.OutboundMessage) {
		a.bus.Publish(TopicOutbound, out)
	})
}

func (a *Agent) HandleMessage(ctx context.Context, sessionID string, message string) (string, error) {
	resp, _, err := a.processMessage(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	}, false, nil, nil)
	return resp, err
}

func (a *Agent) HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error {
	defer close(tokens)
	streamed := false
	content, _, err := a.processMessage(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	}, true, func(token string) {
		streamed = true
		tokens <- token
	}, nil)
	// If the response came through a Chat (non-streaming) fallback (e.g. after tool use),
	// deliver the final content as a single token to honour the streaming contract.
	if err == nil && !streamed && content != "" {
		tokens <- content
	}
	return err
}

func (a *Agent) processMessage(
	ctx context.Context,
	msg bus.InboundMessage,
	streamFinal bool,
	onToken func(string),
	emit func(bus.OutboundMessage),
) (string, *bus.TokenUsage, error) {
	sess, found := a.sessionStore.Get(msg.SessionID)
	if !found {
		sess = session.NewSession(msg.SessionID)
		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save new session", err)
			a.emitError(msg.SessionID, "failed to create session", emit)
			return "", nil, fmt.Errorf("failed to create session: %w", err)
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
			a.emitError(msg.SessionID, fmt.Sprintf("failed to get provider: %v", err), emit)
			return "", nil, err
		}

		resp, streamed, err := a.invokeProvider(ctx, provider, contextMsgs, toolDefs, modelName, streamFinal, onToken, msg.SessionID, emit)
		if err != nil {
			a.logger.Error("LLM chat failed", err)
			a.emitError(msg.SessionID, fmt.Sprintf("LLM error: %v", err), emit)
			return "", nil, err
		}

		if len(resp.ToolCalls) == 0 {
			sess.AddMessage(providers.Message{
				Role:    providers.RoleAssistant,
				Content: resp.Content,
			})
			if err := a.sessionStore.Save(sess); err != nil {
				a.logger.Error("failed to save session", err)
			}

			usage := &bus.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}
			if emit != nil {
				finalContent := resp.Content
				if streamed {
					finalContent = ""
				}
				emit(bus.OutboundMessage{
					SessionID: msg.SessionID,
					Content:   finalContent,
					Role:      bus.RoleAssistant,
					Done:      true,
					Usage:     usage,
				})
			}
			return resp.Content, usage, nil
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
				if emit != nil {
					emit(bus.OutboundMessage{
						SessionID: msg.SessionID,
						Content:   result.ForUser,
						Role:      bus.RoleTool,
						Done:      false,
					})
				}
			}
		}

		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save session", err)
		}
	}

	if emit != nil {
		emit(bus.OutboundMessage{
			SessionID: msg.SessionID,
			Content:   "max tool iterations reached",
			Role:      bus.RoleAssistant,
			Done:      true,
		})
	}
	return "max tool iterations reached", nil, nil
}

func (a *Agent) emitError(sessionID, errMsg string, emit func(bus.OutboundMessage)) {
	if emit == nil {
		return
	}
	emit(bus.OutboundMessage{
		SessionID: sessionID,
		Content:   errMsg,
		Role:      bus.RoleAssistant,
		Done:      true,
	})
}

func (a *Agent) invokeProvider(
	ctx context.Context,
	provider providers.LLMProvider,
	messages []providers.Message,
	toolDefs []tools.ToolDefinition,
	modelName string,
	streamFinal bool,
	onToken func(string),
	sessionID string,
	emit func(bus.OutboundMessage),
) (*providers.LLMResponse, bool, error) {
	sp, ok := provider.(providers.StreamingProvider)
	canStream := ok && streamFinal && len(toolDefs) == 0
	if !canStream {
		resp, err := provider.Chat(ctx, messages, toolDefs, modelName, nil)
		return resp, false, err
	}
	resp, err := sp.ChatStream(ctx, messages, toolDefs, modelName, nil, a.wrapTokenEmitter(sessionID, emit, onToken))
	return resp, err == nil, err
}

func (a *Agent) wrapTokenEmitter(sessionID string, emit func(bus.OutboundMessage), onToken func(string)) func(string) {
	return func(token string) {
		if onToken != nil {
			onToken(token)
		}
		if emit != nil {
			emit(bus.OutboundMessage{
				SessionID:  sessionID,
				Content:    token,
				Role:       bus.RoleAssistant,
				Done:       false,
				TokenDelta: token,
			})
		}
	}
}
