package api

import (
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yuki/flyagi/internal/config"
	"github.com/yuki/flyagi/internal/provider"
	"github.com/yuki/flyagi/internal/ws"
)

// Server holds dependencies for API handlers.
type Server struct {
	cfg      *config.Config
	registry *provider.Registry
	hub      *ws.Hub
}

// NewRouter creates a fully wired Chi router.
func NewRouter(cfg *config.Config, registry *provider.Registry, hub *ws.Hub) *chi.Mux {
	s := &Server{cfg: cfg, registry: registry, hub: hub}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/healthz"))
	r.Use(CORSMiddleware(cfg.AllowedOrigin))

	limiter := NewRateLimiter(10, 30, time.Second)
	r.Use(limiter.Middleware)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/providers", s.handleProviders)
		r.Post("/tts", s.handleTTS)
		r.Post("/stt", s.handleSTT)
		r.Get("/code/tree", s.handleCodeTree)
		r.Get("/code/file", s.handleCodeFile)
	})

	// WebSocket
	r.Get("/ws", hub.ServeWS)

	// SPA static file serving
	spaHandler := spaFileServer("web/dist")
	r.Handle("/*", spaHandler)

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"llm": s.registry.ListLLMs(),
		"tts": s.registry.ListTTS(),
		"stt": s.registry.ListSTT(),
	})
}

func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text     string `json:"text"`
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}

	providerName := req.Provider
	if providerName == "" {
		providerName = s.cfg.DefaultTTSProvider
	}

	tts, err := s.registry.GetTTS(providerName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	reader, contentType, err := tts.Synthesize(r.Context(), req.Text)
	if err != nil {
		slog.Error("TTS synthesis failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "synthesis failed"})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	io.Copy(w, reader)
}

func (s *Server) handleSTT(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "audio file is required"})
		return
	}
	defer file.Close()

	providerName := r.FormValue("provider")
	if providerName == "" {
		providerName = s.cfg.DefaultSTTProvider
	}

	sttProvider, err := s.registry.GetSTT(providerName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	contentType := header.Header.Get("Content-Type")
	text, err := sttProvider.Transcribe(r.Context(), file, contentType)
	if err != nil {
		slog.Error("STT transcription failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transcription failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}

func (s *Server) handleCodeTree(w http.ResponseWriter, r *http.Request) {
	repoPath := s.cfg.RepoPath
	if repoPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo path not configured"})
		return
	}

	var files []string
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden directories and common non-source directories
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(repoPath, path)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		slog.Error("failed to walk repo", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list files"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s *Server) handleCodeFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path parameter is required"})
		return
	}

	// Prevent path traversal
	fullPath := filepath.Join(s.cfg.RepoPath, filepath.Clean(filePath))
	if !strings.HasPrefix(fullPath, filepath.Clean(s.cfg.RepoPath)+string(os.PathSeparator)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "path traversal not allowed"})
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read file"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":    filePath,
		"content": string(content),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func spaFileServer(distPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)
		if path == "/" {
			path = "/index.html"
		}

		fullPath := filepath.Join(distPath, path)

		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			http.ServeFile(w, r, filepath.Join(distPath, "index.html"))
			return
		}

		switch {
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css")
		case strings.HasSuffix(path, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		}

		http.ServeFile(w, r, fullPath)
	}
}

// embeddedSPAHandler is for production with embedded fs.
func embeddedSPAHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		f, err := fsys.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			r.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	}
}
