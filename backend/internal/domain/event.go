package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ParsedEvent represents a single parsed log line stored in the `events`
// table. The `Extra` field captures parser-specific key/value pairs that
// do not have their own column (stored as JSONB in PostgreSQL).
type ParsedEvent struct {
	ID          int64           `json:"id"`
	LogSourceID uuid.UUID       `json:"log_source_id"`
	RawLine     string          `json:"raw_line"`
	Message     *string         `json:"message,omitempty"`
	Hostname    *string         `json:"hostname,omitempty"`
	Process     *string         `json:"process,omitempty"`
	PID         *int            `json:"pid,omitempty"`
	LogLevel    *string         `json:"log_level,omitempty"`
	EventTime   time.Time       `json:"event_time"`
	ReceivedAt  time.Time       `json:"received_at"`
	Extra       json.RawMessage `json:"extra,omitempty"`
}
