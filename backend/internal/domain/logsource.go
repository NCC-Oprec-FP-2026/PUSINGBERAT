package domain

import "time"

type LogSourceStatus string

const (
	LogSourceStatusActive   LogSourceStatus = "active"
	LogSourceStatusInactive LogSourceStatus = "inactive"
	LogSourceStatusError    LogSourceStatus = "error"
)

type LogSourceType string

const (
	LogSourceTypeGeneric LogSourceType = "generic"
	LogSourceTypeSyslog  LogSourceType = "syslog"
	LogSourceTypeNginx   LogSourceType = "nginx"
)

type LogSource struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	FilePath    string          `json:"file_path"`
	LogType     LogSourceType   `json:"log_type"`
	Status      LogSourceStatus `json:"status"`
	Description *string         `json:"description,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
