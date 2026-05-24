package websocket

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// ---------------------------------------------------------------------------
// Tuning constants
// ---------------------------------------------------------------------------

const (
	// writeWait is the maximum time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the maximum time to wait for a pong response from the peer.
	pongWait = 60 * time.Second

	// pingPeriod is how often the server sends a ping frame.
	// Must be less than pongWait.
	pingPeriod = 30 * time.Second

	// maxMessageSize is the maximum size of an inbound message (bytes).
	// We don't expect large inbound messages from the browser.
	maxMessageSize = 512

	// sendBufferSize is the capacity of the per-client outbound channel.
	sendBufferSize = 256
)

// ---------------------------------------------------------------------------
// Upgrader — HTTP → WebSocket
// ---------------------------------------------------------------------------

// Upgrader is the gorilla/websocket Upgrader shared across all connections.
// CheckOrigin allows all origins in development; restrict in production.
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (CORS is handled by Gin middleware)
	},
}

// ---------------------------------------------------------------------------
// Client — one per WebSocket connection
// ---------------------------------------------------------------------------

// Client is a middleman between the WebSocket connection and the Hub.
// Each connected browser tab gets one Client. The readPump and writePump
// goroutines handle bidirectional communication.
type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	remoteAddr string
}

// NewClient creates a Client for a freshly upgraded WebSocket connection.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, sendBufferSize),
		remoteAddr: conn.RemoteAddr().String(),
	}
}

// ---------------------------------------------------------------------------
// readPump — reads inbound messages (mostly handles close/pong)
// ---------------------------------------------------------------------------

// readPump pumps messages from the WebSocket connection to the hub.
// It runs in its own goroutine per client. When the connection closes,
// it unregisters the client from the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
			) {
				slog.Warn("websocket: unexpected close",
					"remote_addr", c.remoteAddr,
					"err", err,
				)
			}
			break
		}
		// We don't process inbound messages from the browser in this SIEM;
		// the read pump exists only to detect disconnection and handle pong.
	}
}

// ---------------------------------------------------------------------------
// writePump — sends outbound messages + ping keepalive
// ---------------------------------------------------------------------------

// writePump pumps messages from the hub to the WebSocket connection.
// It runs in its own goroutine per client. A ping frame is sent
// every pingPeriod; if the peer fails to respond with a pong before
// pongWait expires, readPump's read deadline fires and the client
// is cleaned up.
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
				// Hub closed the channel — send a close frame.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain any queued messages into the same write frame
			// for efficiency (reduces syscall overhead).
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// ---------------------------------------------------------------------------
// ServeWS — Gin-compatible HTTP handler for the /ws upgrade
// ---------------------------------------------------------------------------

// ServeWS upgrades an HTTP request to a WebSocket connection, registers
// the client with the hub, sends the initial "connected" message, and
// starts the read/write pumps.
//
// Usage in router:
//
//	router.GET("/ws", func(c *gin.Context) {
//	    websocket.ServeWS(hub, c.Writer, c.Request)
//	})
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket: upgrade failed", "err", err)
		return
	}

	client := NewClient(hub, conn)
	hub.register <- client

	// Send the initial "connected" envelope (Section 11.1).
	connMsg := NewWSMessage("connected", map[string]string{
		"message": "PUSINGBERAT SIEM connected",
	})
	hub.Broadcast(connMsg)

	// Start the pumps in their own goroutines.
	go client.writePump()
	go client.readPump()
}
