package repository

import "errors"

// ErrNotFound is returned when a row is absent or soft-deleted.
var ErrNotFound = errors.New("not found")
