package ws_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuki/flyagi/internal/ws"
)

type echoHandler struct{}

func (h *echoHandler) HandleMessage(client *ws.Client, env ws.Envelope) {
	client.Send(env) // echo back
}

func TestHub_ConnectAndEcho(t *testing.T) {
	hub := ws.NewHub(&echoHandler{}, "*")

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send a message
	msg := ws.Envelope{
		Type:    "test.ping",
		Payload: json.RawMessage(`{"data":"hello"}`),
	}
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read echo response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, resp, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var echo ws.Envelope
	if err := json.Unmarshal(resp, &echo); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if echo.Type != "test.ping" {
		t.Errorf("expected type %q, got %q", "test.ping", echo.Type)
	}
}
