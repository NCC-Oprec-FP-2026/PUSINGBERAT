package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type EventService interface {
	GetByID(ctx context.Context, id int64) (*domain.Event, error)
	List(ctx context.Context, filter repository.EventFilter) ([]domain.Event, int64, error)
}

type EventHandler struct {
	service EventService
}

func NewEventHandler(service EventService) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/events", h.List)
	group.GET("/events/:id", h.GetByID)
}

func (h *EventHandler) List(c *gin.Context) {
	filter := repository.EventFilter{
		SourceID: c.Query("source_id"),
		Level:    c.Query("level"),
		From:     timeQuery(c, "from"),
		To:       timeQuery(c, "to"),
		Search:   c.Query("search"),
		Page:     intQuery(c, "page", 1),
		PerPage:  intQuery(c, "per_page", 50),
	}

	events, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": events,
		"meta": gin.H{
			"total":    total,
			"page":     filter.Page,
			"per_page": filter.PerPage,
		},
	})
}

func (h *EventHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "id must be an integer",
		})
		return
	}

	event, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": event,
	})
}
