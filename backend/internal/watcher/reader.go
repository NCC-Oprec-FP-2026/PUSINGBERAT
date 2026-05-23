// Package watcher implements the file-watching pipeline that detects new
// log lines, parses them via the parser package, and emits domain.ParsedEvent
// values for downstream persistence.
package watcher

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// ---------------------------------------------------------------------------
// FileReader — offset-aware, rotation-safe log file reader
// ---------------------------------------------------------------------------

// FileReader tracks a read offset into a single file so that only newly
// appended lines are returned on each call to ReadNewLines.
//
// It is safe for sequential use from a single goroutine but NOT for
// concurrent use without external synchronisation.
type FileReader struct {
	filePath string
	offset   int64
	mu       sync.Mutex // guards offset for safe Reset from another goroutine
}

// NewFileReader creates a reader for the given path.
// If seekEnd is true the initial offset is set to the current end-of-file
// so that pre-existing lines are skipped on first read.
func NewFileReader(path string, seekEnd bool) (*FileReader, error) {
	r := &FileReader{filePath: path}

	if seekEnd {
		info, err := os.Stat(path)
		if err != nil {
			// File might not exist yet; start from 0.
			if os.IsNotExist(err) {
				return r, nil
			}
			return nil, fmt.Errorf("reader: stat %s: %w", path, err)
		}
		r.offset = info.Size()
	}

	return r, nil
}

// ReadNewLines opens the file, seeks to the current offset, reads all
// complete lines appended since the last call, and updates the offset.
//
// Truncation handling: if the file is shorter than the stored offset
// (e.g. after `truncate -s 0`), the offset is reset to 0 so the new
// content is read from the beginning.
//
// Returns an empty slice and nil error when the file does not exist —
// this is expected during log rotation when the old file has been
// removed but the new one has not been created yet.
func (r *FileReader) ReadNewLines() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File gone (rotation); nothing to read, offset stays.
			return nil, nil
		}
		return nil, fmt.Errorf("reader: open %s: %w", r.filePath, err)
	}
	defer f.Close()

	// --- Truncation detection ------------------------------------------------
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("reader: stat %s: %w", r.filePath, err)
	}
	if info.Size() < r.offset {
		slog.Info("file truncated, resetting offset",
			"path", r.filePath,
			"old_offset", r.offset,
			"new_size", info.Size(),
		)
		r.offset = 0
	}

	// --- Seek to offset and scan new lines -----------------------------------
	if _, err := f.Seek(r.offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("reader: seek %s: %w", r.filePath, err)
	}

	var lines []string
	scanner := bufio.NewScanner(f)

	// Increase the scanner buffer for potentially long log lines (up to 1 MB).
	const maxLineBytes = 1 << 20
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return lines, fmt.Errorf("reader: scan %s: %w", r.filePath, err)
	}

	// Update offset to current file position.
	newOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return lines, fmt.Errorf("reader: tell %s: %w", r.filePath, err)
	}
	r.offset = newOffset

	return lines, nil
}

// ResetOffset sets the read offset to 0 so the next ReadNewLines call
// reads from the beginning of the file. This is called when a log
// rotation event is detected (RENAME/REMOVE followed by a new file
// appearing at the same path).
func (r *FileReader) ResetOffset() {
	r.mu.Lock()
	r.offset = 0
	r.mu.Unlock()
}

// Offset returns the current byte offset (useful for logging/debugging).
func (r *FileReader) Offset() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.offset
}
