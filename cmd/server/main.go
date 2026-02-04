package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuki/flyagi/internal/api"
	"github.com/yuki/flyagi/internal/config"
	"github.com/yuki/flyagi/internal/provider"
	"github.com/yuki/flyagi/internal/provider/llm"
	"github.com/yuki/flyagi/internal/provider/stt"
	"github.com/yuki/flyagi/internal/provider/tts"
	"github.com/yuki/flyagi/internal/ws"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Build provider registry
	registry := provider.NewRegistry()
	registerProviders(cfg, registry)

	// Build WebSocket hub
	chatHandler := ws.NewChatHandler(registry)
	hub := ws.NewHub(chatHandler, cfg.AllowedOrigin)

	router := api.NewRouter(cfg, registry, hub)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}

func registerProviders(cfg *config.Config, registry *provider.Registry) {
	// LLM providers
	if cfg.AnthropicAPIKey != "" {
		registry.RegisterLLM(llm.NewAnthropicProvider(cfg.AnthropicAPIKey))
		slog.Info("registered LLM provider", "name", "anthropic")
	}
	if cfg.OpenAIAPIKey != "" {
		registry.RegisterLLM(llm.NewOpenAIProvider(cfg.OpenAIAPIKey))
		slog.Info("registered LLM provider", "name", "openai")
	}
	if cfg.GeminiAPIKey != "" {
		p, err := llm.NewGeminiProvider(context.Background(), cfg.GeminiAPIKey)
		if err != nil {
			slog.Error("failed to create Gemini provider", "error", err)
		} else {
			registry.RegisterLLM(p)
			slog.Info("registered LLM provider", "name", "gemini")
		}
	}

	// TTS providers
	if cfg.OpenAIAPIKey != "" {
		registry.RegisterTTS(tts.NewOpenAITTSProvider(cfg.OpenAIAPIKey))
		slog.Info("registered TTS provider", "name", "openai")
	}
	if cfg.ElevenLabsAPIKey != "" {
		registry.RegisterTTS(tts.NewElevenLabsProvider(cfg.ElevenLabsAPIKey))
		slog.Info("registered TTS provider", "name", "elevenlabs")
	}

	// STT providers
	if cfg.OpenAIAPIKey != "" {
		registry.RegisterSTT(stt.NewOpenAISTTProvider(cfg.OpenAIAPIKey))
		slog.Info("registered STT provider", "name", "openai")
	}
	if cfg.GoogleProjectID != "" {
		registry.RegisterSTT(stt.NewGoogleSTTProvider(cfg.GoogleProjectID))
		slog.Info("registered STT provider", "name", "google")
	}
}
