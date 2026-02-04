package tts

import (
	"context"
	"fmt"
	"io"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAITTSProvider implements TTSProvider using OpenAI TTS.
type OpenAITTSProvider struct {
	client *openai.Client
}

// NewOpenAITTSProvider creates a new OpenAI TTS provider.
func NewOpenAITTSProvider(apiKey string) *OpenAITTSProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAITTSProvider{client: &client}
}

func (p *OpenAITTSProvider) Name() string { return "openai" }

func (p *OpenAITTSProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, string, error) {
	resp, err := p.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Model:          openai.SpeechModelTTS1,
		Input:          text,
		Voice:          openai.AudioSpeechNewParamsVoiceAlloy,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	})
	if err != nil {
		return nil, "", fmt.Errorf("openai TTS error: %w", err)
	}
	return resp.Body, "audio/mpeg", nil
}
