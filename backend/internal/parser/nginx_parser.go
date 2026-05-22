package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// NginxParser is a placeholder for the nginx access/error log parser.
// Full implementation is scheduled for a future sprint. For now it falls
// back to the same best-effort logic as GenericParser so that registering
// a log source with log_type="nginx" does not crash the pipeline.
type NginxParser struct{}

// Name implements Parser.
func (p *NginxParser) Name() string { return "nginx" }

// Parse implements Parser.
// TODO(day3-part2): implement proper nginx access + error log regex.
func (p *NginxParser) Parse(line string) (*domain.ParsedEvent, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("nginx: empty line")
	}

	// Fallback: store the whole line as message with current time.
	msg := line
	return &domain.ParsedEvent{
		RawLine:   line,
		Message:   &msg,
		EventTime: time.Now().UTC(),
	}, nil
}
