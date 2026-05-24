package ruleengine_test

import (
	"testing"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/ruleengine"
)

func TestEngine_ThresholdTumblingWindow(t *testing.T) {
	// Setup Rule with Threshold 3 in 60s
	ruleDef := &domain.RuleDefinition{
		ID:       "test-rule",
		Name:     "Test Rule",
		Enabled:  true,
		Severity: domain.SeverityHigh,
		Conditions: []domain.RuleCondition{
			{Field: "message", Operator: "equals", Value: "trigger"},
		},
		Threshold: &domain.RuleThreshold{
			Count:         3,
			WindowSeconds: 60,
			GroupBy:       "hostname",
		},
		Alert: domain.RuleAlert{
			Title: "Test Alert",
		},
	}

	loader := ruleengine.NewRuleLoader()
	loader.LoadFromDefinitions([]*domain.RuleDefinition{ruleDef})

	alertChan := make(chan *domain.Alert, 10)
	engine := ruleengine.NewEngine(loader, alertChan)

	ev := &domain.ParsedEvent{
		Message:  ptr("trigger"),
		Hostname: ptr("server-A"),
	}

	// 1st Match -> No Alert
	engine.Evaluate(ev, "syslog")
	if len(alertChan) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(alertChan))
	}

	// 2nd Match -> No Alert
	engine.Evaluate(ev, "syslog")

	// 3rd Match -> Alert Generated
	engine.Evaluate(ev, "syslog")
	if len(alertChan) != 1 {
		t.Fatalf("expected 1 alert on exact threshold boundary, got %d", len(alertChan))
	}

	// 4th Match -> No Alert (Window should be cleared)
	engine.Evaluate(ev, "syslog")
	if len(alertChan) != 1 {
		t.Fatalf("expected alert spam to be suppressed, still got %d alerts", len(alertChan))
	}
}
