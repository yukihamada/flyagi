package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const elevenLabsBaseURL = "https://api.elevenlabs.io/v1"

// ElevenLabsProvider implements TTSProvider using ElevenLabs.
type ElevenLabsProvider struct {
	apiKey  string
	voiceID string
	client  *http.Client
}

// NewElevenLabsProvider creates a new ElevenLabs TTS provider.
func NewElevenLabsProvider(apiKey string) *ElevenLabsProvider {
	return &ElevenLabsProvider{
		apiKey:  apiKey,
		voiceID: "21m00Tcm4TlvDq8ikWAM", // Rachel - default voice
		client:  &http.Client{},
	}
}

func (p *ElevenLabsProvider) Name() string { return "elevenlabs" }

func (p *ElevenLabsProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, string, error) {
	url := fmt.Sprintf("%s/text-to-speech/%s", elevenLabsBaseURL, p.voiceID)

	body, err := json.Marshal(map[string]any{
		"text":     text,
		"model_id": "eleven_multilingual_v2",
		"voice_settings": map[string]float64{
			"stability":        0.5,
			"similarity_boost": 0.75,
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("elevenlabs marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("elevenlabs request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("elevenlabs request error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("elevenlabs API error: status %d", resp.StatusCode)
	}

	return resp.Body, "audio/mpeg", nil
}
