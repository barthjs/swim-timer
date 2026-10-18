package store

import "errors"

var (
	// ErrInvalidName is returned when a name would address a file outside the store's folder
	// or its file name would be longer than a file system accepts.
	ErrInvalidName = errors.New("store: invalid name")
)
