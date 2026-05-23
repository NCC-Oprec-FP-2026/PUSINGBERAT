package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Regex — compiled once at package init, never inside the parse loop.
// ---------------------------------------------------------------------------

// syslogRe matches the BSD/RFC 3164 syslog format commonly produced by
// rsyslog and systemd-journald:
//
//	May 21 14:30:00 myhost sshd[1234]: Failed password for root
//	May  5 09:01:02 myhost CRON[456]: (root) CMD (/usr/lib/…)
//
// Named groups:
//
//	month  — abbreviated month name  (May)
//	day    — day of month, 1 or 2 digits (5, 21)
//	time   — HH:MM:SS
//	host   — hostname (non-whitespace)
//	proc   — process/tag (captured up to the optional [PID])
//	pid    — numeric PID inside brackets (optional)
//	msg    — the rest of the line after ": "
var syslogRe = regexp.MustCompile(
	`^(?P<month>[A-Z][a-z]{2})\s+(?P<day>\d{1,2})\s+(?P<time>\d{2}:\d{2}:\d{2})\s+` +
		`(?P<host>\S+)\s+` +
		`(?P<proc>[^\s\[:]+)` + // process name (up to [ or : or whitespace)
		`(?:\[(?P<pid>\d+)\])?` + // optional [PID]
		`:\s+(?P<msg>.+)$`,
)

// Pre-compute subexpression indexes once so they are not resolved per call.
var (
	syslogIdxMonth = syslogRe.SubexpIndex("month")
	syslogIdxDay   = syslogRe.SubexpIndex("day")
	syslogIdxTime  = syslogRe.SubexpIndex("time")
	syslogIdxHost  = syslogRe.SubexpIndex("host")
	syslogIdxProc  = syslogRe.SubexpIndex("proc")
	syslogIdxPID   = syslogRe.SubexpIndex("pid")
	syslogIdxMsg   = syslogRe.SubexpIndex("msg")
)

// ---------------------------------------------------------------------------
// SyslogParser
// ---------------------------------------------------------------------------

// SyslogParser parses BSD/RFC 3164 syslog lines. It is stateless and safe
// for concurrent use.
type SyslogParser struct{}

// Name implements Parser.
func (p *SyslogParser) Name() string { return "syslog" }

// Parse implements Parser. It extracts timestamp, hostname, process, PID, and
// message from a BSD syslog line and maps them into a domain.ParsedEvent.
func (p *SyslogParser) Parse(line string) (*domain.ParsedEvent, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("syslog: empty line")
	}

	matches := syslogRe.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("syslog: line did not match expected format: %.80s", line)
	}

	// --- Build the event time ------------------------------------------------
	// BSD syslog timestamps lack a year; we use the current year.
	// Format: "Jan  2 15:04:05"  (Go reference time: Jan 2 15:04:05 2006)
	tsRaw := fmt.Sprintf("%s %s %s", matches[syslogIdxMonth], matches[syslogIdxDay], matches[syslogIdxTime])
	eventTime, err := time.Parse("Jan 2 15:04:05", tsRaw)
	if err != nil {
		return nil, fmt.Errorf("syslog: bad timestamp %q: %w", tsRaw, err)
	}
	eventTime = eventTime.AddDate(time.Now().Year(), 0, 0).UTC()

	// --- Populate the ParsedEvent --------------------------------------------
	host := matches[syslogIdxHost]
	proc := matches[syslogIdxProc]
	msg := matches[syslogIdxMsg]

	ev := &domain.ParsedEvent{
		RawLine:   line,
		Hostname:  &host,
		Process:   &proc,
		Message:   &msg,
		EventTime: eventTime,
	}

	// PID is optional — only set when present.
	if pidStr := matches[syslogIdxPID]; pidStr != "" {
		pid, err := strconv.Atoi(pidStr)
		if err == nil {
			ev.PID = &pid
		}
	}

	return ev, nil
}
