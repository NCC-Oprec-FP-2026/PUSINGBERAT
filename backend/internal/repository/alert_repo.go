package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

type AlertFilter struct {
	Severities   []domain.Severity
	Acknowledged *bool
	From         *time.Time
	To           *time.Time
	Page         int
	PerPage      int
}

type AlertRepo struct {
	db *pgxpool.Pool
}

func NewAlertRepo(db *pgxpool.Pool) *AlertRepo {
	return &AlertRepo{db: db}
}

func (r *AlertRepo) Create(ctx context.Context, alert *domain.Alert) error {
	if alert.Acknowledged && alert.AcknowledgedAt == nil {
		now := time.Now().UTC()
		alert.AcknowledgedAt = &now
	}
	if alert.Severity == "" {
		alert.Severity = domain.SeverityMedium
	}

	const query = `
		INSERT INTO alerts (
			rule_id,
			rule_name,
			event_id,
			log_source_id,
			severity,
			title,
			description,
			raw_line,
			acknowledged,
			acknowledged_at,
			discord_sent
		)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id::text, triggered_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		nullableString(alert.RuleID),
		alert.RuleName,
		nullableInt64(alert.EventID),
		nullableString(alert.LogSourceID),
		string(alert.Severity),
		alert.Title,
		alert.Description,
		alert.RawLine,
		alert.Acknowledged,
		alert.AcknowledgedAt,
		alert.DiscordSent,
	).Scan(&alert.ID, &alert.TriggeredAt)
	if err != nil {
		return fmt.Errorf("create alert: %w", err)
	}

	return nil
}

func (r *AlertRepo) GetByID(ctx context.Context, id string) (*domain.Alert, error) {
	const query = `
		SELECT id::text, rule_id::text, rule_name, event_id, log_source_id::text,
			severity, title, description, raw_line, triggered_at, acknowledged,
			acknowledged_at, discord_sent
		FROM alerts
		WHERE id = $1::uuid
	`

	alert, err := scanAlert(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, notFound(err)
	}
	return alert, nil
}

func (r *AlertRepo) List(ctx context.Context, filter AlertFilter) ([]domain.Alert, int64, error) {
	clauses, args := buildAlertWhere(filter)
	where := whereSQL(clauses)

	var total int64
	countQuery := "SELECT COUNT(*) FROM alerts" + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}

	limit, offset := normalizePagination(filter.Page, filter.PerPage)
	args = append(args, limit, offset)

	query := `
		SELECT id::text, rule_id::text, rule_name, event_id, log_source_id::text,
			severity, title, description, raw_line, triggered_at, acknowledged,
			acknowledged_at, discord_sent
		FROM alerts` + where + orderLimitOffset("ORDER BY triggered_at DESC", len(args)-2)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	alerts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Alert, error) {
		alert, err := scanAlert(row)
		if err != nil {
			return domain.Alert{}, err
		}
		return *alert, nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scan alerts: %w", err)
	}

	return alerts, total, nil
}

func (r *AlertRepo) Acknowledge(ctx context.Context, id string) (*domain.Alert, error) {
	const query = `
		UPDATE alerts
		SET acknowledged = true,
			acknowledged_at = COALESCE(acknowledged_at, NOW())
		WHERE id = $1::uuid
		RETURNING id::text, rule_id::text, rule_name, event_id, log_source_id::text,
			severity, title, description, raw_line, triggered_at, acknowledged,
			acknowledged_at, discord_sent
	`

	alert, err := scanAlert(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, notFound(err)
	}
	return alert, nil
}

func (r *AlertRepo) MarkDiscordSent(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE alerts
		SET discord_sent = true
		WHERE id = $1::uuid
	`, id)
	if err != nil {
		return fmt.Errorf("mark alert discord sent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AlertRepo) ListDiscordPending(ctx context.Context, limit int) ([]domain.Alert, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > maxPerPage {
		limit = maxPerPage
	}

	rows, err := r.db.Query(ctx, `
		SELECT id::text, rule_id::text, rule_name, event_id, log_source_id::text,
			severity, title, description, raw_line, triggered_at, acknowledged,
			acknowledged_at, discord_sent
		FROM alerts
		WHERE discord_sent = false
		ORDER BY triggered_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list discord pending alerts: %w", err)
	}
	defer rows.Close()

	alerts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Alert, error) {
		alert, err := scanAlert(row)
		if err != nil {
			return domain.Alert{}, err
		}
		return *alert, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan discord pending alerts: %w", err)
	}

	return alerts, nil
}

func (r *AlertRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM alerts WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func buildAlertWhere(filter AlertFilter) ([]string, []any) {
	var clauses []string
	var args []any

	if len(filter.Severities) > 0 {
		placeholders := make([]string, 0, len(filter.Severities))
		for _, severity := range filter.Severities {
			if severity == "" {
				continue
			}
			args = append(args, string(severity))
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		if len(placeholders) > 0 {
			clauses = append(clauses, "severity IN ("+strings.Join(placeholders, ", ")+")")
		}
	}
	if filter.Acknowledged != nil {
		args = append(args, *filter.Acknowledged)
		clauses = append(clauses, fmt.Sprintf("acknowledged = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("triggered_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("triggered_at <= $%d", len(args)))
	}

	return clauses, args
}

type alertRow interface {
	Scan(dest ...any) error
}

func scanAlert(row alertRow) (*domain.Alert, error) {
	var alert domain.Alert
	var ruleID pgtype.Text
	var eventID pgtype.Int8
	var logSourceID pgtype.Text
	var severity string
	var description pgtype.Text
	var rawLine pgtype.Text
	var acknowledgedAt pgtype.Timestamptz

	err := row.Scan(
		&alert.ID,
		&ruleID,
		&alert.RuleName,
		&eventID,
		&logSourceID,
		&severity,
		&alert.Title,
		&description,
		&rawLine,
		&alert.TriggeredAt,
		&alert.Acknowledged,
		&acknowledgedAt,
		&alert.DiscordSent,
	)
	if err != nil {
		return nil, err
	}

	alert.RuleID = textPtr(ruleID)
	alert.EventID = int64Ptr(eventID)
	alert.LogSourceID = textPtr(logSourceID)
	alert.Severity = domain.Severity(severity)
	alert.Description = textPtr(description)
	alert.RawLine = textPtr(rawLine)
	alert.AcknowledgedAt = timePtr(acknowledgedAt)
	return &alert, nil
}

func nullableString(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}
