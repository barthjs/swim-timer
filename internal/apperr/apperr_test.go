package apperr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barthjs/swim-timer/internal/apperr"
)

func TestNewValidationError(t *testing.T) {
	t.Parallel()

	t.Run("returns the error for the field", func(t *testing.T) {
		t.Parallel()

		err := apperr.NewValidationError("email", "required")

		assert.EqualError(t, err, "validation failed")
		assert.Equal(t, apperr.FieldErrors{"email": {
			{
				Tag: "required",
			},
		}}, err.Fields)
	})
}
