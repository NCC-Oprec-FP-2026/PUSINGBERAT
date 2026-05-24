package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Default channel buffer size
// ---------------------------------------------------------------------------

// DefaultEventChanSize is the capacity of the buffered event channel
// created by NewRegistry. It provides headroom between the watcher
// goroutines (producers) and the persistence goroutine (consumer).
const DefaultEventChanSize = 1000

// ---------------------------------------------------------------------------
// watcherEntry — internal bookkeeping for one active watcher
// ---------------------------------------------------------------------------

type watcherEntry struct {
	fw     *FileWatcher
	cancel context.CancelFunc
}

// ---------------------------------------------------------------------------
// Registry — thread-safe manager for all active FileWatchers
// ---------------------------------------------------------------------------

// Registry tracks active FileWatchers keyed by LogSource ID. It owns the
// shared event channel and provides methods for dynamic watcher lifecycle
// management (add / remove / bulk start).
//
// All public methods are safe for concurrent use.
type Registry struct {
	mu        sync.Mutex                  // guards `watchers`
	watchers  map[uuid.UUID]*watcherEntry // sourceID → entry
	paths     map[string]uuid.UUID        // filePath → sourceID (duplicate protection)
	eventChan chan *domain.ParsedEvent
	parentCtx context.Context // root context for all watcher goroutines
	hub       Broadcaster     // broadcasts watcher errors to websocket
}

// NewRegistry creates a Registry with a buffered event channel.
// parentCtx is propagated to every watcher goroutine — cancelling it
// stops all watchers.
func NewRegistry(parentCtx context.Context, hub Broadcaster) *Registry {
	return &Registry{
		watchers:  make(map[uuid.UUID]*watcherEntry),
		paths:     make(map[string]uuid.UUID),
		eventChan: make(chan *domain.ParsedEvent, DefaultEventChanSize),
		parentCtx: parentCtx,
		hub:       hub,
	}
}

// EventChan returns a receive-only view of the shared event channel.
// The consumer (persistence goroutine) reads from this channel.
func (r *Registry) EventChan() <-chan *domain.ParsedEvent {
	return r.eventChan
}

// ---------------------------------------------------------------------------
// AddWatcher — register and start watching a single log source
// ---------------------------------------------------------------------------

// AddWatcher creates a FileWatcher for the given log source and starts its
// goroutine. It returns an error if:
//   - A watcher for the same source ID is already active.
//   - A watcher for the same file path is already active (duplicate protection).
//   - FileWatcher construction fails.
func (r *Registry) AddWatcher(source *domain.LogSource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// --- Duplicate checks ----------------------------------------------------
	if _, exists := r.watchers[source.ID]; exists {
		return fmt.Errorf("registry: watcher for source %s already active", source.ID)
	}
	if existingID, exists := r.paths[source.FilePath]; exists {
		return fmt.Errorf("registry: path %s already watched by source %s", source.FilePath, existingID)
	}

	// --- Create the watcher --------------------------------------------------
	fw, err := NewFileWatcher(FileWatcherConfig{
		SourceID:  source.ID,
		FilePath:  source.FilePath,
		LogType:   source.LogType,
		EventChan: r.eventChan,
		Hub:       r.hub,
		SeekEnd:   true, // skip existing content on startup
	})
	if err != nil {
		return fmt.Errorf("registry: create watcher for %s: %w", source.FilePath, err)
	}

	// Derive a cancellable context for this specific watcher.
	ctx, cancel := context.WithCancel(r.parentCtx)
	fw.cancel = cancel

	entry := &watcherEntry{
		fw:     fw,
		cancel: cancel,
	}

	r.watchers[source.ID] = entry
	r.paths[source.FilePath] = source.ID

	// Launch the watcher goroutine.
	go func() {
		if err := fw.Start(ctx); err != nil {
			slog.Error("registry: watcher exited with error",
				"source_id", source.ID,
				"path", source.FilePath,
				"err", err,
			)
		}

		// Cleanup on exit so re-adding is possible.
		r.mu.Lock()
		delete(r.watchers, source.ID)
		delete(r.paths, source.FilePath)
		r.mu.Unlock()

		slog.Info("registry: watcher removed after exit",
			"source_id", source.ID,
			"path", source.FilePath,
		)
	}()

	slog.Info("registry: watcher added",
		"source_id", source.ID,
		"path", source.FilePath,
		"log_type", source.LogType,
	)

	return nil
}

// ---------------------------------------------------------------------------
// RemoveWatcher — stop and deregister a watcher by source ID
// ---------------------------------------------------------------------------

// RemoveWatcher stops the watcher goroutine for the given source ID and
// removes it from the registry. It is idempotent — calling it for an
// unknown source ID is a no-op.
func (r *Registry) RemoveWatcher(sourceID uuid.UUID) {
	r.mu.Lock()
	entry, exists := r.watchers[sourceID]
	if !exists {
		r.mu.Unlock()
		return
	}

	// Cancel the context to signal the goroutine to stop.
	entry.cancel()

	// Remove from maps immediately so a re-add is not blocked.
	filePath := entry.fw.filePath
	delete(r.watchers, sourceID)
	delete(r.paths, filePath)
	r.mu.Unlock()

	slog.Info("registry: watcher removed",
		"source_id", sourceID,
		"path", filePath,
	)
}

// ---------------------------------------------------------------------------
// StartAll — bulk registration at startup
// ---------------------------------------------------------------------------

// StartAll registers and starts watchers for every provided log source.
// It is typically called once at application startup with all active
// sources loaded from the database.
//
// Sources that fail to start (e.g. missing file) are logged and skipped;
// StartAll does not abort on individual failures so that healthy sources
// still run.
func (r *Registry) StartAll(sources []domain.LogSource) {
	for i := range sources {
		src := &sources[i]
		if src.Status != domain.LogSourceStatusActive {
			slog.Debug("registry: skipping inactive source",
				"source_id", src.ID,
				"status", src.Status,
			)
			continue
		}

		if err := r.AddWatcher(src); err != nil {
			slog.Error("registry: failed to start watcher",
				"source_id", src.ID,
				"path", src.FilePath,
				"err", err,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// StopAll — graceful shutdown of all watchers
// ---------------------------------------------------------------------------

// StopAll cancels all active watcher goroutines. It is idempotent and
// safe to call multiple times.
func (r *Registry) StopAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, entry := range r.watchers {
		entry.cancel()
		slog.Info("registry: stopping watcher",
			"source_id", id,
			"path", entry.fw.filePath,
		)
	}
	// Maps are cleaned up by the goroutine exit path in AddWatcher.
}

// ---------------------------------------------------------------------------
// ActiveCount — how many watchers are currently running
// ---------------------------------------------------------------------------

// ActiveCount returns the number of currently active watchers. Useful
// for health checks and debugging.
func (r *Registry) ActiveCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.watchers)
}

// ---------------------------------------------------------------------------
// IsWatching — check if a specific source is being watched
// ---------------------------------------------------------------------------

// IsWatching reports whether a watcher is active for the given source ID.
func (r *Registry) IsWatching(sourceID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.watchers[sourceID]
	return exists
}
