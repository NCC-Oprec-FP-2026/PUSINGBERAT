package parser

import (
	"errors"
	"time"
)

var ErrMalformedLine = errors.New("parser: malformed log line")

type ParsedEvent struct {
	RawLine   string
	Message   string
	Hostname  string
	Process   string
	PID       *int32
	LogLevel  string
	EventTime time.Time
	Extra     map[string]any
}

type Parser interface {
	Parse(line string) (*ParsedEvent, error)
}
