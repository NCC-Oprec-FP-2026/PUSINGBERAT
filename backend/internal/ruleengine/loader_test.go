package ruleengine

import (
	"testing"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeValidYAML(id, name string) []byte {
	return []byte(`
id: ` + id + `
name: ` + name + `
enabled: true
severity: medium
conditions:
  - field: message
    operator: contains
    value: "error"
alert:
  title: "Test Alert"
`)
}

// ---------------------------------------------------------------------------
// ParseYAML tests
// ---------------------------------------------------------------------------

func TestParseYAML_Valid(t *testing.T) {
	data := makeValidYAML("rule-001", "Test Rule")
	def, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.ID != "rule-001" {
		t.Errorf("expected ID %q, got %q", "rule-001", def.ID)
	}
	if def.Name != "Test Rule" {
		t.Errorf("expected name %q, got %q", "Test Rule", def.Name)
	}
	if def.Severity != domain.SeverityMedium {
		t.Errorf("expected severity %q, got %q", domain.SeverityMedium, def.Severity)
	}
	if len(def.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(def.Conditions))
	}
}

func TestParseYAML_WithThreshold(t *testing.T) {
	data := []byte(`
id: rule-thresh-001
name: Threshold Rule
enabled: true
severity: high
conditions:
  - field: message
    operator: contains
    value: "fail"
threshold:
  count: 5
  window_seconds: 60
  group_by: hostname
alert:
  title: "Threshold Fired"
  description: "{{count}} events in {{window_seconds}}s"
`)
	def, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Threshold == nil {
		t.Fatal("expected threshold to be non-nil")
	}
	if def.Threshold.Count != 5 {
		t.Errorf("expected count 5, got %d", def.Threshold.Count)
	}
	if def.Threshold.WindowSeconds != 60 {
		t.Errorf("expected window_seconds 60, got %d", def.Threshold.WindowSeconds)
	}
	if def.Threshold.GroupBy != "hostname" {
		t.Errorf("expected group_by 'hostname', got %q", def.Threshold.GroupBy)
	}
}

func TestParseYAML_MissingID(t *testing.T) {
	data := []byte(`
name: No ID Rule
enabled: true
severity: medium
conditions:
  - field: message
    operator: contains
    value: "error"
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for missing id, got nil")
	}
}

func TestParseYAML_MissingName(t *testing.T) {
	data := []byte(`
id: rule-no-name
enabled: true
severity: medium
conditions:
  - field: message
    operator: contains
    value: "error"
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestParseYAML_InvalidSeverity(t *testing.T) {
	data := []byte(`
id: rule-bad-sev
name: Bad Severity
enabled: true
severity: mega-ultra
conditions:
  - field: message
    operator: contains
    value: "error"
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for invalid severity, got nil")
	}
}

func TestParseYAML_EmptyConditions(t *testing.T) {
	data := []byte(`
id: rule-no-cond
name: No Conditions Rule
enabled: true
severity: medium
conditions: []
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for empty conditions, got nil")
	}
}

func TestParseYAML_MissingAlertTitle(t *testing.T) {
	data := []byte(`
id: rule-no-title
name: No Alert Title
enabled: true
severity: medium
conditions:
  - field: message
    operator: contains
    value: "error"
alert:
  description: "desc only"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for missing alert title, got nil")
	}
}

func TestParseYAML_InvalidConditionOperator(t *testing.T) {
	data := []byte(`
id: rule-bad-op
name: Bad Operator
enabled: true
severity: medium
conditions:
  - field: message
    operator: magic_operator
    value: "error"
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for invalid operator, got nil")
	}
}

func TestParseYAML_InvalidThreshold_ZeroCount(t *testing.T) {
	data := []byte(`
id: rule-zero-count
name: Zero Count Threshold
enabled: true
severity: medium
conditions:
  - field: message
    operator: contains
    value: "error"
threshold:
  count: 0
  window_seconds: 60
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for threshold.count = 0, got nil")
	}
}

func TestParseYAML_InvalidThreshold_ZeroWindow(t *testing.T) {
	data := []byte(`
id: rule-zero-window
name: Zero Window Threshold
enabled: true
severity: medium
conditions:
  - field: message
    operator: contains
    value: "error"
threshold:
  count: 3
  window_seconds: 0
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for threshold.window_seconds = 0, got nil")
	}
}

func TestParseYAML_MissingConditionField(t *testing.T) {
	data := []byte(`
id: rule-no-field
name: Missing Field
enabled: true
severity: medium
conditions:
  - field: ""
    operator: contains
    value: "error"
alert:
  title: "Test"
`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for empty condition field, got nil")
	}
}

func TestParseYAML_InvalidYAML(t *testing.T) {
	data := []byte(`this is: not: valid: yaml: [`)
	_, err := ParseYAML(data)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

// ---------------------------------------------------------------------------
// RuleLoader tests
// ---------------------------------------------------------------------------

func TestRuleLoader_LoadFromDefinitions(t *testing.T) {
	loader := NewRuleLoader()
	defs := []*domain.RuleDefinition{
		{
			ID: "r1", Name: "Rule 1", Enabled: true,
			Severity:   domain.SeverityMedium,
			Conditions: []domain.RuleCondition{{Field: "message", Operator: "contains", Value: "err"}},
			Alert:      domain.RuleAlert{Title: "Alert 1"},
		},
		{
			ID: "r2", Name: "Rule 2", Enabled: true,
			Severity:   domain.SeverityHigh,
			Conditions: []domain.RuleCondition{{Field: "process", Operator: "equals", Value: "nginx"}},
			Alert:      domain.RuleAlert{Title: "Alert 2"},
		},
	}
	loader.LoadFromDefinitions(defs)
	if loader.RuleCount() != 2 {
		t.Errorf("expected 2 rules, got %d", loader.RuleCount())
	}
}

func TestRuleLoader_LoadFromDefinitions_Empty(t *testing.T) {
	loader := NewRuleLoader()
	loader.LoadFromDefinitions(nil)
	if loader.RuleCount() != 0 {
		t.Errorf("expected 0 rules after nil load, got %d", loader.RuleCount())
	}
}

func TestRuleLoader_GetRules_ReturnsSnapshot(t *testing.T) {
	loader := NewRuleLoader()
	def := &domain.RuleDefinition{
		ID: "r1", Name: "Rule 1", Enabled: true,
		Severity:   domain.SeverityMedium,
		Conditions: []domain.RuleCondition{{Field: "message", Operator: "contains", Value: "err"}},
		Alert:      domain.RuleAlert{Title: "Alert 1"},
	}
	loader.LoadFromDefinitions([]*domain.RuleDefinition{def})

	rules1 := loader.GetRules()
	// Mutate the snapshot — the internal slice should be unaffected.
	rules1[0] = nil

	rules2 := loader.GetRules()
	if rules2[0] == nil {
		t.Error("internal slice was mutated by caller — snapshot isolation broken")
	}
}

func TestRuleLoader_LoadFromDB_SkipsDisabled(t *testing.T) {
	loader := NewRuleLoader()
	rules := []domain.Rule{
		{
			Name:        "Enabled Rule",
			YAMLContent: string(makeValidYAML("r-enabled", "Enabled Rule")),
			Enabled:     true,
		},
		{
			Name:        "Disabled Rule",
			YAMLContent: string(makeValidYAML("r-disabled", "Disabled Rule")),
			Enabled:     false,
		},
	}
	if err := loader.LoadFromDB(rules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loader.RuleCount() != 1 {
		t.Errorf("expected 1 rule (disabled skipped), got %d", loader.RuleCount())
	}
}

func TestRuleLoader_LoadFromDB_SkipsInvalidYAML(t *testing.T) {
	loader := NewRuleLoader()
	rules := []domain.Rule{
		{
			Name:        "Valid Rule",
			YAMLContent: string(makeValidYAML("r-valid", "Valid Rule")),
			Enabled:     true,
		},
		{
			Name:        "Invalid YAML Rule",
			YAMLContent: "this: is: not: valid: yaml: [",
			Enabled:     true,
		},
	}
	err := loader.LoadFromDB(rules)
	// Should return an error (skipped rules), but valid ones still load.
	if err == nil {
		t.Fatal("expected error for invalid YAML rule, got nil")
	}
	if loader.RuleCount() != 1 {
		t.Errorf("expected 1 valid rule loaded, got %d", loader.RuleCount())
	}
}

func TestRuleLoader_LoadFromDB_Empty(t *testing.T) {
	loader := NewRuleLoader()
	if err := loader.LoadFromDB([]domain.Rule{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loader.RuleCount() != 0 {
		t.Errorf("expected 0 rules, got %d", loader.RuleCount())
	}
}

func TestRuleLoader_RuleCount_AfterReplace(t *testing.T) {
	loader := NewRuleLoader()
	def1 := &domain.RuleDefinition{
		ID: "r1", Name: "Rule 1", Enabled: true,
		Severity:   domain.SeverityMedium,
		Conditions: []domain.RuleCondition{{Field: "message", Operator: "contains", Value: "err"}},
		Alert:      domain.RuleAlert{Title: "A"},
	}
	loader.LoadFromDefinitions([]*domain.RuleDefinition{def1})
	if loader.RuleCount() != 1 {
		t.Fatalf("expected 1, got %d", loader.RuleCount())
	}
	// Replace with empty
	loader.LoadFromDefinitions(nil)
	if loader.RuleCount() != 0 {
		t.Errorf("expected 0 after replace, got %d", loader.RuleCount())
	}
}

// ---------------------------------------------------------------------------
// isValidOperator tests (white-box, same package)
// ---------------------------------------------------------------------------

func TestIsValidOperator(t *testing.T) {
	valid := []string{"equals", "contains", "starts_with", "ends_with", "regex", "gt", "lt"}
	for _, op := range valid {
		if !isValidOperator(op) {
			t.Errorf("expected operator %q to be valid", op)
		}
	}
	invalid := []string{"", "magic", "like", "in", "not_equals"}
	for _, op := range invalid {
		if isValidOperator(op) {
			t.Errorf("expected operator %q to be invalid", op)
		}
	}
}
