package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type mockAlertService struct {
	getByIDFunc           func(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	listFunc              func(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error)
	acknowledgeFunc       func(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	getSeverityCountsFunc func(ctx context.Context) (map[string]int64, error)
	deleteFunc            func(ctx context.Context, id uuid.UUID) error
}

func (m *mockAlertService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &domain.Alert{}, nil
}

func (m *mockAlertService) List(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, params)
	}
	return []domain.Alert{}, 0, nil
}

func (m *mockAlertService) Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if m.acknowledgeFunc != nil {
		return m.acknowledgeFunc(ctx, id)
	}
	return &domain.Alert{}, nil
}

func (m *mockAlertService) GetSeverityCounts(ctx context.Context) (map[string]int64, error) {
	if m.getSeverityCountsFunc != nil {
		return m.getSeverityCountsFunc(ctx)
	}
	return map[string]int64{}, nil
}

func (m *mockAlertService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestAlertHandler_GetSeverityCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAlertHandler(&mockAlertService{})
	r := gin.New()
	r.GET("/alerts/severitycount", h.GetSeverityCounts)

	req := httptest.NewRequest("GET", "/alerts/severitycount", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAlertHandler(&mockAlertService{})
	r := gin.New()
	r.GET("/alerts/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/alerts/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAlertHandler(&mockAlertService{})
	r := gin.New()
	r.GET("/alerts", h.List)

	req := httptest.NewRequest("GET", "/alerts?page=1&per_page=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Acknowledge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAlertHandler(&mockAlertService{})
	r := gin.New()
	r.PATCH("/alerts/:id/acknowledge", h.Acknowledge)

	req := httptest.NewRequest("PATCH", "/alerts/"+uuid.New().String()+"/acknowledge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAlertHandler(&mockAlertService{})
	r := gin.New()
	r.DELETE("/alerts/:id", h.Delete)

	req := httptest.NewRequest("DELETE", "/alerts/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
