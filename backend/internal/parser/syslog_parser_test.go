package parser

import (
	"errors"
	"testing"
	"time"
)

func fixedSyslogParser() *SyslogParser {
	return &SyslogParser{
		now: func() time.Time {
			return time.Date(2026, time.May, 21, 17, 0, 0, 0, time.UTC)
		},
	}
}

func TestSyslogParserValidLineWithPID(t *testing.T) {
	parser := fixedSyslogParser()

	event, err := parser.Parse("May 21 16:45:12 web01 sshd[1234]: Failed password for root from 10.0.0.1")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if event.Hostname != "web01" {
		t.Fatalf("hostname = %q, want web01", event.Hostname)
	}
	if event.Process != "sshd" {
		t.Fatalf("process = %q, want sshd", event.Process)
	}
	if event.PID == nil || *event.PID != 1234 {
		t.Fatalf("pid = %v, want 1234", event.PID)
	}
	if event.Message != "Failed password for root from 10.0.0.1" {
		t.Fatalf("message = %q", event.Message)
	}
	if event.EventTime.Year() != 2026 || event.EventTime.Month() != time.May || event.EventTime.Day() != 21 {
		t.Fatalf("event time = %s, want 2026-05-21", event.EventTime)
	}
}

func TestSyslogParserValidLineWithoutPID(t *testing.T) {
	parser := fixedSyslogParser()

	event, err := parser.Parse("May  7 01:02:03 app-host systemd: Started Daily apt download activities")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if event.PID != nil {
		t.Fatalf("pid = %v, want nil", *event.PID)
	}
	if event.Process != "systemd" {
		t.Fatalf("process = %q, want systemd", event.Process)
	}
	if event.EventTime.Day() != 7 {
		t.Fatalf("day = %d, want 7", event.EventTime.Day())
	}
}

func TestSyslogParserMalformedLine(t *testing.T) {
	parser := fixedSyslogParser()

	_, err := parser.Parse("this is not syslog")
	if !errors.Is(err, ErrMalformedLine) {
		t.Fatalf("error = %v, want ErrMalformedLine", err)
	}
}

func TestSyslogParserEmptyLine(t *testing.T) {
	parser := fixedSyslogParser()

	_, err := parser.Parse("   ")
	if !errors.Is(err, ErrMalformedLine) {
		t.Fatalf("error = %v, want ErrMalformedLine", err)
	}
}

func TestSyslogParserProcessWithPathLikeName(t *testing.T) {
	parser := fixedSyslogParser()

	event, err := parser.Parse("May 21 16:45:12 host CRON[99]: (root) CMD (/usr/bin/test)")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if event.Process != "CRON" {
		t.Fatalf("process = %q, want CRON", event.Process)
	}
	if event.PID == nil || *event.PID != 99 {
		t.Fatalf("pid = %v, want 99", event.PID)
	}
}
