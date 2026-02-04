package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/yuki/flyagi/internal/git"
	"github.com/yuki/flyagi/internal/github"
	"github.com/yuki/flyagi/internal/provider"
	"github.com/yuki/flyagi/internal/selfmod"
)

// ChatSendPayload is the payload for "chat.send" messages.
type ChatSendPayload struct {
	Messages   []provider.Message `json:"messages"`
	ProviderID string             `json:"provider_id"`
}

// ChatChunkPayload is the payload for "chat.chunk" messages.
type ChatChunkPayload struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// SelfModDiffPayload is the payload for "selfmod.diff" messages sent to the client.
type SelfModDiffPayload struct {
	RequestID   string            `json:"request_id"`
	Description string            `json:"description"`
	Diffs       []selfmod.FileDiff `json:"diffs"`
}

// SelfModApprovePayload is the payload for "selfmod.approve" messages.
type SelfModApprovePayload struct {
	RequestID string `json:"request_id"`
}

// SelfModStatusPayload is the payload for "selfmod.status" messages.
type SelfModStatusPayload struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"` // "applying", "pushing", "pr_created", "error"
	Message   string `json:"message,omitempty"`
	PRURL     string `json:"pr_url,omitempty"`
}

// ChatHandler implements MessageHandler for chat interactions.
type ChatHandler struct {
	registry *provider.Registry
	engine   *selfmod.Engine
	gitSvc   *git.Service
	ghClient *github.Client
	cancels  sync.Map // map[clientID]context.CancelFunc
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(registry *provider.Registry, engine *selfmod.Engine, gitSvc *git.Service, ghClient *github.Client) *ChatHandler {
	return &ChatHandler{
		registry: registry,
		engine:   engine,
		gitSvc:   gitSvc,
		ghClient: ghClient,
	}
}

func (h *ChatHandler) HandleMessage(client *Client, env Envelope) {
	switch env.Type {
	case "chat.send":
		h.handleChatSend(client, env.Payload)
	case "chat.cancel":
		h.handleChatCancel(client)
	case "selfmod.request":
		h.handleSelfModRequest(client, env.Payload)
	case "selfmod.approve":
		h.handleSelfModApprove(client, env.Payload)
	case "selfmod.reject":
		h.handleSelfModReject(client, env.Payload)
	default:
		slog.Warn("unknown message type", "type", env.Type, "client", client.ID)
	}
}

func (h *ChatHandler) handleChatSend(client *Client, payload json.RawMessage) {
	var p ChatSendPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("invalid chat.send payload", "error", err, "client", client.ID)
		return
	}

	// Cancel any existing stream for this client
	h.handleChatCancel(client)

	providerID := p.ProviderID
	if providerID == "" {
		providerID = "anthropic"
	}

	llm, err := h.registry.GetLLM(providerID)
	if err != nil {
		slog.Error("LLM provider not found", "provider", providerID, "error", err)
		sendError(client, "LLM provider not found: "+providerID)
		return
	}

	// Check if the last user message looks like a code change request
	lastMsg := ""
	if len(p.Messages) > 0 {
		lastMsg = p.Messages[len(p.Messages)-1].Content
	}
	if h.engine != nil && isCodeChangeRequest(lastMsg) {
		// Respond with acknowledgment then generate changes
		h.handleSelfModFromChat(client, llm, lastMsg, providerID)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.cancels.Store(client.ID, cancel)

	go func() {
		defer func() {
			h.cancels.Delete(client.ID)
			cancel()
		}()

		err := llm.ChatStream(ctx, p.Messages, func(chunk provider.StreamChunk) error {
			chunkPayload, _ := json.Marshal(ChatChunkPayload{
				Content: chunk.Content,
				Done:    chunk.Done,
			})
			return client.Send(Envelope{
				Type:    "chat.chunk",
				Payload: chunkPayload,
			})
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("chat stream error", "error", err, "client", client.ID)
			sendError(client, "Stream error: "+err.Error())
		}
	}()
}

// handleSelfModFromChat detects code change requests in chat and triggers the selfmod engine.
func (h *ChatHandler) handleSelfModFromChat(client *Client, llm provider.LLMProvider, request string, providerID string) {
	// Send a chat acknowledgment
	ackPayload, _ := json.Marshal(ChatChunkPayload{
		Content: "コード変更を生成しています...\n",
		Done:    false,
	})
	client.Send(Envelope{Type: "chat.chunk", Payload: ackPayload})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		cr, err := h.engine.GenerateChanges(ctx, llm, request)
		if err != nil {
			slog.Error("selfmod generate failed", "error", err)
			errPayload, _ := json.Marshal(ChatChunkPayload{
				Content: fmt.Sprintf("\n変更の生成に失敗しました: %s", err.Error()),
				Done:    true,
			})
			client.Send(Envelope{Type: "chat.chunk", Payload: errPayload})
			return
		}

		// Send done for the chat stream
		donePayload, _ := json.Marshal(ChatChunkPayload{
			Content: fmt.Sprintf("\n変更を生成しました: %s\n以下のDiffを確認して承認/拒否してください。", cr.Description),
			Done:    true,
		})
		client.Send(Envelope{Type: "chat.chunk", Payload: donePayload})

		// Send the diff for approval
		diffPayload, _ := json.Marshal(SelfModDiffPayload{
			RequestID:   cr.ID,
			Description: cr.Description,
			Diffs:       cr.Diffs,
		})
		client.Send(Envelope{Type: "selfmod.diff", Payload: diffPayload})

		slog.Info("selfmod diff sent", "request_id", cr.ID, "changes", len(cr.Changes))
	}()
}

func (h *ChatHandler) handleSelfModRequest(client *Client, payload json.RawMessage) {
	var p struct {
		Request    string `json:"request"`
		ProviderID string `json:"provider_id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("invalid selfmod.request payload", "error", err)
		sendError(client, "Invalid request payload")
		return
	}

	providerID := p.ProviderID
	if providerID == "" {
		providerID = "anthropic"
	}

	llm, err := h.registry.GetLLM(providerID)
	if err != nil {
		sendError(client, "LLM provider not found: "+providerID)
		return
	}

	h.handleSelfModFromChat(client, llm, p.Request, providerID)
}

func (h *ChatHandler) handleSelfModApprove(client *Client, payload json.RawMessage) {
	var p SelfModApprovePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("invalid selfmod.approve payload", "error", err)
		sendError(client, "Invalid approve payload")
		return
	}

	if h.engine == nil {
		sendError(client, "Self-modification not configured")
		return
	}

	go func() {
		cr, ok := h.engine.GetRequest(p.RequestID)
		if !ok {
			h.sendStatus(client, p.RequestID, "error", "変更リクエストが見つかりません", "")
			return
		}

		// If git/github are configured, create branch FIRST (before applying changes)
		var branchName string
		if h.gitSvc != nil && h.ghClient != nil {
			branchName = fmt.Sprintf("selfmod/%s", p.RequestID[:8])
			h.sendStatus(client, p.RequestID, "pushing", "ブランチを作成中...", "")

			if err := h.gitSvc.CreateBranch(branchName); err != nil {
				slog.Error("git branch failed", "error", err)
				h.sendStatus(client, p.RequestID, "error", "ブランチ作成に失敗: "+err.Error(), "")
				return
			}
		}

		// Apply changes to the repo
		h.sendStatus(client, p.RequestID, "applying", "変更を適用中...", "")
		if err := h.engine.ApproveAndApply(p.RequestID); err != nil {
			slog.Error("selfmod apply failed", "error", err)
			h.sendStatus(client, p.RequestID, "error", "変更の適用に失敗: "+err.Error(), "")
			return
		}

		// Commit, push, and create PR
		if h.gitSvc != nil && h.ghClient != nil {
			h.sendStatus(client, p.RequestID, "pushing", "コミットしてpush中...", "")

			commitMsg := fmt.Sprintf("selfmod: %s", cr.Description)
			if _, err := h.gitSvc.CommitAll(commitMsg); err != nil {
				slog.Error("git commit failed", "error", err)
				h.sendStatus(client, p.RequestID, "error", "コミットに失敗: "+err.Error(), "")
				return
			}

			if err := h.gitSvc.Push(branchName); err != nil {
				slog.Error("git push failed", "error", err)
				h.sendStatus(client, p.RequestID, "error", "pushに失敗: "+err.Error(), "")
				return
			}

			// Create PR
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			prTitle := fmt.Sprintf("[selfmod] %s", cr.Description)
			prBody := fmt.Sprintf("## Self-Modification Request\n\n%s\n\nGenerated by FlyAGI self-modification engine.", cr.Description)

			prURL, err := h.ghClient.CreatePR(ctx, prTitle, prBody, branchName, "main")
			if err != nil {
				slog.Error("github PR failed", "error", err)
				h.sendStatus(client, p.RequestID, "error", "PR作成に失敗: "+err.Error(), "")
				return
			}

			h.sendStatus(client, p.RequestID, "pr_created", "PRが作成されました！", prURL)

			// Checkout back to main
			if err := h.gitSvc.CheckoutMain(); err != nil {
				slog.Warn("failed to checkout main after PR", "error", err)
			}
		} else {
			h.sendStatus(client, p.RequestID, "applied", "変更が適用されました（GitHub未設定のためPRは作成されません）", "")
		}
	}()
}

func (h *ChatHandler) handleSelfModReject(client *Client, payload json.RawMessage) {
	var p SelfModApprovePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("invalid selfmod.reject payload", "error", err)
		return
	}

	if h.engine != nil {
		h.engine.Reject(p.RequestID)
	}
	h.sendStatus(client, p.RequestID, "rejected", "変更は拒否されました", "")
}

func (h *ChatHandler) handleChatCancel(client *Client) {
	if cancel, ok := h.cancels.LoadAndDelete(client.ID); ok {
		cancel.(context.CancelFunc)()
	}
}

func (h *ChatHandler) sendStatus(client *Client, requestID, status, message, prURL string) {
	payload, _ := json.Marshal(SelfModStatusPayload{
		RequestID: requestID,
		Status:    status,
		Message:   message,
		PRURL:     prURL,
	})
	client.Send(Envelope{Type: "selfmod.status", Payload: payload})
}

func sendError(client *Client, msg string) {
	errPayload, _ := json.Marshal(map[string]string{"error": msg})
	client.Send(Envelope{
		Type:    "error",
		Payload: errPayload,
	})
}

// isCodeChangeRequest checks if a message looks like a code change request.
func isCodeChangeRequest(msg string) bool {
	msg = strings.ToLower(msg)
	keywords := []string{
		"変更して", "修正して", "追加して", "削除して", "変えて",
		"コードを", "ファイルを", "実装して", "リファクタ",
		"change the", "modify the", "add a", "remove the", "update the",
		"refactor", "implement", "fix the code", "edit the",
		"/change", "/modify", "/selfmod",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
