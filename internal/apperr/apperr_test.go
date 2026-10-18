package apperr_test

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apperr"
)

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	sentinels := map[string]error{
		"not found":        apperr.ErrNotFound,
		"conflict":         apperr.ErrConflict,
		"version conflict": apperr.ErrVersionConflict,
	}

	for name, sentinel := range sentinels {
		t.Run("matches only "+name+" through wrapping", func(t *testing.T) {
			t.Parallel()

			wrapped := fmt.Errorf("loading item: %w", sentinel)

			assert.ErrorIs(t, wrapped, sentinel)
			for otherName, other := range sentinels {
				if otherName != name {
					assert.NotErrorIs(t, wrapped, other)
				}
			}
		})
	}
}

func TestFieldError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field apperr.FieldError
		want  string
	}{
		{
			name: "omits an empty parameter",
			field: apperr.FieldError{
				Tag: "required",
			},
			want: `{"tag":"required"}`,
		},
		{
			name: "includes the parameter",
			field: apperr.FieldError{
				Tag:   "min",
				Param: "8",
			},
			want: `{"tag":"min","param":"8"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.field)

			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

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

	t.Run("is found through wrapping", func(t *testing.T) {
		t.Parallel()

		wrapped := fmt.Errorf("saving item: %w", apperr.NewValidationError("name", "max"))

		validation, ok := errors.AsType[*apperr.ValidationError](wrapped)

		require.True(t, ok)
		assert.Equal(t, "max", validation.Fields["name"][0].Tag)
		assert.False(t, errors.Is(wrapped, apperr.ErrNotFound))
	})
}
