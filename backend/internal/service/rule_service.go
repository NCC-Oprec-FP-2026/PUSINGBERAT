package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type RuleRepository interface {
	Create(ctx context.Context, rule *domain.Rule) error
	GetByID(ctx context.Context, id string) (*domain.Rule, error)
	GetByName(ctx context.Context, name string) (*domain.Rule, error)
	List(ctx context.Context, filter repository.RuleFilter) ([]domain.Rule, int64, error)
	Update(ctx context.Context, rule *domain.Rule) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	Delete(ctx context.Context, id string) error
}

type RuleService struct {
	repo RuleRepository
}

func NewRuleService(repo RuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

func (s *RuleService) Create(ctx context.Context, rule *domain.Rule) error {
	normalizeRule(rule)
	if err := validateRule(rule); err != nil {
		return err
	}
	return s.repo.Create(ctx, rule)
}

func (s *RuleService) GetByID(ctx context.Context, id string) (*domain.Rule, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (s *RuleService) List(ctx context.Context, filter repository.RuleFilter) ([]domain.Rule, int64, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	return s.repo.List(ctx, filter)
}

func (s *RuleService) Update(ctx context.Context, rule *domain.Rule) error {
	if rule == nil {
		return fmt.Errorf("%w: rule is required", ErrValidation)
	}
	if strings.TrimSpace(rule.ID) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	normalizeRule(rule)
	if err := validateRule(rule); err != nil {
		return err
	}
	return s.repo.Update(ctx, rule)
}

func (s *RuleService) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.SetEnabled(ctx, strings.TrimSpace(id), enabled)
}

func (s *RuleService) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

func normalizeRule(rule *domain.Rule) {
	if rule == nil {
		return
	}
	rule.Name = strings.TrimSpace(rule.Name)
	rule.YAMLContent = strings.TrimSpace(rule.YAMLContent)
	if rule.Severity == "" {
		rule.Severity = domain.SeverityMedium
	}
}

func validateRule(rule *domain.Rule) error {
	if rule == nil {
		return fmt.Errorf("%w: rule is required", ErrValidation)
	}
	if rule.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if rule.YAMLContent == "" {
		return fmt.Errorf("%w: yaml_content is required", ErrValidation)
	}

	switch rule.Severity {
	case domain.SeverityInfo, domain.SeverityLow, domain.SeverityMedium, domain.SeverityHigh, domain.SeverityCritical:
	default:
		return fmt.Errorf("%w: severity must be one of info, low, medium, high, critical", ErrValidation)
	}

	return nil
}
