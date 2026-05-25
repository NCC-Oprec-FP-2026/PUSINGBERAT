package ruleengine

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/goccy/go-yaml"
)

const namedErrorFormat = "%s: %v"

// ---------------------------------------------------------------------------
// RuleLoader — thread-safe, hot-swappable rule store
// ---------------------------------------------------------------------------

// RuleLoader maintains an in-memory slice of parsed RuleDefinitions that the
// Engine evaluates against incoming events.  It is safe for concurrent use:
//
//   - LoadFromDirectory reads YAML files from disk (used at startup).
//   - LoadFromDB replaces the in-memory ruleset atomically (used when a
//     rule is created/updated/deleted via the API).
//   - GetRules returns a snapshot of the current ruleset under a read lock.
//
// Hot-swapping is achieved via sync.RWMutex: writers (Load*) take a write
// lock and atomically swap the entire slice; readers (GetRules) take a read
// lock and receive a consistent snapshot.  The engine never needs to restart
// when rules change.
type RuleLoader struct {
	rules []*domain.RuleDefinition
	mu    sync.RWMutex
}

// NewRuleLoader returns an empty RuleLoader ready for use.
func NewRuleLoader() *RuleLoader {
	return &RuleLoader{}
}

// ---------------------------------------------------------------------------
// Read path
// ---------------------------------------------------------------------------

// GetRules returns a snapshot of the currently loaded rules.  The returned
// slice must not be mutated by callers.
func (l *RuleLoader) GetRules() []*domain.RuleDefinition {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a shallow copy so callers cannot corrupt the internal slice.
	out := make([]*domain.RuleDefinition, len(l.rules))
	copy(out, l.rules)
	return out
}

// RuleCount returns the number of currently loaded rules.
func (l *RuleLoader) RuleCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.rules)
}

// ---------------------------------------------------------------------------
// Write paths
// ---------------------------------------------------------------------------

// LoadFromDefinitions atomically replaces the in-memory rule set with the
// provided slice.  This is the primary hot-swap entry point used by
// RuleService after a CRUD operation.
func (l *RuleLoader) LoadFromDefinitions(defs []*domain.RuleDefinition) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rules = defs
	slog.Info("rule loader: rules replaced",
		"count", len(defs),
	)
}

// LoadFromDB parses the YAMLContent of each domain.Rule and atomically
// replaces the in-memory ruleset with the successfully parsed definitions.
// Rules that fail YAML parsing are logged and skipped — a single bad rule
// must never prevent the remaining rules from loading.
func (l *RuleLoader) LoadFromDB(dbRules []domain.Rule) error {
	var defs []*domain.RuleDefinition
	var errs []string

	for i := range dbRules {
		r := &dbRules[i]

		if !r.Enabled {
			slog.Debug("rule loader: skipping disabled rule",
				"name", r.Name,
				"id", r.ID,
			)
			continue
		}

		def, err := ParseYAML([]byte(r.YAMLContent))
		if err != nil {
			errs = append(errs, fmt.Sprintf(namedErrorFormat, r.Name, err))
			slog.Warn("rule loader: failed to parse rule YAML",
				"name", r.Name,
				"id", r.ID,
				"err", err,
			)
			continue
		}

		// Ensure the parsed definition's enabled flag agrees with the DB.
		def.Enabled = r.Enabled
		defs = append(defs, def)
	}

	l.mu.Lock()
	l.rules = defs
	l.mu.Unlock()

	slog.Info("rule loader: loaded rules from DB",
		"loaded", len(defs),
		"skipped", len(errs),
	)

	if len(errs) > 0 {
		return fmt.Errorf("rule loader: %d rule(s) failed to parse: %s",
			len(errs), strings.Join(errs, "; "))
	}
	return nil
}

// LoadFromDirectory walks the given directory for .yaml / .yml files, parses
// each one, and atomically replaces the in-memory ruleset.  This is the
// startup path that reads seed rules from the rules/ directory.
//
// Files that fail to parse are logged and skipped.
func (l *RuleLoader) LoadFromDirectory(dir string) error {
	var defs []*domain.RuleDefinition
	var errs []string

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
			errs = append(errs, fmt.Sprintf(namedErrorFormat, filepath.Base(path), readErr))
			slog.Warn("rule loader: failed to read file",
				"path", path,
				"err", readErr,
			)
			return nil // skip file, continue walking
		}

		def, parseErr := ParseYAML(data)
		if parseErr != nil {
			errs = append(errs, fmt.Sprintf(namedErrorFormat, filepath.Base(path), parseErr))
			slog.Warn("rule loader: failed to parse YAML",
				"path", path,
				"err", parseErr,
			)
			return nil // skip file, continue walking
		}

		if !def.Enabled {
			slog.Debug("rule loader: skipping disabled rule from file",
				"path", path,
				"id", def.ID,
			)
			return nil
		}

		defs = append(defs, def)
		slog.Debug("rule loader: loaded rule from file",
			"path", path,
			"id", def.ID,
			"name", def.Name,
		)
		return nil
	})

	if err != nil {
		return fmt.Errorf("rule loader: walk directory %q: %w", dir, err)
	}

	l.mu.Lock()
	l.rules = defs
	l.mu.Unlock()

	slog.Info("rule loader: loaded rules from directory",
		"dir", dir,
		"loaded", len(defs),
		"skipped", len(errs),
	)

	if len(errs) > 0 {
		return fmt.Errorf("rule loader: %d file(s) failed: %s",
			len(errs), strings.Join(errs, "; "))
	}
	return nil
}

// ---------------------------------------------------------------------------
// YAML parsing + validation
// ---------------------------------------------------------------------------

// ParseYAML unmarshals raw YAML bytes into a RuleDefinition and validates
// that all required fields are present and well-formed.
func ParseYAML(data []byte) (*domain.RuleDefinition, error) {
	var def domain.RuleDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	if err := validateDefinition(&def); err != nil {
		return nil, err
	}

	return &def, nil
}

// validateDefinition checks that all required fields are populated and that
// operators / severity values are valid.
func validateDefinition(def *domain.RuleDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("rule validation: 'id' is required")
	}
	if def.Name == "" {
		return fmt.Errorf("rule validation: 'name' is required")
	}
	if !def.Severity.Valid() {
		return fmt.Errorf("rule validation: invalid severity %q", def.Severity)
	}
	if len(def.Conditions) == 0 {
		return fmt.Errorf("rule validation: at least one condition is required")
	}

	for i, cond := range def.Conditions {
		if cond.Field == "" {
			return fmt.Errorf("rule validation: condition[%d].field is required", i)
		}
		if !isValidOperator(cond.Operator) {
			return fmt.Errorf("rule validation: condition[%d].operator %q is not supported", i, cond.Operator)
		}
		// Value can be empty for some operators (e.g., regex "^$" matching empty strings),
		// but for most operators an empty value is likely a misconfiguration.
		// We do not enforce non-empty here to keep the schema flexible.
	}

	if def.Threshold != nil {
		if def.Threshold.Count < 1 {
			return fmt.Errorf("rule validation: threshold.count must be >= 1, got %d", def.Threshold.Count)
		}
		if def.Threshold.WindowSeconds < 1 {
			return fmt.Errorf("rule validation: threshold.window_seconds must be >= 1, got %d", def.Threshold.WindowSeconds)
		}
	}

	if def.Alert.Title == "" {
		return fmt.Errorf("rule validation: alert.title is required")
	}

	return nil
}

// isValidOperator returns true if the operator string is one of the
// supported comparison operators.
func isValidOperator(op string) bool {
	switch op {
	case "equals", "contains", "starts_with", "ends_with", "regex", "gt", "lt":
		return true
	}
	return false
}
