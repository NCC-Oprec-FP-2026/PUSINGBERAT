package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

// ---------------------------------------------------------------------------
// Service interface (consumed by AlertHandler)
// ---------------------------------------------------------------------------

// AlertService defines the business-logic contract that the handler calls.
type AlertService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	List(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error)
	Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// AlertHandler handles HTTP requests for the /alerts endpoints.
type AlertHandler struct {
	svc AlertService
}

// NewAlertHandler constructs an AlertHandler with the given service.
func NewAlertHandler(svc AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// GetByID handles GET /api/v1/alerts/:id.
func (h *AlertHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	alert, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, alert)
}

// List handles GET /api/v1/alerts with pagination query params.
func (h *AlertHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 50)

	if perPage > 200 {
		perPage = 200
	}
	if page < 1 {
		page = 1
	}

	params := repository.AlertListParams{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	alerts, total, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}

	respondList(c, alerts, total, page, perPage)
}

// Acknowledge handles PATCH /api/v1/alerts/:id/acknowledge.
func (h *AlertHandler) Acknowledge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	alert, err := h.svc.Acknowledge(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, alert)
}

// Delete handles DELETE /api/v1/alerts/:id.
func (h *AlertHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
