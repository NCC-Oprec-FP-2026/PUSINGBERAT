package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByID(ctx context.Context, id int64) (*domain.Event, error)
	List(ctx context.Context, filter repository.EventFilter) ([]domain.Event, int64, error)
	Delete(ctx context.Context, id int64) error
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type EventService struct {
	repo EventRepository
}

func NewEventService(repo EventRepository) *EventService {
	return &EventService{repo: repo}
}

func (s *EventService) Create(ctx context.Context, event *domain.Event) error {
	if event == nil {
		return fmt.Errorf("%w: event is required", ErrValidation)
	}
	event.RawLine = strings.TrimSpace(event.RawLine)
	event.LogSourceID = strings.TrimSpace(event.LogSourceID)
	if event.RawLine == "" {
		return fmt.Errorf("%w: raw_line is required", ErrValidation)
	}
	if event.LogSourceID == "" {
		return fmt.Errorf("%w: log_source_id is required", ErrValidation)
	}
	if event.EventTime.IsZero() {
		event.EventTime = time.Now().UTC()
	}
	return s.repo.Create(ctx, event)
}

func (s *EventService) GetByID(ctx context.Context, id int64) (*domain.Event, error) {
	if id < 1 {
		return nil, fmt.Errorf("%w: id must be positive", ErrValidation)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *EventService) List(ctx context.Context, filter repository.EventFilter) ([]domain.Event, int64, error) {
	filter.SourceID = strings.TrimSpace(filter.SourceID)
	filter.Level = strings.TrimSpace(filter.Level)
	filter.Search = strings.TrimSpace(filter.Search)
	return s.repo.List(ctx, filter)
}

func (s *EventService) Delete(ctx context.Context, id int64) error {
	if id < 1 {
		return fmt.Errorf("%w: id must be positive", ErrValidation)
	}
	return s.repo.Delete(ctx, id)
}

func (s *EventService) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		return 0, fmt.Errorf("%w: cutoff is required", ErrValidation)
	}
	return s.repo.DeleteOlderThan(ctx, cutoff)
}
