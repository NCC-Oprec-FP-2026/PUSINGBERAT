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

type mockRuleService struct {
	createFunc  func(ctx context.Context, rule *domain.Rule) error
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	listFunc    func(ctx context.Context) ([]domain.Rule, error)
	updateFunc  func(ctx context.Context, rule *domain.Rule) error
	toggleFunc  func(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	deleteFunc  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockRuleService) Create(ctx context.Context, rule *domain.Rule) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, rule)
	}
	return nil
}

func (m *mockRuleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &domain.Rule{}, nil
}

func (m *mockRuleService) List(ctx context.Context) ([]domain.Rule, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []domain.Rule{}, nil
}

func (m *mockRuleService) Update(ctx context.Context, rule *domain.Rule) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, rule)
	}
	return nil
}

func (m *mockRuleService) Toggle(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	if m.toggleFunc != nil {
		return m.toggleFunc(ctx, id)
	}
	return &domain.Rule{}, nil
}

func (m *mockRuleService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestRuleHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuleHandler(&mockRuleService{})
	r := gin.New()
	r.POST("/rules", h.Create)

	req := httptest.NewRequest("POST", "/rules", bytes.NewBufferString(`{"name":"test", "yaml_content":"name: test\n", "severity":"high"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestRuleHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuleHandler(&mockRuleService{})
	r := gin.New()
	r.GET("/rules/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/rules/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRuleHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuleHandler(&mockRuleService{})
	r := gin.New()
	r.GET("/rules", h.List)

	req := httptest.NewRequest("GET", "/rules", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRuleHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuleHandler(&mockRuleService{})
	r := gin.New()
	r.PUT("/rules/:id", h.Update)

	req := httptest.NewRequest("PUT", "/rules/"+uuid.New().String(), bytes.NewBufferString(`{"name":"updated", "yaml_content":"name: updated\n", "severity":"high"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRuleHandler_Toggle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuleHandler(&mockRuleService{})
	r := gin.New()
	r.PATCH("/rules/:id/toggle", h.Toggle)

	req := httptest.NewRequest("PATCH", "/rules/"+uuid.New().String()+"/toggle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRuleHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRuleHandler(&mockRuleService{})
	r := gin.New()
	r.DELETE("/rules/:id", h.Delete)

	req := httptest.NewRequest("DELETE", "/rules/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
