package parser

import "strings"

func NewParser(logType string) Parser {
	switch strings.ToLower(strings.TrimSpace(logType)) {
	case "syslog":
		return NewSyslogParser()
	case "generic", "nginx", "":
		return NewGenericParser()
	default:
		return NewGenericParser()
	}
}
