// Package repository implements the data-access layer for the PUSINGBERAT
// SIEM backend. Every function in this package executes raw SQL via the pgx
// driver and returns domain structs. No business logic belongs here.
//
// All queries use parameterized placeholders ($1, $2, …) to prevent SQL
// injection. Functions accept context.Context as their first argument to
// support request-scoped timeouts and cancellation.
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

// LogSourceRepo provides CRUD operations for the log_sources table.
type LogSourceRepo struct {
	pool *pgxpool.Pool
}

// NewLogSourceRepo creates a new LogSourceRepo backed by the given connection pool.
func NewLogSourceRepo(pool *pgxpool.Pool) *LogSourceRepo {
	return &LogSourceRepo{pool: pool}
}

// Create inserts a new LogSource into the database. The ID, CreatedAt, and
// UpdatedAt fields are populated by PostgreSQL defaults and scanned back
// into the provided struct.
func (r *LogSourceRepo) Create(ctx context.Context, ls *domain.LogSource) error {
	query := `
		INSERT INTO log_sources (name, file_path, log_type, status, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		ls.Name,
		ls.FilePath,
		ls.LogType,
		ls.Status,
		ls.Description,
	).Scan(&ls.ID, &ls.CreatedAt, &ls.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("logSourceRepo.Create: %w", domain.ErrConflict)
		}
		return fmt.Errorf("logSourceRepo.Create: %w", err)
	}
	return nil
}

// GetByID retrieves a single LogSource by its UUID primary key.
func (r *LogSourceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error) {
	query := `
		SELECT id, name, file_path, log_type, status, description, created_at, updated_at
		FROM log_sources
		WHERE id = $1`

	ls := &domain.LogSource{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ls.ID,
		&ls.Name,
		&ls.FilePath,
		&ls.LogType,
		&ls.Status,
		&ls.Description,
		&ls.CreatedAt,
		&ls.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("logSourceRepo.GetByID: %w", err)
	}
	return ls, nil
}

// List returns all log sources ordered by creation time (newest first).
func (r *LogSourceRepo) List(ctx context.Context) ([]domain.LogSource, error) {
	query := `
		SELECT id, name, file_path, log_type, status, description, created_at, updated_at
		FROM log_sources
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("logSourceRepo.List: %w", err)
	}
	defer rows.Close()

	var sources []domain.LogSource
	for rows.Next() {
		var ls domain.LogSource
		if err := rows.Scan(
			&ls.ID,
			&ls.Name,
			&ls.FilePath,
			&ls.LogType,
			&ls.Status,
			&ls.Description,
			&ls.CreatedAt,
			&ls.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("logSourceRepo.List scan: %w", err)
		}
		sources = append(sources, ls)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("logSourceRepo.List rows: %w", err)
	}
	return sources, nil
}

// Update modifies the mutable fields of an existing LogSource. The updated_at
// column is refreshed automatically by the database trigger.
func (r *LogSourceRepo) Update(ctx context.Context, ls *domain.LogSource) error {
	query := `
		UPDATE log_sources
		SET name = $2, file_path = $3, log_type = $4, status = $5, description = $6
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		ls.ID,
		ls.Name,
		ls.FilePath,
		ls.LogType,
		ls.Status,
		ls.Description,
	).Scan(&ls.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("logSourceRepo.Update: %w", domain.ErrConflict)
		}
		return fmt.Errorf("logSourceRepo.Update: %w", err)
	}
	return nil
}

// Delete removes a LogSource by ID. Associated events are cascade-deleted
// by the ON DELETE CASCADE foreign key constraint in PostgreSQL.
func (r *LogSourceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM log_sources WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("logSourceRepo.Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
