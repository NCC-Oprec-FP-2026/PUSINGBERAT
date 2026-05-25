package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// expectEvent is a test helper that waits for an event with a specific message
// to arrive on the channel within the given timeout.
func expectEvent(t *testing.T, ch <-chan *domain.ParsedEvent, expectedMsg string, timeout time.Duration) {
	t.Helper()
	select {
	case ev := <-ch:
		if ev.Message == nil || *ev.Message != expectedMsg {
			t.Errorf("expected message %q, got %q", expectedMsg, *ev.Message)
		}
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for event with message %q", expectedMsg)
	}
}

// 2. Log Rotation Recovery Test
func TestFileWatcher_LogRotation(t *testing.T) {
	// Create a temporary directory and an empty file
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	err := os.WriteFile(logFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	eventChan := make(chan *domain.ParsedEvent, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := FileWatcherConfig{
		SourceID:  uuid.New(),
		FilePath:  logFile,
		LogType:   "generic",
		EventChan: eventChan,
		SeekEnd:   false,
	}

	fw, err := NewFileWatcher(cfg)
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}

	// Start watcher in background
	go func() {
		if err := fw.Start(ctx); err != nil {
			t.Errorf("Start failed: %v", err)
		}
	}()

	// Wait for the watcher to attach to the file
	time.Sleep(100 * time.Millisecond)

	// Append initial data
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open file for append: %v", err)
	}
	f.WriteString("line1\nline2\n")
	f.Close()

	// Consume the first two lines
	expectEvent(t, eventChan, "line1", 2*time.Second)
	expectEvent(t, eventChan, "line2", 2*time.Second)

	// Simulate log rotation: rename the current file
	rotatedFile := filepath.Join(dir, "app.log.1")
	if err := os.Rename(logFile, rotatedFile); err != nil {
		t.Fatalf("failed to rename file: %v", err)
	}

	// Logging daemon creates the new file immediately after rotation
	f2, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}
	f2.Close()

	// Wait for the watcher to handle the rename event (it sleeps for 200ms)
	time.Sleep(300 * time.Millisecond)

	// Write new data to the new file (triggers a Write event)
	if err := os.WriteFile(logFile, []byte("line3\n"), 0644); err != nil {
		t.Fatalf("failed to write to new file: %v", err)
	}

	// Assert the watcher recovers, handles rotation, and reads the new data
	expectEvent(t, eventChan, "line3", 5*time.Second)
}

// 3. File Deletion Test
func TestFileWatcher_FileDeletion(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "delete_target.log")

	if err := os.WriteFile(logFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	eventChan := make(chan *domain.ParsedEvent, 10)
	ctx, cancel := context.WithCancel(context.Background())
	// Use defer to ensure we shut down gracefully at test end if needed
	defer cancel()

	cfg := FileWatcherConfig{
		SourceID:  uuid.New(),
		FilePath:  logFile,
		LogType:   "generic",
		EventChan: eventChan,
		SeekEnd:   false,
	}

	fw, err := NewFileWatcher(cfg)
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		// Wait for context cancellation, or return early if Start errors out unexpectedly
		_ = fw.Start(ctx)
		close(done)
	}()

	// Wait for the watcher to attach to the file
	time.Sleep(100 * time.Millisecond)

	// Append initial data to ensure watcher is fully hooked up and active
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open file for append: %v", err)
	}
	f.WriteString("line1\n")
	f.Close()

	// Consume the initial line to ensure watcher is fully hooked up and active
	expectEvent(t, eventChan, "line1", 2*time.Second)

	// Delete the active watched file
	if err := os.Remove(logFile); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	// Give the watcher's event loop time to process the fsnotify.Remove event
	// It will log the error and attempt handleRotation. We assert that it does
	// NOT panic the testing suite.
	time.Sleep(500 * time.Millisecond)

	// Cancel the context to gracefully shut down the goroutine
	cancel()

	// Wait for the goroutine to actually exit
	select {
	case <-done:
		// Success! It cleanly exited.
	case <-time.After(2 * time.Second):
		t.Fatalf("goroutine failed to exit after context cancellation")
	}
}
