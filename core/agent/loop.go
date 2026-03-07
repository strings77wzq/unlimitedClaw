package agent

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/core/usage"
	"golang.org/x/sync/errgroup"
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

	// Guard against infinite loops if the LLM repeatedly calls tools without converging.
	// Default maxToolIterations=25 prevents runaway agents while allowing complex tasks.
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

			tokenUsage := &bus.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}

			// Record usage for cost tracking
			if a.tracker != nil && tokenUsage != nil {
				a.tracker.Record(msg.SessionID, modelName, usage.TokenUsage{
					PromptTokens:     tokenUsage.PromptTokens,
					CompletionTokens: tokenUsage.CompletionTokens,
					TotalTokens:      tokenUsage.TotalTokens,
				})
			}

			if emit != nil {
				finalContent := resp.Content
				// When streaming, tokens already emitted via onToken callback.
				// Final emit only carries Done flag + Usage for completion signal.
				if streamed {
					finalContent = ""
				}
				emit(bus.OutboundMessage{
					SessionID: msg.SessionID,
					Content:   finalContent,
					Role:      bus.RoleAssistant,
					Done:      true,
					Usage:     tokenUsage,
				})
			}
			return resp.Content, tokenUsage, nil
		}

		sess.AddMessage(providers.Message{
			Role:      providers.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute tool calls in parallel for better performance
		var wg errgroup.Group
		results := make([]*tools.ToolResult, len(resp.ToolCalls))
		errors := make([]error, len(resp.ToolCalls))

		for i, tc := range resp.ToolCalls {
			i, tc := i, tc // capture loop variables
			wg.Go(func() error {
				tool, found := a.toolRegistry.Get(tc.Name)
				if !found {
					errors[i] = fmt.Errorf("tool %q not found", tc.Name)
					return nil
				}

				result, err := tool.Execute(ctx, tc.Arguments)
				results[i] = result
				errors[i] = err
				return nil
			})
		}

		// Wait for all tool executions to complete
		if err := wg.Wait(); err != nil {
			a.logger.Error("tool execution failed", err)
		}

		// Process results in order (but execution was parallel)
		for i, tc := range resp.ToolCalls {
			if errors[i] != nil {
				sess.AddMessage(providers.Message{
					Role:       providers.RoleTool,
					Content:    fmt.Sprintf("tool execution error: %v", errors[i]),
					ToolCallID: tc.ID,
				})
				continue
			}

			result := results[i]
			if result == nil {
				continue
			}

			sess.AddMessage(providers.Message{
				Role:       providers.RoleTool,
				Content:    result.ForLLM,
				ToolCallID: tc.ID,
			})

			if result.ForUser != "" && !result.Silent {
				// ForUser: user-visible feedback (e.g., "Searched for X", "Downloaded file Y")
				// Silent: suppress display for noisy tools that produce too much output
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
	// Streaming is disabled when tools are present because mid-stream tool call arguments
	// require buffering the entire stream to parse JSON, eliminating any latency benefit.
	// The streaming contract only applies to final text responses without tool calls.
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
