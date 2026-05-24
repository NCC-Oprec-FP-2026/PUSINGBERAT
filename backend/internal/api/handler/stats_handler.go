package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

// ---------------------------------------------------------------------------
// Service interfaces (consumed by StatsHandler)
// ---------------------------------------------------------------------------

// StatsEventService defines the EventService methods required by StatsHandler.
type StatsEventService interface {
	GetStatsOverview(ctx context.Context) (*repository.StatsOverview, error)
	GetEventsTimeline(ctx context.Context) ([]repository.TimelinePoint, error)
	GetTopSources(ctx context.Context) ([]repository.TopSource, error)
}

// StatsAlertService defines the AlertService methods required by StatsHandler.
type StatsAlertService interface {
	GetAlertsBySeverity(ctx context.Context) (map[string]int64, error)
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// StatsHandler handles HTTP requests for the /stats endpoints.
type StatsHandler struct {
	eventSvc StatsEventService
	alertSvc StatsAlertService
}

// NewStatsHandler constructs a StatsHandler with the given services.
func NewStatsHandler(eventSvc StatsEventService, alertSvc StatsAlertService) *StatsHandler {
	return &StatsHandler{
		eventSvc: eventSvc,
		alertSvc: alertSvc,
	}
}

// Overview handles GET /api/v1/stats/overview.
func (h *StatsHandler) Overview(c *gin.Context) {
	stats, err := h.eventSvc.GetStatsOverview(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, stats)
}

// EventsTimeline handles GET /api/v1/stats/events/timeline.
func (h *StatsHandler) EventsTimeline(c *gin.Context) {
	timeline, err := h.eventSvc.GetEventsTimeline(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, timeline)
}

// AlertsBySeverity handles GET /api/v1/stats/alerts/by-severity.
func (h *StatsHandler) AlertsBySeverity(c *gin.Context) {
	severity, err := h.alertSvc.GetAlertsBySeverity(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, severity)
}

// TopSources handles GET /api/v1/stats/top-sources.
func (h *StatsHandler) TopSources(c *gin.Context) {
	sources, err := h.eventSvc.GetTopSources(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, sources)
}
