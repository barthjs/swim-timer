package handlers

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var v *validator.Validate

func init() {
	v = validator.New(validator.WithRequiredStructEnabled())
}

// Error represents a validation error with rule and optional parameters
type Error struct {
	Rule   string            `json:"rule"`
	Params map[string]string `json:"params,omitempty"`
}

// Errors holds field-level validation errors
type Errors map[string]Error

// validate validates a struct and returns validation errors with rule details
// Returns nil if validation passes.
func validate(data interface{}) Errors {
	err := v.Struct(data)
	if err == nil {
		return nil
	}

	errs := make(Errors)
	for _, err := range err.(validator.ValidationErrors) {
		rule := err.Tag()
		params := extractParams(err)

		namespace := err.Namespace()
		field := namespace
		if i := strings.Index(namespace, "."); i != -1 {
			field = namespace[i+1:]
		}

		errs[field] = Error{
			Rule:   rule,
			Params: params,
		}
	}

	return errs
}

// extractParams extracts validation parameters from the error tag
// For example: min=1 -> {min: "1"}, max=255 -> {max: "255"}
func extractParams(err validator.FieldError) map[string]string {
	params := make(map[string]string)
	tag := err.Tag()
	param := err.Param()

	switch tag {
	case "min", "max", "gte", "lte":
		if param != "" {
			params[tag] = param
		}
	case "oneof":
		if param != "" {
			params["values"] = strings.Join(strings.Fields(param), ", ")
		}
	case "len":
		if param != "" {
			params["length"] = param
		}
	}

	return params
}
