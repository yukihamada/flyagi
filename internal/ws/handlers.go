package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/yuki/flyagi/internal/provider"
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

// ProviderSelectPayload is the payload for "provider.select" messages.
type ProviderSelectPayload struct {
	Type string `json:"type"` // "llm", "tts", "stt"
	Name string `json:"name"`
}

// SelfModApprovePayload is the payload for "selfmod.approve" messages.
type SelfModApprovePayload struct {
	RequestID string `json:"request_id"`
}

// ChatHandler implements MessageHandler for chat interactions.
type ChatHandler struct {
	registry  *provider.Registry
	cancels   sync.Map // map[clientID]context.CancelFunc
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(registry *provider.Registry) *ChatHandler {
	return &ChatHandler{
		registry: registry,
	}
}

func (h *ChatHandler) HandleMessage(client *Client, env Envelope) {
	switch env.Type {
	case "chat.send":
		h.handleChatSend(client, env.Payload)
	case "chat.cancel":
		h.handleChatCancel(client)
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
				// Context cancelled - not an error
				return
			}
			slog.Error("chat stream error", "error", err, "client", client.ID)
			sendError(client, "Stream error: "+err.Error())
		}
	}()
}

func (h *ChatHandler) handleChatCancel(client *Client) {
	if cancel, ok := h.cancels.LoadAndDelete(client.ID); ok {
		cancel.(context.CancelFunc)()
	}
}

func sendError(client *Client, msg string) {
	errPayload, _ := json.Marshal(map[string]string{"error": msg})
	client.Send(Envelope{
		Type:    "error",
		Payload: errPayload,
	})
}
