package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/ruleengine"
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

// EventService orchestrates ParsedEvent operations and hosts the background
// persistence goroutine that drains the watcher pipeline's event channel.
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

// ---------------------------------------------------------------------------
// Background persistence goroutine
// ---------------------------------------------------------------------------

// StartPersistenceWorker launches a background goroutine that reads parsed
// events from the provided channel and persists them to the database via
// EventRepository.Create.
//
// The goroutine runs until ctx is cancelled or eventChan is closed. Parse
// or DB errors are logged but never crash the worker — one bad event must
// not stop the pipeline.
//
// This is called once from main.go after DI wiring is complete.
func (s *EventService) StartPersistenceWorker(
	ctx context.Context, 
	eventChan <-chan *domain.ParsedEvent,
	engine *ruleengine.Engine,
	resolveLogType func(uuid.UUID) string,
) {
	go func() {
		slog.Info("event persistence worker started")
		var saved, dropped int64

		for {
			select {
			case <-ctx.Done():
				slog.Info("event persistence worker stopping",
					"saved", saved,
					"dropped", dropped,
				)
				return

			case ev, ok := <-eventChan:
				if !ok {
					slog.Info("event persistence worker: channel closed",
						"saved", saved,
						"dropped", dropped,
					)
					return
				}

				if err := s.repo.Create(ctx, ev); err != nil {
					dropped++
					slog.Error("event persistence worker: save failed",
						"source_id", ev.LogSourceID,
						"err", err,
					)
					continue
				}
				saved++

				// Resolve log_type
				logType := ""
				if resolveLogType != nil {
					logType = resolveLogType(ev.LogSourceID)
				}

				// Forward to Rule Engine
				if engine != nil {
					engine.Evaluate(ev, logType)
				}

				if saved%100 == 0 {
					slog.Debug("event persistence worker progress",
						"saved", saved,
						"dropped", dropped,
					)
				}
			}
		}
	}()
}
