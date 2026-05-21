package domain

import "time"

type Alert struct {
	ID             string     `json:"id"`
	RuleID         *string    `json:"rule_id,omitempty"`
	RuleName       string     `json:"rule_name"`
	EventID        *int64     `json:"event_id,omitempty"`
	LogSourceID    *string    `json:"log_source_id,omitempty"`
	Severity       Severity   `json:"severity"`
	Title          string     `json:"title"`
	Description    *string    `json:"description,omitempty"`
	RawLine        *string    `json:"raw_line,omitempty"`
	TriggeredAt    time.Time  `json:"triggered_at"`
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	DiscordSent    bool       `json:"discord_sent"`
}
