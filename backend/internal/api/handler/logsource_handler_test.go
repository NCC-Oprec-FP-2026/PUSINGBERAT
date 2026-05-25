package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type mockLogSourceService struct {
	createFunc         func(ctx context.Context, ls *domain.LogSource) error
	getByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.LogSource, error)
	listFunc           func(ctx context.Context) ([]domain.LogSource, error)
	updateFunc         func(ctx context.Context, ls *domain.LogSource) error
	updateMetadataFunc func(ctx context.Context, id uuid.UUID, name *string, description *string) (*domain.LogSource, error)
	deleteFunc         func(ctx context.Context, id uuid.UUID) error
}

func (m *mockLogSourceService) Create(ctx context.Context, ls *domain.LogSource) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, ls)
	}
	return nil
}

func (m *mockLogSourceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &domain.LogSource{}, nil
}

func (m *mockLogSourceService) List(ctx context.Context) ([]domain.LogSource, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []domain.LogSource{}, nil
}

func (m *mockLogSourceService) Update(ctx context.Context, ls *domain.LogSource) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, ls)
	}
	return nil
}

func (m *mockLogSourceService) UpdateMetadata(ctx context.Context, id uuid.UUID, name *string, description *string) (*domain.LogSource, error) {
	if m.updateMetadataFunc != nil {
		return m.updateMetadataFunc(ctx, id, name, description)
	}
	return &domain.LogSource{}, nil
}

func (m *mockLogSourceService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestLogSourceHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLogSourceHandler(&mockLogSourceService{})
	r := gin.New()
	r.POST("/sources", h.Create)

	req := httptest.NewRequest("POST", "/sources", bytes.NewBufferString(`{"name":"test", "file_path":"/tmp/test.log", "log_type":"generic"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestLogSourceHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLogSourceHandler(&mockLogSourceService{})
	r := gin.New()
	r.GET("/sources/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/sources/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLogSourceHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLogSourceHandler(&mockLogSourceService{})
	r := gin.New()
	r.GET("/sources", h.List)

	req := httptest.NewRequest("GET", "/sources", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLogSourceHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLogSourceHandler(&mockLogSourceService{})
	r := gin.New()
	r.PATCH("/sources/:id", h.Update)

	req := httptest.NewRequest("PATCH", "/sources/"+uuid.New().String(), bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLogSourceHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLogSourceHandler(&mockLogSourceService{})
	r := gin.New()
	r.DELETE("/sources/:id", h.Delete)

	req := httptest.NewRequest("DELETE", "/sources/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
