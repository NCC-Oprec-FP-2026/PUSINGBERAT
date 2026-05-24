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
// The UpdateMetadata method is the PATCH-specific entry point that only
// touches metadata (name, description) without affecting file_path or the
// active watcher goroutine.
type LogSourceService interface {
	Create(ctx context.Context, ls *domain.LogSource) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error)
	List(ctx context.Context) ([]domain.LogSource, error)
	Update(ctx context.Context, ls *domain.LogSource) error
	UpdateMetadata(ctx context.Context, id uuid.UUID, name *string, description *string) (*domain.LogSource, error)
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

// updateLogSourceMetadataRequest is the JSON body for PATCH /sources/:id.
// Only metadata fields (name, description) are accepted. Changing file_path
// or log_type requires a delete + re-create to properly reconfigure the
// watcher pipeline.
type updateLogSourceMetadataRequest struct {
	Name        *string `json:"name,omitempty"`
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
//
// This endpoint only accepts metadata updates (name, description).
// It does NOT accept file_path, log_type, or status changes — those would
// require reconfiguring the watcher pipeline, which is a more complex
// operation. This design ensures that the active filesystem watcher
// goroutine for this source is never broken or restarted by a metadata
// update.
//
// If the caller sends fields other than name/description, they are silently
// ignored (following the "tolerant reader" pattern for PATCH semantics).
func (h *LogSourceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	var req updateLogSourceMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON body: "+err.Error())
		return
	}

	// Validate that at least one metadata field was provided.
	if req.Name == nil && req.Description == nil {
		respondBadRequest(c, "at least one metadata field (name, description) is required")
		return
	}

	updated, err := h.svc.UpdateMetadata(c.Request.Context(), id, req.Name, req.Description)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, updated)
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
