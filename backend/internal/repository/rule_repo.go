package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// RuleRepo provides CRUD operations for the rules table.
type RuleRepo struct {
	pool *pgxpool.Pool
}

// NewRuleRepo creates a new RuleRepo backed by the given connection pool.
func NewRuleRepo(pool *pgxpool.Pool) *RuleRepo {
	return &RuleRepo{pool: pool}
}

// Create inserts a new Rule into the database. Server-generated fields
// (id, created_at, updated_at) are scanned back into the provided struct.
func (r *RuleRepo) Create(ctx context.Context, rule *domain.Rule) error {
	query := `
		INSERT INTO rules (name, description, yaml_content, severity, enabled)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		rule.Name,
		rule.Description,
		rule.YAMLContent,
		rule.Severity,
		rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("ruleRepo.Create: %w", domain.ErrConflict)
		}
		return fmt.Errorf("ruleRepo.Create: %w", err)
	}
	return nil
}

// GetByID retrieves a single Rule by its UUID primary key.
func (r *RuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	query := `
		SELECT id, name, description, yaml_content, severity, enabled, created_at, updated_at
		FROM rules
		WHERE id = $1`

	rule := &domain.Rule{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&rule.YAMLContent,
		&rule.Severity,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("ruleRepo.GetByID: %w", err)
	}
	return rule, nil
}

// GetByName retrieves a single Rule by its unique name.
// Returns domain.ErrNotFound if no rule with that name exists.
func (r *RuleRepo) GetByName(ctx context.Context, name string) (*domain.Rule, error) {
	query := `
		SELECT id, name, description, yaml_content, severity, enabled, created_at, updated_at
		FROM rules
		WHERE name = $1`

	rule := &domain.Rule{}
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&rule.YAMLContent,
		&rule.Severity,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("ruleRepo.GetByName: %w", err)
	}
	return rule, nil
}

// List returns all rules ordered by creation time (newest first).
func (r *RuleRepo) List(ctx context.Context) ([]domain.Rule, error) {
	query := `
		SELECT id, name, description, yaml_content, severity, enabled, created_at, updated_at
		FROM rules
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ruleRepo.List: %w", err)
	}
	defer rows.Close()

	var rules []domain.Rule
	for rows.Next() {
		var rule domain.Rule
		if err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.YAMLContent,
			&rule.Severity,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ruleRepo.List scan: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ruleRepo.List rows: %w", err)
	}
	return rules, nil
}

// Update replaces the mutable fields of an existing Rule. The updated_at
// column is refreshed automatically by the database trigger.
func (r *RuleRepo) Update(ctx context.Context, rule *domain.Rule) error {
	query := `
		UPDATE rules
		SET name = $2, description = $3, yaml_content = $4, severity = $5, enabled = $6
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.YAMLContent,
		rule.Severity,
		rule.Enabled,
	).Scan(&rule.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("ruleRepo.Update: %w", domain.ErrConflict)
		}
		return fmt.Errorf("ruleRepo.Update: %w", err)
	}
	return nil
}

// Delete removes a Rule by ID. Associated alerts keep their rule_name due
// to the ON DELETE SET NULL foreign key (rule_id becomes NULL).
func (r *RuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM rules WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ruleRepo.Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListEnabled returns only enabled rules (used by the rule engine at load time).
func (r *RuleRepo) ListEnabled(ctx context.Context) ([]domain.Rule, error) {
	query := `
		SELECT id, name, description, yaml_content, severity, enabled, created_at, updated_at
		FROM rules
		WHERE enabled = true
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ruleRepo.ListEnabled: %w", err)
	}
	defer rows.Close()

	var rules []domain.Rule
	for rows.Next() {
		var rule domain.Rule
		if err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.YAMLContent,
			&rule.Severity,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ruleRepo.ListEnabled scan: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ruleRepo.ListEnabled rows: %w", err)
	}
	return rules, nil
}
