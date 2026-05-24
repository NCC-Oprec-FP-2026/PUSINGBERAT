package ruleengine_test

import (
	"testing"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/ruleengine"
)

func ptr[T any](v T) *T { return &v }

func TestMatchAllConditions(t *testing.T) {
	ev := &domain.ParsedEvent{
		Message:  ptr("Failed password for root from 192.168.1.50 port 22 ssh2"),
		Process:  ptr("sshd"),
		PID:      ptr(1234),
		Hostname: ptr("web-server-01"),
	}

	tests := []struct {
		name       string
		conditions []domain.RuleCondition
		want       bool
	}{
		{
			name: "Equals Match",
			conditions: []domain.RuleCondition{
				{Field: "process", Operator: "equals", Value: "sshd"},
			},
			want: true,
		},
		{
			name: "Contains Match",
			conditions: []domain.RuleCondition{
				{Field: "message", Operator: "contains", Value: "Failed password"},
			},
			want: true,
		},
		{
			name: "Numeric GT Match",
			conditions: []domain.RuleCondition{
				{Field: "pid", Operator: "gt", Value: "1000"},
			},
			want: true,
		},
		{
			name: "Multiple Conditions (AND) - Match",
			conditions: []domain.RuleCondition{
				{Field: "process", Operator: "equals", Value: "sshd"},
				{Field: "message", Operator: "contains", Value: "Failed password"},
			},
			want: true,
		},
		{
			name: "Multiple Conditions (AND) - Mismatch",
			conditions: []domain.RuleCondition{
				{Field: "process", Operator: "equals", Value: "sshd"},
				{Field: "message", Operator: "contains", Value: "Accepted password"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ruleengine.MatchAllConditions(tt.conditions, ev)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
