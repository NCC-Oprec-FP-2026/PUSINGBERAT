package domain

// ---------------------------------------------------------------------------
// SeverityLevel enum
// ---------------------------------------------------------------------------

// SeverityLevel mirrors the PostgreSQL `severity_level` enum.
// It is shared between the Rule and Alert domain types.
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// Valid returns true when the value is one of the known enum members.
func (s SeverityLevel) Valid() bool {
	switch s {
	case SeverityInfo, SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	}
	return false
}
