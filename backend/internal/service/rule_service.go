package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Repository interface (consumed by RuleService)
// ---------------------------------------------------------------------------

// RuleRepository defines the persistence contract that RuleService requires.
type RuleRepository interface {
	Create(ctx context.Context, rule *domain.Rule) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	List(ctx context.Context) ([]domain.Rule, error)
	ListEnabled(ctx context.Context) ([]domain.Rule, error)
	Update(ctx context.Context, rule *domain.Rule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// RuleService orchestrates Rule CRUD operations with input validation.
type RuleService struct {
	repo RuleRepository
}

// NewRuleService constructs a RuleService with the given repository.
func NewRuleService(repo RuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

// Create validates and persists a new Rule.
func (s *RuleService) Create(ctx context.Context, rule *domain.Rule) error {
	if err := s.validate(rule); err != nil {
		return err
	}

	// Default severity to "medium" when not explicitly set.
	if rule.Severity == "" {
		rule.Severity = domain.SeverityMedium
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return fmt.Errorf("ruleService.Create: %w", err)
	}
	return nil
}

// GetByID retrieves a single Rule by ID.
func (s *RuleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ruleService.GetByID: %w", err)
	}
	return rule, nil
}

// List returns all rules.
func (s *RuleService) List(ctx context.Context) ([]domain.Rule, error) {
	rules, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("ruleService.List: %w", err)
	}
	return rules, nil
}

// Update validates and persists changes to an existing Rule.
func (s *RuleService) Update(ctx context.Context, rule *domain.Rule) error {
	if err := s.validate(rule); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, rule); err != nil {
		return fmt.Errorf("ruleService.Update: %w", err)
	}
	return nil
}

// Toggle flips the enabled flag of an existing Rule.
func (s *RuleService) Toggle(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ruleService.Toggle: %w", err)
	}
	rule.Enabled = !rule.Enabled
	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("ruleService.Toggle: %w", err)
	}
	return rule, nil
}

// Delete removes a Rule by ID.
func (s *RuleService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("ruleService.Delete: %w", err)
	}
	return nil
}

// validate runs input checks before creating or updating a Rule.
func (s *RuleService) validate(rule *domain.Rule) error {
	if strings.TrimSpace(rule.Name) == "" {
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if strings.TrimSpace(rule.YAMLContent) == "" {
		return fmt.Errorf("%w: yaml_content is required", domain.ErrValidation)
	}
	if rule.Severity != "" && !rule.Severity.Valid() {
		return fmt.Errorf("%w: invalid severity %q", domain.ErrValidation, rule.Severity)
	}
	return nil
}
