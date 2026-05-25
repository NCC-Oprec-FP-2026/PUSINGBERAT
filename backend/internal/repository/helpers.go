package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultPage    = 1
	defaultPerPage = 50
	maxPerPage     = 200
)

var ErrNotFound = errors.New("repository: not found")

type Pagination struct {
	Page    int
	PerPage int
}

func normalizePagination(page, perPage int) (limit, offset int) {
	if page < 1 {
		page = defaultPage
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return perPage, (page - 1) * perPage
}

func whereSQL(clauses []string) string {
	if len(clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(clauses, " AND ")
}

func orderLimitOffset(orderBy string, argCount int) string {
	return fmt.Sprintf(" %s LIMIT $%d OFFSET $%d", orderBy, argCount+1, argCount+2)
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func int32Ptr(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	return &v.Int32
}

func int64Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func timePtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}

func marshalJSONB(v map[string]any) ([]byte, error) {
	if v == nil {
		v = map[string]any{}
	}
	return json.Marshal(v)
}

func unmarshalJSONB(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
