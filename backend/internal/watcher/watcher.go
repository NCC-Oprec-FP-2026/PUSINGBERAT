package watcher

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/parser"
)

type EventWriter interface {
	Create(ctx context.Context, event *domain.Event) error
}

type FileWatcher struct {
	source       domain.LogSource
	eventWriter  EventWriter
	parser       parser.Parser
	fallback     parser.Parser
	pollInterval time.Duration
}

func NewFileWatcher(source domain.LogSource, eventWriter EventWriter) *FileWatcher {
	return &FileWatcher{
		source:       source,
		eventWriter:  eventWriter,
		parser:       parser.NewParser(string(source.LogType)),
		fallback:     parser.NewGenericParser(),
		pollInterval: time.Second,
	}
}

func (w *FileWatcher) Run(ctx context.Context) {
	reader := NewLineReader(resolveContainerPath(w.source.FilePath))
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		w.readAndPersist(ctx, reader)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *FileWatcher) readAndPersist(ctx context.Context, reader *LineReader) {
	lines, err := reader.ReadNewLines()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("WARN: watcher read failed source=%s path=%s: %v", w.source.ID, w.source.FilePath, err)
		}
		return
	}

	for _, line := range lines {
		event, err := w.parseLine(line)
		if err != nil {
			continue
		}

		if err := w.eventWriter.Create(ctx, event); err != nil {
			log.Printf("WARN: persist parsed event failed source=%s: %v", w.source.ID, err)
		}
	}
}

func (w *FileWatcher) parseLine(line string) (*domain.Event, error) {
	parsed, err := w.parser.Parse(line)
	if err != nil {
		parsed, err = w.fallback.Parse(line)
		if err != nil {
			return nil, err
		}
	}

	event := &domain.Event{
		LogSourceID: w.source.ID,
		RawLine:     parsed.RawLine,
		EventTime:   parsed.EventTime,
		Extra:       parsed.Extra,
	}

	if parsed.Message != "" {
		event.Message = stringPtr(parsed.Message)
	}
	if parsed.Hostname != "" {
		event.Hostname = stringPtr(parsed.Hostname)
	}
	if parsed.Process != "" {
		event.Process = stringPtr(parsed.Process)
	}
	if parsed.PID != nil {
		event.PID = parsed.PID
	}
	if parsed.LogLevel != "" {
		event.LogLevel = stringPtr(parsed.LogLevel)
	}

	return event, nil
}

func resolveContainerPath(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}

	if strings.HasPrefix(path, "/var/log/") {
		hostPath := "/host" + path
		if _, err := os.Stat(hostPath); err == nil {
			return hostPath
		}
	}

	return path
}

func stringPtr(value string) *string {
	return &value
}
