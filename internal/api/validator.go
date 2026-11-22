package api

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/barthjs/swim-timer/internal/apperr"
)

// v validates the request DTOs.
var v = newValidator()

// newValidator builds the validator once, since it caches the metadata of every struct it checks.
// The resources register their own rules in an init function of their file.
func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled(), validator.WithTagNameFuncBlankOmit())

	// Report errors under the wire-format field name: the JSON tag for request bodies,
	// falling back to the form tag for multipart/form uploads.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "" {
			name = strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
		}
		// An embedded struct without a name of its own is flattened into its parent on the wire
		if fld.Anonymous && name == "" {
			return ""
		}
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	// The future and past tags compare a time with the clock of the API, which validate passes in the context
	if err := v.RegisterValidationCtx("future", validateFuture); err != nil {
		panic(err)
	}
	if err := v.RegisterValidationCtx("past", validatePast); err != nil {
		panic(err)
	}

	return v
}

// validationClockKey is the context key under which validate hands the clock of the API to the rules,
// so a rule that compares with the current time follows the clock of the tests.
type validationClockKey struct{}

// validateFuture is the "future" tag, which reports whether a time lies in the future of the clock that validate passes.
func validateFuture(ctx context.Context, fl validator.FieldLevel) bool {
	now, ok := ctx.Value(validationClockKey{}).(func() time.Time)
	if !ok {
		return false
	}
	t, ok := fieldTime(fl)
	if !ok {
		return false
	}
	return t.After(now())
}

// validatePast is the "past" tag, which reports whether a time does not lie in the future of the clock that validate passes.
func validatePast(ctx context.Context, fl validator.FieldLevel) bool {
	now, ok := ctx.Value(validationClockKey{}).(func() time.Time)
	if !ok {
		return false
	}
	t, ok := fieldTime(fl)
	if !ok {
		return false
	}
	return !t.After(now())
}

// fieldTime returns the time a field holds as a time.Time or an RFC 3339 string.
func fieldTime(fl validator.FieldLevel) (time.Time, bool) {
	switch value := fl.Field().Interface().(type) {
	case time.Time:
		return value, true
	case string:
		t, err := time.Parse(time.RFC3339, value)
		return t, err == nil
	default:
		return time.Time{}, false
	}
}

// FieldError and FieldErrors are aliases of the apperr types, so handlers build them with short names
// and services return the same structures through apperr.ValidationError.
type (
	// FieldError is a single validation failure of a field.
	FieldError = apperr.FieldError
	// FieldErrors groups validation failures by field name.
	FieldErrors = apperr.FieldErrors
)

// validate validates a struct against the clock of the API and returns its field errors,
// or an error for a value the validator cannot check.
func (a *API) validate(ctx context.Context, data any) (FieldErrors, error) {
	err := v.StructCtx(context.WithValue(ctx, validationClockKey{}, a.now), data)
	if err == nil {
		return nil, nil
	}

	validationErrors, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return nil, fmt.Errorf("validating %T: %w", data, err)
	}

	result := make(FieldErrors)
	for _, validationError := range validationErrors {
		field := fieldPath(validationError.Namespace())

		result[field] = append(result[field], FieldError{
			Tag:   validationError.Tag(),
			Param: validationError.Param(),
		})
	}

	return result, nil
}

// fieldPath turns a validator namespace into the dotted field path.
func fieldPath(namespace string) string {
	_, rest, found := strings.Cut(namespace, ".")
	if !found {
		return namespace
	}
	return rest
}
