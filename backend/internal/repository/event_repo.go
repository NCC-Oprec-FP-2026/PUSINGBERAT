package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// EventRepo provides CRUD operations for the events table.
type EventRepo struct {
	pool *pgxpool.Pool
}

// NewEventRepo creates a new EventRepo backed by the given connection pool.
func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

// Create inserts a new ParsedEvent. The ID and ReceivedAt columns are
// populated by PostgreSQL (BIGSERIAL / DEFAULT NOW()) and scanned back.
func (r *EventRepo) Create(ctx context.Context, ev *domain.ParsedEvent) error {
	query := `
		INSERT INTO events (log_source_id, raw_line, message, hostname, process, pid, log_level, event_time, extra)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, received_at`

	err := r.pool.QueryRow(ctx, query,
		ev.LogSourceID,
		ev.RawLine,
		ev.Message,
		ev.Hostname,
		ev.Process,
		ev.PID,
		ev.LogLevel,
		ev.EventTime,
		ev.Extra,
	).Scan(&ev.ID, &ev.ReceivedAt)

	if err != nil {
		return fmt.Errorf("eventRepo.Create: %w", err)
	}
	return nil
}

// GetByID retrieves a single event by its BIGSERIAL primary key.
func (r *EventRepo) GetByID(ctx context.Context, id int64) (*domain.ParsedEvent, error) {
	query := `
		SELECT id, log_source_id, raw_line, message, hostname, process, pid,
		       log_level, event_time, received_at, extra
		FROM events
		WHERE id = $1`

	ev := &domain.ParsedEvent{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ev.ID,
		&ev.LogSourceID,
		&ev.RawLine,
		&ev.Message,
		&ev.Hostname,
		&ev.Process,
		&ev.PID,
		&ev.LogLevel,
		&ev.EventTime,
		&ev.ReceivedAt,
		&ev.Extra,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("eventRepo.GetByID: %w", err)
	}
	return ev, nil
}

// ListParams holds optional filter/pagination parameters for List.
type EventListParams struct {
	Limit  int
	Offset int
}

// List retrieves events ordered by event_time descending with pagination.
func (r *EventRepo) List(ctx context.Context, params EventListParams) ([]domain.ParsedEvent, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 200 {
		params.Limit = 200
	}

	// Count total rows for pagination metadata.
	var total int64
	countQuery := `SELECT COUNT(*) FROM events`
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("eventRepo.List count: %w", err)
	}

	query := `
		SELECT id, log_source_id, raw_line, message, hostname, process, pid,
		       log_level, event_time, received_at, extra
		FROM events
		ORDER BY event_time DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("eventRepo.List: %w", err)
	}
	defer rows.Close()

	var events []domain.ParsedEvent
	for rows.Next() {
		var ev domain.ParsedEvent
		if err := rows.Scan(
			&ev.ID,
			&ev.LogSourceID,
			&ev.RawLine,
			&ev.Message,
			&ev.Hostname,
			&ev.Process,
			&ev.PID,
			&ev.LogLevel,
			&ev.EventTime,
			&ev.ReceivedAt,
			&ev.Extra,
		); err != nil {
			return nil, 0, fmt.Errorf("eventRepo.List scan: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("eventRepo.List rows: %w", err)
	}
	return events, total, nil
}
