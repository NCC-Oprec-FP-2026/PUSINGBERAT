package repository

import (
	"errors"
)

var ErrNotFound = errors.New("repository: not found")

type Pagination struct {
	Page    int
	PerPage int
}
