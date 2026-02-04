package stt

import (
	"context"
	"fmt"
	"io"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAISTTProvider implements STTProvider using OpenAI Whisper.
type OpenAISTTProvider struct {
	client *openai.Client
}

// NewOpenAISTTProvider creates a new OpenAI Whisper STT provider.
func NewOpenAISTTProvider(apiKey string) *OpenAISTTProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAISTTProvider{client: &client}
}

func (p *OpenAISTTProvider) Name() string { return "openai" }

func (p *OpenAISTTProvider) Transcribe(ctx context.Context, audio io.Reader, _ string) (string, error) {
	transcription, err := p.client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:  audio,
		Model: openai.AudioModelWhisper1,
	})
	if err != nil {
		return "", fmt.Errorf("openai STT error: %w", err)
	}

	return transcription.Text, nil
}
