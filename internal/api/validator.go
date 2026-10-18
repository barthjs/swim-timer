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
var v *validator.Validate

// init builds the validator once, since it caches the metadata of every struct it checks.
func init() {
	v = validator.New(validator.WithRequiredStructEnabled(), validator.WithTagNameFuncBlankOmit())

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

	// The future tag compares a time with the clock of the API, which validate passes in the context
	if err := v.RegisterValidationCtx("future", validateFuture); err != nil {
		panic(err)
	}

	// The S3 fields depend on the driver one struct level up, which a required_if tag cannot reach,
	// since it only sees sibling fields. A struct-level validation reports them as per-field 422 errors.
	v.RegisterStructValidation(validateStorageSettings, StorageSettings{})

	// The validator has no rule for a positive integer in a string, which the lanes of a preset are keyed by
	if err := v.RegisterValidation("lane", validateLaneNumber); err != nil {
		panic(err)
	}

	// The validator has no rule for a comma-separated list of networks, which the network settings are written as
	if err := v.RegisterValidation("networks", validateNetworks); err != nil {
		panic(err)
	}

	// The origins and the credentials of CORS follow the rules of the config package, see ServerSettings
	if err := v.RegisterValidation("origin", validateOrigin); err != nil {
		panic(err)
	}
	v.RegisterStructValidation(validateServerSettings, ServerSettings{})
}

// validationClockKey is the context key under which validate hands the clock of the API to the rules,
// so a rule that compares with the current time follows the clock of the tests.
type validationClockKey struct{}

// validateFuture reports whether a time lies in the future of the clock that validate passes in the context.
// It is the "future" tag, such as for an expiry that would otherwise be over from the start.
// Without a clock in the context it fails, since the validator is only called through validate.
func validateFuture(ctx context.Context, fl validator.FieldLevel) bool {
	now, ok := ctx.Value(validationClockKey{}).(func() time.Time)
	if !ok {
		return false
	}
	t, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	return t.After(now())
}

// FieldError and FieldErrors are aliases of the apperr types, so handlers build them with short names
// and services return the same structures through apperr.ValidationError.
type (
	// FieldError is a single validation failure of a field.
	FieldError = apperr.FieldError
	// FieldErrors groups validation failures by field name.
	FieldErrors = apperr.FieldErrors
)

// validate validates a struct against the clock of the API and returns structured field errors. Any other error
// of the validator, such as for a value that is not a struct, is a programming error and returned as an error.
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
