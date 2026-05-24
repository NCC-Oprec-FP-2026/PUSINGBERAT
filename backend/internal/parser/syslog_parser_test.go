package parser

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func currentYear() int { return time.Now().Year() }

// ---------------------------------------------------------------------------
// SyslogParser — valid lines
// ---------------------------------------------------------------------------

func TestSyslogParser_ValidLine_WithPID(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	ev, err := p.Parse("May 21 14:30:00 myhost sshd[1234]: Failed password for root from 10.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Hostname
	if ev.Hostname == nil || *ev.Hostname != "myhost" {
		t.Errorf("hostname = %v, want %q", ev.Hostname, "myhost")
	}

	// Process
	if ev.Process == nil || *ev.Process != "sshd" {
		t.Errorf("process = %v, want %q", ev.Process, "sshd")
	}

	// PID
	if ev.PID == nil || *ev.PID != 1234 {
		t.Errorf("pid = %v, want %d", ev.PID, 1234)
	}

	// Message
	if ev.Message == nil || *ev.Message != "Failed password for root from 10.0.0.1" {
		t.Errorf("message = %v, want %q", ev.Message, "Failed password for root from 10.0.0.1")
	}

	// Timestamp — year is patched to the current year
	wantTime := time.Date(currentYear(), time.May, 21, 14, 30, 0, 0, time.UTC)
	if !ev.EventTime.Equal(wantTime) {
		t.Errorf("event_time = %v, want %v", ev.EventTime, wantTime)
	}

	// RawLine preserved
	if !strings.Contains(ev.RawLine, "Failed password") {
		t.Errorf("raw_line should contain original text")
	}
}

func TestSyslogParser_ValidLine_WithoutPID(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// Some syslog entries have no [PID] block.
	ev, err := p.Parse("Jan  5 09:01:02 webserver nginx: 127.0.0.1 GET /health 200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.Hostname == nil || *ev.Hostname != "webserver" {
		t.Errorf("hostname = %v, want %q", ev.Hostname, "webserver")
	}
	if ev.Process == nil || *ev.Process != "nginx" {
		t.Errorf("process = %v, want %q", ev.Process, "nginx")
	}
	if ev.PID != nil {
		t.Errorf("pid = %v, want nil", ev.PID)
	}
	if ev.Message == nil || *ev.Message != "127.0.0.1 GET /health 200" {
		t.Errorf("message = %v, want %q", ev.Message, "127.0.0.1 GET /health 200")
	}

	wantTime := time.Date(currentYear(), time.January, 5, 9, 1, 2, 0, time.UTC)
	if !ev.EventTime.Equal(wantTime) {
		t.Errorf("event_time = %v, want %v", ev.EventTime, wantTime)
	}
}

func TestSyslogParser_ValidLine_CRON(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	ev, err := p.Parse("Dec 31 23:59:59 prodhost CRON[99999]: (root) CMD (/usr/lib/cron/run)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.Process == nil || *ev.Process != "CRON" {
		t.Errorf("process = %v, want %q", ev.Process, "CRON")
	}
	if ev.PID == nil || *ev.PID != 99999 {
		t.Errorf("pid = %v, want %d", ev.PID, 99999)
	}
}

// ---------------------------------------------------------------------------
// SyslogParser — malformed / edge-case lines
// ---------------------------------------------------------------------------

func TestSyslogParser_EmptyLine(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	_, err := p.Parse("")
	if err == nil {
		t.Fatal("expected error for empty line, got nil")
	}
}

func TestSyslogParser_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	_, err := p.Parse("   \t  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only line, got nil")
	}
}

func TestSyslogParser_MalformedLine_NoTimestamp(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	_, err := p.Parse("this is not a syslog line at all")
	if err == nil {
		t.Fatal("expected error for non-syslog line, got nil")
	}
}

func TestSyslogParser_MalformedLine_TruncatedTimestamp(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	_, err := p.Parse("May 21 14:")
	if err == nil {
		t.Fatal("expected error for truncated timestamp, got nil")
	}
}

func TestSyslogParser_MalformedLine_MissingMessage(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// Timestamp + host + proc but no ": message"
	_, err := p.Parse("May 21 14:30:00 host proc")
	if err == nil {
		t.Fatal("expected error for line without message delimiter, got nil")
	}
}

func TestSyslogParser_LeadingTrailingWhitespace(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	ev, err := p.Parse("  May 21 14:30:00 host sshd[1]: hello world  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Message == nil || *ev.Message != "hello world" {
		t.Errorf("message = %v, want %q", ev.Message, "hello world")
	}
}

func TestSyslogParser_Name(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}
	if p.Name() != "syslog" {
		t.Errorf("Name() = %q, want %q", p.Name(), "syslog")
	}
}

// ---------------------------------------------------------------------------
// SyslogParser — single-digit day with double-space padding
// ---------------------------------------------------------------------------

func TestSyslogParser_SingleDigitDay(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// rsyslog pads single-digit days with a leading space: "May  5"
	ev, err := p.Parse("May  5 12:00:00 box kernel: [42.123] eth0: link up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantTime := time.Date(currentYear(), time.May, 5, 12, 0, 0, 0, time.UTC)
	if !ev.EventTime.Equal(wantTime) {
		t.Errorf("event_time = %v, want %v", ev.EventTime, wantTime)
	}
}

// ---------------------------------------------------------------------------
// SyslogParser — message with colons
// ---------------------------------------------------------------------------

func TestSyslogParser_MessageWithColons(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// The message itself contains colons — the regex must capture everything
	// after the first ": " delimiter.
	ev, err := p.Parse("Jun 10 08:00:00 host app[1]: key: value: extra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.Message == nil || *ev.Message != "key: value: extra" {
		t.Errorf("message = %v, want %q", ev.Message, "key: value: extra")
	}
}

// ---------------------------------------------------------------------------
// Race-safety test — exercised with `go test -race`
// ---------------------------------------------------------------------------

func TestSyslogParser_ConcurrentSafety(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	lines := []string{
		"May 21 14:30:00 host1 sshd[1]: msg1",
		"Jun 10 08:00:00 host2 nginx: msg2",
		"Dec 31 23:59:59 host3 CRON[99]: msg3",
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			line := lines[idx%len(lines)]
			ev, err := p.Parse(line)
			if err != nil {
				t.Errorf("concurrent parse error: %v", err)
				return
			}
			if ev.Message == nil {
				t.Error("concurrent parse returned nil message")
			}
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// GenericParser — basic smoke tests
// ---------------------------------------------------------------------------

func TestGenericParser_ISOTimestamp(t *testing.T) {
	t.Parallel()
	p := &GenericParser{}

	ev, err := p.Parse("2026-05-21T14:30:00Z This is the message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Message == nil || *ev.Message != "This is the message" {
		t.Errorf("message = %v, want %q", ev.Message, "This is the message")
	}

	wantTime := time.Date(2026, time.May, 21, 14, 30, 0, 0, time.UTC)
	if !ev.EventTime.Equal(wantTime) {
		t.Errorf("event_time = %v, want %v", ev.EventTime, wantTime)
	}
}

func TestGenericParser_NoTimestamp_Fallback(t *testing.T) {
	t.Parallel()
	p := &GenericParser{}

	ev, err := p.Parse("just a random line with no timestamp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Message should be the entire line.
	if ev.Message == nil || *ev.Message != "just a random line with no timestamp" {
		t.Errorf("message = %v, want original line", ev.Message)
	}

	// EventTime should be approximately now (within 5 seconds).
	diff := time.Since(ev.EventTime)
	if diff > 5*time.Second || diff < -5*time.Second {
		t.Errorf("event_time %v is too far from now", ev.EventTime)
	}
}

func TestGenericParser_EmptyLine(t *testing.T) {
	t.Parallel()
	p := &GenericParser{}

	_, err := p.Parse("")
	if err == nil {
		t.Fatal("expected error for empty line, got nil")
	}
}

func TestGenericParser_Name(t *testing.T) {
	t.Parallel()
	p := &GenericParser{}
	if p.Name() != "generic" {
		t.Errorf("Name() = %q, want %q", p.Name(), "generic")
	}
}

// ---------------------------------------------------------------------------
// Factory — New()
// ---------------------------------------------------------------------------

func TestNew_KnownTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		logType  string
		wantName string
	}{
		{"syslog", "syslog"},
		{"nginx", "nginx"},
		{"generic", "generic"},
		{"unknown-type", "generic"}, // fallback
		{"", "generic"},             // empty string → fallback
	}

	for _, tt := range tests {
		p := New(tt.logType)
		if p.Name() != tt.wantName {
			t.Errorf("New(%q).Name() = %q, want %q", tt.logType, p.Name(), tt.wantName)
		}
	}
}

func TestIsSupported(t *testing.T) {
	t.Parallel()

	if !IsSupported("syslog") {
		t.Error("IsSupported(syslog) should be true")
	}
	if !IsSupported("nginx") {
		t.Error("IsSupported(nginx) should be true")
	}
	if !IsSupported("generic") {
		t.Error("IsSupported(generic) should be true")
	}
	if IsSupported("banana") {
		t.Error("IsSupported(banana) should be false")
	}
}

// ---------------------------------------------------------------------------
// SyslogParser — additional edge cases
// ---------------------------------------------------------------------------

func TestSyslogParser_MalformedLine_UnknownMonth(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// Unknown month format 'jan' instead of 'Jan' (case-sensitive)
	_, err := p.Parse("jan  2 15:04:05 host app: msg")
	if err == nil {
		t.Fatal("expected error for unknown month format, got nil")
	}
}

func TestSyslogParser_MalformedLine_MissingProcess(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// Hostname is there, but no process name before the message colon
	_, err := p.Parse("Jan  2 15:04:05 host : msg")
	if err == nil {
		t.Fatal("expected error for missing process name, got nil")
	}
}

func TestSyslogParser_CompletelyMalformed(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// Gibberish
	_, err := p.Parse("This is completely malformed and has no structure")
	if err == nil {
		t.Fatal("expected error for completely malformed line, got nil")
	}
}

func TestSyslogParser_MissingPIDBrackets(t *testing.T) {
	t.Parallel()
	p := &SyslogParser{}

	// Process has number but no brackets (e.g. sshd1234)
	ev, err := p.Parse("Jan  2 15:04:05 host sshd1234: msg")
	if err != nil {
		t.Fatalf("unexpected error for process without brackets: %v", err)
	}
	if ev.Process == nil || *ev.Process != "sshd1234" {
		t.Errorf("process = %v, want %q", ev.Process, "sshd1234")
	}
	if ev.PID != nil {
		t.Errorf("pid = %v, want nil", ev.PID)
	}
}
