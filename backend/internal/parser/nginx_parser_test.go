package parser

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNginxParser_ValidAccessLog(t *testing.T) {
	t.Parallel()
	p := &NginxParser{}

	line := `127.0.0.1 - - [05/May/2025:12:34:56 +0000] "GET / HTTP/1.1" 200 612`
	ev, err := p.Parse(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.Hostname == nil || *ev.Hostname != "127.0.0.1" {
		t.Errorf("hostname = %v, want %q", ev.Hostname, "127.0.0.1")
	}

	if ev.Message == nil || *ev.Message != line {
		t.Errorf("message = %v, want %q", ev.Message, line)
	}

	wantTime := time.Date(2025, time.May, 5, 12, 34, 56, 0, time.UTC)
	if !ev.EventTime.Equal(wantTime) {
		t.Errorf("event_time = %v, want %v", ev.EventTime, wantTime)
	}

	var extra map[string]interface{}
	if err := json.Unmarshal(ev.Extra, &extra); err != nil {
		t.Fatalf("failed to unmarshal extra: %v", err)
	}

	if extra["ip"] != "127.0.0.1" {
		t.Errorf("extra[ip] = %v, want %q", extra["ip"], "127.0.0.1")
	}
	if extra["method"] != "GET" {
		t.Errorf("extra[method] = %v, want %q", extra["method"], "GET")
	}
	if extra["path"] != "/" {
		t.Errorf("extra[path] = %v, want %q", extra["path"], "/")
	}
	if extra["status_code"] != "200" {
		t.Errorf("extra[status_code] = %v, want %q", extra["status_code"], "200")
	}
	if extra["bytes"] != "612" {
		t.Errorf("extra[bytes] = %v, want %q", extra["bytes"], "612")
	}
}

func TestNginxParser_MalformedLine(t *testing.T) {
	t.Parallel()
	p := &NginxParser{}

	_, err := p.Parse("this is not an nginx log line")
	if err == nil {
		t.Fatal("expected error for malformed line, got nil")
	}
	if !strings.Contains(err.Error(), "did not match expected format") {
		t.Errorf("expected format error, got: %v", err)
	}
}

func TestNginxParser_ErrorLogLine(t *testing.T) {
	t.Parallel()
	p := &NginxParser{}

	_, err := p.Parse("2025/05/05 12:34:56 [error] 123#123: *456 connect() failed")
	if err == nil {
		t.Fatal("expected error for error log line (unsupported by regex), got nil")
	}
}

func TestNginxParser_EmptyLine(t *testing.T) {
	t.Parallel()
	p := &NginxParser{}

	_, err := p.Parse("   \t  ")
	if err == nil {
		t.Fatal("expected error for empty line, got nil")
	}
	if !strings.Contains(err.Error(), "empty line") {
		t.Errorf("expected empty line error, got: %v", err)
	}
}
