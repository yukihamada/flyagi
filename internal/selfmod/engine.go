package selfmod

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/yuki/flyagi/internal/provider"
)

// protectedPaths are files that cannot be modified.
var protectedPaths = map[string]bool{
	"Dockerfile":             true,
	"fly.toml":               true,
	".github":                true,
	".env":                   true,
	".env.local":             true,
	".env.production":        true,
}

// FileChange represents a single file modification.
type FileChange struct {
	Path       string `json:"path"`
	Action     string `json:"action"` // "create", "modify", "delete"
	NewContent string `json:"new_content,omitempty"`
}

// ChangeRequest represents a pending code modification.
type ChangeRequest struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	Changes     []FileChange `json:"changes"`
	Diffs       []FileDiff   `json:"diffs"`
	Status      string       `json:"status"` // "pending", "approved", "rejected", "applied"
	CreatedAt   time.Time    `json:"created_at"`
}

// FileDiff represents a unified diff for a file.
type FileDiff struct {
	Path string `json:"path"`
	Diff string `json:"diff"`
}

// Engine handles self-modification of the codebase.
type Engine struct {
	mu       sync.Mutex
	repoPath string
	requests sync.Map // map[string]*ChangeRequest
	history  []*ChangeRequest
	histMu   sync.RWMutex
}

// NewEngine creates a new self-modification engine.
func NewEngine(repoPath string) *Engine {
	return &Engine{repoPath: repoPath}
}

const systemPrompt = `You are a code modification assistant. When the user asks for code changes, respond with a JSON object containing file modifications.

Response format:
{
  "description": "Brief description of changes",
  "changes": [
    {
      "path": "relative/path/to/file.go",
      "action": "create|modify|delete",
      "new_content": "full file content for create/modify actions"
    }
  ]
}

Rules:
- Only output valid JSON, no markdown or explanations
- Use relative paths from the project root
- For "modify" action, provide the complete new file content
- For "delete" action, new_content can be omitted
- Never modify: Dockerfile, fly.toml, .github/, .env files
- Keep changes minimal and focused`

// GenerateChanges asks the LLM to generate code modifications.
func (e *Engine) GenerateChanges(ctx context.Context, llm provider.LLMProvider, userRequest string) (*ChangeRequest, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Collect codebase context
	codeContext, err := e.collectContext()
	if err != nil {
		return nil, fmt.Errorf("failed to collect context: %w", err)
	}

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Project structure:\n%s\n\nRequest: %s", codeContext, userRequest)},
	}

	var response strings.Builder
	err = llm.ChatStream(ctx, messages, func(chunk provider.StreamChunk) error {
		response.WriteString(chunk.Content)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse LLM response
	var llmResp struct {
		Description string       `json:"description"`
		Changes     []FileChange `json:"changes"`
	}

	responseText := response.String()
	// Try to extract JSON from response
	responseText = extractJSON(responseText)

	if err := json.Unmarshal([]byte(responseText), &llmResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w (response: %s)", err, truncate(responseText, 200))
	}

	// Validate changes
	for _, change := range llmResp.Changes {
		if err := e.validateChange(change); err != nil {
			return nil, fmt.Errorf("invalid change: %w", err)
		}
	}

	// Generate diffs
	diffs, err := e.generateDiffs(llmResp.Changes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diffs: %w", err)
	}

	cr := &ChangeRequest{
		ID:          uuid.New().String(),
		Description: llmResp.Description,
		Changes:     llmResp.Changes,
		Diffs:       diffs,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	e.requests.Store(cr.ID, cr)

	e.histMu.Lock()
	e.history = append(e.history, cr)
	e.histMu.Unlock()

	return cr, nil
}

// ApproveAndApply applies an approved change request to the filesystem.
func (e *Engine) ApproveAndApply(requestID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	val, ok := e.requests.Load(requestID)
	if !ok {
		return fmt.Errorf("change request %q not found", requestID)
	}
	cr := val.(*ChangeRequest)

	if cr.Status != "pending" {
		return fmt.Errorf("change request is %s, not pending", cr.Status)
	}

	for _, change := range cr.Changes {
		fullPath := filepath.Join(e.repoPath, change.Path)

		switch change.Action {
		case "create", "modify":
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			if err := os.WriteFile(fullPath, []byte(change.NewContent), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", change.Path, err)
			}
			slog.Info("applied change", "action", change.Action, "path", change.Path)
		case "delete":
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete %s: %w", change.Path, err)
			}
			slog.Info("applied change", "action", "delete", "path", change.Path)
		}
	}

	cr.Status = "approved"
	return nil
}

// Reject marks a change request as rejected.
func (e *Engine) Reject(requestID string) error {
	val, ok := e.requests.Load(requestID)
	if !ok {
		return fmt.Errorf("change request %q not found", requestID)
	}
	cr := val.(*ChangeRequest)
	cr.Status = "rejected"
	return nil
}

// GetRequest returns a change request by ID.
func (e *Engine) GetRequest(id string) (*ChangeRequest, bool) {
	val, ok := e.requests.Load(id)
	if !ok {
		return nil, false
	}
	return val.(*ChangeRequest), true
}

// History returns all change requests.
func (e *Engine) History() []*ChangeRequest {
	e.histMu.RLock()
	defer e.histMu.RUnlock()
	result := make([]*ChangeRequest, len(e.history))
	copy(result, e.history)
	return result
}

func (e *Engine) collectContext() (string, error) {
	var sb strings.Builder
	sb.WriteString("File tree:\n")

	err := filepath.Walk(e.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(e.repoPath, path)
		sb.WriteString("  " + rel + "\n")
		return nil
	})
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func (e *Engine) validateChange(change FileChange) error {
	// Check protected paths
	cleanPath := filepath.Clean(change.Path)
	for protected := range protectedPaths {
		if strings.HasPrefix(cleanPath, protected) {
			return fmt.Errorf("cannot modify protected path: %s", change.Path)
		}
	}

	// Prevent path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", change.Path)
	}

	// Check action
	switch change.Action {
	case "create", "modify", "delete":
		// valid
	default:
		return fmt.Errorf("invalid action: %s", change.Action)
	}

	// Limit file size (1MB)
	if len(change.NewContent) > 1<<20 {
		return fmt.Errorf("file too large: %s (%d bytes)", change.Path, len(change.NewContent))
	}

	return nil
}

func (e *Engine) generateDiffs(changes []FileChange) ([]FileDiff, error) {
	dmp := diffmatchpatch.New()
	var diffs []FileDiff

	for _, change := range changes {
		fullPath := filepath.Join(e.repoPath, change.Path)
		var oldContent string

		if change.Action == "modify" || change.Action == "delete" {
			data, err := os.ReadFile(fullPath)
			if err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read %s: %w", change.Path, err)
			}
			oldContent = string(data)
		}

		newContent := change.NewContent
		if change.Action == "delete" {
			newContent = ""
		}

		d := dmp.DiffMain(oldContent, newContent, true)
		patch := dmp.PatchMake(oldContent, d)
		diffText := dmp.PatchToText(patch)

		diffs = append(diffs, FileDiff{
			Path: change.Path,
			Diff: diffText,
		})
	}

	return diffs, nil
}

func extractJSON(s string) string {
	// Try to find JSON block in markdown code fences
	if idx := strings.Index(s, "```json"); idx != -1 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
	} else if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
