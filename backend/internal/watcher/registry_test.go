package watcher

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// TestRegistry_GoroutineLeak programmatically checks runtime.NumGoroutine().
// It adds and removes a watcher 100 times and asserts that the active
// goroutine count returns to baseline, proving no goroutines are leaked.
func TestRegistry_GoroutineLeak(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg := NewRegistry(ctx, nil)

	// Settle before taking baseline
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	sourceID := uuid.New()
	source := &domain.LogSource{
		ID:       sourceID,
		FilePath: "/tmp/leak_test.log",
		LogType:  "generic",
	}

	for i := 0; i < 100; i++ {
		err := reg.AddWatcher(source)
		if err != nil {
			t.Fatalf("AddWatcher failed on iter %d: %v", i, err)
		}

		// Give the goroutine a tiny moment to start up and register itself
		time.Sleep(2 * time.Millisecond)

		// Remove it, triggering the cancel() function inside the entry
		reg.RemoveWatcher(sourceID)

		// Give it a tiny moment to exit
		time.Sleep(2 * time.Millisecond)
	}

	// Wait for any straggling cleanup routines to finish
	time.Sleep(200 * time.Millisecond)
	final := runtime.NumGoroutine()

	// Allow a small variance (Go runtime bg tasks) but definitely not +100
	if final > baseline+5 {
		t.Errorf("goroutine leak detected! baseline=%d, final=%d", baseline, final)
	}
}

func TestRegistry_Methods(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg := NewRegistry(ctx, nil)

	if reg.EventChan() == nil {
		t.Error("expected non-nil event chan")
	}

	source := &domain.LogSource{
		ID:       uuid.New(),
		FilePath: "/tmp/leak_test_methods.log",
		LogType:  "generic",
		Status:   domain.LogSourceStatusActive,
	}

	// Test StartAll
	reg.StartAll([]domain.LogSource{*source})
	
	if reg.ActiveCount() != 1 {
		t.Errorf("expected 1 active watcher, got %d", reg.ActiveCount())
	}
	
	if !reg.IsWatching(source.ID) {
		t.Error("expected IsWatching to be true")
	}

	// Test StopAll
	reg.StopAll()
	time.Sleep(100 * time.Millisecond)

	if reg.ActiveCount() != 0 {
		t.Errorf("expected 0 active watchers after StopAll, got %d", reg.ActiveCount())
	}
}
