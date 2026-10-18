// Package apperr defines sentinel errors and per-field validation errors,
// so every layer reports and matches failures the same way.
package apperr

import "errors"

var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when data violates a uniqueness constraint.
	ErrConflict = errors.New("conflict")
	// ErrVersionConflict is returned when a resource was modified concurrently
	// between being read and written (optimistic locking failure).
	ErrVersionConflict = errors.New("resource was modified concurrently")
)

// FieldError represents a single validation failure for a field.
type FieldError struct {
	Tag   string `json:"tag"`             // validation rule, such as "required" or "min"
	Param string `json:"param,omitempty"` // rule parameter, such as "8"
}

// FieldErrors groups validation failures by field name.
type FieldErrors map[string][]FieldError

// ValidationError carries structured, per-field validation failures.
type ValidationError struct {
	Fields FieldErrors
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return "validation failed"
}

// NewValidationError creates a ValidationError for a single field and rule.
func NewValidationError(field, tag string) *ValidationError {
	return &ValidationError{
		Fields: FieldErrors{
			field: {
				{
					Tag: tag,
				},
			},
		},
	}
}
