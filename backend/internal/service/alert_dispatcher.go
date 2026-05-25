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

type alertDispatcherStats struct {
	saved       int64
	dropped     int64
	wsBroadcast int64
	discordOK   int64
	discordFail int64
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
		stats := &alertDispatcherStats{}

		for {
			select {
			case <-ctx.Done():
				stats.log("alert dispatcher stopping")
				return

			case alert, ok := <-d.alertChan:
				if !ok {
					slog.Info("alert dispatcher: channel closed",
						"saved", stats.saved,
						"dropped", stats.dropped,
					)
					return
				}

				d.handleAlert(ctx, alert, stats)
			}
		}
	}()
}

func (d *AlertDispatcher) handleAlert(ctx context.Context, alert *domain.Alert, stats *alertDispatcherStats) {
	if !d.persistAlert(ctx, alert, stats) {
		return
	}

	d.broadcastAlert(alert, stats)
	d.sendDiscordNotification(ctx, alert, stats)

	if stats.saved%50 == 0 {
		stats.log("alert dispatcher progress")
	}
}

func (d *AlertDispatcher) persistAlert(ctx context.Context, alert *domain.Alert, stats *alertDispatcherStats) bool {
	if err := d.repo.Create(ctx, alert); err != nil {
		stats.dropped++
		slog.Error("alert dispatcher: failed to persist alert",
			"rule_name", alert.RuleName,
			"title", alert.Title,
			"err", err,
		)
		return false
	}

	stats.saved++
	slog.Info("alert dispatcher: alert saved to database",
		"alert_id", alert.ID,
		"rule_name", alert.RuleName,
		"severity", alert.Severity,
		"title", alert.Title,
	)
	return true
}

func (d *AlertDispatcher) broadcastAlert(alert *domain.Alert, stats *alertDispatcherStats) {
	if d.wsHub == nil {
		return
	}

	msg := ws.NewWSMessage("alert", alert)
	d.wsHub.Broadcast(msg)
	stats.wsBroadcast++
	slog.Debug("alert dispatcher: WebSocket broadcast sent",
		"alert_id", alert.ID,
		"ws_clients", d.wsHub.ClientCount(),
	)
}

func (d *AlertDispatcher) sendDiscordNotification(ctx context.Context, alert *domain.Alert, stats *alertDispatcherStats) {
	if d.discord == nil || !d.discord.Enabled() {
		return
	}

	if err := d.discord.Send(alert); err != nil {
		stats.discordFail++
		slog.Error("alert dispatcher: Discord delivery failed",
			"alert_id", alert.ID,
			"err", err,
		)
		return
	}

	stats.discordOK++
	if err := d.repo.MarkDiscordSent(ctx, alert.ID); err != nil {
		slog.Error("alert dispatcher: failed to mark discord_sent",
			"alert_id", alert.ID,
			"err", err,
		)
	}
}

func (s alertDispatcherStats) log(message string) {
	slog.Info(message,
		"saved", s.saved,
		"dropped", s.dropped,
		"ws_broadcast", s.wsBroadcast,
		"discord_ok", s.discordOK,
		"discord_fail", s.discordFail,
	)
}
