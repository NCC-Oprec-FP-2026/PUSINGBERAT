package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// genericRe is a best-effort regex that tries to extract a leading
// timestamp in several common formats, followed by the remainder as the
// message body.  It is compiled once at init — never inside the hot loop.
//
// Supported timestamp prefixes:
//   - ISO 8601        : 2026-05-21T14:30:00Z  or  2026-05-21T14:30:00+07:00
//   - Date + time     : 2026-05-21 14:30:00
//   - Syslog-like     : May 21 14:30:00
var genericRe = regexp.MustCompile(
	`^(?P<timestamp>` +
		`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?` + // ISO / date-time
		`|` +
		`[A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}` + // Syslog-like
		`)` +
		`\s+(?P<message>.+)$`,
)

// genericTimestampLayouts are tried in order when parsing the extracted
// timestamp substring. The first successful parse wins.
var genericTimestampLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"Jan  2 15:04:05",
	"Jan 2 15:04:05",
}

// GenericParser is a best-effort fallback parser. It attempts to split a
// line into a timestamp prefix and a message tail. When even that fails
// the entire line is stored as the message with time.Now() as the event
// timestamp so no data is silently lost.
type GenericParser struct{}

// Name implements Parser.
func (p *GenericParser) Name() string { return "generic" }

// Parse implements Parser.
func (p *GenericParser) Parse(line string) (*domain.ParsedEvent, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("generic: empty line")
	}

	ev := &domain.ParsedEvent{
		RawLine: line,
	}

	matches := genericRe.FindStringSubmatch(line)
	if matches == nil {
		// No recognisable timestamp — store the entire line as message
		// and use the current wall clock as event time.
		msg := line
		ev.Message = &msg
		ev.EventTime = time.Now().UTC()
		return ev, nil
	}

	tsStr := matches[genericRe.SubexpIndex("timestamp")]
	msgStr := matches[genericRe.SubexpIndex("message")]

	ev.Message = &msgStr
	ev.EventTime = parseTimestamp(tsStr)

	return ev, nil
}

// parseTimestamp tries every layout in genericTimestampLayouts. If none
// succeed it returns time.Now().UTC() to guarantee the event is never
// stored with a zero-value time.
func parseTimestamp(raw string) time.Time {
	for _, layout := range genericTimestampLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			// Syslog timestamps lack a year — patch with current year.
			if t.Year() == 0 {
				t = t.AddDate(time.Now().Year(), 0, 0)
			}
			return t.UTC()
		}
	}
	return time.Now().UTC()
}
