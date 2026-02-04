package stt

import (
	"context"
	"fmt"
	"io"
	"strings"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
)

// GoogleSTTProvider implements STTProvider using Google Cloud Speech-to-Text.
type GoogleSTTProvider struct {
	projectID string
}

// NewGoogleSTTProvider creates a new Google STT provider.
func NewGoogleSTTProvider(projectID string) *GoogleSTTProvider {
	return &GoogleSTTProvider{projectID: projectID}
}

func (p *GoogleSTTProvider) Name() string { return "google" }

func (p *GoogleSTTProvider) Transcribe(ctx context.Context, audio io.Reader, contentType string) (string, error) {
	client, err := speech.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("google STT client error: %w", err)
	}
	defer client.Close()

	data, err := io.ReadAll(audio)
	if err != nil {
		return "", fmt.Errorf("google STT read error: %w", err)
	}

	encoding := speechpb.RecognitionConfig_WEBM_OPUS
	switch contentType {
	case "audio/wav":
		encoding = speechpb.RecognitionConfig_LINEAR16
	case "audio/mp3", "audio/mpeg":
		encoding = speechpb.RecognitionConfig_MP3
	case "audio/ogg":
		encoding = speechpb.RecognitionConfig_OGG_OPUS
	}

	resp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        encoding,
			SampleRateHertz: 16000,
			LanguageCode:    "ja-JP",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: data,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("google STT recognize error: %w", err)
	}

	var sb strings.Builder
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			sb.WriteString(alt.Transcript)
		}
	}

	return sb.String(), nil
}
