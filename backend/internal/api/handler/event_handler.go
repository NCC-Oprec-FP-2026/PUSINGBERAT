package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

// ---------------------------------------------------------------------------
// Service interface (consumed by EventHandler)
// ---------------------------------------------------------------------------

// EventService defines the business-logic contract that the handler calls.
type EventService interface {
	GetByID(ctx context.Context, id int64) (*domain.ParsedEvent, error)
	List(ctx context.Context, params repository.EventListParams) ([]domain.ParsedEvent, int64, error)
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// EventHandler handles HTTP requests for the /events endpoints.
type EventHandler struct {
	svc EventService
}

// NewEventHandler constructs an EventHandler with the given service.
func NewEventHandler(svc EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

// GetByID handles GET /api/v1/events/:id.
func (h *EventHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid event ID: must be a number")
		return
	}

	ev, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, ev)
}

// List handles GET /api/v1/events with pagination query params.
func (h *EventHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 50)

	if perPage > 200 {
		perPage = 200
	}
	if page < 1 {
		page = 1
	}

	params := repository.EventListParams{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	events, total, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}

	respondList(c, events, total, page, perPage)
}

// queryInt is a small helper that reads an integer query parameter with a
// fallback default.
func queryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
