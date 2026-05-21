package domain

import "errors"

// Sentinel errors used across the repository and service layers.
// Handlers inspect these to choose the correct HTTP status code.
var (
	// ErrNotFound indicates a requested entity does not exist.
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates a uniqueness constraint violation (e.g. duplicate file_path).
	ErrConflict = errors.New("entity already exists")

	// ErrValidation indicates a client-provided value failed input validation.
	ErrValidation = errors.New("validation error")
)
