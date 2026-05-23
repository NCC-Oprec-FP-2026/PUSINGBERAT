// Package domain defines the core data structures and enumerations used
// across every layer of the PUSINGBERAT SIEM backend. Domain types are pure
// value objects — they carry no behaviour beyond serialisation hints (JSON
// tags) and are intentionally free of any external dependency.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// LogSourceStatus enum
// ---------------------------------------------------------------------------

// LogSourceStatus mirrors the PostgreSQL `log_source_status` enum.
type LogSourceStatus string

const (
	LogSourceStatusActive   LogSourceStatus = "active"
	LogSourceStatusInactive LogSourceStatus = "inactive"
	LogSourceStatusError    LogSourceStatus = "error"
)

// Valid returns true when the value is one of the known enum members.
func (s LogSourceStatus) Valid() bool {
	switch s {
	case LogSourceStatusActive, LogSourceStatusInactive, LogSourceStatusError:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// LogSource struct
// ---------------------------------------------------------------------------

// LogSource represents a registered log file that the system watches for new
// lines.  It maps 1-to-1 with the `log_sources` PostgreSQL table.
type LogSource struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	FilePath    string          `json:"file_path"`
	LogType     string          `json:"log_type"`
	Status      LogSourceStatus `json:"status"`
	Description *string         `json:"description,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
