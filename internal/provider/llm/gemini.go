package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"github.com/yuki/flyagi/internal/provider"
)

// GeminiProvider implements LLMProvider for Google Gemini.
type GeminiProvider struct {
	client *genai.Client
	model  string
}

// NewGeminiProvider creates a new Google Gemini provider.
func NewGeminiProvider(ctx context.Context, apiKey string) (*GeminiProvider, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	return &GeminiProvider{
		client: client,
		model:  "gemini-2.0-flash",
	}, nil
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) ChatStream(ctx context.Context, messages []provider.Message, onChunk func(provider.StreamChunk) error) error {
	var systemInstruction string
	var contents []*genai.Content
	for _, m := range messages {
		switch m.Role {
		case "system":
			systemInstruction = m.Content
		case "user":
			contents = append(contents, &genai.Content{
				Role: "user",
				Parts: []*genai.Part{
					genai.NewPartFromText(m.Content),
				},
			})
		case "assistant":
			contents = append(contents, &genai.Content{
				Role: "model",
				Parts: []*genai.Part{
					genai.NewPartFromText(m.Content),
				},
			})
		}
	}

	config := &genai.GenerateContentConfig{}
	if systemInstruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(systemInstruction),
			},
		}
	}

	for result, err := range p.client.Models.GenerateContentStream(ctx, p.model, contents, config) {
		if err != nil {
			return fmt.Errorf("gemini stream error: %w", err)
		}
		for _, candidate := range result.Candidates {
			if candidate.Content != nil {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						if err := onChunk(provider.StreamChunk{Content: part.Text}); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return onChunk(provider.StreamChunk{Done: true})
}
