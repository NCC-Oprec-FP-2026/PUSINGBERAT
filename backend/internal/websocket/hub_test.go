package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHub_Run_And_Register(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Give the hub a moment to start
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.ClientCount())
	}

	// Mock client
	c := &Client{
		send:       make(chan []byte, 10),
		remoteAddr: "127.0.0.1:12345",
	}

	// Register
	hub.Register() <- c
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	// Broadcast
	msg := NewWSMessage("test", "payload")
	hub.Broadcast(msg)
	time.Sleep(10 * time.Millisecond)

	select {
	case data := <-c.send:
		var parsed WSMessage
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if parsed.Type != "test" {
			t.Fatalf("expected 'test', got %s", parsed.Type)
		}
	default:
		t.Fatal("expected message in client send channel")
	}

	// BroadcastRaw
	raw := []byte(`{"type":"raw"}`)
	hub.BroadcastRaw(raw)
	time.Sleep(10 * time.Millisecond)

	select {
	case data := <-c.send:
		if string(data) != string(raw) {
			t.Fatalf("expected %s, got %s", raw, data)
		}
	default:
		t.Fatal("expected message in client send channel")
	}

	// Unregister
	hub.Unregister() <- c
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestHub_SlowClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Mock slow client with small buffer
	c := &Client{
		send:       make(chan []byte, 1),
		remoteAddr: "127.0.0.1:12345",
	}

	hub.Register() <- c
	time.Sleep(10 * time.Millisecond)

	// Fill the buffer
	hub.BroadcastRaw([]byte("msg1"))
	hub.BroadcastRaw([]byte("msg2")) // Should trigger drop and unregister

	time.Sleep(10 * time.Millisecond)
	if hub.ClientCount() != 0 {
		t.Fatalf("expected client to be dropped, got %d", hub.ClientCount())
	}
}
