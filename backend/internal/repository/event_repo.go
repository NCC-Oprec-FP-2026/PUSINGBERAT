package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type EventFilter struct {
	SourceID string
	Level    string
	From     *time.Time
	To       *time.Time
	Search   string
	Page     int
	PerPage  int
}

type EventRepo struct {
	db *pgxpool.Pool
}

func NewEventRepo(db *pgxpool.Pool) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) Create(ctx context.Context, event *domain.Event) error {
	extra, err := marshalJSONB(event.Extra)
	if err != nil {
		return fmt.Errorf("marshal event extra: %w", err)
	}

	const query = `
		INSERT INTO events (
			log_source_id,
			raw_line,
			message,
			hostname,
			process,
			pid,
			log_level,
			event_time,
			extra
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, received_at
	`

	err = r.db.QueryRow(
		ctx,
		query,
		event.LogSourceID,
		event.RawLine,
		event.Message,
		event.Hostname,
		event.Process,
		event.PID,
		event.LogLevel,
		event.EventTime,
		extra,
	).Scan(&event.ID, &event.ReceivedAt)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	return nil
}

func (r *EventRepo) GetByID(ctx context.Context, id int64) (*domain.Event, error) {
	const query = `
		SELECT id, log_source_id::text, raw_line, message, hostname, process, pid,
			log_level, event_time, received_at, extra
		FROM events
		WHERE id = $1
	`

	event, err := scanEvent(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, notFound(err)
	}
	return event, nil
}

func (r *EventRepo) List(ctx context.Context, filter EventFilter) ([]domain.Event, int64, error) {
	clauses, args := buildEventWhere(filter)
	where := whereSQL(clauses)

	var total int64
	countQuery := "SELECT COUNT(*) FROM events" + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	limit, offset := normalizePagination(filter.Page, filter.PerPage)
	args = append(args, limit, offset)

	query := `
		SELECT id, log_source_id::text, raw_line, message, hostname, process, pid,
			log_level, event_time, received_at, extra
		FROM events` + where + orderLimitOffset("ORDER BY event_time DESC, id DESC", len(args)-2)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Event, error) {
		event, err := scanEvent(row)
		if err != nil {
			return domain.Event{}, err
		}
		return *event, nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scan events: %w", err)
	}

	return events, total, nil
}

func (r *EventRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM events WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *EventRepo) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM events WHERE received_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old events: %w", err)
	}
	return tag.RowsAffected(), nil
}

func buildEventWhere(filter EventFilter) ([]string, []any) {
	var clauses []string
	var args []any

	if strings.TrimSpace(filter.SourceID) != "" {
		args = append(args, strings.TrimSpace(filter.SourceID))
		clauses = append(clauses, fmt.Sprintf("log_source_id = $%d::uuid", len(args)))
	}
	if strings.TrimSpace(filter.Level) != "" {
		args = append(args, strings.TrimSpace(filter.Level))
		clauses = append(clauses, fmt.Sprintf("log_level = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("event_time >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("event_time <= $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		clauses = append(clauses, fmt.Sprintf("(message ILIKE $%d OR raw_line ILIKE $%d)", len(args), len(args)))
	}

	return clauses, args
}

type eventRow interface {
	Scan(dest ...any) error
}

func scanEvent(row eventRow) (*domain.Event, error) {
	var event domain.Event
	var message pgtype.Text
	var hostname pgtype.Text
	var process pgtype.Text
	var pid pgtype.Int4
	var logLevel pgtype.Text
	var rawExtra []byte

	err := row.Scan(
		&event.ID,
		&event.LogSourceID,
		&event.RawLine,
		&message,
		&hostname,
		&process,
		&pid,
		&logLevel,
		&event.EventTime,
		&event.ReceivedAt,
		&rawExtra,
	)
	if err != nil {
		return nil, err
	}

	event.Message = textPtr(message)
	event.Hostname = textPtr(hostname)
	event.Process = textPtr(process)
	event.PID = int32Ptr(pid)
	event.LogLevel = textPtr(logLevel)

	event.Extra, err = unmarshalJSONB(rawExtra)
	if err != nil {
		return nil, fmt.Errorf("unmarshal event extra: %w", err)
	}

	return &event, nil
}
