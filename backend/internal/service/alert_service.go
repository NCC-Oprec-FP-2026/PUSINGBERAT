package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type AlertRepository interface {
	Create(ctx context.Context, alert *domain.Alert) error
	GetByID(ctx context.Context, id string) (*domain.Alert, error)
	List(ctx context.Context, filter repository.AlertFilter) ([]domain.Alert, int64, error)
	Acknowledge(ctx context.Context, id string) (*domain.Alert, error)
	MarkDiscordSent(ctx context.Context, id string) error
	ListDiscordPending(ctx context.Context, limit int) ([]domain.Alert, error)
	Delete(ctx context.Context, id string) error
}

type AlertService struct {
	repo AlertRepository
}

func NewAlertService(repo AlertRepository) *AlertService {
	return &AlertService{repo: repo}
}

func (s *AlertService) Create(ctx context.Context, alert *domain.Alert) error {
	if alert == nil {
		return fmt.Errorf("%w: alert is required", ErrValidation)
	}
	alert.RuleName = strings.TrimSpace(alert.RuleName)
	alert.Title = strings.TrimSpace(alert.Title)
	if alert.RuleName == "" {
		return fmt.Errorf("%w: rule_name is required", ErrValidation)
	}
	if alert.Title == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if alert.Severity == "" {
		alert.Severity = domain.SeverityMedium
	}
	return s.repo.Create(ctx, alert)
}

func (s *AlertService) GetByID(ctx context.Context, id string) (*domain.Alert, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (s *AlertService) List(ctx context.Context, filter repository.AlertFilter) ([]domain.Alert, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *AlertService) Acknowledge(ctx context.Context, id string) (*domain.Alert, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.Acknowledge(ctx, strings.TrimSpace(id))
}

func (s *AlertService) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

func (s *AlertService) MarkDiscordSent(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.MarkDiscordSent(ctx, strings.TrimSpace(id))
}

func (s *AlertService) ListDiscordPending(ctx context.Context, limit int) ([]domain.Alert, error) {
	return s.repo.ListDiscordPending(ctx, limit)
}
