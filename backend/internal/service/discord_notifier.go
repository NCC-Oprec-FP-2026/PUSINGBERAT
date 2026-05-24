// Package service — discord_notifier.go implements the Discord webhook
// notification client for the PUSINGBERAT alerting pipeline.
//
// When an alert is dispatched, this notifier sends a richly formatted
// Discord embed to the configured webhook URL. It uses a 3-attempt
// quadratic backoff retry strategy (1s, 4s, 9s) as specified in
// Section 7.4 of the architecture document.

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Discord Payload Structs (Section 7.3)
// ---------------------------------------------------------------------------

// DiscordPayload is the top-level JSON body sent to the Discord webhook API.
type DiscordPayload struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

// DiscordEmbed represents a single Discord embed block with color-coded
// severity, structured fields, and an ISO 8601 timestamp.
type DiscordEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Timestamp   string              `json:"timestamp"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
}

// DiscordEmbedField is a key-value pair rendered inside the embed body.
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// DiscordEmbedFooter is the small text at the bottom of the embed.
type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

// ---------------------------------------------------------------------------
// Severity → Color mapping (Section 7.3)
// ---------------------------------------------------------------------------

// severityToColor maps domain severity levels to Discord embed color codes.
func severityToColor(s domain.SeverityLevel) int {
	switch s {
	case domain.SeverityCritical:
		return 0xFF0000 // Red
	case domain.SeverityHigh:
		return 0xFF6600 // Orange
	case domain.SeverityMedium:
		return 0xFFCC00 // Yellow
	case domain.SeverityLow:
		return 0x0099FF // Blue
	case domain.SeverityInfo:
		return 0x999999 // Grey
	default:
		return 0x999999 // Grey
	}
}

// severityEmoji returns a visual indicator for chat readability.
func severityEmoji(s domain.SeverityLevel) string {
	switch s {
	case domain.SeverityCritical:
		return "🔴"
	case domain.SeverityHigh:
		return "🟠"
	case domain.SeverityMedium:
		return "🟡"
	case domain.SeverityLow:
		return "🔵"
	default:
		return "⚪"
	}
}

// ---------------------------------------------------------------------------
// DiscordNotifier
// ---------------------------------------------------------------------------

const (
	// maxRetries is the maximum number of delivery attempts per alert.
	maxRetries = 3

	// discordHTTPTimeout is the per-request timeout for webhook POSTs.
	discordHTTPTimeout = 10 * time.Second
)

// DiscordNotifier delivers alert notifications to a Discord channel via
// an incoming webhook. It is safe for concurrent use.
type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

// NewDiscordNotifier creates a notifier for the given webhook URL.
// If webhookURL is empty the notifier is effectively a no-op; Send
// returns nil immediately so the dispatcher never blocks.
func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: discordHTTPTimeout,
		},
	}
}

// Enabled reports whether a webhook URL has been configured.
func (n *DiscordNotifier) Enabled() bool {
	return n.webhookURL != ""
}

// ---------------------------------------------------------------------------
// Send — public entry point with quadratic backoff (Section 7.4)
// ---------------------------------------------------------------------------

// Send delivers the alert to Discord with up to 3 attempts using quadratic
// backoff (0s, 1s, 4s delays before each attempt).
//
// If the webhook URL is empty, Send returns nil immediately.
// If all 3 attempts fail, it returns the last error so the dispatcher
// can leave discord_sent = false in the database.
func (n *DiscordNotifier) Send(alert *domain.Alert) error {
	if !n.Enabled() {
		return nil
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Quadratic backoff: 1s, 4s, 9s
			delay := time.Duration(attempt*attempt) * time.Second
			slog.Debug("discord notifier: retrying",
				"attempt", attempt+1,
				"delay", delay,
				"alert_id", alert.ID,
			)
			time.Sleep(delay)
		}

		if err := n.post(alert); err != nil {
			lastErr = err
			slog.Warn("discord notifier: attempt failed",
				"attempt", attempt+1,
				"alert_id", alert.ID,
				"err", err,
			)
			continue
		}

		slog.Info("discord notifier: alert delivered",
			"alert_id", alert.ID,
			"rule_name", alert.RuleName,
			"severity", alert.Severity,
			"attempts", attempt+1,
		)
		return nil
	}

	return fmt.Errorf("discord notifier: all %d attempts failed for alert %s: %w",
		maxRetries, alert.ID, lastErr)
}

// ---------------------------------------------------------------------------
// post — single HTTP POST to the Discord webhook
// ---------------------------------------------------------------------------

// post builds the embed payload and executes a single HTTP POST to the
// Discord webhook endpoint.
func (n *DiscordNotifier) post(alert *domain.Alert) error {
	payload := n.buildPayload(alert)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success; 200 is also acceptable.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Read the error body for diagnostics (cap at 512 bytes).
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("discord returned HTTP %d: %s", resp.StatusCode, string(respBody))
}

// ---------------------------------------------------------------------------
// buildPayload — construct a rich Discord embed from an Alert
// ---------------------------------------------------------------------------

func (n *DiscordNotifier) buildPayload(alert *domain.Alert) DiscordPayload {
	emoji := severityEmoji(alert.Severity)
	title := fmt.Sprintf("%s %s", emoji, alert.Title)

	// Build description from the alert's description field, falling back
	// to a sensible default.
	desc := fmt.Sprintf("**Rule:** %s\n**Severity:** %s",
		alert.RuleName, alert.Severity)
	if alert.Description != nil && *alert.Description != "" {
		desc = *alert.Description + "\n\n" + desc
	}

	embed := DiscordEmbed{
		Title:       title,
		Description: desc,
		Color:       severityToColor(alert.Severity),
		Timestamp:   alert.TriggeredAt.Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: "PUSINGBERAT SIEM",
		},
	}

	// Add structured fields for quick scanning in the Discord channel.
	embed.Fields = append(embed.Fields, DiscordEmbedField{
		Name:   "Alert ID",
		Value:  fmt.Sprintf("`%s`", alert.ID),
		Inline: true,
	})

	if alert.LogSourceID != nil {
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "Log Source",
			Value:  fmt.Sprintf("`%s`", alert.LogSourceID),
			Inline: true,
		})
	}

	if alert.RawLine != nil && *alert.RawLine != "" {
		// Truncate long raw lines for Discord's 1024-char field limit.
		rawLine := *alert.RawLine
		if len(rawLine) > 200 {
			rawLine = rawLine[:200] + "…"
		}
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "Raw Log Line",
			Value:  fmt.Sprintf("```\n%s\n```", rawLine),
			Inline: false,
		})
	}

	return DiscordPayload{
		Embeds: []DiscordEmbed{embed},
	}
}
