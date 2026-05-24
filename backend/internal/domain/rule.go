package domain

import (
	"time"

	"github.com/google/uuid"
)

// Rule represents a YAML-based detection rule stored in the `rules` table.
// The raw YAML is persisted in YAMLContent so that users can edit it from
// the UI; the extracted metadata columns (name, severity, enabled) power
// efficient queries without parsing YAML on every request.
type Rule struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	YAMLContent string        `json:"yaml_content"`
	Severity    SeverityLevel `json:"severity"`
	Enabled     bool          `json:"enabled"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// YAML schema types — used to unmarshal YAML rule definitions from disk
// or from the Rule.YAMLContent field.  These mirror the YAML format
// documented in README §6.1 and are intentionally separate from the
// database-facing Rule struct above.
// ---------------------------------------------------------------------------

// RuleDefinition is the top-level struct that a single YAML rule file
// unmarshals into.  Every field uses a yaml struct tag so that
// goccy/go-yaml can map YAML keys to Go fields unambiguously.
type RuleDefinition struct {
	// ID is the rule's unique string identifier (e.g. "ssh-brute-force-001").
	ID string `yaml:"id" json:"id"`

	// Name is the human-readable rule name shown in alerts and the UI.
	Name string `yaml:"name" json:"name"`

	// Description provides additional context about what this rule detects.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Enabled controls whether the engine evaluates this rule.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Severity is the alert severity produced when this rule fires.
	Severity SeverityLevel `yaml:"severity" json:"severity"`

	// LogTypes restricts which log source types this rule evaluates against.
	// An empty slice means the rule applies to ALL log types.
	LogTypes []string `yaml:"log_types,omitempty" json:"log_types,omitempty"`

	// Conditions is the list of field-level match predicates.  ALL
	// conditions must match for the rule to fire (AND logic).
	Conditions []RuleCondition `yaml:"conditions" json:"conditions"`

	// Threshold is optional.  When present, the rule only fires once
	// the condition-match count within the sliding window reaches
	// Threshold.Count.  When absent, the rule fires immediately on
	// every matching event.
	Threshold *RuleThreshold `yaml:"threshold,omitempty" json:"threshold,omitempty"`

	// Alert defines the title and description templates for the
	// generated alert.  Templates may contain {{field}} placeholders
	// that are interpolated at alert-generation time.
	Alert RuleAlert `yaml:"alert" json:"alert"`
}

// RuleCondition represents a single predicate evaluated against a parsed
// event field.  The Operator determines the comparison type and the Value
// provides the expected operand.
type RuleCondition struct {
	// Field is the ParsedEvent field name to evaluate.  Supported values:
	//   message, hostname, process, pid, log_level, raw_line
	Field string `yaml:"field" json:"field"`

	// Operator is the comparison function.  One of:
	//   equals, contains, starts_with, ends_with, regex, gt, lt
	Operator string `yaml:"operator" json:"operator"`

	// Value is the expected operand for the comparison.
	// For gt/lt operators this is parsed as a numeric string at match time.
	Value string `yaml:"value" json:"value"`
}

// RuleThreshold defines the sliding-window parameters for threshold-based
// rules.  When omitted from YAML (nil pointer), the rule fires on every
// single matching event.
type RuleThreshold struct {
	// Count is the minimum number of condition matches required within
	// the window before an alert is generated.
	Count int `yaml:"count" json:"count"`

	// WindowSeconds is the sliding window size in seconds.
	WindowSeconds int `yaml:"window_seconds" json:"window_seconds"`

	// GroupBy is an optional event field used to partition threshold
	// counters.  For example, "hostname" counts matches per-host.
	// When empty, all matches are counted together.
	GroupBy string `yaml:"group_by,omitempty" json:"group_by,omitempty"`
}

// RuleAlert defines the alert output template.  The Title and Description
// fields may contain {{field}} placeholders (e.g. {{hostname}}, {{count}},
// {{window_seconds}}) that are resolved when the alert is generated.
type RuleAlert struct {
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}
