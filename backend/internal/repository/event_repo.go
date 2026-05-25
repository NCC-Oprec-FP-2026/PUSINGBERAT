package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"


	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Dashboard DTO types
// ---------------------------------------------------------------------------

// StatsOverview holds the four top-level counters for the dashboard stat cards.
type StatsOverview struct {
	TotalEvents24h  int64 `json:"total_events_24h"`
	TotalAlerts24h  int64 `json:"total_alerts_24h"`
	CriticalAlerts  int64 `json:"critical_alerts"`
	ActiveSources   int64 `json:"active_sources"`
}

// TimelinePoint represents a single hourly bucket in the events timeline chart.
type TimelinePoint struct {
	Hour  time.Time `json:"hour"`
	Count int64     `json:"count"`
}

// TopSource represents a log source ranked by event count for the dashboard.
type TopSource struct {
	SourceID   uuid.UUID `json:"source_id"`
	SourceName string    `json:"source_name"`
	Count      int64     `json:"count"`
}

// ---------------------------------------------------------------------------
// Filtered list parameters
// ---------------------------------------------------------------------------

// EventFilterParams extends basic pagination with optional filters matching
// the GET /api/v1/events query parameters defined in README §10.3.
type EventFilterParams struct {
	Limit    int
	Offset   int
	SourceID *uuid.UUID // filter by log_source_id
	Level    *string    // filter by log_level (e.g. "error")
	From     *time.Time // event_time >= from
	To       *time.Time // event_time <= to
	Search   *string    // ILIKE search on message column
}

// DBQuery defines the minimal set of methods EventRepo needs from a pgx pool.
// This interface allows for easy mocking in unit tests.
type DBQuery interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// EventRepo handles database operations for ParsedEvents and dashboard statistics.
type EventRepo struct {
	pool DBQuery
}

// NewEventRepo constructs a new EventRepo using the provided database pool.
func NewEventRepo(pool DBQuery) *EventRepo {
	return &EventRepo{pool: pool}
}

// Create inserts a new ParsedEvent. The ID and ReceivedAt columns are
// populated by PostgreSQL (BIGSERIAL / DEFAULT NOW()) and scanned back.
func (r *EventRepo) Create(ctx context.Context, ev *domain.ParsedEvent) error {
	query := `
		INSERT INTO events (log_source_id, raw_line, message, hostname, process, pid, log_level, event_time, extra)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, received_at`

	// Default nil Extra to an empty JSON object so the NOT NULL constraint
	// is satisfied. PostgreSQL's column DEFAULT only applies when the column
	// is omitted from the INSERT — explicitly passing NULL triggers a
	// constraint violation.
	extra := ev.Extra
	if extra == nil {
		extra = []byte(`{}`)
	}

	err := r.pool.QueryRow(ctx, query,
		ev.LogSourceID,
		ev.RawLine,
		ev.Message,
		ev.Hostname,
		ev.Process,
		ev.PID,
		ev.LogLevel,
		ev.EventTime,
		extra,
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

// ---------------------------------------------------------------------------
// ListEvents — paginated + filtered event listing (Day 6)
// ---------------------------------------------------------------------------

// buildFilterClause constructs a dynamic WHERE clause and parameter list from
// the supplied EventFilterParams. The paramIdx argument is the 1-based index
// of the next free placeholder (callers typically pass 1).
func buildFilterClause(p EventFilterParams, paramIdx int) (string, []any) {
	var clauses []string
	var args []any

	if p.SourceID != nil {
		clauses = append(clauses, "log_source_id = $"+strconv.Itoa(paramIdx))
		args = append(args, *p.SourceID)
		paramIdx++
	}
	if p.Level != nil {
		clauses = append(clauses, "log_level = $"+strconv.Itoa(paramIdx))
		args = append(args, *p.Level)
		paramIdx++
	}
	if p.From != nil {
		clauses = append(clauses, "event_time >= $"+strconv.Itoa(paramIdx))
		args = append(args, *p.From)
		paramIdx++
	}
	if p.To != nil {
		clauses = append(clauses, "event_time <= $"+strconv.Itoa(paramIdx))
		args = append(args, *p.To)
		paramIdx++
	}
	if p.Search != nil && *p.Search != "" {
		clauses = append(clauses, "message ILIKE $"+strconv.Itoa(paramIdx))
		args = append(args, "%"+*p.Search+"%")
		paramIdx++
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// ListEvents retrieves events ordered by event_time DESC with optional
// filters (source_id, level, from, to, search) and LIMIT/OFFSET pagination.
func (r *EventRepo) ListEvents(ctx context.Context, params EventFilterParams) ([]domain.ParsedEvent, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 200 {
		params.Limit = 200
	}

	whereClause, whereArgs := buildFilterClause(params, 1)

	// --- total count ---
	var total int64
	countQuery := "SELECT COUNT(*) FROM events" + whereClause
	if err := r.pool.QueryRow(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("eventRepo.ListEvents count: %w", err)
	}

	// --- data query ---
	nextIdx := len(whereArgs) + 1
	dataQuery := `
		SELECT id, log_source_id, raw_line, message, hostname, process, pid,
		       log_level, event_time, received_at, extra
		FROM events` + whereClause + `
		ORDER BY event_time DESC
		LIMIT $` + strconv.Itoa(nextIdx) + ` OFFSET $` + strconv.Itoa(nextIdx+1)

	dataArgs := append(whereArgs, params.Limit, params.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("eventRepo.ListEvents: %w", err)
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
			return nil, 0, fmt.Errorf("eventRepo.ListEvents scan: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("eventRepo.ListEvents rows: %w", err)
	}
	return events, total, nil
}

// ---------------------------------------------------------------------------
// GetStatsOverview — concurrent 4-stat dashboard query (Day 6)
// ---------------------------------------------------------------------------

// GetStatsOverview returns the four top-level statistics for the dashboard
// stat cards. Each counter is fetched concurrently via its own connection from
// the pool to minimise wall-clock latency.
func (r *EventRepo) GetStatsOverview(ctx context.Context) (*StatsOverview, error) {
	var (
		stats StatsOverview
		wg    sync.WaitGroup
		mu    sync.Mutex
		errs  []error
	)

	collect := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}

	// 1. Total events in the last 24 hours.
	collect(func() error {
		return r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM events WHERE event_time > NOW() - INTERVAL '24 hours'`,
		).Scan(&stats.TotalEvents24h)
	})

	// 2. Total alerts in the last 24 hours.
	collect(func() error {
		return r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM alerts WHERE triggered_at > NOW() - INTERVAL '24 hours'`,
		).Scan(&stats.TotalAlerts24h)
	})

	// 3. Unacknowledged critical alerts (uses partial index idx_alerts_unack).
	collect(func() error {
		return r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM alerts WHERE severity = 'critical' AND acknowledged = false`,
		).Scan(&stats.CriticalAlerts)
	})

	// 4. Active log sources.
	collect(func() error {
		return r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM log_sources WHERE status = 'active'`,
		).Scan(&stats.ActiveSources)
	})

	wg.Wait()

	if len(errs) > 0 {
		return nil, fmt.Errorf("eventRepo.GetStatsOverview: %w", errs[0])
	}
	return &stats, nil
}

// ---------------------------------------------------------------------------
// GetEventsTimeline — hourly event counts for the last 24 h (Day 6)
// ---------------------------------------------------------------------------

// GetEventsTimeline returns event counts grouped by hour for the last 24
// hours, suitable for the dashboard line chart. Uses the date_trunc('hour',
// event_time) pattern from README §20.1 and leverages idx_events_event_time.
func (r *EventRepo) GetEventsTimeline(ctx context.Context) ([]TimelinePoint, error) {
	query := `
		SELECT date_trunc('hour', event_time) AS hour,
		       COUNT(*) AS count
		FROM events
		WHERE event_time > NOW() - INTERVAL '24 hours'
		GROUP BY 1
		ORDER BY 1`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("eventRepo.GetEventsTimeline: %w", err)
	}
	defer rows.Close()

	var points []TimelinePoint
	for rows.Next() {
		var p TimelinePoint
		if err := rows.Scan(&p.Hour, &p.Count); err != nil {
			return nil, fmt.Errorf("eventRepo.GetEventsTimeline scan: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("eventRepo.GetEventsTimeline rows: %w", err)
	}
	return points, nil
}

// ---------------------------------------------------------------------------
// GetTopSources — top 5 log sources by event count (Day 6)
// ---------------------------------------------------------------------------

// GetTopSources returns the 5 log sources with the highest event count,
// joining log_sources for the human-readable name.
func (r *EventRepo) GetTopSources(ctx context.Context) ([]TopSource, error) {
	query := `
		SELECT e.log_source_id,
		       COALESCE(ls.name, 'Unknown') AS source_name,
		       COUNT(*) AS count
		FROM events e
		LEFT JOIN log_sources ls ON ls.id = e.log_source_id
		GROUP BY e.log_source_id, ls.name
		ORDER BY count DESC
		LIMIT 5`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("eventRepo.GetTopSources: %w", err)
	}
	defer rows.Close()

	var sources []TopSource
	for rows.Next() {
		var s TopSource
		if err := rows.Scan(&s.SourceID, &s.SourceName, &s.Count); err != nil {
			return nil, fmt.Errorf("eventRepo.GetTopSources scan: %w", err)
		}
		sources = append(sources, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("eventRepo.GetTopSources rows: %w", err)
	}
	return sources, nil
}
