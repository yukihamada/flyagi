package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuki/flyagi/internal/api"
	"github.com/yuki/flyagi/internal/config"
	"github.com/yuki/flyagi/internal/provider"
	"github.com/yuki/flyagi/internal/ws"
)

func newTestServer(t *testing.T) (*httptest.Server, *config.Config) {
	t.Helper()

	tmpDir := t.TempDir()
	// Create a test file in the repo
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644)

	cfg := &config.Config{
		Port:               "0",
		RepoPath:           tmpDir,
		DefaultLLMProvider: "test",
		DefaultTTSProvider: "test",
		DefaultSTTProvider: "test",
		AllowedOrigin:      "*",
	}

	reg := provider.NewRegistry()
	handler := ws.NewChatHandler(reg, nil, nil, nil)
	hub := ws.NewHub(handler, "*")
	router := api.NewRouter(cfg, reg, hub)

	return httptest.NewServer(router), cfg
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestHealthzEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestProvidersEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/providers")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if _, ok := body["llm"]; !ok {
		t.Error("expected llm field in response")
	}
}

func TestCodeTreeEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/code/tree")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	files, ok := body["files"].([]any)
	if !ok {
		t.Fatal("expected files array in response")
	}
	if len(files) == 0 {
		t.Error("expected at least one file")
	}
}

func TestCodeFileEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/code/file?path=src/main.go")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["content"] != "package main" {
		t.Errorf("unexpected content: %v", body["content"])
	}
}

func TestCodeFileEndpoint_PathTraversal(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/code/file?path=../../etc/passwd")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}
