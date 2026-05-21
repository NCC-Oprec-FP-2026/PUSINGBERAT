package domain

import (
	"time"

	"github.com/google/uuid"
)

// Rule represents a YAML-based detection rule stored in the `rules` table.
// The raw YAML is persisted in YAMLContent so that users can edit it from
// the UI; the extracted metadata columns (name, severity, enabled) power
// efficient queries without parsing YAML on every request.
type Rule struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	YAMLContent string        `json:"yaml_content"`
	Severity    SeverityLevel `json:"severity"`
	Enabled     bool          `json:"enabled"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
