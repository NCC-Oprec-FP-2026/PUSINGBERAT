// --- internal/service/alert_dispatcher.go ---

package service

import (
	"context"
	"log/slog"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// AlertDispatcher reads alerts from a channel and persists them to the
// database. Future sprints will add WebSocket broadcasting and Discord
// webhook delivery here.
type AlertDispatcher struct {
	repo      AlertRepository
	alertChan <-chan *domain.Alert
}

// NewAlertDispatcher creates a dispatcher that reads from the given channel
// and writes alerts using the provided repository.
func NewAlertDispatcher(repo AlertRepository, alertChan <-chan *domain.Alert) *AlertDispatcher {
	return &AlertDispatcher{
		repo:      repo,
		alertChan: alertChan,
	}
}

// Start launches the background goroutine that drains the alert channel.
// It runs until ctx is cancelled or the channel is closed.
// Called once from main.go after DI wiring is complete.
func (d *AlertDispatcher) Start(ctx context.Context) {
	go func() {
		slog.Info("alert dispatcher started")
		var saved, dropped int64

		for {
			select {
			case <-ctx.Done():
				slog.Info("alert dispatcher stopping",
					"saved", saved,
					"dropped", dropped,
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

				// TODO (Day 5+): WebSocket broadcast
				// wsHub.Broadcast(alertJSON)
				slog.Debug("alert dispatcher: WebSocket broadcast not wired yet")

				// TODO (Day 5+): Discord webhook notification
				// discord.Send(alert)
				slog.Debug("alert dispatcher: Discord notification not wired yet")

				if saved%50 == 0 {
					slog.Info("alert dispatcher progress",
						"saved", saved,
						"dropped", dropped,
					)
				}
			}
		}
	}()
}
