// Package parser provides log-line parsers that convert raw text lines into
// domain.ParsedEvent structs. Each parser targets a specific log format
// (syslog, nginx, generic fallback). Implementations are stateless and safe
// for concurrent use.
package parser

import "github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"

// Parser is the contract every log-format parser must satisfy.
// Implementations must be safe for concurrent use from multiple goroutines.
type Parser interface {
	// Parse converts a single raw log line into a ParsedEvent.
	// It returns a non-nil error when the line cannot be meaningfully
	// parsed (the caller should log the error and skip the line).
	// The returned ParsedEvent will NOT have LogSourceID or ReceivedAt
	// set — those are populated downstream by the watcher pipeline.
	Parse(line string) (*domain.ParsedEvent, error)

	// Name returns a human-readable identifier for this parser
	// (e.g. "syslog", "nginx", "generic"). Used in structured logging.
	Name() string
}
