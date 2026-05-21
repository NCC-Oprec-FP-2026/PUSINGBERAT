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

type LogSourceFilter struct {
	Status  domain.LogSourceStatus
	LogType domain.LogSourceType
	Search  string
	Page    int
	PerPage int
}

type LogSourceRepo struct {
	db *pgxpool.Pool
}

func NewLogSourceRepo(db *pgxpool.Pool) *LogSourceRepo {
	return &LogSourceRepo{db: db}
}

func (r *LogSourceRepo) Create(ctx context.Context, source *domain.LogSource) error {
	const query = `
		INSERT INTO log_sources (name, file_path, log_type, status, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at, updated_at
	`

	logType := source.LogType
	if logType == "" {
		logType = domain.LogSourceTypeGeneric
	}

	status := source.Status
	if status == "" {
		status = domain.LogSourceStatusActive
	}

	err := r.db.QueryRow(
		ctx,
		query,
		source.Name,
		source.FilePath,
		string(logType),
		string(status),
		source.Description,
	).Scan(&source.ID, &source.CreatedAt, &source.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create log source: %w", err)
	}

	source.LogType = logType
	source.Status = status
	return nil
}

func (r *LogSourceRepo) GetByID(ctx context.Context, id string) (*domain.LogSource, error) {
	const query = `
		SELECT id::text, name, file_path, log_type, status, description, created_at, updated_at
		FROM log_sources
		WHERE id = $1::uuid
	`

	source, err := scanLogSource(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, notFound(err)
	}
	return source, nil
}

func (r *LogSourceRepo) GetByFilePath(ctx context.Context, filePath string) (*domain.LogSource, error) {
	const query = `
		SELECT id::text, name, file_path, log_type, status, description, created_at, updated_at
		FROM log_sources
		WHERE file_path = $1
	`

	source, err := scanLogSource(r.db.QueryRow(ctx, query, filePath))
	if err != nil {
		return nil, notFound(err)
	}
	return source, nil
}

func (r *LogSourceRepo) List(ctx context.Context, filter LogSourceFilter) ([]domain.LogSource, int64, error) {
	clauses, args := buildLogSourceWhere(filter)
	where := whereSQL(clauses)

	var total int64
	countQuery := "SELECT COUNT(*) FROM log_sources" + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count log sources: %w", err)
	}

	limit, offset := normalizePagination(filter.Page, filter.PerPage)
	args = append(args, limit, offset)

	query := `
		SELECT id::text, name, file_path, log_type, status, description, created_at, updated_at
		FROM log_sources` + where + orderLimitOffset("ORDER BY created_at DESC", len(args)-2)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list log sources: %w", err)
	}
	defer rows.Close()

	sources, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LogSource, error) {
		source, err := scanLogSource(row)
		if err != nil {
			return domain.LogSource{}, err
		}
		return *source, nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scan log sources: %w", err)
	}

	return sources, total, nil
}

func (r *LogSourceRepo) Update(ctx context.Context, source *domain.LogSource) error {
	const query = `
		UPDATE log_sources
		SET name = $2,
			file_path = $3,
			log_type = $4,
			status = $5,
			description = $6
		WHERE id = $1::uuid
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		source.ID,
		source.Name,
		source.FilePath,
		string(source.LogType),
		string(source.Status),
		source.Description,
	).Scan(&source.UpdatedAt)
	if err != nil {
		return notFound(fmt.Errorf("update log source: %w", err))
	}
	return nil
}

func (r *LogSourceRepo) UpdateStatus(ctx context.Context, id string, status domain.LogSourceStatus) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE log_sources
		SET status = $2
		WHERE id = $1::uuid
	`, id, string(status))
	if err != nil {
		return fmt.Errorf("update log source status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *LogSourceRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM log_sources WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete log source: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func buildLogSourceWhere(filter LogSourceFilter) ([]string, []any) {
	var clauses []string
	var args []any

	if filter.Status != "" {
		args = append(args, string(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.LogType != "" {
		args = append(args, string(filter.LogType))
		clauses = append(clauses, fmt.Sprintf("log_type = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR file_path ILIKE $%d)", len(args), len(args)))
	}

	return clauses, args
}

type logSourceRow interface {
	Scan(dest ...any) error
}

func scanLogSource(row logSourceRow) (*domain.LogSource, error) {
	var source domain.LogSource
	var logType string
	var status string
	var description pgtype.Text

	err := row.Scan(
		&source.ID,
		&source.Name,
		&source.FilePath,
		&logType,
		&status,
		&description,
		&source.CreatedAt,
		&source.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	source.LogType = domain.LogSourceType(logType)
	source.Status = domain.LogSourceStatus(status)
	source.Description = textPtr(description)
	return &source, nil
}
