package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func TestDiscordNotifier_Disabled(t *testing.T) {
	n := NewDiscordNotifier("")
	if n.Enabled() {
		t.Error("expected notifier to be disabled")
	}

	err := n.Send(&domain.Alert{})
	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}
}

func TestDiscordNotifier_SendSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload DiscordPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if len(payload.Embeds) != 1 {
			t.Fatalf("expected 1 embed, got %d", len(payload.Embeds))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	n := NewDiscordNotifier(ts.URL)
	n.client = ts.Client() // Use the test server client if needed, or just let it use default

	alert := &domain.Alert{
		Title:       "Test Alert",
		Severity:    domain.SeverityCritical,
		RuleName:    "Test Rule",
		TriggeredAt: time.Now(),
	}

	if err := n.Send(alert); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDiscordNotifier_SendRetry(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	n := NewDiscordNotifier(ts.URL)

	alert := &domain.Alert{
		Title:       "Test Alert",
		Severity:    domain.SeverityCritical,
		RuleName:    "Test Rule",
		TriggeredAt: time.Now(),
	}

	// Because of backoff (1s, 4s), this test might take 5 seconds.
	// Let's speed it up by modifying the maxRetries or just letting it run.
	// Actually, the backoff is hardcoded. It will take 1s + 4s = 5s.
	if err := n.Send(alert); err != nil {
		t.Errorf("unexpected error after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestDiscordNotifier_SendFailure(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	n := NewDiscordNotifier(ts.URL)

	alert := &domain.Alert{
		Title:       "Test Alert",
		Severity:    domain.SeverityCritical,
		RuleName:    "Test Rule",
		TriggeredAt: time.Now(),
	}

	err := n.Send(alert)
	if err == nil {
		t.Error("expected error on total failure")
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
