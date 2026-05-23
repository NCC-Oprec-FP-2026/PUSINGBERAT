package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Service interface (consumed by LogSourceHandler)
// ---------------------------------------------------------------------------

// LogSourceService defines the business-logic contract that the handler calls.
type LogSourceService interface {
	Create(ctx context.Context, ls *domain.LogSource) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error)
	List(ctx context.Context) ([]domain.LogSource, error)
	Update(ctx context.Context, ls *domain.LogSource) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// LogSourceHandler handles HTTP requests for the /sources endpoints.
type LogSourceHandler struct {
	svc LogSourceService
}

// NewLogSourceHandler constructs a LogSourceHandler with the given service.
func NewLogSourceHandler(svc LogSourceService) *LogSourceHandler {
	return &LogSourceHandler{svc: svc}
}

// createLogSourceRequest is the JSON body for POST /sources.
type createLogSourceRequest struct {
	Name        string  `json:"name"`
	FilePath    string  `json:"file_path"`
	LogType     string  `json:"log_type"`
	Description *string `json:"description,omitempty"`
}

// updateLogSourceRequest is the JSON body for PATCH /sources/:id.
type updateLogSourceRequest struct {
	Name        *string `json:"name,omitempty"`
	FilePath    *string `json:"file_path,omitempty"`
	LogType     *string `json:"log_type,omitempty"`
	Status      *string `json:"status,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Create handles POST /api/v1/sources.
func (h *LogSourceHandler) Create(c *gin.Context) {
	var req createLogSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON body: "+err.Error())
		return
	}

	ls := &domain.LogSource{
		Name:        req.Name,
		FilePath:    req.FilePath,
		LogType:     req.LogType,
		Description: req.Description,
	}

	if err := h.svc.Create(c.Request.Context(), ls); err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusCreated, ls)
}

// GetByID handles GET /api/v1/sources/:id.
func (h *LogSourceHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	ls, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, ls)
}

// List handles GET /api/v1/sources.
func (h *LogSourceHandler) List(c *gin.Context) {
	sources, err := h.svc.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	// LogSources are few enough that full pagination is unnecessary.
	respondList(c, sources, int64(len(sources)), 1, len(sources))
}

// Update handles PATCH /api/v1/sources/:id.
func (h *LogSourceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	// Fetch existing entity to apply partial updates.
	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	var req updateLogSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON body: "+err.Error())
		return
	}

	// Apply only the fields that the client sent.
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.FilePath != nil {
		existing.FilePath = *req.FilePath
	}
	if req.LogType != nil {
		existing.LogType = *req.LogType
	}
	if req.Status != nil {
		existing.Status = domain.LogSourceStatus(*req.Status)
	}
	if req.Description != nil {
		existing.Description = req.Description
	}

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/sources/:id.
func (h *LogSourceHandler) Delete(c *gin.Context) {
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
