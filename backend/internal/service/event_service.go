package service

import (
	"context"
	"fmt"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

// ---------------------------------------------------------------------------
// Repository interface (consumed by EventService)
// ---------------------------------------------------------------------------

// EventRepository defines the persistence contract that EventService requires.
type EventRepository interface {
	Create(ctx context.Context, ev *domain.ParsedEvent) error
	GetByID(ctx context.Context, id int64) (*domain.ParsedEvent, error)
	List(ctx context.Context, params repository.EventListParams) ([]domain.ParsedEvent, int64, error)
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// EventService orchestrates ParsedEvent operations.
type EventService struct {
	repo EventRepository
}

// NewEventService constructs an EventService with the given repository.
func NewEventService(repo EventRepository) *EventService {
	return &EventService{repo: repo}
}

// Create persists a new parsed event. Called internally by the log watcher
// pipeline — not directly by an HTTP handler.
func (s *EventService) Create(ctx context.Context, ev *domain.ParsedEvent) error {
	if err := s.repo.Create(ctx, ev); err != nil {
		return fmt.Errorf("eventService.Create: %w", err)
	}
	return nil
}

// GetByID retrieves a single event by its BIGSERIAL ID.
func (s *EventService) GetByID(ctx context.Context, id int64) (*domain.ParsedEvent, error) {
	ev, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("eventService.GetByID: %w", err)
	}
	return ev, nil
}

// List returns paginated events.
func (s *EventService) List(ctx context.Context, params repository.EventListParams) ([]domain.ParsedEvent, int64, error) {
	events, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("eventService.List: %w", err)
	}
	return events, total, nil
}
