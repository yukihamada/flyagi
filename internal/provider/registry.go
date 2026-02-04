package provider

import (
	"fmt"
	"sync"
)

// Registry manages available providers.
type Registry struct {
	mu   sync.RWMutex
	llms map[string]LLMProvider
	tts  map[string]TTSProvider
	stt  map[string]STTProvider
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		llms: make(map[string]LLMProvider),
		tts:  make(map[string]TTSProvider),
		stt:  make(map[string]STTProvider),
	}
}

// RegisterLLM registers a language model provider.
func (r *Registry) RegisterLLM(p LLMProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llms[p.Name()] = p
}

// RegisterTTS registers a text-to-speech provider.
func (r *Registry) RegisterTTS(p TTSProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tts[p.Name()] = p
}

// RegisterSTT registers a speech-to-text provider.
func (r *Registry) RegisterSTT(p STTProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stt[p.Name()] = p
}

// GetLLM returns the named LLM provider.
func (r *Registry) GetLLM(name string) (LLMProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.llms[name]
	if !ok {
		return nil, fmt.Errorf("LLM provider %q not found", name)
	}
	return p, nil
}

// GetTTS returns the named TTS provider.
func (r *Registry) GetTTS(name string) (TTSProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.tts[name]
	if !ok {
		return nil, fmt.Errorf("TTS provider %q not found", name)
	}
	return p, nil
}

// GetSTT returns the named STT provider.
func (r *Registry) GetSTT(name string) (STTProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.stt[name]
	if !ok {
		return nil, fmt.Errorf("STT provider %q not found", name)
	}
	return p, nil
}

// ListLLMs returns names of all registered LLM providers.
func (r *Registry) ListLLMs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.llms))
	for name := range r.llms {
		names = append(names, name)
	}
	return names
}

// ListTTS returns names of all registered TTS providers.
func (r *Registry) ListTTS() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tts))
	for name := range r.tts {
		names = append(names, name)
	}
	return names
}

// ListSTT returns names of all registered STT providers.
func (r *Registry) ListSTT() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.stt))
	for name := range r.stt {
		names = append(names, name)
	}
	return names
}
