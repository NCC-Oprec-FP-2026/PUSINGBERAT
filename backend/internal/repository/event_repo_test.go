package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func strPtr(s string) *string { return &s }
func timePtr(t time.Time) *time.Time { return &t }

func TestBuildFilterClause(t *testing.T) {
	id := uuid.New()
	t1 := time.Now()
	t2 := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name       string
		params     EventFilterParams
		paramIdx   int
		wantClause string
		wantArgs   int
	}{
		{
			name:       "empty params",
			params:     EventFilterParams{},
			paramIdx:   1,
			wantClause: "",
			wantArgs:   0,
		},
		{
			name: "all params",
			params: EventFilterParams{
				SourceID: &id,
				Level:    strPtr("error"),
				From:     timePtr(t1),
				To:       timePtr(t2),
				Search:   strPtr("fail"),
			},
			paramIdx:   1,
			wantClause: " WHERE log_source_id = $1 AND log_level = $2 AND event_time >= $3 AND event_time <= $4 AND message ILIKE $5",
			wantArgs:   5,
		},
		{
			name: "only search with different starting index",
			params: EventFilterParams{
				Search: strPtr("test"),
			},
			paramIdx:   3,
			wantClause: " WHERE message ILIKE $3",
			wantArgs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := buildFilterClause(tt.params, tt.paramIdx)
			if clause != tt.wantClause {
				t.Errorf("got clause %q, want %q", clause, tt.wantClause)
			}
			if len(args) != tt.wantArgs {
				t.Errorf("got %d args, want %d", len(args), tt.wantArgs)
			}
		})
	}
}

func TestEventRepo_Create(t *testing.T) {
	
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewEventRepo(mock)

	id := uuid.New()
	ev := &domain.ParsedEvent{
		LogSourceID: id,
		Hostname:    strPtr("web01"),
	}

	mock.ExpectQuery(`INSERT INTO events`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "received_at"}).AddRow(int64(1), time.Now()))

	err = repo.Create(context.Background(), ev)
	if err != nil {
		t.Errorf("error was not expected while inserting: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestEventRepo_GetTopSources(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewEventRepo(mock)

	mock.ExpectQuery(`SELECT e\.log_source_id, COALESCE\(ls\.name, 'Unknown'\) AS source_name, COUNT\(\*\) AS count FROM events e LEFT JOIN log_sources ls ON ls\.id = e\.log_source_id GROUP BY e\.log_source_id, ls\.name ORDER BY count DESC LIMIT 5`).
		WillReturnRows(pgxmock.NewRows([]string{"log_source_id", "source_name", "count"}).AddRow(uuid.New(), "syslog", int64(10)))

	_, err = repo.GetTopSources(context.Background())
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

type mockDBQuery struct {
	QueryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	QueryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	ExecFunc     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (m *mockDBQuery) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return nil
}

func (m *mockDBQuery) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, args...)
	}
	return nil, nil
}

func (m *mockDBQuery) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

type mockRow struct {
	val int64
}

func (m *mockRow) Scan(dest ...any) error {
	*dest[0].(*int64) = m.val
	return nil
}

func TestEventRepo_GetStatsOverview(t *testing.T) {
	repo := NewEventRepo(&mockDBQuery{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			if strings.Contains(sql, "events WHERE event_time") {
				return &mockRow{val: 100}
			}
			if strings.Contains(sql, "alerts WHERE triggered_at") && !strings.Contains(sql, "severity") {
				return &mockRow{val: 5}
			}
			if strings.Contains(sql, "severity") {
				return &mockRow{val: 2}
			}
			if strings.Contains(sql, "log_sources") {
				return &mockRow{val: 3}
			}
			return &mockRow{val: 0}
		},
	})

	stats, err := repo.GetStatsOverview(context.Background())
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	
	if stats != nil && stats.TotalEvents24h != 100 {
		t.Errorf("expected 100, got %d", stats.TotalEvents24h)
	}

	}


