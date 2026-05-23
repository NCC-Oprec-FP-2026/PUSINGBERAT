// Package handler implements the HTTP layer for the PUSINGBERAT SIEM API.
// Handlers parse requests, call the service layer, and serialise responses
// using the standard envelope format defined in the architecture document.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Standard JSON response envelopes (§10.2 of the architecture doc)
// ---------------------------------------------------------------------------

// SuccessResponse wraps a successful response with optional pagination meta.
type SuccessResponse struct {
	Data any             `json:"data"`
	Meta *PaginationMeta `json:"meta,omitempty"`
}

// PaginationMeta is included in list responses for pagination UI.
type PaginationMeta struct {
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ---------------------------------------------------------------------------
// Response helpers — keep handler methods short and consistent
// ---------------------------------------------------------------------------

// respondData writes a success JSON response with the given status code.
func respondData(c *gin.Context, status int, data any) {
	c.JSON(status, SuccessResponse{Data: data})
}

// respondList writes a paginated list JSON response.
func respondList(c *gin.Context, data any, total int64, page, perPage int) {
	c.JSON(http.StatusOK, SuccessResponse{
		Data: data,
		Meta: &PaginationMeta{
			Total:   total,
			Page:    page,
			PerPage: perPage,
		},
	})
}

// respondError maps domain sentinel errors to HTTP status codes and writes
// a structured JSON error response.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
		})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: err.Error(),
		})
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "an unexpected error occurred",
		})
	}
}

// respondBadRequest writes a 400 response with a custom message.
func respondBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   "bad_request",
		Message: message,
	})
}
