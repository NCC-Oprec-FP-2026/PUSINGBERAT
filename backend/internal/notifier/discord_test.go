package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func TestSeverityColor(t *testing.T) {
	tests := []struct {
		severity domain.SeverityLevel
		want     int
	}{
		{domain.SeverityCritical, 0x992D22},
		{domain.SeverityHigh, 0xE74C3C},
		{domain.SeverityMedium, 0xF1C40F},
		{domain.SeverityLow, 0x3498DB},
		{domain.SeverityInfo, 0x95A5A6},
	}

	for _, tt := range tests {
		if got := severityColor(tt.severity); got != tt.want {
			t.Fatalf("severityColor(%q) = %d, want %d", tt.severity, got, tt.want)
		}
	}
}

func TestDiscordNotifierRetriesAndSendsEmbed(t *testing.T) {
	calls := 0
	var payload discordPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if calls < 3 {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	notifier := NewDiscordNotifier(server.URL)
	notifier.backoffs = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}

	description := "Five failed SSH logins"
	rawLine := "May 23 12:00:05 host sshd[4321]: Failed password for root"
	alert := domain.Alert{
		RuleName:    "SSH Brute Force",
		Severity:    domain.SeverityHigh,
		Title:       "SSH brute force suspected",
		Description: &description,
		RawLine:     &rawLine,
		TriggeredAt: time.Date(2026, 5, 23, 12, 0, 5, 0, time.UTC),
	}

	if err := notifier.Send(context.Background(), alert); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	if len(payload.Embeds) != 1 {
		t.Fatalf("embeds = %d, want 1", len(payload.Embeds))
	}
	if payload.Embeds[0].Color != 0xE74C3C {
		t.Fatalf("embed color = %d, want high severity red", payload.Embeds[0].Color)
	}
	if payload.Embeds[0].Title != alert.Title {
		t.Fatalf("embed title = %q, want %q", payload.Embeds[0].Title, alert.Title)
	}
}
