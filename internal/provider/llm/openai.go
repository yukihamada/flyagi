package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/yuki/flyagi/internal/provider"
)

// OpenAIProvider implements LLMProvider for GPT-4o.
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider creates a new OpenAI GPT provider.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIProvider{
		client: &client,
		model:  "gpt-4o",
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []provider.Message, onChunk func(provider.StreamChunk) error) error {
	var chatMessages []openai.ChatCompletionMessageParamUnion
	for _, m := range messages {
		switch m.Role {
		case "system":
			chatMessages = append(chatMessages, openai.SystemMessage(m.Content))
		case "user":
			chatMessages = append(chatMessages, openai.UserMessage(m.Content))
		case "assistant":
			chatMessages = append(chatMessages, openai.AssistantMessage(m.Content))
		}
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(p.model),
		Messages: chatMessages,
	})

	for stream.Next() {
		chunk := stream.Current()
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if err := onChunk(provider.StreamChunk{Content: choice.Delta.Content}); err != nil {
					return err
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("openai stream error: %w", err)
	}

	return onChunk(provider.StreamChunk{Done: true})
}
