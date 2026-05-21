package ruleengine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type Engine struct {
	rules []RuleDefinition

	mu      sync.Mutex
	windows map[string][]time.Time
}

func NewEngine(rules []RuleDefinition) *Engine {
	return &Engine{
		rules:   rules,
		windows: make(map[string][]time.Time),
	}
}

func (e *Engine) Evaluate(ctx context.Context, event *domain.Event) ([]domain.Alert, error) {
	if event == nil {
		return nil, nil
	}

	var alerts []domain.Alert
	for _, rule := range e.rules {
		select {
		case <-ctx.Done():
			return alerts, ctx.Err()
		default:
		}

		ok, err := ruleConditionsMatch(rule, event)
		if err != nil {
			return alerts, err
		}
		if !ok {
			continue
		}

		if rule.Threshold != nil {
			ok = e.thresholdReached(rule, event)
			if !ok {
				continue
			}
		}

		alerts = append(alerts, buildAlert(rule, event))
	}

	return alerts, nil
}

func (e *Engine) thresholdReached(rule RuleDefinition, event *domain.Event) bool {
	window, err := time.ParseDuration(rule.Threshold.Window)
	if err != nil || window <= 0 {
		window = time.Minute
	}

	groupValue := "global"
	if strings.TrimSpace(rule.Threshold.GroupBy) != "" {
		groupValue = fieldValue(event, rule.Threshold.GroupBy)
		if groupValue == "" {
			groupValue = "unknown"
		}
	}

	key := rule.Name + "|" + rule.Threshold.GroupBy + "|" + groupValue
	now := event.EventTime
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.Add(-window)

	e.mu.Lock()
	defer e.mu.Unlock()

	var kept []time.Time
	for _, ts := range e.windows[key] {
		if !ts.Before(cutoff) {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, now)
	e.windows[key] = kept

	return len(kept) >= rule.Threshold.Count
}

func buildAlert(rule RuleDefinition, event *domain.Event) domain.Alert {
	title := rule.Alert.Title
	if title == "" {
		title = rule.Name
	}

	description := rule.Alert.Description
	if description == "" {
		description = rule.Description
	}

	var ruleID *string
	if rule.DatabaseID != "" {
		ruleID = &rule.DatabaseID
	}

	var eventID *int64
	if event.ID > 0 {
		eventID = &event.ID
	}

	logSourceID := event.LogSourceID
	rawLine := event.RawLine

	return domain.Alert{
		RuleID:      ruleID,
		RuleName:    rule.Name,
		EventID:     eventID,
		LogSourceID: &logSourceID,
		Severity:    rule.Severity,
		Title:       title,
		Description: &description,
		RawLine:     &rawLine,
	}
}

type AlertWriter interface {
	Create(ctx context.Context, alert *domain.Alert) error
}

type AlertDispatcher struct {
	alerts     <-chan domain.Alert
	writer     AlertWriter
	downstream []chan<- domain.Alert
}

func NewAlertDispatcher(alerts <-chan domain.Alert, writer AlertWriter, downstream ...chan<- domain.Alert) *AlertDispatcher {
	return &AlertDispatcher{
		alerts:     alerts,
		writer:     writer,
		downstream: downstream,
	}
}

func (d *AlertDispatcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert, ok := <-d.alerts:
			if !ok {
				return
			}
			if err := d.writer.Create(ctx, &alert); err != nil {
				log.Printf("WARN: persist alert failed rule=%s: %v", alert.RuleName, err)
				continue
			}
			for _, ch := range d.downstream {
				select {
				case ch <- alert:
				default:
				}
			}
		}
	}
}

func FormatAlertLog(alert domain.Alert) string {
	return fmt.Sprintf("[%s] %s", alert.Severity, alert.Title)
}
