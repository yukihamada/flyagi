package provider_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/yuki/flyagi/internal/provider"
)

type mockLLM struct{ name string }

func (m *mockLLM) Name() string { return m.name }
func (m *mockLLM) ChatStream(_ context.Context, _ []provider.Message, onChunk func(provider.StreamChunk) error) error {
	return onChunk(provider.StreamChunk{Content: "hello", Done: true})
}

type mockTTS struct{ name string }

func (m *mockTTS) Name() string { return m.name }
func (m *mockTTS) Synthesize(_ context.Context, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader("audio")), "audio/mp3", nil
}

type mockSTT struct{ name string }

func (m *mockSTT) Name() string { return m.name }
func (m *mockSTT) Transcribe(_ context.Context, _ io.Reader, _ string) (string, error) {
	return "transcribed text", nil
}

func TestRegistry_RegisterAndGetLLM(t *testing.T) {
	reg := provider.NewRegistry()
	reg.RegisterLLM(&mockLLM{name: "test-llm"})

	p, err := reg.GetLLM("test-llm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "test-llm" {
		t.Errorf("expected name %q, got %q", "test-llm", p.Name())
	}
}

func TestRegistry_GetLLM_NotFound(t *testing.T) {
	reg := provider.NewRegistry()
	_, err := reg.GetLLM("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestRegistry_RegisterAndGetTTS(t *testing.T) {
	reg := provider.NewRegistry()
	reg.RegisterTTS(&mockTTS{name: "test-tts"})

	p, err := reg.GetTTS("test-tts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "test-tts" {
		t.Errorf("expected name %q, got %q", "test-tts", p.Name())
	}
}

func TestRegistry_RegisterAndGetSTT(t *testing.T) {
	reg := provider.NewRegistry()
	reg.RegisterSTT(&mockSTT{name: "test-stt"})

	p, err := reg.GetSTT("test-stt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "test-stt" {
		t.Errorf("expected name %q, got %q", "test-stt", p.Name())
	}
}

func TestRegistry_ListProviders(t *testing.T) {
	reg := provider.NewRegistry()
	reg.RegisterLLM(&mockLLM{name: "llm-a"})
	reg.RegisterLLM(&mockLLM{name: "llm-b"})
	reg.RegisterTTS(&mockTTS{name: "tts-a"})
	reg.RegisterSTT(&mockSTT{name: "stt-a"})

	llms := reg.ListLLMs()
	if len(llms) != 2 {
		t.Errorf("expected 2 LLMs, got %d", len(llms))
	}

	ttsList := reg.ListTTS()
	if len(ttsList) != 1 {
		t.Errorf("expected 1 TTS, got %d", len(ttsList))
	}

	stts := reg.ListSTT()
	if len(stts) != 1 {
		t.Errorf("expected 1 STT, got %d", len(stts))
	}
}
