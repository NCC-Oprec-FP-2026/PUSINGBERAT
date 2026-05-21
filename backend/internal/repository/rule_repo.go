package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type RuleFilter struct {
	Enabled  *bool
	Severity domain.Severity
	Search   string
	Page     int
	PerPage  int
}

type RuleRepo struct {
	db *pgxpool.Pool
}

func NewRuleRepo(db *pgxpool.Pool) *RuleRepo {
	return &RuleRepo{db: db}
}

func (r *RuleRepo) Create(ctx context.Context, rule *domain.Rule) error {
	const query = `
		INSERT INTO rules (name, description, yaml_content, severity, enabled)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at, updated_at
	`

	severity := rule.Severity
	if severity == "" {
		severity = domain.SeverityMedium
	}

	err := r.db.QueryRow(
		ctx,
		query,
		rule.Name,
		rule.Description,
		rule.YAMLContent,
		string(severity),
		rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create rule: %w", err)
	}

	rule.Severity = severity
	return nil
}

func (r *RuleRepo) GetByID(ctx context.Context, id string) (*domain.Rule, error) {
	const query = `
		SELECT id::text, name, description, yaml_content, severity, enabled, created_at, updated_at
		FROM rules
		WHERE id = $1::uuid
	`

	rule, err := scanRule(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, notFound(err)
	}
	return rule, nil
}

func (r *RuleRepo) GetByName(ctx context.Context, name string) (*domain.Rule, error) {
	const query = `
		SELECT id::text, name, description, yaml_content, severity, enabled, created_at, updated_at
		FROM rules
		WHERE name = $1
	`

	rule, err := scanRule(r.db.QueryRow(ctx, query, name))
	if err != nil {
		return nil, notFound(err)
	}
	return rule, nil
}

func (r *RuleRepo) List(ctx context.Context, filter RuleFilter) ([]domain.Rule, int64, error) {
	clauses, args := buildRuleWhere(filter)
	where := whereSQL(clauses)

	var total int64
	countQuery := "SELECT COUNT(*) FROM rules" + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rules: %w", err)
	}

	limit, offset := normalizePagination(filter.Page, filter.PerPage)
	args = append(args, limit, offset)

	query := `
		SELECT id::text, name, description, yaml_content, severity, enabled, created_at, updated_at
		FROM rules` + where + orderLimitOffset("ORDER BY updated_at DESC", len(args)-2)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	rules, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Rule, error) {
		rule, err := scanRule(row)
		if err != nil {
			return domain.Rule{}, err
		}
		return *rule, nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scan rules: %w", err)
	}

	return rules, total, nil
}

func (r *RuleRepo) Update(ctx context.Context, rule *domain.Rule) error {
	const query = `
		UPDATE rules
		SET name = $2,
			description = $3,
			yaml_content = $4,
			severity = $5,
			enabled = $6
		WHERE id = $1::uuid
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.YAMLContent,
		string(rule.Severity),
		rule.Enabled,
	).Scan(&rule.UpdatedAt)
	if err != nil {
		return notFound(fmt.Errorf("update rule: %w", err))
	}
	return nil
}

func (r *RuleRepo) SetEnabled(ctx context.Context, id string, enabled bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE rules
		SET enabled = $2
		WHERE id = $1::uuid
	`, id, enabled)
	if err != nil {
		return fmt.Errorf("set rule enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RuleRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM rules WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func buildRuleWhere(filter RuleFilter) ([]string, []any) {
	var clauses []string
	var args []any

	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		clauses = append(clauses, fmt.Sprintf("enabled = $%d", len(args)))
	}
	if filter.Severity != "" {
		args = append(args, string(filter.Severity))
		clauses = append(clauses, fmt.Sprintf("severity = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", len(args), len(args)))
	}

	return clauses, args
}

type ruleRow interface {
	Scan(dest ...any) error
}

func scanRule(row ruleRow) (*domain.Rule, error) {
	var rule domain.Rule
	var description pgtype.Text
	var severity string

	err := row.Scan(
		&rule.ID,
		&rule.Name,
		&description,
		&rule.YAMLContent,
		&severity,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	rule.Description = textPtr(description)
	rule.Severity = domain.Severity(severity)
	return &rule, nil
}
