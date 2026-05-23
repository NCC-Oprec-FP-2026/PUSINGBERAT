package ruleengine

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

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

func (e *Engine) Evaluate(ctx context.Context, event *domain.ParsedEvent) ([]domain.Alert, error) {
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

func (e *Engine) thresholdReached(rule RuleDefinition, event *domain.ParsedEvent) bool {
	window := time.Duration(rule.Threshold.WindowSeconds) * time.Second
	if rule.Threshold.Window != "" {
		parsed, err := time.ParseDuration(rule.Threshold.Window)
		if err == nil {
			window = parsed
		}
	}
	if window <= 0 {
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

func buildAlert(rule RuleDefinition, event *domain.ParsedEvent) domain.Alert {
	title := rule.Alert.Title
	if title == "" {
		title = rule.Name
	}
	title = renderAlertTemplate(title, rule, event)

	description := rule.Alert.Description
	if description == "" {
		description = rule.Description
	}
	description = renderAlertTemplate(description, rule, event)

	var ruleID *uuid.UUID
	if rule.DatabaseID != uuid.Nil {
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
	MarkDiscordSent(ctx context.Context, id uuid.UUID) error
}

type AlertNotifier interface {
	Send(ctx context.Context, alert domain.Alert) error
}

type AlertDispatcher struct {
	alerts     <-chan domain.Alert
	writer     AlertWriter
	notifier   AlertNotifier
	downstream []chan<- domain.Alert
}

func NewAlertDispatcher(alerts <-chan domain.Alert, writer AlertWriter, notifier AlertNotifier, downstream ...chan<- domain.Alert) *AlertDispatcher {
	return &AlertDispatcher{
		alerts:     alerts,
		writer:     writer,
		notifier:   notifier,
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
			if d.notifier == nil {
				continue
			}
			if err := d.notifier.Send(ctx, alert); err != nil {
				log.Printf("WARN: send discord notification failed alert=%s rule=%s: %v", alert.ID, alert.RuleName, err)
				continue
			}
			if err := d.writer.MarkDiscordSent(ctx, alert.ID); err != nil {
				log.Printf("WARN: mark discord sent failed alert=%s rule=%s: %v", alert.ID, alert.RuleName, err)
			}
		}
	}
}

func FormatAlertLog(alert domain.Alert) string {
	return fmt.Sprintf("[%s] %s", alert.Severity, alert.Title)
}

func renderAlertTemplate(template string, rule RuleDefinition, event *domain.ParsedEvent) string {
	replacements := map[string]string{
		"raw_line": event.RawLine,
		"message":  fieldValue(event, "message"),
		"hostname": fieldValue(event, "hostname"),
		"host":     fieldValue(event, "hostname"),
		"process":  fieldValue(event, "process"),
		"pid":      fieldValue(event, "pid"),
	}
	if rule.Threshold != nil {
		replacements["count"] = strconv.Itoa(rule.Threshold.Count)
		if rule.Threshold.WindowSeconds > 0 {
			replacements["window_seconds"] = strconv.Itoa(rule.Threshold.WindowSeconds)
		} else if rule.Threshold.Window != "" {
			replacements["window"] = rule.Threshold.Window
		}
	}

	out := template
	for key, value := range replacements {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}
