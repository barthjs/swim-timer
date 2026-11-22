package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rejection is a change to a valid configuration that Validate must reject with an error naming wantErr.
type rejection struct {
	name    string
	change  func(*Config)
	wantErr string
}

// assertRejects applies each change to the default configuration and checks that Validate rejects it.
func assertRejects(t *testing.T, tests []rejection) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testConfig(t)
			tt.change(&cfg)

			err := Validate(&cfg)

			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// testConfig returns the validated configuration built from the struct-tag defaults alone.
func testConfig(t *testing.T) Config {
	t.Helper()

	cfg, err := Defaults()
	require.NoError(t, err)
	require.NoError(t, Validate(&cfg))
	return cfg
}
