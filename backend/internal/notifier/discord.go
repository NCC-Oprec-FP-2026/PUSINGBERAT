package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
	backoffs   []time.Duration
}

func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: strings.TrimSpace(webhookURL),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		backoffs: []time.Duration{time.Second, 4 * time.Second, 9 * time.Second},
	}
}

func (n *DiscordNotifier) Send(ctx context.Context, alert domain.Alert) error {
	if n == nil || n.webhookURL == "" {
		return nil
	}

	payload, err := json.Marshal(discordPayloadFromAlert(alert))
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	var lastErr error
	totalAttempts := len(n.backoffs) + 1
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		if attempt > 1 {
			delay := n.backoffs[attempt-2]
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create discord request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := n.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("discord webhook returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return fmt.Errorf("send discord notification: %w", lastErr)
}

type discordPayload struct {
	Username string         `json:"username,omitempty"`
	Embeds   []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color"`
	Fields      []discordField `json:"fields,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

func discordPayloadFromAlert(alert domain.Alert) discordPayload {
	description := ""
	if alert.Description != nil {
		description = *alert.Description
	}
	if description == "" && alert.RawLine != nil {
		description = *alert.RawLine
	}

	triggeredAt := alert.TriggeredAt
	if triggeredAt.IsZero() {
		triggeredAt = time.Now().UTC()
	}

	fields := []discordField{
		{Name: "Severity", Value: strings.ToUpper(string(alert.Severity)), Inline: true},
		{Name: "Rule", Value: alert.RuleName, Inline: true},
	}
	if alert.LogSourceID != nil {
		fields = append(fields, discordField{Name: "Log Source", Value: alert.LogSourceID.String(), Inline: true})
	}
	if alert.RawLine != nil && *alert.RawLine != "" {
		fields = append(fields, discordField{Name: "Raw Line", Value: truncateDiscordValue(*alert.RawLine), Inline: false})
	}

	return discordPayload{
		Username: "PUSINGBERAT SIEM",
		Embeds: []discordEmbed{
			{
				Title:       alert.Title,
				Description: description,
				Color:       severityColor(alert.Severity),
				Fields:      fields,
				Timestamp:   triggeredAt.UTC().Format(time.RFC3339),
			},
		},
	}
}

func severityColor(severity domain.SeverityLevel) int {
	switch severity {
	case domain.SeverityCritical:
		return 0x992D22
	case domain.SeverityHigh:
		return 0xE74C3C
	case domain.SeverityMedium:
		return 0xF1C40F
	case domain.SeverityLow:
		return 0x3498DB
	case domain.SeverityInfo:
		return 0x95A5A6
	default:
		return 0x95A5A6
	}
}

func truncateDiscordValue(value string) string {
	const maxFieldValue = 1024
	if len(value) <= maxFieldValue {
		return value
	}
	return value[:maxFieldValue-3] + "..."
}
