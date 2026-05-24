package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/ruleengine"
)

// ---------------------------------------------------------------------------
// Repository interface (consumed by RuleService)
// ---------------------------------------------------------------------------

// RuleRepository defines the persistence contract that RuleService requires.
type RuleRepository interface {
	Create(ctx context.Context, rule *domain.Rule) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	GetByName(ctx context.Context, name string) (*domain.Rule, error)
	List(ctx context.Context) ([]domain.Rule, error)
	ListEnabled(ctx context.Context) ([]domain.Rule, error)
	Update(ctx context.Context, rule *domain.Rule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// RuleService orchestrates Rule CRUD operations with input validation.
// When a RuleLoader is attached (via SetLoader), every mutation that changes
// the "enabled" state of any rule triggers an atomic in-memory hot-reload
// of the active ruleset. This ensures the Engine always evaluates against
// the latest state without requiring a server restart.
//
// Concurrency contract:
//   - Toggle/Create/Update/Delete call reloadActiveRules which acquires the
//     RuleLoader's write lock (sync.RWMutex) for an atomic swap.
//   - The Engine's Evaluate loop acquires the read lock via GetRules.
//   - This means a hot-reload briefly blocks evaluation (write lock), and
//     evaluations block hot-reload (read locks). Both are fast (<1ms) so
//     contention is negligible.
type RuleService struct {
	repo   RuleRepository
	loader *ruleengine.RuleLoader // nil until SetLoader is called
}

// NewRuleService constructs a RuleService with the given repository.
func NewRuleService(repo RuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

// SetLoader attaches the rule engine's RuleLoader to this service. After
// this call, every mutation that affects the active ruleset will
// automatically trigger an atomic in-memory reload.
//
// This method is called once during startup wiring in main.go, after both
// the RuleService and the RuleLoader have been constructed.
func (s *RuleService) SetLoader(loader *ruleengine.RuleLoader) {
	s.loader = loader
}

// Create validates and persists a new Rule. If a RuleLoader is attached,
// the in-memory ruleset is hot-reloaded after the database write.
func (s *RuleService) Create(ctx context.Context, rule *domain.Rule) error {
	if err := s.validate(rule); err != nil {
		return err
	}

	// Default severity to "medium" when not explicitly set.
	if rule.Severity == "" {
		rule.Severity = domain.SeverityMedium
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return fmt.Errorf("ruleService.Create: %w", err)
	}

	// Hot-reload: the new rule may be enabled, so refresh the engine's
	// in-memory ruleset to include it immediately.
	s.reloadActiveRules(ctx, "Create")

	return nil
}

// GetByID retrieves a single Rule by ID.
func (s *RuleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ruleService.GetByID: %w", err)
	}
	return rule, nil
}

// List returns all rules.
func (s *RuleService) List(ctx context.Context) ([]domain.Rule, error) {
	rules, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("ruleService.List: %w", err)
	}
	return rules, nil
}

// ListEnabled returns only enabled rules.
func (s *RuleService) ListEnabled(ctx context.Context) ([]domain.Rule, error) {
	rules, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("ruleService.ListEnabled: %w", err)
	}
	return rules, nil
}

// Update validates and persists changes to an existing Rule. If a RuleLoader
// is attached, the in-memory ruleset is hot-reloaded after the database
// write.
func (s *RuleService) Update(ctx context.Context, rule *domain.Rule) error {
	if err := s.validate(rule); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, rule); err != nil {
		return fmt.Errorf("ruleService.Update: %w", err)
	}

	// Hot-reload: the update may have changed the rule's enabled flag,
	// YAML content, or severity — all of which affect evaluation.
	s.reloadActiveRules(ctx, "Update")

	return nil
}

// Toggle flips the enabled flag of an existing Rule and performs an atomic
// in-memory hot-reload of the active ruleset.
//
// Concurrency safety:
//   - The database update is authoritative. If the DB write succeeds, the
//     in-memory reload fetches the latest state from the DB.
//   - The RuleLoader.LoadFromDB method acquires a write lock on the
//     sync.RWMutex, swaps the entire slice atomically, and releases the
//     lock. Any concurrent Engine.Evaluate calls will either complete with
//     the old ruleset (if they acquired the read lock first) or block
//     briefly until the swap finishes. This guarantees that no evaluation
//     ever sees a partially-updated ruleset.
//   - If the reload fails (e.g. DB is temporarily unreachable), the old
//     in-memory ruleset remains intact — the engine never gets into a
//     broken state.
func (s *RuleService) Toggle(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	// 1. Fetch the current rule from the database.
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ruleService.Toggle: %w", err)
	}

	// 2. Flip the enabled flag.
	rule.Enabled = !rule.Enabled

	slog.Info("ruleService.Toggle: flipping rule",
		"rule_id", id,
		"rule_name", rule.Name,
		"new_enabled", rule.Enabled,
	)

	// 3. Persist the toggled state to the database.
	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("ruleService.Toggle: %w", err)
	}

	// 4. Atomic in-memory hot-reload of the active ruleset.
	//    This is the critical step that makes the toggle effective
	//    immediately without a server restart.
	s.reloadActiveRules(ctx, "Toggle")

	return rule, nil
}

// Delete removes a Rule by ID. If a RuleLoader is attached, the in-memory
// ruleset is hot-reloaded to remove the deleted rule from evaluation.
func (s *RuleService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("ruleService.Delete: %w", err)
	}

	// Hot-reload: the deleted rule must be removed from the engine's
	// in-memory ruleset immediately.
	s.reloadActiveRules(ctx, "Delete")

	return nil
}

// ---------------------------------------------------------------------------
// In-Memory Hot-Reload (§6.3 of Architecture Document)
// ---------------------------------------------------------------------------

// reloadActiveRules fetches all enabled rules from the database, parses
// their YAML content, and atomically replaces the RuleLoader's in-memory
// ruleset. This is the single function that ensures consistency between
// the database and the engine's evaluation state.
//
// If the RuleLoader is nil (not attached yet), this is a no-op.
// If the reload fails, the error is logged but not propagated — the
// caller's database mutation already succeeded, and the engine will
// continue with the previous (slightly stale) ruleset until the next
// successful reload.
//
// The 'caller' parameter is used for structured logging to identify
// which operation triggered the reload.
func (s *RuleService) reloadActiveRules(ctx context.Context, caller string) {
	if s.loader == nil {
		return
	}

	// Fetch all enabled rules from the authoritative source (database).
	enabledRules, err := s.repo.ListEnabled(ctx)
	if err != nil {
		slog.Error("ruleService.reloadActiveRules: failed to fetch enabled rules",
			"caller", caller,
			"err", err,
		)
		return
	}

	// LoadFromDB parses each rule's YAML, filters out parse failures,
	// and atomically swaps the in-memory slice under a write lock.
	if err := s.loader.LoadFromDB(enabledRules); err != nil {
		// Non-fatal: some rules had YAML parse errors but the rest loaded.
		slog.Warn("ruleService.reloadActiveRules: some rules failed to parse",
			"caller", caller,
			"err", err,
		)
	}

	slog.Info("ruleService.reloadActiveRules: hot-reload complete",
		"caller", caller,
		"active_rules", s.loader.RuleCount(),
	)
}

// ---------------------------------------------------------------------------
// Rule Seeding from YAML directory
// ---------------------------------------------------------------------------

// SeedFromDirectory reads all .yaml/.yml files from the given directory,
// parses each one into a RuleDefinition, and inserts a corresponding Rule
// row into the database if one with the same name does not already exist.
//
// This is called once at startup from main.go. It is idempotent — running
// it multiple times will not create duplicate rules.
func (s *RuleService) SeedFromDirectory(ctx context.Context, dir string) error {
	slog.Info("rule seeding: scanning directory", "dir", dir)

	var seeded, skipped, errCount int

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			slog.Warn("rule seeding: failed to read file",
				"path", path, "err", readErr)
			errCount++
			return nil
		}

		// Parse YAML to extract metadata and validate schema.
		def, parseErr := ruleengine.ParseYAML(data)
		if parseErr != nil {
			slog.Warn("rule seeding: failed to parse or validate YAML",
				"path", path, "err", parseErr)
			errCount++
			return nil
		}

		if def.Name == "" {
			slog.Warn("rule seeding: skipping file with empty name", "path", path)
			errCount++
			return nil
		}

		// Check if rule already exists by name.
		_, getErr := s.repo.GetByName(ctx, def.Name)
		if getErr == nil {
			slog.Debug("rule seeding: rule already exists, skipping",
				"name", def.Name, "path", path)
			skipped++
			return nil
		}
		if !errors.Is(getErr, domain.ErrNotFound) {
			slog.Error("rule seeding: DB lookup failed",
				"name", def.Name, "err", getErr)
			errCount++
			return nil
		}

		// Build domain.Rule from the parsed definition.
		var descPtr *string
		if def.Description != "" {
			d := def.Description
			descPtr = &d
		}

		severity := def.Severity
		if severity == "" {
			severity = domain.SeverityMedium
		}

		rule := &domain.Rule{
			Name:        def.Name,
			Description: descPtr,
			YAMLContent: string(data),
			Severity:    severity,
			Enabled:     def.Enabled,
		}

		if createErr := s.repo.Create(ctx, rule); createErr != nil {
			slog.Error("rule seeding: failed to insert rule",
				"name", def.Name, "err", createErr)
			errCount++
			return nil
		}

		slog.Info("rule seeding: rule created",
			"name", def.Name, "id", rule.ID, "severity", rule.Severity)
		seeded++
		return nil
	})

	if err != nil {
		return fmt.Errorf("rule seeding: walk directory %q: %w", dir, err)
	}

	slog.Info("rule seeding complete",
		"seeded", seeded, "skipped", skipped, "errors", errCount)
	return nil
}

// validate runs input checks before creating or updating a Rule.
func (s *RuleService) validate(rule *domain.Rule) error {
	if strings.TrimSpace(rule.Name) == "" {
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if strings.TrimSpace(rule.YAMLContent) == "" {
		return fmt.Errorf("%w: yaml_content is required", domain.ErrValidation)
	}
	if rule.Severity != "" && !rule.Severity.Valid() {
		return fmt.Errorf("%w: invalid severity %q", domain.ErrValidation, rule.Severity)
	}
	return nil
}
