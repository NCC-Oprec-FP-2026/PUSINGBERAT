package parser

import (
	"regexp"
	"strings"
	"time"
)

var genericTimestampRegex = regexp.MustCompile(`^\s*(\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s*(.*)$`)

type GenericParser struct {
	now func() time.Time
}

func NewGenericParser() *GenericParser {
	return &GenericParser{now: time.Now}
}

func (p *GenericParser) Parse(line string) (*ParsedEvent, error) {
	raw := strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(raw) == "" {
		return nil, ErrMalformedLine
	}

	eventTime := p.now()
	message := strings.TrimSpace(raw)

	if matches := genericTimestampRegex.FindStringSubmatch(raw); matches != nil {
		if parsedTime, ok := parseGenericTimestamp(matches[1]); ok {
			eventTime = parsedTime
			if strings.TrimSpace(matches[2]) != "" {
				message = strings.TrimSpace(matches[2])
			}
		}
	}

	return &ParsedEvent{
		RawLine:   raw,
		Message:   message,
		EventTime: eventTime,
		Extra: map[string]any{
			"parser": "generic",
		},
	}, nil
}

func parseGenericTimestamp(raw string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.999999999",
	}

	normalized := raw
	if strings.Contains(normalized, " ") && strings.ContainsAny(normalized, "Z+-") {
		normalized = strings.Replace(normalized, " ", "T", 1)
	}

	for _, layout := range layouts {
		if value, err := time.Parse(layout, normalized); err == nil {
			return value, true
		}
	}

	return time.Time{}, false
}
