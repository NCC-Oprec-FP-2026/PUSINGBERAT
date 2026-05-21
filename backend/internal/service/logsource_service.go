// Package service implements the business-logic layer for the PUSINGBERAT SIEM
// backend. Each service struct depends on repository interfaces (defined here,
// at the point of consumption) so that concrete implementations can be swapped
// for mocks in tests.
//
// Services are intentionally thin for Day 2: they validate input, delegate to
// repositories, and wrap errors with context. Future sprints will add watcher
// registry notifications, rule engine reloads, and WS broadcasts here.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Repository interface (consumed by LogSourceService)
// ---------------------------------------------------------------------------

// LogSourceRepository defines the persistence contract that LogSourceService
// requires. The concrete implementation lives in package repository.
type LogSourceRepository interface {
	Create(ctx context.Context, ls *domain.LogSource) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error)
	List(ctx context.Context) ([]domain.LogSource, error)
	Update(ctx context.Context, ls *domain.LogSource) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// LogSourceService orchestrates LogSource CRUD operations with input
// validation and error wrapping.
type LogSourceService struct {
	repo LogSourceRepository
}

// NewLogSourceService constructs a LogSourceService with the given repository.
func NewLogSourceService(repo LogSourceRepository) *LogSourceService {
	return &LogSourceService{repo: repo}
}

// Create validates and persists a new LogSource.
func (s *LogSourceService) Create(ctx context.Context, ls *domain.LogSource) error {
	if err := s.validate(ls); err != nil {
		return err
	}

	// Default status to "active" when not explicitly set.
	if ls.Status == "" {
		ls.Status = domain.LogSourceStatusActive
	}
	// Default log_type to "generic" when not explicitly set.
	if ls.LogType == "" {
		ls.LogType = "generic"
	}

	if err := s.repo.Create(ctx, ls); err != nil {
		return fmt.Errorf("logSourceService.Create: %w", err)
	}
	return nil
}

// GetByID retrieves a single LogSource by ID.
func (s *LogSourceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error) {
	ls, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("logSourceService.GetByID: %w", err)
	}
	return ls, nil
}

// List returns all registered log sources.
func (s *LogSourceService) List(ctx context.Context) ([]domain.LogSource, error) {
	sources, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("logSourceService.List: %w", err)
	}
	return sources, nil
}

// Update validates and persists changes to an existing LogSource.
func (s *LogSourceService) Update(ctx context.Context, ls *domain.LogSource) error {
	if err := s.validate(ls); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, ls); err != nil {
		return fmt.Errorf("logSourceService.Update: %w", err)
	}
	return nil
}

// Delete removes a LogSource by ID.
func (s *LogSourceService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("logSourceService.Delete: %w", err)
	}
	return nil
}

// validate runs input checks before creating or updating a LogSource.
func (s *LogSourceService) validate(ls *domain.LogSource) error {
	if strings.TrimSpace(ls.Name) == "" {
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if strings.TrimSpace(ls.FilePath) == "" {
		return fmt.Errorf("%w: file_path is required", domain.ErrValidation)
	}
	if ls.Status != "" && !ls.Status.Valid() {
		return fmt.Errorf("%w: invalid status %q", domain.ErrValidation, ls.Status)
	}
	validLogTypes := map[string]bool{"generic": true, "syslog": true, "nginx": true}
	if ls.LogType != "" && !validLogTypes[ls.LogType] {
		return fmt.Errorf("%w: invalid log_type %q", domain.ErrValidation, ls.LogType)
	}
	return nil
}
