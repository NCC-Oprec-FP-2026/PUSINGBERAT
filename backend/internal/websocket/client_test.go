package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestServeWS(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	}))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer ws.Close()

	// The client should receive a "connected" message
	ws.SetReadDeadline(time.Now().Add(time.Second))
	var msg WSMessage
	if err := ws.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read json: %v", err)
	}

	if msg.Type != "connected" {
		t.Fatalf("expected 'connected' msg type, got %s", msg.Type)
	}

	// We can broadcast something
	hub.BroadcastRaw([]byte(`{"type":"test"}`))

	ws.SetReadDeadline(time.Now().Add(time.Second))
	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read raw: %v", err)
	}

	if string(p) != `{"type":"test"}` {
		t.Fatalf("expected `{\"type\":\"test\"}`, got %s", string(p))
	}

	// Wait for ping
	ws.SetReadDeadline(time.Now().Add(pingPeriod + time.Second))
	ws.SetPingHandler(func(appData string) error {
		ws.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		return nil
	})
	
	// Test client disconnect
	ws.Close()
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after disconnect, got %d", hub.ClientCount())
	}
}
