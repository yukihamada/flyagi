package provider

import (
	"context"
	"io"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// StreamChunk represents a chunk of streaming LLM response.
type StreamChunk struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// LLMProvider defines the interface for language model providers.
type LLMProvider interface {
	// Name returns the provider identifier.
	Name() string
	// ChatStream sends messages and streams the response via the callback.
	// The callback is called for each chunk. Return an error to stop streaming.
	ChatStream(ctx context.Context, messages []Message, onChunk func(StreamChunk) error) error
}

// TTSProvider defines the interface for text-to-speech providers.
type TTSProvider interface {
	// Name returns the provider identifier.
	Name() string
	// Synthesize converts text to audio. The returned reader contains audio data (mp3/opus).
	Synthesize(ctx context.Context, text string) (io.ReadCloser, string, error) // reader, contentType, error
}

// STTProvider defines the interface for speech-to-text providers.
type STTProvider interface {
	// Name returns the provider identifier.
	Name() string
	// Transcribe converts audio to text.
	Transcribe(ctx context.Context, audio io.Reader, contentType string) (string, error)
}
