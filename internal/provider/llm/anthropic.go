package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/yuki/flyagi/internal/provider"
)

// AnthropicProvider implements LLMProvider for Claude.
type AnthropicProvider struct {
	client *anthropic.Client
	model  string
}

// NewAnthropicProvider creates a new Anthropic Claude provider.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{
		client: &client,
		model:  "claude-sonnet-4-20250514",
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) ChatStream(ctx context.Context, messages []provider.Message, onChunk func(provider.StreamChunk) error) error {
	// Separate system message from conversation messages
	var systemPrompt string
	var convMessages []anthropic.MessageParam
	for _, m := range messages {
		switch m.Role {
		case "system":
			systemPrompt = m.Content
		case "user":
			convMessages = append(convMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(m.Content),
			))
		case "assistant":
			convMessages = append(convMessages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(m.Content),
			))
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		Messages:  convMessages,
	}
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	stream := p.client.Messages.NewStreaming(ctx, params)
	for stream.Next() {
		event := stream.Current()
		if event.Type == "content_block_delta" {
			if delta := event.Delta; delta.Text != "" {
				if err := onChunk(provider.StreamChunk{Content: delta.Text}); err != nil {
					return err
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("anthropic stream error: %w", err)
	}

	return onChunk(provider.StreamChunk{Done: true})
}
