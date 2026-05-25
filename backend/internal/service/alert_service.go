package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

// ---------------------------------------------------------------------------
// Repository interface (consumed by AlertService)
// ---------------------------------------------------------------------------

// AlertRepository defines the persistence contract that AlertService requires.
type AlertRepository interface {
	Create(ctx context.Context, a *domain.Alert) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	List(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error)
	Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	// GetSeverityCounts(ctx context.Context) (map[string]int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MarkDiscordSent(ctx context.Context, id uuid.UUID) error
	GetAlertsBySeverity(ctx context.Context) (map[string]int64, error)
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// AlertService orchestrates Alert operations.
type AlertService struct {
	repo AlertRepository
}

// NewAlertService constructs an AlertService with the given repository.
func NewAlertService(repo AlertRepository) *AlertService {
	return &AlertService{repo: repo}
}

// Create persists a new alert. Called by the rule engine pipeline.
func (s *AlertService) Create(ctx context.Context, a *domain.Alert) error {
	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("alertService.Create: %w", err)
	}
	return nil
}

// GetByID retrieves a single alert.
func (s *AlertService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("alertService.GetByID: %w", err)
	}
	return a, nil
}

// List returns paginated alerts.
func (s *AlertService) List(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error) {
	alerts, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("alertService.List: %w", err)
	}
	return alerts, total, nil
}

// Acknowledge marks an alert as acknowledged.
func (s *AlertService) Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	a, err := s.repo.Acknowledge(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("alertService.Acknowledge: %w", err)
	}
	return a, nil
}

// GetSeverityCounts retrieves a map of severity levels to alert counts from the repository.
func (s *AlertService) GetSeverityCounts(ctx context.Context) (map[string]int64, error) {
	counts, err := s.repo.GetAlertsBySeverity(ctx)
	if err != nil {
		return nil, fmt.Errorf("alertService.GetSeverityCounts: %w", err)
	}
	return counts, nil
}

// Delete removes an alert by ID.
func (s *AlertService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("alertService.Delete: %w", err)
	}
	return nil
}

// GetAlertsBySeverity returns alert counts grouped by severity level.
func (s *AlertService) GetAlertsBySeverity(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetAlertsBySeverity(ctx)
}
