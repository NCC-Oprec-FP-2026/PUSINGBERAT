package ruleengine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type RuleRepository interface {
	Create(ctx context.Context, rule *domain.Rule) error
	List(ctx context.Context) ([]domain.Rule, error)
	ListEnabled(ctx context.Context) ([]domain.Rule, error)
}

type RuleDefinition struct {
	DatabaseID  uuid.UUID            `yaml:"-"`
	ID          string               `yaml:"id"`
	Name        string               `yaml:"name"`
	Description string               `yaml:"description"`
	Severity    domain.SeverityLevel `yaml:"severity"`
	Enabled     *bool                `yaml:"enabled"`
	LogTypes    []string             `yaml:"log_types"`
	Conditions  []Condition          `yaml:"conditions"`
	Threshold   *Threshold           `yaml:"threshold"`
	Alert       AlertTemplate        `yaml:"alert"`
	yamlContent string               `yaml:"-"`
}

type Condition struct {
	Field    string `yaml:"field"`
	Operator string `yaml:"operator"`
	Value    string `yaml:"value"`
}

type Threshold struct {
	Count         int    `yaml:"count"`
	Window        string `yaml:"window"`
	WindowSeconds int    `yaml:"window_seconds"`
	GroupBy       string `yaml:"group_by"`
}

type AlertTemplate struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

func LoadRuleDefinitionsFromDir(dir string) ([]RuleDefinition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read rules dir: %w", err)
	}

	var rules []RuleDefinition
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read rule %s: %w", path, err)
		}
		if strings.TrimSpace(string(raw)) == "" {
			continue
		}

		rule, err := ParseRuleDefinition(raw)
		if err != nil {
			return nil, fmt.Errorf("parse rule %s: %w", path, err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func ParseRuleDefinition(raw []byte) (RuleDefinition, error) {
	var rule RuleDefinition
	if err := yaml.Unmarshal(raw, &rule); err != nil {
		return RuleDefinition{}, err
	}
	rule.yamlContent = string(raw)
	normalizeRuleDefinition(&rule)
	if err := validateRuleDefinition(rule); err != nil {
		return RuleDefinition{}, err
	}
	return rule, nil
}

func SeedRules(ctx context.Context, repo RuleRepository, dir string) error {
	rules, err := LoadRuleDefinitionsFromDir(dir)
	if err != nil {
		return err
	}

	existingRules, err := repo.List(ctx)
	if err != nil {
		return err
	}
	existingByName := make(map[string]struct{}, len(existingRules))
	for _, rule := range existingRules {
		existingByName[strings.ToLower(strings.TrimSpace(rule.Name))] = struct{}{}
	}

	for _, rule := range rules {
		if _, exists := existingByName[strings.ToLower(strings.TrimSpace(rule.Name))]; exists {
			continue
		}

		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}

		description := rule.Description
		dbRule := &domain.Rule{
			Name:        rule.Name,
			Description: &description,
			YAMLContent: rule.yamlContent,
			Severity:    rule.Severity,
			Enabled:     enabled,
		}
		if err := repo.Create(ctx, dbRule); err != nil {
			return err
		}
		existingByName[strings.ToLower(strings.TrimSpace(rule.Name))] = struct{}{}
	}

	return nil
}

func LoadEnabledRulesFromDB(ctx context.Context, repo RuleRepository) ([]RuleDefinition, error) {
	dbRules, err := repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	rules := make([]RuleDefinition, 0, len(dbRules))
	for _, dbRule := range dbRules {
		rule, err := ParseRuleDefinition([]byte(dbRule.YAMLContent))
		if err != nil {
			return nil, fmt.Errorf("parse DB rule %s: %w", dbRule.Name, err)
		}
		rule.DatabaseID = dbRule.ID
		rules = append(rules, rule)
	}

	return rules, nil
}

func normalizeRuleDefinition(rule *RuleDefinition) {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Description = strings.TrimSpace(rule.Description)
	if rule.Severity == "" {
		rule.Severity = domain.SeverityMedium
	}
	for i := range rule.Conditions {
		rule.Conditions[i].Field = strings.TrimSpace(rule.Conditions[i].Field)
		rule.Conditions[i].Operator = strings.ToLower(strings.TrimSpace(rule.Conditions[i].Operator))
		rule.Conditions[i].Value = strings.TrimSpace(rule.Conditions[i].Value)
	}
}

func validateRuleDefinition(rule RuleDefinition) error {
	if rule.Name == "" {
		return errors.New("name is required")
	}
	if len(rule.Conditions) == 0 {
		return errors.New("at least one condition is required")
	}
	for _, condition := range rule.Conditions {
		if condition.Field == "" || condition.Operator == "" {
			return errors.New("condition field and operator are required")
		}
	}
	if rule.Threshold != nil && rule.Threshold.Count < 1 {
		return errors.New("threshold count must be positive")
	}
	return nil
}
