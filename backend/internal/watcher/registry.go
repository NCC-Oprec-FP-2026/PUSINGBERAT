package watcher

import (
	"context"
	"log"
	"sync"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type Registry struct {
	eventWriter EventWriter

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewRegistry(eventWriter EventWriter) *Registry {
	return &Registry{
		eventWriter: eventWriter,
		cancels:     make(map[string]context.CancelFunc),
	}
}

func (r *Registry) Start(source *domain.LogSource) {
	if source == nil || source.ID == "" {
		return
	}

	r.Stop(source.ID)

	ctx, cancel := context.WithCancel(context.Background())
	watcher := NewFileWatcher(*source, r.eventWriter)

	r.mu.Lock()
	r.cancels[source.ID] = cancel
	r.mu.Unlock()

	go func() {
		log.Printf("INFO: watcher started source=%s path=%s type=%s", source.ID, source.FilePath, source.LogType)
		watcher.Run(ctx)
		log.Printf("INFO: watcher stopped source=%s", source.ID)
	}()
}

func (r *Registry) Stop(sourceID string) {
	r.mu.Lock()
	cancel, ok := r.cancels[sourceID]
	if ok {
		delete(r.cancels, sourceID)
	}
	r.mu.Unlock()

	if ok {
		cancel()
	}
}

func (r *Registry) StopAll() {
	r.mu.Lock()
	cancels := r.cancels
	r.cancels = make(map[string]context.CancelFunc)
	r.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}
