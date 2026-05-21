package domain

import "time"

type Event struct {
	ID          int64          `json:"id"`
	LogSourceID string         `json:"log_source_id"`
	RawLine     string         `json:"raw_line"`
	Message     *string        `json:"message,omitempty"`
	Hostname    *string        `json:"hostname,omitempty"`
	Process     *string        `json:"process,omitempty"`
	PID         *int32         `json:"pid,omitempty"`
	LogLevel    *string        `json:"log_level,omitempty"`
	EventTime   time.Time      `json:"event_time"`
	ReceivedAt  time.Time      `json:"received_at"`
	Extra       map[string]any `json:"extra"`
}
