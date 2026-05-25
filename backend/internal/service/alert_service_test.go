package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock AlertRepository
// ---------------------------------------------------------------------------

type mockAlertRepo struct {
	createFn           func(ctx context.Context, a *domain.Alert) error
	getByIDFn          func(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	listFn             func(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error)
	acknowledgeFn      func(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	deleteFn           func(ctx context.Context, id uuid.UUID) error
	markDiscordSentFn  func(ctx context.Context, id uuid.UUID) error
	getBySeverityFn    func(ctx context.Context) (map[string]int64, error)
}

func (m *mockAlertRepo) Create(ctx context.Context, a *domain.Alert) error {
	if m.createFn != nil {
		return m.createFn(ctx, a)
	}
	return nil
}
func (m *mockAlertRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.Alert{ID: id, Title: "test alert"}, nil
}
func (m *mockAlertRepo) List(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return []domain.Alert{}, 0, nil
}
func (m *mockAlertRepo) Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if m.acknowledgeFn != nil {
		return m.acknowledgeFn(ctx, id)
	}
	return &domain.Alert{ID: id, Acknowledged: true}, nil
}
func (m *mockAlertRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockAlertRepo) MarkDiscordSent(ctx context.Context, id uuid.UUID) error {
	if m.markDiscordSentFn != nil {
		return m.markDiscordSentFn(ctx, id)
	}
	return nil
}
func (m *mockAlertRepo) GetAlertsBySeverity(ctx context.Context) (map[string]int64, error) {
	if m.getBySeverityFn != nil {
		return m.getBySeverityFn(ctx)
	}
	return map[string]int64{"info": 0, "low": 0, "medium": 0, "high": 0, "critical": 0}, nil
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAlertService_Create_Happy(t *testing.T) {
	svc := NewAlertService(&mockAlertRepo{})
	a := &domain.Alert{Title: "test", Severity: domain.SeverityHigh}
	if err := svc.Create(context.Background(), a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAlertService_Create_RepoError(t *testing.T) {
	repoErr := errors.New("db error")
	svc := NewAlertService(&mockAlertRepo{
		createFn: func(_ context.Context, _ *domain.Alert) error { return repoErr },
	})
	err := svc.Create(context.Background(), &domain.Alert{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestAlertService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	svc := NewAlertService(&mockAlertRepo{
		getByIDFn: func(_ context.Context, got uuid.UUID) (*domain.Alert, error) {
			if got != id {
				return nil, domain.ErrNotFound
			}
			return &domain.Alert{ID: id, Title: "found"}, nil
		},
	})
	a, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != id {
		t.Errorf("expected id %v, got %v", id, a.ID)
	}
}

func TestAlertService_GetByID_NotFound(t *testing.T) {
	svc := NewAlertService(&mockAlertRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Alert, error) {
			return nil, domain.ErrNotFound
		},
	})
	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestAlertService_List_ReturnsPaginated(t *testing.T) {
	expected := []domain.Alert{{Title: "a"}, {Title: "b"}}
	svc := NewAlertService(&mockAlertRepo{
		listFn: func(_ context.Context, _ repository.AlertListParams) ([]domain.Alert, int64, error) {
			return expected, int64(len(expected)), nil
		},
	})
	got, total, err := svc.List(context.Background(), repository.AlertListParams{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Errorf("expected %d alerts, got %d", len(expected), len(got))
	}
	if total != int64(len(expected)) {
		t.Errorf("expected total %d, got %d", len(expected), total)
	}
}

func TestAlertService_List_RepoError(t *testing.T) {
	svc := NewAlertService(&mockAlertRepo{
		listFn: func(_ context.Context, _ repository.AlertListParams) ([]domain.Alert, int64, error) {
			return nil, 0, errors.New("db error")
		},
	})
	_, _, err := svc.List(context.Background(), repository.AlertListParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Acknowledge
// ---------------------------------------------------------------------------

func TestAlertService_Acknowledge_Happy(t *testing.T) {
	id := uuid.New()
	svc := NewAlertService(&mockAlertRepo{
		acknowledgeFn: func(_ context.Context, got uuid.UUID) (*domain.Alert, error) {
			return &domain.Alert{ID: got, Acknowledged: true}, nil
		},
	})
	a, err := svc.Acknowledge(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.Acknowledged {
		t.Error("expected Acknowledged to be true")
	}
}

func TestAlertService_Acknowledge_NotFound(t *testing.T) {
	svc := NewAlertService(&mockAlertRepo{
		acknowledgeFn: func(_ context.Context, _ uuid.UUID) (*domain.Alert, error) {
			return nil, domain.ErrNotFound
		},
	})
	_, err := svc.Acknowledge(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAlertService_Delete_Happy(t *testing.T) {
	svc := NewAlertService(&mockAlertRepo{})
	if err := svc.Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAlertService_Delete_NotFound(t *testing.T) {
	svc := NewAlertService(&mockAlertRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return domain.ErrNotFound },
	})
	err := svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetAlertsBySeverity
// ---------------------------------------------------------------------------

func TestAlertService_GetAlertsBySeverity_ReturnsMap(t *testing.T) {
	expected := map[string]int64{"info": 1, "low": 2, "medium": 3, "high": 4, "critical": 5}
	svc := NewAlertService(&mockAlertRepo{
		getBySeverityFn: func(_ context.Context) (map[string]int64, error) {
			return expected, nil
		},
	})
	got, err := svc.GetAlertsBySeverity(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for k, v := range expected {
		if got[k] != v {
			t.Errorf("expected %s=%d, got %d", k, v, got[k])
		}
	}
}

func TestAlertService_GetAlertsBySeverity_RepoError(t *testing.T) {
	svc := NewAlertService(&mockAlertRepo{
		getBySeverityFn: func(_ context.Context) (map[string]int64, error) {
			return nil, errors.New("db error")
		},
	})
	_, err := svc.GetAlertsBySeverity(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
