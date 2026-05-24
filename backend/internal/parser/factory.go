package parser

import "fmt"

// supportedTypes lists every log_type string recognised by New.
// Useful for validation in the service layer.
var supportedTypes = map[string]bool{
	"syslog":  true,
	"nginx":   true,
	"generic": true,
}

// New returns the Parser implementation that matches the given logType.
// Unknown types fall back to GenericParser so the watcher pipeline never
// panics on a misconfigured log source.
func New(logType string) Parser {
	switch logType {
	case "syslog":
		return &SyslogParser{}
	case "nginx":
		return &NginxParser{}
	default:
		return &GenericParser{}
	}
}

// IsSupported reports whether logType is a recognised parser type.
func IsSupported(logType string) bool {
	return supportedTypes[logType]
}

// SupportedTypes returns a human-readable list of all known log types.
func SupportedTypes() []string {
	out := make([]string, 0, len(supportedTypes))
	for k := range supportedTypes {
		out = append(out, k)
	}
	return out
}

// ErrUnsupportedType is returned (via wrapping) when callers want to
// distinguish "parsed with fallback" from "truly unknown" if needed.
var ErrUnsupportedType = fmt.Errorf("unsupported log type")
