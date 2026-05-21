package ruleengine

import (
	"context"
	"testing"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func testEvent() *domain.Event {
	message := "Failed password for root from 10.0.0.1"
	host := "web01"
	process := "sshd"
	pid := int32(1234)
	level := "warn"

	return &domain.Event{
		ID:          42,
		LogSourceID: "source-1",
		RawLine:     "May 21 10:00:00 web01 sshd[1234]: Failed password for root from 10.0.0.1",
		Message:     &message,
		Hostname:    &host,
		Process:     &process,
		PID:         &pid,
		LogLevel:    &level,
		EventTime:   time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
		Extra: map[string]any{
			"log_type": "syslog",
			"status":   503,
		},
	}
}

func TestConditionOperators(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		want      bool
	}{
		{
			name:      "contains",
			condition: Condition{Field: "message", Operator: "contains", Value: "failed password"},
			want:      true,
		},
		{
			name:      "equals",
			condition: Condition{Field: "process", Operator: "equals", Value: "sshd"},
			want:      true,
		},
		{
			name:      "not_equals",
			condition: Condition{Field: "hostname", Operator: "not_equals", Value: "db01"},
			want:      true,
		},
		{
			name:      "regex",
			condition: Condition{Field: "message", Operator: "regex", Value: `root from \d+\.\d+\.\d+\.\d+`},
			want:      true,
		},
		{
			name:      "starts_with",
			condition: Condition{Field: "message", Operator: "starts_with", Value: "Failed"},
			want:      true,
		},
		{
			name:      "ends_with",
			condition: Condition{Field: "message", Operator: "ends_with", Value: "10.0.0.1"},
			want:      true,
		},
		{
			name:      "greater_than",
			condition: Condition{Field: "extra.status", Operator: "greater_than", Value: "500"},
			want:      true,
		},
		{
			name:      "less_than",
			condition: Condition{Field: "pid", Operator: "less_than", Value: "2000"},
			want:      true,
		},
	}

	event := testEvent()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := conditionMatches(tt.condition, event)
			if err != nil {
				t.Fatalf("conditionMatches returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("conditionMatches = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuleLogTypeFiltering(t *testing.T) {
	rule := RuleDefinition{
		Name:     "Syslog only",
		LogTypes: []string{"nginx"},
		Conditions: []Condition{
			{Field: "message", Operator: "contains", Value: "Failed"},
		},
	}

	ok, err := ruleConditionsMatch(rule, testEvent())
	if err != nil {
		t.Fatalf("ruleConditionsMatch returned error: %v", err)
	}
	if ok {
		t.Fatal("ruleConditionsMatch = true, want false for different log_type")
	}
}

func TestThresholdBelowAndAtBoundary(t *testing.T) {
	rule := RuleDefinition{
		Name:       "Boundary Rule",
		DatabaseID: "rule-id",
		Severity:   domain.SeverityHigh,
		Conditions: []Condition{
			{Field: "message", Operator: "contains", Value: "Failed password"},
		},
		Threshold: &Threshold{
			Count:   3,
			Window:  "1m",
			GroupBy: "hostname",
		},
	}
	engine := NewEngine([]RuleDefinition{rule})

	event := testEvent()
	for i := 0; i < 2; i++ {
		event.EventTime = event.EventTime.Add(time.Second)
		alerts, err := engine.Evaluate(context.Background(), event)
		if err != nil {
			t.Fatalf("Evaluate returned error: %v", err)
		}
		if len(alerts) != 0 {
			t.Fatalf("event %d produced %d alerts, want 0", i+1, len(alerts))
		}
	}

	event.EventTime = event.EventTime.Add(time.Second)
	alerts, err := engine.Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("third event produced %d alerts, want 1", len(alerts))
	}
}

func TestThresholdOutsideWindowDoesNotAlert(t *testing.T) {
	rule := RuleDefinition{
		Name:       "Window Rule",
		DatabaseID: "rule-id",
		Severity:   domain.SeverityHigh,
		Conditions: []Condition{
			{Field: "message", Operator: "contains", Value: "Failed password"},
		},
		Threshold: &Threshold{
			Count:   2,
			Window:  "10s",
			GroupBy: "hostname",
		},
	}
	engine := NewEngine([]RuleDefinition{rule})

	event := testEvent()
	if alerts, err := engine.Evaluate(context.Background(), event); err != nil || len(alerts) != 0 {
		t.Fatalf("first Evaluate alerts=%d err=%v, want 0 nil", len(alerts), err)
	}

	event.EventTime = event.EventTime.Add(11 * time.Second)
	alerts, err := engine.Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("outside-window event produced %d alerts, want 0", len(alerts))
	}
}
