package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

// Envelope is the wire format for all WebSocket messages.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client represents a connected WebSocket client.
type Client struct {
	ID   string
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub manages WebSocket connections and message routing.
type Hub struct {
	mu            sync.RWMutex
	clients       map[string]*Client
	handler       MessageHandler
	allowedOrigin string
}

// MessageHandler processes incoming WebSocket messages.
type MessageHandler interface {
	HandleMessage(client *Client, env Envelope)
}

// NewHub creates a new WebSocket hub.
func NewHub(handler MessageHandler, allowedOrigin string) *Hub {
	return &Hub{
		clients:       make(map[string]*Client),
		handler:       handler,
		allowedOrigin: allowedOrigin,
	}
}

// ServeWS handles WebSocket upgrade requests.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if h.allowedOrigin == "*" || h.allowedOrigin == "" {
				return true
			}
			origin := r.Header.Get("Origin")
			return origin == h.allowedOrigin
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	client := &Client{
		ID:   uuid.New().String(),
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register(client)

	go client.writePump()
	go client.readPump()
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID] = c
	slog.Info("client connected", "id", c.ID)
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c.ID]; ok {
		delete(h.clients, c.ID)
		close(c.send)
		slog.Info("client disconnected", "id", c.ID)
	}
}

// Send sends an envelope to a specific client.
func (c *Client) Send(env Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
		return nil
	default:
		return ErrSendBufferFull
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("websocket read error", "id", c.ID, "error", err)
			}
			break
		}

		var env Envelope
		if err := json.Unmarshal(message, &env); err != nil {
			slog.Warn("invalid message format", "id", c.ID, "error", err)
			continue
		}

		c.hub.handler.HandleMessage(c, env)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var ErrSendBufferFull = &sendBufferFullError{}

type sendBufferFullError struct{}

func (e *sendBufferFullError) Error() string { return "send buffer full" }
