package selfmod_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuki/flyagi/internal/provider"
	"github.com/yuki/flyagi/internal/selfmod"
)

type mockLLM struct {
	response string
}

func (m *mockLLM) Name() string { return "mock" }
func (m *mockLLM) ChatStream(_ context.Context, _ []provider.Message, onChunk func(provider.StreamChunk) error) error {
	if err := onChunk(provider.StreamChunk{Content: m.response}); err != nil {
		return err
	}
	return onChunk(provider.StreamChunk{Done: true})
}

func TestEngine_GenerateAndApply(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing file
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)

	llmResponse, _ := json.Marshal(map[string]any{
		"description": "Add hello world",
		"changes": []map[string]string{
			{
				"path":        "src/main.go",
				"action":      "modify",
				"new_content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
			},
		},
	})

	engine := selfmod.NewEngine(tmpDir)
	llm := &mockLLM{response: string(llmResponse)}

	cr, err := engine.GenerateChanges(context.Background(), llm, "Add hello world")
	if err != nil {
		t.Fatalf("GenerateChanges failed: %v", err)
	}

	if cr.Status != "pending" {
		t.Errorf("expected status pending, got %q", cr.Status)
	}
	if len(cr.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(cr.Changes))
	}
	if len(cr.Diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(cr.Diffs))
	}
	if cr.Diffs[0].Diff == "" {
		t.Error("expected non-empty diff")
	}

	// Apply changes
	if err := engine.ApproveAndApply(cr.ID); err != nil {
		t.Fatalf("ApproveAndApply failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(tmpDir, "src", "main.go"))
	if string(content) != "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n" {
		t.Errorf("unexpected file content: %s", content)
	}

	// Verify status
	req, ok := engine.GetRequest(cr.ID)
	if !ok {
		t.Fatal("request not found")
	}
	if req.Status != "approved" {
		t.Errorf("expected approved, got %q", req.Status)
	}
}

func TestEngine_RejectChange(t *testing.T) {
	tmpDir := t.TempDir()

	llmResponse, _ := json.Marshal(map[string]any{
		"description": "Test change",
		"changes": []map[string]string{
			{"path": "test.txt", "action": "create", "new_content": "test"},
		},
	})

	engine := selfmod.NewEngine(tmpDir)
	llm := &mockLLM{response: string(llmResponse)}

	cr, err := engine.GenerateChanges(context.Background(), llm, "Create test file")
	if err != nil {
		t.Fatalf("GenerateChanges failed: %v", err)
	}

	if err := engine.Reject(cr.ID); err != nil {
		t.Fatalf("Reject failed: %v", err)
	}

	req, _ := engine.GetRequest(cr.ID)
	if req.Status != "rejected" {
		t.Errorf("expected rejected, got %q", req.Status)
	}
}

func TestEngine_ProtectedPaths(t *testing.T) {
	tmpDir := t.TempDir()

	llmResponse, _ := json.Marshal(map[string]any{
		"description": "Modify Dockerfile",
		"changes": []map[string]string{
			{"path": "Dockerfile", "action": "modify", "new_content": "FROM scratch"},
		},
	})

	engine := selfmod.NewEngine(tmpDir)
	llm := &mockLLM{response: string(llmResponse)}

	_, err := engine.GenerateChanges(context.Background(), llm, "Change Dockerfile")
	if err == nil {
		t.Fatal("expected error for protected path")
	}
}

func TestEngine_History(t *testing.T) {
	tmpDir := t.TempDir()

	llmResponse, _ := json.Marshal(map[string]any{
		"description": "Test",
		"changes": []map[string]string{
			{"path": "test.txt", "action": "create", "new_content": "test"},
		},
	})

	engine := selfmod.NewEngine(tmpDir)
	llm := &mockLLM{response: string(llmResponse)}

	engine.GenerateChanges(context.Background(), llm, "First")
	engine.GenerateChanges(context.Background(), llm, "Second")

	history := engine.History()
	if len(history) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(history))
	}
}
