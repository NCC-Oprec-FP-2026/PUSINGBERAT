// Package websocket implements the WebSocket hub for real-time alert
// broadcasting in the PUSINGBERAT SIEM. The Hub manages client
// connections and fans out messages to all connected browsers.
//
// Architecture: Section 7.2 + Section 11
package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// WSMessage — envelope for all WebSocket messages (Section 11.2)
// ---------------------------------------------------------------------------

// WSMessage is the standard envelope wrapping every message sent over the
// WebSocket connection. The frontend expects this shape for all message types.
type WSMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

// NewWSMessage creates a message envelope with the current UTC timestamp.
func NewWSMessage(msgType string, payload interface{}) WSMessage {
	return WSMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// Hub — central coordinator for all WebSocket clients
// ---------------------------------------------------------------------------

// Hub maintains the set of active clients and broadcasts messages to them.
// It is safe for concurrent use — the Run() goroutine serialises all
// register/unregister/broadcast operations through channels.
type Hub struct {
	// clients holds the set of currently connected clients.
	clients map[*Client]bool

	// broadcast receives raw JSON bytes to fan out to every client.
	broadcast chan []byte

	// register receives new clients from the upgrade handler.
	register chan *Client

	// unregister receives clients that have disconnected.
	unregister chan *Client

	// mu protects clients for read access from ClientCount().
	mu sync.RWMutex
}

// NewHub creates a Hub ready to be started with Run().
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub event loop. It must be launched in its own goroutine
// before any clients can connect. It runs until the process exits.
func (h *Hub) Run() {
	slog.Info("websocket hub started")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			count := len(h.clients)
			h.mu.Unlock()

			slog.Info("websocket: client connected",
				"remote_addr", client.remoteAddr,
				"clients_total", count,
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			count := len(h.clients)
			h.mu.Unlock()

			slog.Info("websocket: client disconnected",
				"remote_addr", client.remoteAddr,
				"clients_total", count,
			)

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full — drop it to prevent blocking the hub.
					close(client.send)
					delete(h.clients, client)
					slog.Warn("websocket: dropped slow client",
						"remote_addr", client.remoteAddr,
					)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast serialises the given message envelope to JSON and pushes it
// to all connected clients. It is safe to call from any goroutine.
func (h *Hub) Broadcast(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("websocket: failed to marshal broadcast message", "err", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		slog.Warn("websocket: broadcast channel full, message dropped")
	}
}

// BroadcastRaw pushes pre-serialised JSON bytes to all connected clients.
func (h *Hub) BroadcastRaw(data []byte) {
	select {
	case h.broadcast <- data:
	default:
		slog.Warn("websocket: broadcast channel full, message dropped")
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Register returns the registration channel (used by the upgrade handler).
func (h *Hub) Register() chan<- *Client {
	return h.register
}

// Unregister returns the unregistration channel (used by Client.readPump).
func (h *Hub) Unregister() chan<- *Client {
	return h.unregister
}
