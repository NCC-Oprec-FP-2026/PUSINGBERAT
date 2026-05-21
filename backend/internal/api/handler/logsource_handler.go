package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/service"
)

type LogSourceService interface {
	Create(ctx context.Context, source *domain.LogSource) error
	GetByID(ctx context.Context, id string) (*domain.LogSource, error)
	List(ctx context.Context, filter repository.LogSourceFilter) ([]domain.LogSource, int64, error)
	Update(ctx context.Context, source *domain.LogSource) error
	Delete(ctx context.Context, id string) error
}

type LogSourceHandler struct {
	service LogSourceService
}

func NewLogSourceHandler(service LogSourceService) *LogSourceHandler {
	return &LogSourceHandler{service: service}
}

func (h *LogSourceHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/sources", h.List)
	group.POST("/sources", h.Create)
	group.GET("/sources/:id", h.GetByID)
	group.PATCH("/sources/:id", h.Update)
	group.DELETE("/sources/:id", h.Delete)
}

type createLogSourceRequest struct {
	Name        string                 `json:"name"`
	FilePath    string                 `json:"file_path"`
	LogType     domain.LogSourceType   `json:"log_type"`
	Status      domain.LogSourceStatus `json:"status"`
	Description *string                `json:"description"`
}

func (h *LogSourceHandler) Create(c *gin.Context) {
	var req createLogSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid JSON body",
		})
		return
	}

	source := &domain.LogSource{
		Name:        req.Name,
		FilePath:    req.FilePath,
		LogType:     req.LogType,
		Status:      req.Status,
		Description: req.Description,
	}

	if err := h.service.Create(c.Request.Context(), source); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": source,
	})
}

func (h *LogSourceHandler) GetByID(c *gin.Context) {
	source, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": source,
	})
}

func (h *LogSourceHandler) List(c *gin.Context) {
	filter := repository.LogSourceFilter{
		Status:  domain.LogSourceStatus(c.Query("status")),
		LogType: domain.LogSourceType(c.Query("log_type")),
		Search:  c.Query("search"),
		Page:    intQuery(c, "page", 1),
		PerPage: intQuery(c, "per_page", 50),
	}

	sources, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": sources,
		"meta": gin.H{
			"total":    total,
			"page":     filter.Page,
			"per_page": filter.PerPage,
		},
	})
}

func (h *LogSourceHandler) Update(c *gin.Context) {
	var req createLogSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid JSON body",
		})
		return
	}

	source := &domain.LogSource{
		ID:          c.Param("id"),
		Name:        req.Name,
		FilePath:    req.FilePath,
		LogType:     req.LogType,
		Status:      req.Status,
		Description: req.Description,
	}

	if err := h.service.Update(c.Request.Context(), source); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": source,
	})
}

func (h *LogSourceHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func intQuery(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func writeServiceError(c *gin.Context, err error) {
	var pgErr *pgconn.PgError

	switch {
	case errors.Is(err, service.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "resource already exists",
		})
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "resource not found",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "internal server error",
		})
	}
}
