package ruleengine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func TestAlertDispatcherLeavesDiscordSentFalseOnNotifierFailure(t *testing.T) {
	alerts := make(chan domain.Alert, 1)
	downstream := make(chan domain.Alert, 1)
	writer := &fakeAlertWriter{}
	notifier := fakeAlertNotifier{err: errors.New("webhook down")}
	dispatcher := NewAlertDispatcher(alerts, writer, notifier, downstream)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dispatcher.Run(ctx)

	alerts <- domain.Alert{
		RuleName: "SSH Brute Force",
		Severity: domain.SeverityHigh,
		Title:    "SSH brute force suspected",
	}

	select {
	case <-downstream:
	case <-time.After(time.Second):
		t.Fatal("alert was not broadcast downstream")
	}

	time.Sleep(20 * time.Millisecond)
	if writer.markDiscordSentCalled.Load() {
		t.Fatal("MarkDiscordSent was called after notifier failure")
	}
}

func TestAlertDispatcherMarksDiscordSentOnNotifierSuccess(t *testing.T) {
	alerts := make(chan domain.Alert, 1)
	writer := &fakeAlertWriter{}
	dispatcher := NewAlertDispatcher(alerts, writer, fakeAlertNotifier{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dispatcher.Run(ctx)

	alerts <- domain.Alert{
		RuleName: "Failed Login",
		Severity: domain.SeverityMedium,
		Title:    "Failed login detected",
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if writer.markDiscordSentCalled.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("MarkDiscordSent was not called after notifier success")
}

type fakeAlertWriter struct {
	markDiscordSentCalled atomic.Bool
}

func (w *fakeAlertWriter) Create(_ context.Context, alert *domain.Alert) error {
	alert.ID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	return nil
}

func (w *fakeAlertWriter) MarkDiscordSent(_ context.Context, id uuid.UUID) error {
	if id != uuid.MustParse("00000000-0000-0000-0000-000000000001") {
		return errors.New("unexpected alert id")
	}
	w.markDiscordSentCalled.Store(true)
	return nil
}

type fakeAlertNotifier struct {
	err error
}

func (n fakeAlertNotifier) Send(_ context.Context, _ domain.Alert) error {
	return n.err
}
