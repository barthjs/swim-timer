package store

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidName is returned when a name would address a file outside the store's folder
	// or its file name would be longer than a file system accepts.
	ErrInvalidName = errors.New("store: invalid name")
	// ErrUnreadableFile is returned together with the readable items of a list
	// when one of its files cannot be read or decoded, so one broken file does not hide the others.
	ErrUnreadableFile = errors.New("store: unreadable file")
)

// DataFileError reports a stored file whose content cannot be read.
// Line and Column are empty if the error concerns the whole file.
type DataFileError struct {
	File   string
	Line   int
	Column string
	Err    error
}

// Error describes the problem with the file and its position in it.
func (e *DataFileError) Error() string {
	position := e.File
	if e.Line > 0 {
		position += fmt.Sprintf(", line %d", e.Line)
	}
	if e.Column != "" {
		position += ", column " + e.Column
	}
	return fmt.Sprintf("%s: %v", position, e.Err)
}

// Unwrap returns the underlying error.
func (e *DataFileError) Unwrap() error {
	return e.Err
}
