package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type RuleService interface {
	Create(ctx context.Context, rule *domain.Rule) error
	GetByID(ctx context.Context, id string) (*domain.Rule, error)
	List(ctx context.Context, filter repository.RuleFilter) ([]domain.Rule, int64, error)
	Update(ctx context.Context, rule *domain.Rule) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	Delete(ctx context.Context, id string) error
}

type RuleHandler struct {
	service RuleService
}

func NewRuleHandler(service RuleService) *RuleHandler {
	return &RuleHandler{service: service}
}

func (h *RuleHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/rules", h.List)
	group.POST("/rules", h.Create)
	group.GET("/rules/:id", h.GetByID)
	group.PUT("/rules/:id", h.Update)
	group.PATCH("/rules/:id/toggle", h.Toggle)
	group.DELETE("/rules/:id", h.Delete)
}

type ruleRequest struct {
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	YAMLContent string          `json:"yaml_content"`
	Severity    domain.Severity `json:"severity"`
	Enabled     *bool           `json:"enabled"`
}

type toggleRuleRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *RuleHandler) Create(c *gin.Context) {
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid JSON body",
		})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &domain.Rule{
		Name:        req.Name,
		Description: req.Description,
		YAMLContent: req.YAMLContent,
		Severity:    req.Severity,
		Enabled:     enabled,
	}

	if err := h.service.Create(c.Request.Context(), rule); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": rule,
	})
}

func (h *RuleHandler) List(c *gin.Context) {
	filter := repository.RuleFilter{
		Enabled:  boolQuery(c, "enabled"),
		Severity: domain.Severity(c.Query("severity")),
		Search:   c.Query("search"),
		Page:     intQuery(c, "page", 1),
		PerPage:  intQuery(c, "per_page", 50),
	}

	rules, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rules,
		"meta": gin.H{
			"total":    total,
			"page":     filter.Page,
			"per_page": filter.PerPage,
		},
	})
}

func (h *RuleHandler) GetByID(c *gin.Context) {
	rule, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rule,
	})
}

func (h *RuleHandler) Update(c *gin.Context) {
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid JSON body",
		})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &domain.Rule{
		ID:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
		YAMLContent: req.YAMLContent,
		Severity:    req.Severity,
		Enabled:     enabled,
	}

	if err := h.service.Update(c.Request.Context(), rule); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rule,
	})
}

func (h *RuleHandler) Toggle(c *gin.Context) {
	var req toggleRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid JSON body",
		})
		return
	}

	if err := h.service.SetEnabled(c.Request.Context(), c.Param("id"), req.Enabled); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":      c.Param("id"),
			"enabled": req.Enabled,
		},
	})
}

func (h *RuleHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
