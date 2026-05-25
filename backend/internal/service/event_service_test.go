package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock EventRepository
// ---------------------------------------------------------------------------

type mockEventRepo struct {
	createFn           func(ctx context.Context, ev *domain.ParsedEvent) error
	getByIDFn          func(ctx context.Context, id int64) (*domain.ParsedEvent, error)
	listEventsFn       func(ctx context.Context, params repository.EventFilterParams) ([]domain.ParsedEvent, int64, error)
	getStatsOverviewFn func(ctx context.Context) (*repository.StatsOverview, error)
	getTimelineFn      func(ctx context.Context) ([]repository.TimelinePoint, error)
	getTopSourcesFn    func(ctx context.Context) ([]repository.TopSource, error)
}

func (m *mockEventRepo) Create(ctx context.Context, ev *domain.ParsedEvent) error {
	if m.createFn != nil {
		return m.createFn(ctx, ev)
	}
	return nil
}
func (m *mockEventRepo) GetByID(ctx context.Context, id int64) (*domain.ParsedEvent, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.ParsedEvent{ID: id}, nil
}
func (m *mockEventRepo) ListEvents(ctx context.Context, params repository.EventFilterParams) ([]domain.ParsedEvent, int64, error) {
	if m.listEventsFn != nil {
		return m.listEventsFn(ctx, params)
	}
	return []domain.ParsedEvent{}, 0, nil
}
func (m *mockEventRepo) GetStatsOverview(ctx context.Context) (*repository.StatsOverview, error) {
	if m.getStatsOverviewFn != nil {
		return m.getStatsOverviewFn(ctx)
	}
	return &repository.StatsOverview{}, nil
}
func (m *mockEventRepo) GetEventsTimeline(ctx context.Context) ([]repository.TimelinePoint, error) {
	if m.getTimelineFn != nil {
		return m.getTimelineFn(ctx)
	}
	return []repository.TimelinePoint{}, nil
}
func (m *mockEventRepo) GetTopSources(ctx context.Context) ([]repository.TopSource, error) {
	if m.getTopSourcesFn != nil {
		return m.getTopSourcesFn(ctx)
	}
	return []repository.TopSource{}, nil
}


// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestEventService_Create_Happy(t *testing.T) {
	svc := NewEventService(&mockEventRepo{})
	ev := &domain.ParsedEvent{LogSourceID: uuid.New()}
	if err := svc.Create(context.Background(), ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventService_Create_RepoError(t *testing.T) {
	repoErr := errors.New("db error")
	svc := NewEventService(&mockEventRepo{
		createFn: func(_ context.Context, _ *domain.ParsedEvent) error { return repoErr },
	})
	err := svc.Create(context.Background(), &domain.ParsedEvent{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestEventService_GetByID_Found(t *testing.T) {
	svc := NewEventService(&mockEventRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.ParsedEvent, error) {
			return &domain.ParsedEvent{ID: id}, nil
		},
	})
	ev, err := svc.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.ID != 42 {
		t.Errorf("expected ID 42, got %d", ev.ID)
	}
}

func TestEventService_GetByID_NotFound(t *testing.T) {
	svc := NewEventService(&mockEventRepo{
		getByIDFn: func(_ context.Context, _ int64) (*domain.ParsedEvent, error) {
			return nil, domain.ErrNotFound
		},
	})
	_, err := svc.GetByID(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListEvents
// ---------------------------------------------------------------------------

func TestEventService_ListEvents_ReturnsEvents(t *testing.T) {
	expected := []domain.ParsedEvent{{ID: 1}, {ID: 2}}
	svc := NewEventService(&mockEventRepo{
		listEventsFn: func(_ context.Context, _ repository.EventFilterParams) ([]domain.ParsedEvent, int64, error) {
			return expected, 2, nil
		},
	})
	got, total, err := svc.ListEvents(context.Background(), repository.EventFilterParams{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 events, got %d", len(got))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestEventService_ListEvents_RepoError(t *testing.T) {
	svc := NewEventService(&mockEventRepo{
		listEventsFn: func(_ context.Context, _ repository.EventFilterParams) ([]domain.ParsedEvent, int64, error) {
			return nil, 0, errors.New("db error")
		},
	})
	_, _, err := svc.ListEvents(context.Background(), repository.EventFilterParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetStatsOverview
// ---------------------------------------------------------------------------

func TestEventService_GetStatsOverview_Happy(t *testing.T) {
	expected := &repository.StatsOverview{TotalEvents24h: 100, TotalAlerts24h: 5}
	svc := NewEventService(&mockEventRepo{
		getStatsOverviewFn: func(_ context.Context) (*repository.StatsOverview, error) {
			return expected, nil
		},
	})
	got, err := svc.GetStatsOverview(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalEvents24h != expected.TotalEvents24h {
		t.Errorf("expected TotalEvents24h %d, got %d", expected.TotalEvents24h, got.TotalEvents24h)
	}
}

func TestEventService_GetStatsOverview_RepoError(t *testing.T) {
	svc := NewEventService(&mockEventRepo{
		getStatsOverviewFn: func(_ context.Context) (*repository.StatsOverview, error) {
			return nil, errors.New("db error")
		},
	})
	_, err := svc.GetStatsOverview(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetEventsTimeline
// ---------------------------------------------------------------------------

func TestEventService_GetEventsTimeline_Happy(t *testing.T) {
	expected := []repository.TimelinePoint{{Count: 10}, {Count: 20}}
	svc := NewEventService(&mockEventRepo{
		getTimelineFn: func(_ context.Context) ([]repository.TimelinePoint, error) {
			return expected, nil
		},
	})
	got, err := svc.GetEventsTimeline(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 timeline points, got %d", len(got))
	}
}

func TestEventService_GetEventsTimeline_RepoError(t *testing.T) {
	svc := NewEventService(&mockEventRepo{
		getTimelineFn: func(_ context.Context) ([]repository.TimelinePoint, error) {
			return nil, errors.New("db error")
		},
	})
	_, err := svc.GetEventsTimeline(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetTopSources
// ---------------------------------------------------------------------------

func TestEventService_GetTopSources_Happy(t *testing.T) {
	expected := []repository.TopSource{{SourceName: "nginx", Count: 42}}
	svc := NewEventService(&mockEventRepo{
		getTopSourcesFn: func(_ context.Context) ([]repository.TopSource, error) {
			return expected, nil
		},
	})
	got, err := svc.GetTopSources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].SourceName != "nginx" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestEventService_GetTopSources_RepoError(t *testing.T) {
	svc := NewEventService(&mockEventRepo{
		getTopSourcesFn: func(_ context.Context) ([]repository.TopSource, error) {
			return nil, errors.New("db error")
		},
	})
	_, err := svc.GetTopSources(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEventService_StartPersistenceWorker(t *testing.T) {
	svc := NewEventService(&mockEventRepo{
		createFn: func(ctx context.Context, ev *domain.ParsedEvent) error { return nil },
	})
	eventChan := make(chan *domain.ParsedEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc.StartPersistenceWorker(ctx, eventChan, nil, func(u uuid.UUID) string { return "syslog" })
	
	eventChan <- &domain.ParsedEvent{}
	close(eventChan)
	// let it process
	time.Sleep(100 * time.Millisecond)
}
