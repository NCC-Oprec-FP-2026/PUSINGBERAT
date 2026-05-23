package websocket

import (
	"context"
	"encoding/json"
	"log"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	clients    map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 100),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			return
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if h.clients[client] {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}
}

func (h *Hub) BroadcastAlert(alert domain.Alert) {
	message, err := json.Marshal(map[string]any{
		"type": "alert",
		"data": alert,
	})
	if err != nil {
		log.Printf("WARN: marshal websocket alert failed alert=%s: %v", alert.ID, err)
		return
	}

	select {
	case h.broadcast <- message:
	default:
		log.Printf("WARN: websocket broadcast queue full; dropping alert=%s", alert.ID)
	}
}
