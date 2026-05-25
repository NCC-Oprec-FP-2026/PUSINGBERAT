package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type mockEventService struct {
	getStatsOverview  func(ctx context.Context) (*repository.StatsOverview, error)
	getEventsTimeline func(ctx context.Context) ([]repository.TimelinePoint, error)
	getTopSources     func(ctx context.Context) ([]repository.TopSource, error)
}

func (m *mockEventService) GetStatsOverview(ctx context.Context) (*repository.StatsOverview, error) {
	if m.getStatsOverview != nil {
		return m.getStatsOverview(ctx)
	}
	return &repository.StatsOverview{}, nil
}

func (m *mockEventService) GetEventsTimeline(ctx context.Context) ([]repository.TimelinePoint, error) {
	if m.getEventsTimeline != nil {
		return m.getEventsTimeline(ctx)
	}
	return []repository.TimelinePoint{}, nil
}

func (m *mockEventService) GetTopSources(ctx context.Context) ([]repository.TopSource, error) {
	if m.getTopSources != nil {
		return m.getTopSources(ctx)
	}
	return []repository.TopSource{}, nil
}

type mockStatsAlertService struct {
	getAlertsBySeverity func(ctx context.Context) (map[string]int64, error)
}

func (m *mockStatsAlertService) GetAlertsBySeverity(ctx context.Context) (map[string]int64, error) {
	if m.getAlertsBySeverity != nil {
		return m.getAlertsBySeverity(ctx)
	}
	return map[string]int64{}, nil
}

func TestStatsHandler_Overview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewStatsHandler(&mockEventService{}, &mockStatsAlertService{})
	r := gin.New()
	r.GET("/overview", h.Overview)

	req := httptest.NewRequest("GET", "/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStatsHandler_EventsTimeline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewStatsHandler(&mockEventService{}, &mockStatsAlertService{})
	r := gin.New()
	r.GET("/timeline", h.EventsTimeline)

	req := httptest.NewRequest("GET", "/timeline", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStatsHandler_TopSources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewStatsHandler(&mockEventService{}, &mockStatsAlertService{})
	r := gin.New()
	r.GET("/sources", h.TopSources)

	req := httptest.NewRequest("GET", "/sources", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStatsHandler_AlertsBySeverity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewStatsHandler(&mockEventService{}, &mockStatsAlertService{})
	r := gin.New()
	r.GET("/alerts", h.AlertsBySeverity)

	req := httptest.NewRequest("GET", "/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
