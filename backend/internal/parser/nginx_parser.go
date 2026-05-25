package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

var nginxAccessRe = regexp.MustCompile(
	`^(?P<ip>\S+) \S+ \S+ \[(?P<time>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) \S+" (?P<status>\d+) (?P<bytes>\d+)`,
)

var (
	nginxIdxIP     = nginxAccessRe.SubexpIndex("ip")
	nginxIdxTime   = nginxAccessRe.SubexpIndex("time")
	nginxIdxMethod = nginxAccessRe.SubexpIndex("method")
	nginxIdxPath   = nginxAccessRe.SubexpIndex("path")
	nginxIdxStatus = nginxAccessRe.SubexpIndex("status")
	nginxIdxBytes  = nginxAccessRe.SubexpIndex("bytes")
)

// NginxParser is a parser for nginx access logs.
type NginxParser struct{}

// Name implements Parser.
func (p *NginxParser) Name() string { return "nginx" }

// Parse implements Parser.
func (p *NginxParser) Parse(line string) (*domain.ParsedEvent, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("nginx: empty line")
	}

	matches := nginxAccessRe.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("nginx: line did not match expected format: %.80s", line)
	}

	tsRaw := matches[nginxIdxTime]
	// Nginx time format: 05/May/2025:12:34:56 +0000
	eventTime, err := time.Parse("02/Jan/2006:15:04:05 -0700", tsRaw)
	if err != nil {
		return nil, fmt.Errorf("nginx: bad timestamp %q: %w", tsRaw, err)
	}

	ip := matches[nginxIdxIP]
	method := matches[nginxIdxMethod]
	path := matches[nginxIdxPath]
	status := matches[nginxIdxStatus]
	bytesStr := matches[nginxIdxBytes]

	extraMap := map[string]interface{}{
		"ip":          ip,
		"method":      method,
		"path":        path,
		"status_code": status,
		"bytes":       bytesStr,
	}

	extraBytes, err := json.Marshal(extraMap)
	if err != nil {
		return nil, fmt.Errorf("nginx: failed to marshal extra fields: %w", err)
	}

	msg := line
	return &domain.ParsedEvent{
		RawLine:   line,
		Message:   &msg,
		Hostname:  &ip,
		EventTime: eventTime.UTC(),
		Extra:     extraBytes,
	}, nil
}
