package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestValidateFuture(t *testing.T) {
	t.Parallel()
	f := apitest.Faker(t)
	clock := f.Date()
	a := &API{
		now: func() time.Time {
			return clock
		},
	}

	type request struct {
		ExpiresAt *time.Time `json:"expiresAt" validate:"omitempty,future"`
	}

	tests := []struct {
		name      string
		expiresAt *time.Time
		valid     bool
	}{
		{
			name:      "accepts a missing time",
			expiresAt: nil,
			valid:     true,
		},
		{
			name:      "accepts a time after the clock",
			expiresAt: new(clock.Add(time.Duration(f.IntRange(1, 1000)) * time.Hour)),
			valid:     true,
		},
		{
			name:      "rejects the time of the clock",
			expiresAt: new(clock),
			valid:     false,
		},
		{
			name:      "rejects a time before the clock",
			expiresAt: new(clock.Add(-time.Duration(f.IntRange(1, 1000)) * time.Hour)),
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errs, err := a.validate(t.Context(), request{
				ExpiresAt: tt.expiresAt,
			})
			require.NoError(t, err)
			if tt.valid {
				assert.Nil(t, errs)
				return
			}
			assert.Equal(t, FieldErrors{
				"expiresAt": {
					{
						Tag: "future",
					},
				},
			}, errs)
		})
	}
}

func TestAPI_Validate(t *testing.T) {
	t.Parallel()

	t.Run("returns no errors for a valid struct", func(t *testing.T) {
		t.Parallel()
		type request struct {
			Name string `json:"name" validate:"required"`
		}

		errs, err := newBareAPI().validate(t.Context(), request{
			Name: apitest.Faker(t).Word(),
		})

		require.NoError(t, err)
		assert.Nil(t, errs)
	})

	t.Run("reports the invalid fields", func(t *testing.T) {
		t.Parallel()
		type request struct {
			Name string `json:"name" validate:"required"`
		}

		errs, err := newBareAPI().validate(t.Context(), request{})

		require.NoError(t, err)
		assert.Equal(t, FieldErrors{
			"name": {
				{
					Tag: "required",
				},
			},
		}, errs)
	})

	t.Run("names fields by their wire name", func(t *testing.T) {
		t.Parallel()
		type request struct {
			JSON     string `json:"json_name" validate:"required"`
			Options  string `json:"options_name,omitempty" validate:"required"`
			Form     string `form:"form_name" validate:"required"`
			Untagged string `validate:"required"`
			Skipped  string `json:"-" validate:"required"`
		}

		errs, err := newBareAPI().validate(t.Context(), request{})

		require.NoError(t, err)
		assert.Equal(t, FieldErrors{
			"json_name": {
				{
					Tag: "required",
				},
			},
			"options_name": {
				{
					Tag: "required",
				},
			},
			"form_name": {
				{
					Tag: "required",
				},
			},
			"Untagged": {
				{
					Tag: "required",
				},
			},
			"Skipped": {
				{
					Tag: "required",
				},
			},
		}, errs)
	})

	t.Run("names nested fields by their path", func(t *testing.T) {
		t.Parallel()
		type inner struct {
			Count int `json:"count" validate:"min=1"`
		}
		type embedded struct {
			Name string `json:"name" validate:"required"`
		}
		type request struct {
			embedded
			Inner inner   `json:"inner"`
			Items []inner `json:"items" validate:"dive"`
		}

		errs, err := newBareAPI().validate(t.Context(), request{
			Items: []inner{
				{
					Count: 0,
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, FieldErrors{
			"name": {
				{
					Tag: "required",
				},
			},
			"inner.count": {
				{
					Tag:   "min",
					Param: "1",
				},
			},
			"items[0].count": {
				{
					Tag:   "min",
					Param: "1",
				},
			},
		}, errs)
	})

	t.Run("rejects a value that is not a struct", func(t *testing.T) {
		t.Parallel()
		errs, err := newBareAPI().validate(t.Context(), apitest.Faker(t).Word())

		require.Error(t, err, "a value the validator cannot check is a programming error, not a valid value")
		assert.Nil(t, errs)
	})
}
