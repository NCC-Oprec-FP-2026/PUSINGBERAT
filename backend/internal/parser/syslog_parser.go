package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var syslogRegex = regexp.MustCompile(`^([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+([A-Za-z0-9_./-]+)(?:\[(\d+)\])?:\s*(.*)$`)

type SyslogParser struct {
	now func() time.Time
}

func NewSyslogParser() *SyslogParser {
	return &SyslogParser{now: time.Now}
}

func (p *SyslogParser) Parse(line string) (*ParsedEvent, error) {
	raw := strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(raw) == "" {
		return nil, ErrMalformedLine
	}

	matches := syslogRegex.FindStringSubmatch(raw)
	if matches == nil {
		return nil, ErrMalformedLine
	}

	eventTime, err := parseSyslogTimestamp(matches[1], p.now())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedLine, err)
	}

	var pid *int32
	if matches[4] != "" {
		parsedPID, err := strconv.ParseInt(matches[4], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid pid", ErrMalformedLine)
		}
		value := int32(parsedPID)
		pid = &value
	}

	message := strings.TrimSpace(matches[5])
	return &ParsedEvent{
		RawLine:   raw,
		Message:   message,
		Hostname:  matches[2],
		Process:   matches[3],
		PID:       pid,
		EventTime: eventTime,
		Extra: map[string]any{
			"parser": "syslog",
		},
	}, nil
}

func parseSyslogTimestamp(raw string, now time.Time) (time.Time, error) {
	value, err := time.ParseInLocation("Jan 2 15:04:05", raw, now.Location())
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(
		now.Year(),
		value.Month(),
		value.Day(),
		value.Hour(),
		value.Minute(),
		value.Second(),
		0,
		now.Location(),
	), nil
}
