package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type AlertService interface {
	GetByID(ctx context.Context, id string) (*domain.Alert, error)
	List(ctx context.Context, filter repository.AlertFilter) ([]domain.Alert, int64, error)
	Acknowledge(ctx context.Context, id string) (*domain.Alert, error)
	Delete(ctx context.Context, id string) error
}

type AlertHandler struct {
	service AlertService
}

func NewAlertHandler(service AlertService) *AlertHandler {
	return &AlertHandler{service: service}
}

func (h *AlertHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/alerts", h.List)
	group.GET("/alerts/:id", h.GetByID)
	group.PATCH("/alerts/:id/acknowledge", h.Acknowledge)
	group.DELETE("/alerts/:id", h.Delete)
}

func (h *AlertHandler) List(c *gin.Context) {
	filter := repository.AlertFilter{
		Severities:   severityList(c.Query("severity")),
		Acknowledged: boolQuery(c, "acknowledged"),
		From:         timeQuery(c, "from"),
		To:           timeQuery(c, "to"),
		Page:         intQuery(c, "page", 1),
		PerPage:      intQuery(c, "per_page", 50),
	}

	alerts, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alerts,
		"meta": gin.H{
			"total":    total,
			"page":     filter.Page,
			"per_page": filter.PerPage,
		},
	})
}

func (h *AlertHandler) GetByID(c *gin.Context) {
	alert, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alert,
	})
}

func (h *AlertHandler) Acknowledge(c *gin.Context) {
	alert, err := h.service.Acknowledge(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alert,
	})
}

func (h *AlertHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func severityList(raw string) []domain.Severity {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]domain.Severity, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, domain.Severity(trimmed))
		}
	}
	return out
}
