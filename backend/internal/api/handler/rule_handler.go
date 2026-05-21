package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Service interface (consumed by RuleHandler)
// ---------------------------------------------------------------------------

// RuleService defines the business-logic contract that the handler calls.
type RuleService interface {
	Create(ctx context.Context, rule *domain.Rule) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	List(ctx context.Context) ([]domain.Rule, error)
	Update(ctx context.Context, rule *domain.Rule) error
	Toggle(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// RuleHandler handles HTTP requests for the /rules endpoints.
type RuleHandler struct {
	svc RuleService
}

// NewRuleHandler constructs a RuleHandler with the given service.
func NewRuleHandler(svc RuleService) *RuleHandler {
	return &RuleHandler{svc: svc}
}

// createRuleRequest is the JSON body for POST /rules.
type createRuleRequest struct {
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	YAMLContent string               `json:"yaml_content"`
	Severity    domain.SeverityLevel `json:"severity"`
	Enabled     *bool                `json:"enabled,omitempty"`
}

// updateRuleRequest is the JSON body for PUT /rules/:id.
type updateRuleRequest struct {
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	YAMLContent string               `json:"yaml_content"`
	Severity    domain.SeverityLevel `json:"severity"`
	Enabled     *bool                `json:"enabled,omitempty"`
}

// Create handles POST /api/v1/rules.
func (h *RuleHandler) Create(c *gin.Context) {
	var req createRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON body: "+err.Error())
		return
	}

	rule := &domain.Rule{
		Name:        req.Name,
		Description: req.Description,
		YAMLContent: req.YAMLContent,
		Severity:    req.Severity,
		Enabled:     true, // default
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusCreated, rule)
}

// GetByID handles GET /api/v1/rules/:id.
func (h *RuleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, rule)
}

// List handles GET /api/v1/rules.
func (h *RuleHandler) List(c *gin.Context) {
	rules, err := h.svc.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	respondList(c, rules, int64(len(rules)), 1, len(rules))
}

// Update handles PUT /api/v1/rules/:id (full replacement).
func (h *RuleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	var req updateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON body: "+err.Error())
		return
	}

	rule := &domain.Rule{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		YAMLContent: req.YAMLContent,
		Severity:    req.Severity,
		Enabled:     true, // default
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := h.svc.Update(c.Request.Context(), rule); err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, rule)
}

// Toggle handles PATCH /api/v1/rules/:id/toggle.
func (h *RuleHandler) Toggle(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "invalid UUID")
		return
	}

	rule, err := h.svc.Toggle(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respondData(c, http.StatusOK, rule)
}

// Delete handles DELETE /api/v1/rules/:id.
func (h *RuleHandler) Delete(c *gin.Context) {
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
