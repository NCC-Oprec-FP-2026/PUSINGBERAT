package ruleengine

import (
	"sync"
	"testing"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// 1. Test MatchAllConditions with all operators
func TestMatchAllConditions(t *testing.T) {
	ev := &domain.ParsedEvent{
		Hostname: strPtr("web-01"),
		Message:  strPtr("Failed password for root from 192.168.1.10"),
		Process:  strPtr("sshd"),
		PID:      intPtr(1234),
		LogLevel: strPtr("error"),
		RawLine:  "May 21 14:30:00 web-01 sshd[1234]: Failed password for root from 192.168.1.10",
	}

	tests := []struct {
		name       string
		conditions []domain.RuleCondition
		wantMatch  bool
		wantErr    bool
	}{
		{
			name:       "equals match",
			conditions: []domain.RuleCondition{{Field: "process", Operator: "equals", Value: "sshd"}},
			wantMatch:  true,
		},
		{
			name:       "equals mismatch",
			conditions: []domain.RuleCondition{{Field: "process", Operator: "equals", Value: "nginx"}},
			wantMatch:  false,
		},
		{
			name:       "contains match",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "contains", Value: "Failed password"}},
			wantMatch:  true,
		},
		{
			name:       "contains mismatch",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "contains", Value: "Accepted password"}},
			wantMatch:  false,
		},
		{
			name:       "starts_with match",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "starts_with", Value: "Failed"}},
			wantMatch:  true,
		},
		{
			name:       "starts_with mismatch",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "starts_with", Value: "password"}},
			wantMatch:  false,
		},
		{
			name:       "ends_with match",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "ends_with", Value: "192.168.1.10"}},
			wantMatch:  true,
		},
		{
			name:       "ends_with mismatch",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "ends_with", Value: "root"}},
			wantMatch:  false,
		},
		{
			name:       "regex match",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "regex", Value: `from \d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`}},
			wantMatch:  true,
		},
		{
			name:       "regex mismatch",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "regex", Value: `from [a-z]+`}},
			wantMatch:  false,
		},
		{
			name:       "regex invalid",
			conditions: []domain.RuleCondition{{Field: "message", Operator: "regex", Value: `[`}},
			wantMatch:  false,
			wantErr:    true,
		},
		{
			name:       "gt match",
			conditions: []domain.RuleCondition{{Field: "pid", Operator: "gt", Value: "1000"}},
			wantMatch:  true,
		},
		{
			name:       "gt mismatch",
			conditions: []domain.RuleCondition{{Field: "pid", Operator: "gt", Value: "2000"}},
			wantMatch:  false,
		},
		{
			name:       "gt invalid value",
			conditions: []domain.RuleCondition{{Field: "pid", Operator: "gt", Value: "abc"}},
			wantMatch:  false,
			wantErr:    true,
		},
		{
			name:       "lt match",
			conditions: []domain.RuleCondition{{Field: "pid", Operator: "lt", Value: "2000"}},
			wantMatch:  true,
		},
		{
			name:       "lt mismatch",
			conditions: []domain.RuleCondition{{Field: "pid", Operator: "lt", Value: "1000"}},
			wantMatch:  false,
		},
		{
			name: "multiple conditions all match",
			conditions: []domain.RuleCondition{
				{Field: "process", Operator: "equals", Value: "sshd"},
				{Field: "hostname", Operator: "equals", Value: "web-01"},
				{Field: "pid", Operator: "gt", Value: "1000"},
			},
			wantMatch: true,
		},
		{
			name: "multiple conditions one mismatch",
			conditions: []domain.RuleCondition{
				{Field: "process", Operator: "equals", Value: "sshd"},
				{Field: "hostname", Operator: "equals", Value: "db-01"},
			},
			wantMatch: false,
		},
		{
			name:       "unsupported operator",
			conditions: []domain.RuleCondition{{Field: "process", Operator: "magic", Value: "sshd"}},
			wantMatch:  false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchAllConditions(tt.conditions, ev)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchAllConditions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantMatch {
				t.Errorf("MatchAllConditions() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

// 2. Sliding Window Threshold Logic
func TestEngine_ThresholdLogic(t *testing.T) {
	loader := NewRuleLoader()
	ruleID := "test-rule-1"
	rule := &domain.RuleDefinition{
		ID:       ruleID,
		Name:     "Test Rule",
		Enabled:  true,
		Severity: domain.SeverityMedium,
		Conditions: []domain.RuleCondition{
			{Field: "message", Operator: "contains", Value: "fail"},
		},
		Threshold: &domain.RuleThreshold{
			Count:         3,
			WindowSeconds: 5,
			GroupBy:       "hostname",
		},
		Alert: domain.RuleAlert{
			Title: "Test Alert",
		},
	}
	loader.LoadFromDefinitions([]*domain.RuleDefinition{rule})

	alertChan := make(chan *domain.Alert, 10)
	engine := NewEngine(loader, alertChan)

	ev1 := &domain.ParsedEvent{Hostname: strPtr("host1"), Message: strPtr("fail 1"), ID: 1}
	ev2 := &domain.ParsedEvent{Hostname: strPtr("host1"), Message: strPtr("fail 2"), ID: 2}
	ev3 := &domain.ParsedEvent{Hostname: strPtr("host1"), Message: strPtr("fail 3"), ID: 3}
	evOtherHost := &domain.ParsedEvent{Hostname: strPtr("host2"), Message: strPtr("fail 1"), ID: 4}

	// 1st event -> Below boundary
	engine.Evaluate(ev1, "syslog")
	if len(alertChan) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertChan))
	}

	// Event from other host -> Separate window
	engine.Evaluate(evOtherHost, "syslog")
	if len(alertChan) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertChan))
	}

	// 2nd event -> Below boundary
	engine.Evaluate(ev2, "syslog")
	if len(alertChan) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertChan))
	}

	// 3rd event -> Exactly at boundary (count = 3)
	engine.Evaluate(ev3, "syslog")
	if len(alertChan) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alertChan))
	}
	alert := <-alertChan
	if alert.RuleName != "Test Rule" {
		t.Errorf("expected alert for 'Test Rule', got %v", alert.RuleName)
	}

	// 4th event -> Post boundary reset (it should start counting from 1 again)
	ev4 := &domain.ParsedEvent{Hostname: strPtr("host1"), Message: strPtr("fail 4"), ID: 5}
	engine.Evaluate(ev4, "syslog")
	if len(alertChan) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertChan))
	}
}

// 2b. Test Time Eviction Logic
func TestEngine_ThresholdTimeEviction(t *testing.T) {
	loader := NewRuleLoader()
	ruleID := "test-rule-2"
	rule := &domain.RuleDefinition{
		ID:       ruleID,
		Name:     "Eviction Rule",
		Enabled:  true,
		Severity: domain.SeverityMedium,
		Conditions: []domain.RuleCondition{
			{Field: "message", Operator: "contains", Value: "fail"},
		},
		Threshold: &domain.RuleThreshold{
			Count:         2,
			WindowSeconds: 1, // small window
		},
		Alert: domain.RuleAlert{Title: "Test"},
	}
	loader.LoadFromDefinitions([]*domain.RuleDefinition{rule})

	alertChan := make(chan *domain.Alert, 10)
	engine := NewEngine(loader, alertChan)

	ev1 := &domain.ParsedEvent{Message: strPtr("fail")}
	engine.Evaluate(ev1, "syslog")

	// Wait for the window to expire
	time.Sleep(1200 * time.Millisecond)

	// Now send 2nd event. The first should be evicted.
	engine.Evaluate(ev1, "syslog")
	if len(alertChan) != 0 {
		t.Errorf("expected 0 alerts (evicted), got %d", len(alertChan))
	}

	// Send another event immediately (now we have 2 in the fresh window)
	engine.Evaluate(ev1, "syslog")
	if len(alertChan) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alertChan))
	}
	<-alertChan // drain
}

// 3. Concurrency test (safe under go test -race)
func TestEngine_Concurrency(t *testing.T) {
	loader := NewRuleLoader()
	rule := &domain.RuleDefinition{
		ID:       "concurrent-rule",
		Name:     "Concurrent Rule",
		Enabled:  true,
		Severity: domain.SeverityHigh,
		Conditions: []domain.RuleCondition{
			{Field: "message", Operator: "contains", Value: "trigger"},
		},
		Threshold: &domain.RuleThreshold{
			Count:         10, // Takes 10 events to trigger
			WindowSeconds: 5,
			GroupBy:       "hostname",
		},
		Alert: domain.RuleAlert{Title: "Concurrent Alert {{hostname}}"},
	}
	loader.LoadFromDefinitions([]*domain.RuleDefinition{rule})

	alertChan := make(chan *domain.Alert, 1000)
	engine := NewEngine(loader, alertChan)

	ev := &domain.ParsedEvent{Hostname: strPtr("hostA"), Message: strPtr("trigger"), ID: 1}

	var wg sync.WaitGroup
	// Fire 100 events concurrently. We expect exactly 10 alerts because each takes 10 to trigger and resets.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			engine.Evaluate(ev, "syslog")
		}()
	}
	wg.Wait()

	if len(alertChan) != 10 {
		t.Errorf("expected exactly 10 alerts, got %d", len(alertChan))
	}
}
