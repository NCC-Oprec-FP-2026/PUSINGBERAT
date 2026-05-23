// Package service implements the business-logic layer for the PUSINGBERAT SIEM
// backend. Each service struct depends on repository interfaces (defined here,
// at the point of consumption) so that concrete implementations can be swapped
// for mocks in tests.
package service

import (
	"context"
	"fmt"
	"log/slog"
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
// WatcherRegistry interface (optional dependency)
// ---------------------------------------------------------------------------

// WatcherRegistry defines the contract for managing file watchers. The
// concrete implementation lives in package watcher. This is defined at the
// point of consumption to avoid a hard dependency from service → watcher.
type WatcherRegistry interface {
	AddWatcher(source *domain.LogSource) error
	RemoveWatcher(sourceID uuid.UUID)
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// LogSourceService orchestrates LogSource CRUD operations with input
// validation, error wrapping, and optional watcher registry notifications.
type LogSourceService struct {
	repo     LogSourceRepository
	registry WatcherRegistry // nil when watcher pipeline is not enabled
}

// NewLogSourceService constructs a LogSourceService with the given repository.
// The watcher registry can be set later via SetRegistry once the pipeline is
// initialised (avoids circular init order issues in main.go).
func NewLogSourceService(repo LogSourceRepository) *LogSourceService {
	return &LogSourceService{repo: repo}
}

// SetRegistry attaches a WatcherRegistry to this service. After this call,
// Create and Delete will automatically start/stop watchers.
func (s *LogSourceService) SetRegistry(reg WatcherRegistry) {
	s.registry = reg
}

// Create validates and persists a new LogSource. When a WatcherRegistry is
// attached, it also starts watching the new file path.
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

	// --- Watcher Pipeline Integration ----------------------------------------
	// If the source is active and a registry is configured, start watching.
	if s.registry != nil && ls.Status == domain.LogSourceStatusActive {
		if err := s.registry.AddWatcher(ls); err != nil {
			// Log the error but do NOT fail the HTTP request — the source is
			// already persisted. The watcher can be manually restarted later.
			slog.Error("logSourceService.Create: failed to start watcher",
				"source_id", ls.ID,
				"path", ls.FilePath,
				"err", err,
			)
		}
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

// Delete removes a LogSource by ID. When a WatcherRegistry is attached,
// it also stops the associated watcher.
func (s *LogSourceService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("logSourceService.Delete: %w", err)
	}

	// --- Watcher Pipeline Integration ----------------------------------------
	if s.registry != nil {
		s.registry.RemoveWatcher(id)
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
