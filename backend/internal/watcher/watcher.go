package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/parser"
)

// ---------------------------------------------------------------------------
// FileWatcher — watches a single file and emits ParsedEvents
// ---------------------------------------------------------------------------

// FileWatcher ties together an fsnotify watcher, an offset-aware reader,
// and a log parser. One goroutine per watched file runs the select loop.
type FileWatcher struct {
	sourceID  uuid.UUID
	filePath  string
	logType   string
	reader    *FileReader
	parser    parser.Parser
	eventChan chan<- *domain.ParsedEvent

	cancel context.CancelFunc // stored so the registry can stop this watcher
}

// FileWatcherConfig holds the parameters needed to construct a FileWatcher.
type FileWatcherConfig struct {
	SourceID  uuid.UUID
	FilePath  string
	LogType   string
	EventChan chan<- *domain.ParsedEvent
	// SeekEnd controls whether existing file content is skipped.
	// Set to true during initial startup so historical lines are not
	// re-ingested; false when a watcher is added via the API for
	// a brand-new file.
	SeekEnd bool
}

// NewFileWatcher creates a FileWatcher ready to be started.
func NewFileWatcher(cfg FileWatcherConfig) (*FileWatcher, error) {
	reader, err := NewFileReader(cfg.FilePath, cfg.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("watcher: create reader for %s: %w", cfg.FilePath, err)
	}

	p := parser.New(cfg.LogType)

	return &FileWatcher{
		sourceID:  cfg.SourceID,
		filePath:  cfg.FilePath,
		logType:   cfg.LogType,
		reader:    reader,
		parser:    p,
		eventChan: cfg.EventChan,
	}, nil
}

// Start runs the fsnotify event loop in the current goroutine. It blocks
// until ctx is cancelled. The caller is expected to launch this in a
// separate goroutine.
func (fw *FileWatcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watcher: fsnotify.NewWatcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(fw.filePath); err != nil {
		// File might not exist yet — log and continue; the re-watch
		// loop below will retry.
		slog.Warn("watcher: initial Add failed, will retry on rotation",
			"path", fw.filePath,
			"err", err,
		)
	}

	slog.Info("watcher started",
		"source_id", fw.sourceID,
		"path", fw.filePath,
		"parser", fw.parser.Name(),
		"offset", fw.reader.Offset(),
	)

	// Debounce timer prevents bursts of WRITE events from triggering
	// redundant read passes. When a WRITE arrives, we reset the timer;
	// reading only happens when the timer fires (i.e. writes have
	// settled for the debounce duration).
	const debounceDuration = 50 * time.Millisecond
	debounce := time.NewTimer(debounceDuration)
	debounce.Stop() // don't fire until the first WRITE arrives
	defer debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("watcher stopping (context cancelled)",
				"source_id", fw.sourceID,
				"path", fw.filePath,
			)
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil // channel closed
			}

			switch {
			case event.Has(fsnotify.Write):
				// Debounce: reset the timer so we batch rapid writes.
				debounce.Reset(debounceDuration)

			case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
				// Log rotation: the old file was renamed or removed.
				slog.Info("log rotation detected",
					"source_id", fw.sourceID,
					"path", fw.filePath,
					"op", event.Op.String(),
				)
				fw.handleRotation(watcher)
			}

		case <-debounce.C:
			// Debounce fired — read and process new lines.
			fw.readAndParse()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			slog.Error("watcher: fsnotify error",
				"source_id", fw.sourceID,
				"path", fw.filePath,
				"err", err,
			)
		}
	}
}

// readAndParse reads new lines from the file and sends parsed events
// to the event channel. Parse errors are logged and skipped — one bad
// line never stops the pipeline.
func (fw *FileWatcher) readAndParse() {
	lines, err := fw.reader.ReadNewLines()
	if err != nil {
		slog.Error("watcher: read error",
			"source_id", fw.sourceID,
			"path", fw.filePath,
			"err", err,
		)
		return
	}

	now := time.Now().UTC()
	for _, line := range lines {
		ev, err := fw.parser.Parse(line)
		if err != nil {
			slog.Debug("watcher: parse skip",
				"source_id", fw.sourceID,
				"parser", fw.parser.Name(),
				"err", err,
			)
			continue
		}

		// Populate the fields the parser deliberately leaves blank.
		ev.LogSourceID = fw.sourceID
		ev.ReceivedAt = now

		// Non-blocking send: if the channel is full, log a warning
		// and drop the event rather than blocking the watcher goroutine.
		select {
		case fw.eventChan <- ev:
		default:
			slog.Warn("watcher: event channel full, dropping event",
				"source_id", fw.sourceID,
				"path", fw.filePath,
			)
		}
	}
}

// handleRotation is called when a RENAME or REMOVE event is detected.
// It waits briefly for the new file to appear, resets the read offset,
// and re-adds the path to fsnotify.
func (fw *FileWatcher) handleRotation(watcher *fsnotify.Watcher) {
	// Brief delay for the logging daemon to create the new file.
	time.Sleep(200 * time.Millisecond)

	fw.reader.ResetOffset()

	// Best-effort re-add — the file may not exist yet.
	if err := watcher.Add(fw.filePath); err != nil {
		slog.Warn("watcher: re-add after rotation failed, will retry next event",
			"source_id", fw.sourceID,
			"path", fw.filePath,
			"err", err,
		)
	} else {
		slog.Info("watcher: re-watching after rotation",
			"source_id", fw.sourceID,
			"path", fw.filePath,
		)
	}
}

// Stop cancels the watcher's context, causing Start to return.
func (fw *FileWatcher) Stop() {
	if fw.cancel != nil {
		fw.cancel()
	}
}
