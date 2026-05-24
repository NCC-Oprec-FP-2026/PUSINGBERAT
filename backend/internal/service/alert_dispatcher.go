// Package service — alert_dispatcher.go is the central alert pipeline
// goroutine. It reads alerts from the rule engine's channel and
// sequentially:
//   1. Persists the alert to PostgreSQL.
//   2. Broadcasts the alert to all WebSocket clients.
//   3. Sends the alert to Discord (with retry).
//
// Architecture: Section 7.1 (Real-Time Alert Flow)
package service

import (
	"context"
	"log/slog"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	ws "github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/websocket"
)

// AlertDispatcher reads alerts from a channel and fans them out to the
// database, WebSocket hub, and Discord webhook.
type AlertDispatcher struct {
	repo     AlertRepository
	alertChan <-chan *domain.Alert
	wsHub    *ws.Hub
	discord  *DiscordNotifier
}

// NewAlertDispatcher creates a dispatcher wired to all three output sinks.
// wsHub and discord may be nil — the dispatcher gracefully skips them.
func NewAlertDispatcher(
	repo AlertRepository,
	alertChan <-chan *domain.Alert,
	wsHub *ws.Hub,
	discord *DiscordNotifier,
) *AlertDispatcher {
	return &AlertDispatcher{
		repo:      repo,
		alertChan: alertChan,
		wsHub:     wsHub,
		discord:   discord,
	}
}

// Start launches the background goroutine that drains the alert channel.
// It runs until ctx is cancelled or the channel is closed.
// Called once from main.go after DI wiring is complete.
func (d *AlertDispatcher) Start(ctx context.Context) {
	go func() {
		slog.Info("alert dispatcher started",
			"websocket", d.wsHub != nil,
			"discord", d.discord != nil && d.discord.Enabled(),
		)
		var saved, dropped, wsBroadcast, discordOK, discordFail int64

		for {
			select {
			case <-ctx.Done():
				slog.Info("alert dispatcher stopping",
					"saved", saved,
					"dropped", dropped,
					"ws_broadcast", wsBroadcast,
					"discord_ok", discordOK,
					"discord_fail", discordFail,
				)
				return

			case alert, ok := <-d.alertChan:
				if !ok {
					slog.Info("alert dispatcher: channel closed",
						"saved", saved,
						"dropped", dropped,
					)
					return
				}

				// -------------------------------------------------------
				// Step 1: Persist to PostgreSQL
				// -------------------------------------------------------
				if err := d.repo.Create(ctx, alert); err != nil {
					dropped++
					slog.Error("alert dispatcher: failed to persist alert",
						"rule_name", alert.RuleName,
						"title", alert.Title,
						"err", err,
					)
					continue
				}

				saved++
				slog.Info("alert dispatcher: alert saved to database",
					"alert_id", alert.ID,
					"rule_name", alert.RuleName,
					"severity", alert.Severity,
					"title", alert.Title,
				)

				// -------------------------------------------------------
				// Step 2: WebSocket broadcast (Section 11.2/11.3)
				// -------------------------------------------------------
				if d.wsHub != nil {
					msg := ws.NewWSMessage("alert", alert)
					d.wsHub.Broadcast(msg)
					wsBroadcast++
					slog.Debug("alert dispatcher: WebSocket broadcast sent",
						"alert_id", alert.ID,
						"ws_clients", d.wsHub.ClientCount(),
					)
				}

				// -------------------------------------------------------
				// Step 3: Discord webhook (Section 7.3/7.4)
				// -------------------------------------------------------
				if d.discord != nil && d.discord.Enabled() {
					if err := d.discord.Send(alert); err != nil {
						discordFail++
						slog.Error("alert dispatcher: Discord delivery failed",
							"alert_id", alert.ID,
							"err", err,
						)
						// discord_sent defaults to false in DB, so no
						// explicit update needed on failure.
					} else {
						discordOK++
						// Mark as sent in the database.
						if err := d.repo.MarkDiscordSent(ctx, alert.ID); err != nil {
							slog.Error("alert dispatcher: failed to mark discord_sent",
								"alert_id", alert.ID,
								"err", err,
							)
						}
					}
				}

				if saved%50 == 0 {
					slog.Info("alert dispatcher progress",
						"saved", saved,
						"dropped", dropped,
						"ws_broadcast", wsBroadcast,
						"discord_ok", discordOK,
						"discord_fail", discordFail,
					)
				}
			}
		}
	}()
}
