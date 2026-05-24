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
type RuleService struct {
	repo RuleRepository
}

// NewRuleService constructs a RuleService with the given repository.
func NewRuleService(repo RuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

// Create validates and persists a new Rule.
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

// Update validates and persists changes to an existing Rule.
func (s *RuleService) Update(ctx context.Context, rule *domain.Rule) error {
	if err := s.validate(rule); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, rule); err != nil {
		return fmt.Errorf("ruleService.Update: %w", err)
	}
	return nil
}

// Toggle flips the enabled flag of an existing Rule.
func (s *RuleService) Toggle(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ruleService.Toggle: %w", err)
	}
	rule.Enabled = !rule.Enabled
	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("ruleService.Toggle: %w", err)
	}
	return rule, nil
}

// Delete removes a Rule by ID.
func (s *RuleService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("ruleService.Delete: %w", err)
	}
	return nil
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
