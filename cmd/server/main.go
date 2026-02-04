package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuki/flyagi/internal/api"
	"github.com/yuki/flyagi/internal/config"
	"github.com/yuki/flyagi/internal/git"
	"github.com/yuki/flyagi/internal/github"
	"github.com/yuki/flyagi/internal/provider"
	"github.com/yuki/flyagi/internal/provider/llm"
	"github.com/yuki/flyagi/internal/provider/stt"
	"github.com/yuki/flyagi/internal/provider/tts"
	"github.com/yuki/flyagi/internal/selfmod"
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

	// Initialize self-modification services
	var engine *selfmod.Engine
	var gitSvc *git.Service
	var ghClient *github.Client

	if cfg.RepoPath != "" {
		engine = selfmod.NewEngine(cfg.RepoPath)
		slog.Info("selfmod engine initialized", "repo_path", cfg.RepoPath)
	}

	if cfg.GitHubToken != "" && cfg.GitHubOwner != "" && cfg.GitHubRepo != "" {
		gitSvc = git.NewService(cfg.RepoPath, cfg.GitHubToken)
		ghClient = github.NewClient(cfg.GitHubToken, cfg.GitHubOwner, cfg.GitHubRepo)

		// Clone or open the repo
		cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", cfg.GitHubOwner, cfg.GitHubRepo)
		if err := gitSvc.CloneOrOpen(cloneURL); err != nil {
			slog.Error("failed to clone/open repo", "error", err)
			// Non-fatal: continue without git
			gitSvc = nil
			ghClient = nil
		} else {
			slog.Info("git service initialized", "owner", cfg.GitHubOwner, "repo", cfg.GitHubRepo)
		}
	}

	// Build WebSocket hub
	chatHandler := ws.NewChatHandler(registry, engine, gitSvc, ghClient)
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
