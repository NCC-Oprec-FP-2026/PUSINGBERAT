// Package service implements the business-logic layer for the PUSINGBERAT SIEM
// backend. Each service struct depends on repository interfaces (defined here,
// at the point of consumption) so that concrete implementations can be swapped
// for mocks in tests.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Repository interface (consumed by LogSourceService)
// ---------------------------------------------------------------------------

// LogSourceRepository defines the persistence contract that LogSourceService
// requires. The concrete implementation lives in package repository.
type LogSourceRepository interface {
	Create(ctx context.Context, ls *domain.LogSource) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error)
	List(ctx context.Context) ([]domain.LogSource, error)
	Update(ctx context.Context, ls *domain.LogSource) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// WatcherRegistry interface (optional dependency)
// ---------------------------------------------------------------------------

// WatcherRegistry defines the contract for managing file watchers. The
// concrete implementation lives in package watcher. This is defined at the
// point of consumption to avoid a hard dependency from service → watcher.
type WatcherRegistry interface {
	AddWatcher(source *domain.LogSource) error
	RemoveWatcher(sourceID uuid.UUID)
}

// ---------------------------------------------------------------------------
// Security: File path allow-list (§15.2 of Architecture Document)
// ---------------------------------------------------------------------------

// allowedPathPrefixes defines the directories under which log source file
// paths are permitted. Any path outside these prefixes is rejected to
// prevent an attacker from reading arbitrary host files (e.g. /etc/shadow).
//
//   - /var/log    — the standard Linux log directory (host-native paths)
//   - /host/logs  — the Docker bind-mount target (Docker mounts /var/log here)
//   - /tmp        — convenience for dev/testing only
//
// These prefixes are checked AFTER filepath.Clean normalises the path, so
// traversal payloads like "/var/log/../../etc/shadow" are neutralised.
var allowedPathPrefixes = []string{
	"/var/log",
	"/host/logs",
	"/tmp", // dev/test convenience; remove in hardened production builds
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// LogSourceService orchestrates LogSource CRUD operations with input
// validation, error wrapping, and optional watcher registry notifications.
type LogSourceService struct {
	repo     LogSourceRepository
	registry WatcherRegistry // nil when watcher pipeline is not enabled
}

// NewLogSourceService constructs a LogSourceService with the given repository.
// The watcher registry can be set later via SetRegistry once the pipeline is
// initialised (avoids circular init order issues in main.go).
func NewLogSourceService(repo LogSourceRepository) *LogSourceService {
	return &LogSourceService{repo: repo}
}

// SetRegistry attaches a WatcherRegistry to this service. After this call,
// Create and Delete will automatically start/stop watchers.
func (s *LogSourceService) SetRegistry(reg WatcherRegistry) {
	s.registry = reg
}

// Create validates and persists a new LogSource. When a WatcherRegistry is
// attached, it also starts watching the new file path.
func (s *LogSourceService) Create(ctx context.Context, ls *domain.LogSource) error {
	if err := s.validate(ls); err != nil {
		return err
	}

	// Default status to "active" when not explicitly set.
	if ls.Status == "" {
		ls.Status = domain.LogSourceStatusActive
	}
	// Default log_type to "generic" when not explicitly set.
	if ls.LogType == "" {
		ls.LogType = "generic"
	}

	if err := s.repo.Create(ctx, ls); err != nil {
		return fmt.Errorf("logSourceService.Create: %w", err)
	}

	// --- Watcher Pipeline Integration ----------------------------------------
	// If the source is active and a registry is configured, start watching.
	if s.registry != nil && ls.Status == domain.LogSourceStatusActive {
		if err := s.registry.AddWatcher(ls); err != nil {
			// Log the error but do NOT fail the HTTP request — the source is
			// already persisted. The watcher can be manually restarted later.
			slog.Error("logSourceService.Create: failed to start watcher",
				"source_id", ls.ID,
				"path", ls.FilePath,
				"err", err,
			)
		}
	}

	return nil
}

// GetByID retrieves a single LogSource by ID.
func (s *LogSourceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error) {
	ls, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("logSourceService.GetByID: %w", err)
	}
	return ls, nil
}

// List returns all registered log sources.
func (s *LogSourceService) List(ctx context.Context) ([]domain.LogSource, error) {
	sources, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("logSourceService.List: %w", err)
	}
	return sources, nil
}

// UpdateMetadata updates only the metadata fields of an existing LogSource
// (name, description). It intentionally does NOT modify the file_path or
// log_type — changing those requires a delete + re-create cycle to ensure
// the watcher pipeline is properly reconfigured.
//
// This is the handler-facing method for PATCH /api/v1/sources/:id. It
// updates the database row without breaking or restarting the active
// filesystem watcher goroutine, because the watcher operates on the
// file_path that remains unchanged.
func (s *LogSourceService) UpdateMetadata(ctx context.Context, id uuid.UUID, name *string, description *string) (*domain.LogSource, error) {
	// 1. Fetch the existing entity — confirms it exists.
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("logSourceService.UpdateMetadata: %w", err)
	}

	// 2. Apply only the metadata fields the client sent.
	changed := false
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: name must not be empty", domain.ErrValidation)
		}
		existing.Name = trimmed
		changed = true
	}
	if description != nil {
		existing.Description = description
		changed = true
	}

	// Nothing to do if no metadata fields were provided.
	if !changed {
		return existing, nil
	}

	// 3. Persist. The repo.Update writes all columns but that is fine
	//    because we only mutated metadata fields on the existing struct.
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("logSourceService.UpdateMetadata: %w", err)
	}

	slog.Info("logSourceService.UpdateMetadata: metadata updated",
		"source_id", id,
		"name", existing.Name,
	)

	return existing, nil
}

// Update validates and persists changes to an existing LogSource.
// This is the full-replacement update used internally; the HTTP PATCH
// handler should prefer UpdateMetadata for partial updates.
func (s *LogSourceService) Update(ctx context.Context, ls *domain.LogSource) error {
	if err := s.validate(ls); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, ls); err != nil {
		return fmt.Errorf("logSourceService.Update: %w", err)
	}
	return nil
}

// Delete removes a LogSource by ID. When a WatcherRegistry is attached,
// it also stops the associated watcher.
func (s *LogSourceService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("logSourceService.Delete: %w", err)
	}

	// --- Watcher Pipeline Integration ----------------------------------------
	if s.registry != nil {
		s.registry.RemoveWatcher(id)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Input Validation (§15.2 — Security Guardrails)
// ---------------------------------------------------------------------------

// validate runs input checks before creating or updating a LogSource.
// This includes the security-critical file path validation that prevents
// path traversal attacks and restricts watched directories to the allow-list.
func (s *LogSourceService) validate(ls *domain.LogSource) error {
	// --- Name validation ---
	if strings.TrimSpace(ls.Name) == "" {
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	}

	// --- File path validation ---
	if strings.TrimSpace(ls.FilePath) == "" {
		return fmt.Errorf("%w: file_path is required", domain.ErrValidation)
	}

	if err := ValidateFilePath(ls.FilePath); err != nil {
		return err
	}

	// --- Status validation ---
	if ls.Status != "" && !ls.Status.Valid() {
		return fmt.Errorf("%w: invalid status %q", domain.ErrValidation, ls.Status)
	}

	// --- Log type validation ---
	validLogTypes := map[string]bool{"generic": true, "syslog": true, "nginx": true}
	if ls.LogType != "" && !validLogTypes[ls.LogType] {
		return fmt.Errorf("%w: invalid log_type %q", domain.ErrValidation, ls.LogType)
	}

	return nil
}

// ValidateFilePath enforces the file path security guardrails defined in
// §15.2 of the architecture document. It is exported so the handler layer
// can optionally call it for early validation (fail-fast), though the
// service layer always calls it via validate() as the authoritative check.
//
// The validation performs three checks in order:
//
//  1. The path must be absolute (begins with '/').
//  2. The path is cleaned via filepath.Clean to neutralise traversal
//     sequences like "../" or "/./" and redundant separators.
//  3. The cleaned path must begin with one of the allowed directory
//     prefixes (/var/log or /host/logs). The check uses prefix + separator
//     matching to prevent partial-prefix attacks (e.g. "/var/logfake").
//
// If any check fails, a domain.ErrValidation error is returned with a
// human-readable message describing the rejection reason.
func ValidateFilePath(path string) error {
	// ---- Check 1: Must be an absolute path ----
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: file_path must be an absolute path (got %q)",
			domain.ErrValidation, path)
	}

	// ---- Check 2: Normalise to neutralise traversal sequences ----
	// filepath.Clean resolves "..", ".", double slashes, and trailing slashes.
	// For example:
	//   "/var/log/../etc/shadow" → "/etc/shadow"
	//   "/var/log/./syslog"     → "/var/log/syslog"
	//   "/var/log//auth.log"    → "/var/log/auth.log"
	cleaned := filepath.Clean(path)

	// ---- Check 3: Prefix allow-list check ----
	// We verify the cleaned path starts with an allowed prefix. To prevent
	// partial-prefix attacks we require that the cleaned path either:
	//   a) equals the prefix exactly (i.e. watching the root of the prefix), OR
	//   b) the character immediately after the prefix is a path separator.
	//
	// Without this check, an attacker could register "/var/logfake/evil"
	// which starts with "/var/log" but is NOT actually under /var/log/.
	allowed := false
	for _, prefix := range allowedPathPrefixes {
		if cleaned == prefix {
			// Exact match — the user is watching the prefix directory itself.
			allowed = true
			break
		}
		if strings.HasPrefix(cleaned, prefix+"/") {
			// The cleaned path is strictly inside the allowed directory.
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf(
			"%w: file_path must be under one of the allowed directories (%s); got %q (cleaned: %q)",
			domain.ErrValidation,
			strings.Join(allowedPathPrefixes, ", "),
			path,
			cleaned,
		)
	}

	return nil
}
