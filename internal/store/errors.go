package store

import "errors"

var (
	// ErrInvalidName is returned when a name would address a file outside the store's folder
	// or its file name would be longer than a file system accepts.
	ErrInvalidName = errors.New("store: invalid name")
	// ErrUnreadableFile is returned together with the readable items of a list when one of its files
	// cannot be read or decoded, so one broken file does not hide the others.
	ErrUnreadableFile = errors.New("store: unreadable file")
)
